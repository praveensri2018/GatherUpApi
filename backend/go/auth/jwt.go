/* Place: backend/go/auth/jwt.go */
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret []byte
	ttl    time.Duration
}

type Claims struct {
	UserID string   `json:"sub"`
	Roles  []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

// NewJWTManager creates JWT manager with HMAC secret and TTL.
func NewJWTManager(secret string, ttl time.Duration) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// Generate creates a signed JWT string and returns expiry.
func (m *JWTManager) Generate(userID string, roles []string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.ttl)
	claims := Claims{
		UserID: userID,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Subject:   userID,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := tok.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return ss, exp, nil
}

// Verify parses and validates a token and returns claims.
func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}
