// Package secrets encrypts what must be readable again but must not sit in the
// database in the clear, such as the password of the mail server. The key comes
// from outside the database (APP_SECRET_KEY), so a stolen database or backup
// does not give the secrets away.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// KeySize is the size of a key: AES-256.
const KeySize = 32

// sealedPrefix marks the format of a sealed value, so a later format can be told
// apart from this one.
const sealedPrefix = "v1."

// ErrUnreadable is returned for a sealed value that cannot be opened: it was
// sealed with another key, under another purpose, or has been tampered with.
var ErrUnreadable = errors.New("the secret cannot be read with this key")

// ParseKey reads a key written in base64, as `openssl rand -base64 32` makes.
func ParseKey(text string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
	if err != nil {
		return nil, errors.New("must be written in base64")
	}
	if len(key) != KeySize {
		return nil, fmt.Errorf("must be %d bytes, got %d", KeySize, len(key))
	}

	return key, nil
}

// Box seals and opens secrets with one key. It is safe for concurrent use.
type Box struct {
	aead cipher.AEAD
}

// NewBox returns a box using a key of KeySize bytes.
func NewBox(key []byte) (*Box, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("a key must be %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("making the cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("making the cipher: %w", err)
	}

	return &Box{aead: aead}, nil
}

// Seal encrypts a secret for storing. The purpose says what it is for: a value
// sealed for one purpose cannot be opened for another, so a stored value cannot
// be moved to where something else would read it.
func (b *Box) Seal(secret []byte, purpose string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("making a nonce: %w", err)
	}

	sealed := b.aead.Seal(nonce, nonce, secret, []byte(purpose))
	return sealedPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

// Open decrypts what Seal made for the same purpose. It returns ErrUnreadable
// when that is not possible.
func (b *Box) Open(sealed, purpose string) ([]byte, error) {
	encoded, ok := strings.CutPrefix(sealed, sealedPrefix)
	if !ok {
		return nil, ErrUnreadable
	}
	raw, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(raw) < b.aead.NonceSize() {
		return nil, ErrUnreadable
	}

	nonce, ciphertext := raw[:b.aead.NonceSize()], raw[b.aead.NonceSize():]
	secret, err := b.aead.Open(nil, nonce, ciphertext, []byte(purpose))
	if err != nil {
		return nil, ErrUnreadable
	}

	return secret, nil
}
