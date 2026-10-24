package auth

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/argon2"
)

func TestHashUsesArgon2idWithTheOWASPSettings(t *testing.T) {
	encoded, err := NewPasswordHasher(1).Hash(context.Background(), "correct horse battery staple")
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$"), encoded)
	assert.False(t, NeedsRehash(encoded))
}

func TestHashSaltsEveryPassword(t *testing.T) {
	hasher := NewPasswordHasher(1)

	first, err := hasher.Hash(context.Background(), "same password")
	require.NoError(t, err)
	second, err := hasher.Hash(context.Background(), "same password")
	require.NoError(t, err)

	assert.NotEqual(t, first, second)
}

func TestVerifyAcceptsOnlyTheRightPassword(t *testing.T) {
	ctx := context.Background()
	hasher := NewPasswordHasher(1)
	encoded, err := hasher.Hash(ctx, "correct horse battery staple")
	require.NoError(t, err)

	ok, err := hasher.Verify(ctx, encoded, "correct horse battery staple")
	require.NoError(t, err)
	assert.True(t, ok)

	for _, wrong := range []string{"", "correct horse battery stapl", "Correct horse battery staple", "correct horse battery staple "} {
		ok, err := hasher.Verify(ctx, encoded, wrong)
		require.NoError(t, err)
		assert.False(t, ok, "%q", wrong)
	}
}

func TestVerifyReadsTheSettingsFromTheHash(t *testing.T) {
	// A hash made with other, older settings still verifies, and is flagged
	// for rehashing.
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("old password"), salt, 1, 8*1024, 1, 32)
	old := "$argon2id$v=19$m=8192,t=1,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)

	ok, err := NewPasswordHasher(1).Verify(context.Background(), old, "old password")
	require.NoError(t, err)
	assert.True(t, ok)
	assert.True(t, NeedsRehash(old))
}

func TestVerifyRefusesMalformedHashes(t *testing.T) {
	hasher := NewPasswordHasher(1)

	for _, malformed := range []string{
		"",
		"plain text",
		"$argon2i$v=19$m=19456,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=18$m=19456,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=0,t=2,p=1$c2FsdA$a2V5",
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$a2V5",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA$",
		"$argon2id$v=19$m=19456,t=2,p=1$c2FsdA",
	} {
		ok, err := hasher.Verify(context.Background(), malformed, "anything")
		assert.ErrorIs(t, err, errMalformedHash, "%q", malformed)
		assert.False(t, ok)
		assert.True(t, NeedsRehash(malformed))
	}
}

func TestVerifyNothingTakesAsLongAsAVerify(t *testing.T) {
	ctx := context.Background()
	hasher := NewPasswordHasher(1)
	encoded, err := hasher.Hash(ctx, "correct horse battery staple")
	require.NoError(t, err)
	require.NoError(t, hasher.VerifyNothing(ctx, "warm up"))

	started := time.Now()
	_, err = hasher.Verify(ctx, encoded, "guess")
	require.NoError(t, err)
	verify := time.Since(started)

	started = time.Now()
	require.NoError(t, hasher.VerifyNothing(ctx, "guess"))
	nothing := time.Since(started)

	// Both run one Argon2id derivation. Timings vary with machine load, so
	// only a comparable amount of work is checked, not equal times.
	assert.Greater(t, nothing, verify/4, "VerifyNothing does the work of a real verification")
}

func TestHashingWaitsForAFreeSlot(t *testing.T) {
	hasher := NewPasswordHasher(1)
	hasher.slots <- struct{}{}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := hasher.Hash(ctx, "password")
	assert.ErrorIs(t, err, context.DeadlineExceeded, "with every slot taken, hashing waits and gives up with the context")

	<-hasher.slots
	_, err = hasher.Hash(context.Background(), "password")
	assert.NoError(t, err, "a freed slot is used")
}
