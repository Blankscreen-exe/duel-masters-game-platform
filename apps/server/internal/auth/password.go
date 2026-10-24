package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Argon2id settings: OWASP's recommended minimum of 19 MiB of memory, two
// passes and one lane, with a 16-byte salt and a 32-byte key.
const (
	argonMemoryKiB uint32 = 19 * 1024
	argonPasses    uint32 = 2
	argonLanes     uint8  = 1
	argonSaltBytes        = 16
	argonKeyBytes  uint32 = 32
)

// timingPassword is hashed once to give VerifyNothing a hash to check against.
const timingPassword = "timing-equalisation-placeholder"

// errMalformedHash is returned for a stored hash this package cannot read.
var errMalformedHash = errors.New("malformed password hash")

// argonParams are the settings a hash was made with, read back from it.
type argonParams struct {
	memoryKiB uint32
	passes    uint32
	lanes     uint8
}

var currentParams = argonParams{memoryKiB: argonMemoryKiB, passes: argonPasses, lanes: argonLanes}

// PasswordHasher hashes and checks passwords with Argon2id. Each hash takes 19
// MiB of memory, so at most a fixed number run at once, and a burst of
// sign-ups or logins waits its turn rather than exhausting the server's
// memory. It is safe for concurrent use.
type PasswordHasher struct {
	slots chan struct{}

	timingOnce sync.Once
	timingHash string
	timingErr  error
}

// NewPasswordHasher returns a hasher running at most concurrency hashes at
// once.
func NewPasswordHasher(concurrency int) *PasswordHasher {
	return &PasswordHasher{slots: make(chan struct{}, max(1, concurrency))}
}

// Hash returns the password's hash in the standard encoded form,
// "$argon2id$v=19$m=19456,t=2,p=1$<salt>$<key>", which records the settings it
// was made with.
func (h *PasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	salt := make([]byte, argonSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("making a password salt: %w", err)
	}

	key, err := h.derive(ctx, password, salt, currentParams, argonKeyBytes)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, currentParams.memoryKiB, currentParams.passes, currentParams.lanes,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify reports whether password matches encoded, using the settings encoded
// records. It returns an error for a hash it cannot read.
func (h *PasswordHasher) Verify(ctx context.Context, encoded, password string) (bool, error) {
	params, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	candidate, err := h.derive(ctx, password, salt, params, uint32(len(key)))
	if err != nil {
		return false, err
	}

	return subtle.ConstantTimeCompare(candidate, key) == 1, nil
}

// VerifyNothing spends as long as a real Verify. Logins naming an unknown
// account call it, so how long a login takes does not reveal whether the
// account exists.
func (h *PasswordHasher) VerifyNothing(ctx context.Context, password string) error {
	h.timingOnce.Do(func() {
		h.timingHash, h.timingErr = h.Hash(ctx, timingPassword)
	})
	if h.timingErr != nil {
		return h.timingErr
	}

	_, err := h.Verify(ctx, h.timingHash, password)
	return err
}

// NeedsRehash reports whether encoded was made with other settings than the
// current ones, so the password should be hashed again after a successful
// login. An unreadable hash needs rehashing too.
func NeedsRehash(encoded string) bool {
	params, _, key, err := decodeHash(encoded)
	return err != nil || params != currentParams || uint32(len(key)) != argonKeyBytes
}

// derive runs Argon2id once a slot is free, or gives up when ctx ends first.
func (h *PasswordHasher) derive(ctx context.Context, password string, salt []byte, params argonParams, keyBytes uint32) ([]byte, error) {
	select {
	case h.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-h.slots }()

	return argon2.IDKey([]byte(password), salt, params.passes, params.memoryKiB, params.lanes, keyBytes), nil
}

// decodeHash reads the settings, salt and key back from an encoded hash.
func decodeHash(encoded string) (params argonParams, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, errMalformedHash
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memoryKiB, &params.passes, &params.lanes); err != nil {
		return argonParams{}, nil, nil, errMalformedHash
	}
	if params.memoryKiB == 0 || params.passes == 0 || params.lanes == 0 {
		return argonParams{}, nil, nil, errMalformedHash
	}

	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil || len(salt) == 0 {
		return argonParams{}, nil, nil, errMalformedHash
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return argonParams{}, nil, nil, errMalformedHash
	}

	return params, salt, key, nil
}
