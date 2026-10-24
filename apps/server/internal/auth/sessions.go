package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

// sessionTokenBytes is the size of a session token: 256 random bits.
const sessionTokenBytes = 32

// SessionDuration is how long a login lasts.
const SessionDuration = 30 * 24 * time.Hour

// Session is a login: the token the browser keeps in its cookie, and when it
// stops working.
type Session struct {
	Token     string
	ExpiresAt time.Time
}

// sessionRecord is a row of the sessions table. ID is the SHA-256 of the
// token, so a leaked database holds no usable tokens.
type sessionRecord struct {
	ID        string `gorm:"primaryKey"`
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

func (sessionRecord) TableName() string { return "sessions" }

// newSessionToken returns a new random token, safe to put in a cookie.
func newSessionToken() (string, error) {
	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("making a session token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// sessionID is what the database stores for a token.
func sessionID(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
