/* Place: backend/go/auth/tokens.go */
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"time"
)

// GenerateRefreshToken returns raw token and its hash (base64 url encoded).
func GenerateRefreshToken(n int) (raw string, hash string, err error) {
	b := make([]byte, n)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = base64.RawURLEncoding.EncodeToString(h[:])
	return raw, hash, nil
}

// HashRefreshToken computes hash for a given raw token.
func HashRefreshToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(h[:])
}

// RefreshTokenExpiry returns expiry time for ttl.
func RefreshTokenExpiry(ttl time.Duration) time.Time {
	return time.Now().UTC().Add(ttl)
}
