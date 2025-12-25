package service

import (
	"context"
	"errors"
	"gatherup/auth"
	"gatherup/repository"
	"strings"
	"time"
)

type AuthConfig struct {
	BcryptCost        int
	RefreshTokenBytes int
	RefreshTTL        time.Duration
	OTPDigits         int
	OTPTTL            time.Duration
}

type SMSClient interface {
	SendOTP(ctx context.Context, mobile, otp string) error
}

type AuthService struct {
	repo       *repository.UserRepo
	jwtManager *auth.JWTManager
	cfg        *AuthConfig
	smsClient  SMSClient
	otpStore   *OTPStore
}

func NewAuthService(repo *repository.UserRepo, jwtMgr *auth.JWTManager, cfg *AuthConfig, smsClient SMSClient) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtManager: jwtMgr,
		cfg:        cfg,
		smsClient:  smsClient,
		otpStore:   NewOTPStore(),
	}
}

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrRefreshTokenNotFound = errors.New("refresh token not found or revoked/expired")
var ErrUserAlreadyExists = errors.New("user already exists")
var ErrUsernameAlreadyExists = errors.New("username already exists")

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrInvalidOTP     = errors.New("invalid otp")
	ErrOTPExpired     = errors.New("otp expired")
	ErrOTPNotFound    = errors.New("otp not found")
	ErrOTPNotVerified = errors.New("otp not verified")
)

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

func (s *AuthService) ResetPassword(ctx context.Context, mobile, newPassword string) error {
	if newPassword == "" {
		return errors.New("password required")
	}
	mobileNorm := NormalizeMobile(mobile)
	if mobileNorm == "" {
		return errors.New("invalid mobile")
	}
	phash, err := auth.HashPassword(newPassword, s.cfg.BcryptCost)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordByIdentifier(ctx, mobileNorm, phash); err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	return nil
}

func (s *AuthService) Register(
	ctx context.Context,
	mobile string,
	password string,
	username string,
	fullName string,
	deviceInfo string,
) (
	accessToken string,
	accessExp time.Time,
	refreshRaw string,
	refreshExpiry time.Time,
	err error,
) {

	if mobile == "" || password == "" || username == "" || fullName == "" {
		err = errors.New("mobile, password, username and fullname are required")
		return
	}
	mobileNorm := NormalizeMobile(mobile)
	uid, _, err := s.repo.GetCredentialByIdentifier(ctx, mobileNorm)
	if err != nil {
		return
	}
	if uid != "" {
		err = ErrUserAlreadyExists
		return
	}
	taken, err := s.repo.IsUsernameTaken(ctx, username)
	if err != nil {
		return
	}
	if taken {
		err = ErrUsernameAlreadyExists
		return
	}
	phash, err := auth.HashPassword(password, s.cfg.BcryptCost)
	if err != nil {
		return
	}
	userID, err := s.repo.CreateUserWithPassword(
		ctx,
		mobile,
		mobileNorm,
		username,
		fullName,
		phash,
	)
	if err != nil {
		return
	}
	accessToken, accessExp, err = s.jwtManager.Generate(userID, nil)
	if err != nil {
		return
	}
	raw, hash, err := auth.GenerateRefreshToken(s.cfg.RefreshTokenBytes)
	if err != nil {
		return
	}
	refreshExpiry = auth.RefreshTokenExpiry(s.cfg.RefreshTTL)

	var devicePtr *string
	if deviceInfo != "" {
		devicePtr = &deviceInfo
	}

	if _, err = s.repo.SaveRefreshToken(
		ctx,
		userID,
		hash,
		devicePtr,
		refreshExpiry,
	); err != nil {
		return
	}

	refreshRaw = raw
	return
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
