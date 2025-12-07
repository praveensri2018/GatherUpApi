package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"

	"gatherup/service"
)

var mobileRe = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

type registerReq struct {
	MobileNumber string `json:"mobile_number"`
	Password     string `json:"password"`
}

type loginReq struct {
	MobileNumber string `json:"mobile_number"`
	Password     string `json:"password"`
	DeviceInfo   string `json:"device_info,omitempty"`
}

type revokeReq struct {
	RefreshToken string `json:"refresh_token"`
}

type tokenResp struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type simpleReq struct {
	MobileOrEmail string `json:"mobile_or_email,omitempty"`
	OTP           string `json:"otp,omitempty"`
	Password      string `json:"password,omitempty"`
	NewPassword   string `json:"new_password,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	Token         string `json:"token,omitempty"`
}

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.MobileNumber == "" || req.Password == "" {
		ErrorJSON(w, http.StatusBadRequest, "mobile_number and password required")
		return
	}
	if !mobileRe.MatchString(req.MobileNumber) {
		ErrorJSON(w, http.StatusBadRequest, "invalid mobile_number format")
		return
	}

	id, err := h.svc.Register(ctx, req.MobileNumber, req.Password)
	if err != nil {
		// return generic message to client
		ErrorJSON(w, http.StatusBadRequest, "register failed")
		return
	}
	JSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.MobileNumber == "" || req.Password == "" {
		ErrorJSON(w, http.StatusBadRequest, "mobile_number and password required")
		return
	}
	if !mobileRe.MatchString(req.MobileNumber) {
		ErrorJSON(w, http.StatusBadRequest, "invalid mobile_number format")
		return
	}
	access, accessExp, refreshRaw, _, err := h.svc.Login(ctx, req.MobileNumber, req.Password, req.DeviceInfo)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			ErrorJSON(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "login failed")
		return
	}
	JSON(w, http.StatusOK, tokenResp{AccessToken: access, RefreshToken: refreshRaw, ExpiresAt: accessExp})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req revokeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.RefreshToken == "" {
		ErrorJSON(w, http.StatusBadRequest, "refresh_token required")
		return
	}
	newAccess, accessExp, newRefreshRaw, _, err := h.svc.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrRefreshTokenNotFound) {
			ErrorJSON(w, http.StatusUnauthorized, "refresh token invalid or expired")
			return
		}
		ErrorJSON(w, http.StatusInternalServerError, "refresh failed")
		return
	}
	JSON(w, http.StatusOK, tokenResp{AccessToken: newAccess, RefreshToken: newRefreshRaw, ExpiresAt: accessExp})
}

func (h *AuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req revokeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.RefreshToken == "" {
		ErrorJSON(w, http.StatusBadRequest, "refresh_token required")
		return
	}
	if err := h.svc.Revoke(ctx, req.RefreshToken); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "revoke failed")
		return
	}
	JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
