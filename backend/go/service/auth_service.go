package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"gatherup/auth"
	"gatherup/repository"
)

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

// 👇 NEW: exported sentinel for duplicate user
var ErrUserAlreadyExists = errors.New("user already exists")

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

func (s *AuthService) Register(ctx context.Context, mobile, password string) (string, error) {
	if mobile == "" || password == "" {
		return "", errors.New("mobile and password are required")
	}
	mobileNorm := NormalizeMobile(mobile)

	uid, _, err := s.repo.GetCredentialByIdentifier(ctx, mobileNorm)
	if err != nil {
		return "", err
	}
	if uid != "" {
		// 🔴 use sentinel instead of new error every time
		return "", ErrUserAlreadyExists
	}

	phash, err := auth.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return "", err
	}

	userID, err := s.repo.CreateUserWithPassword(ctx, mobile, mobileNorm, phash)
	if err != nil {
		return "", err
	}
	return userID, nil
}

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
	// mark last_used and then rotate
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

func (s *AuthService) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return ErrRefreshTokenNotFound
	}
	hash := auth.HashRefreshToken(raw)
	row, err := s.repo.GetRefreshTokenRow(ctx, hash)
	if err != nil {
		return err
	}
	if row == nil {
		return ErrRefreshTokenNotFound
	}
	return s.repo.RevokeRefreshToken(ctx, row.ID)
}
