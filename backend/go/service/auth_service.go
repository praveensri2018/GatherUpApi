/* Place: backend/go/service/auth_service.go */
package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gatherup/auth"
	"gatherup/repository"
)

// Exported config so other packages (cmd/server) can construct it.
type AuthConfig struct {
	BcryptCost        int
	RefreshTokenBytes int
	RefreshTTL        time.Duration
}

type AuthService struct {
	repo       *repository.UserRepo
	jwtManager *auth.JWTManager
	cfg        *AuthConfig
}

func NewAuthService(repo *repository.UserRepo, jwtMgr *auth.JWTManager, cfg *AuthConfig) *AuthService {
	return &AuthService{repo: repo, jwtManager: jwtMgr, cfg: cfg}
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrRefreshTokenNotFound = errors.New("refresh token not found or revoked/expired")

// NormalizeMobile removes non-digit characters except leading +.
// Keep this small helper here for phase-1; consider moving to a shared util package later.
func NormalizeMobile(m string) string {
	if m == "" {
		return ""
	}
	m = strings.TrimSpace(m)
	var sb strings.Builder
	for i, r := range m {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		} else if r == '+' && i == 0 {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Register creates a user with hashed password using user_credentials (not on users table).
func (s *AuthService) Register(ctx context.Context, mobile, password string) (string, error) {
	if mobile == "" || password == "" {
		return "", errors.New("mobile and password are required")
	}
	mobileNorm := NormalizeMobile(mobile)

	// check existing credential (by normalized mobile)
	uid, _, err := s.repo.GetCredentialByIdentifier(ctx, mobileNorm)
	if err != nil {
		return "", err
	}
	if uid != "" {
		return "", errors.New("user already exists")
	}

	phash, err := auth.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return "", err
	}

	// create user row and credential row in a single transaction
	userID, err := s.repo.CreateUserWithPassword(ctx, mobile, mobileNorm, phash)
	if err != nil {
		return "", err
	}
	return userID, nil
}

// Login verifies credentials and issues tokens.
func (s *AuthService) Login(ctx context.Context, mobile, password, deviceInfo string) (accessToken string, accessExp time.Time, refreshRaw string, refreshExpiry time.Time, err error) {
	if mobile == "" || password == "" {
		err = ErrInvalidCredentials
		return
	}
	mobileNorm := NormalizeMobile(mobile)

	userID, pwHash, err := s.repo.GetCredentialByIdentifier(ctx, mobileNorm)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	if userID == "" {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}

	if err := auth.ComparePassword(pwHash, password); err != nil {
		return "", time.Time{}, "", time.Time{}, ErrInvalidCredentials
	}

	accessToken, accessExp, err = s.jwtManager.Generate(userID, nil)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}

	raw, hash, err := auth.GenerateRefreshToken(s.cfg.RefreshTokenBytes)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	expiry := auth.RefreshTokenExpiry(s.cfg.RefreshTTL)
	if _, err := s.repo.SaveRefreshToken(ctx, userID, hash, &deviceInfo, expiry); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	return accessToken, accessExp, raw, expiry, nil
}

// Refresh validates provided refresh token, rotates it and returns new tokens.
func (s *AuthService) Refresh(ctx context.Context, raw string) (newAccess string, accessExp time.Time, newRaw string, newExpiry time.Time, err error) {
	if raw == "" {
		return "", time.Time{}, "", time.Time{}, ErrRefreshTokenNotFound
	}
	hash := auth.HashRefreshToken(raw)
	row, err := s.repo.GetRefreshTokenRow(ctx, hash)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	if row == nil || row.Revoked {
		return "", time.Time{}, "", time.Time{}, ErrRefreshTokenNotFound
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return "", time.Time{}, "", time.Time{}, ErrRefreshTokenNotFound
	}
	newAccess, accessExp, err = s.jwtManager.Generate(row.UserID, nil)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	newRaw, newHash, err := auth.GenerateRefreshToken(s.cfg.RefreshTokenBytes)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	newExpiry = auth.RefreshTokenExpiry(s.cfg.RefreshTTL)
	if _, err := s.repo.RotateRefreshToken(ctx, row.ID, newHash, newExpiry); err != nil {
		return "", time.Time{}, "", time.Time{}, err
	}
	return newAccess, accessExp, newRaw, newExpiry, nil
}
