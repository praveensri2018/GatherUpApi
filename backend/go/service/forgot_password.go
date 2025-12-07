package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"gatherup/auth"
	"gatherup/repository"
	"math/big"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// -------------------- OTP store (in-memory) --------------------

type otpRecord struct {
	Code      string
	ExpiresAt time.Time
	Verified  bool
}

type OTPStore struct {
	mu       sync.Mutex
	byMobile map[string]otpRecord // key: normalized mobile
}

func NewOTPStore() *OTPStore {
	return &OTPStore{
		byMobile: make(map[string]otpRecord),
	}
}

func (s *OTPStore) Set(mobile, code string, expiry time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byMobile[mobile] = otpRecord{
		Code:      code,
		ExpiresAt: expiry,
		Verified:  false,
	}
}

func (s *OTPStore) Verify(mobile, code string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.byMobile[mobile]
	if !ok {
		return ErrOTPNotFound
	}
	if now.After(rec.ExpiresAt) {
		delete(s.byMobile, mobile)
		return ErrOTPExpired
	}
	if rec.Code != code {
		return ErrInvalidOTP
	}

	rec.Verified = true
	s.byMobile[mobile] = rec
	return nil
}

func (s *OTPStore) RequireVerified(mobile string, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.byMobile[mobile]
	if !ok {
		return ErrOTPNotFound
	}
	if now.After(rec.ExpiresAt) {
		delete(s.byMobile, mobile)
		return ErrOTPExpired
	}
	if !rec.Verified {
		return ErrOTPNotVerified
	}

	// One-time use – remove after success
	delete(s.byMobile, mobile)
	return nil
}

// -------------------- Fast2SMS client --------------------

type Fast2SMSClient struct {
	apiKey     string
	httpClient *http.Client
}

func NewFast2SMSClient(apiKey string) *Fast2SMSClient {
	return &Fast2SMSClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Fast2SMSClient) SendOTP(ctx context.Context, mobile, otp string) error {
	if c.apiKey == "" {
		return errors.New("fast2sms api key is empty")
	}

	msg := fmt.Sprintf("Your GatherUp OTP is %s. It is valid for 10 minutes.", otp)

	params := url.Values{}
	params.Set("authorization", c.apiKey)
	params.Set("route", "q")
	params.Set("message", msg)
	params.Set("numbers", mobile)
	params.Set("flash", "0")

	endpoint := "https://www.fast2sms.com/dev/bulkV2?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create fast2sms request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call fast2sms: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var body map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&body)
		return fmt.Errorf("fast2sms returned status %d body=%v", resp.StatusCode, body)
	}

	return nil
}

// -------------------- AuthService forgot-password methods --------------------

// helper: random N-digit OTP
func (s *AuthService) generateOTP() (string, error) {
	nDigits := s.cfg.OTPDigits
	if nDigits <= 0 {
		nDigits = 6
	}
	max := int64(1)
	for i := 0; i < nDigits; i++ {
		max *= 10
	}
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", nDigits, n.Int64()), nil
}

// SendForgotOTP validates mobile, checks user exists, stores OTP and sends SMS.
func (s *AuthService) SendForgotOTP(ctx context.Context, mobile string) error {
	mobileNorm := NormalizeMobile(mobile)
	if mobileNorm == "" {
		return errors.New("invalid mobile")
	}

	uid, _, err := s.repo.GetCredentialByIdentifier(ctx, mobileNorm)
	if err != nil {
		return err
	}
	if uid == "" {
		return ErrUserNotFound
	}

	code, err := s.generateOTP()
	if err != nil {
		return err
	}

	expiry := time.Now().UTC().Add(s.cfg.OTPTTL)
	s.otpStore.Set(mobileNorm, code, expiry)

	return s.smsClient.SendOTP(ctx, mobileNorm, code)
}

// VerifyForgotOTP marks the OTP as verified if correct.
func (s *AuthService) VerifyForgotOTP(ctx context.Context, mobile, otp string) error {
	mobileNorm := NormalizeMobile(mobile)
	if mobileNorm == "" {
		return errors.New("invalid mobile")
	}
	return s.otpStore.Verify(mobileNorm, otp, time.Now().UTC())
}

// ResetPasswordAfterOTP requires verified OTP, then updates password hash in DB.
func (s *AuthService) ResetPasswordAfterOTP(ctx context.Context, mobile, newPassword string) error {
	if newPassword == "" {
		return errors.New("password required")
	}
	mobileNorm := NormalizeMobile(mobile)
	if mobileNorm == "" {
		return errors.New("invalid mobile")
	}

	// ensure OTP verified
	if err := s.otpStore.RequireVerified(mobileNorm, time.Now().UTC()); err != nil {
		return err
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
