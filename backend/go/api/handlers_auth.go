/* Place: backend/go/api/handlers_auth.go */
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"gatherup/service"
)

var mobileRe = regexp.MustCompile(`^\+?[0-9]{7,15}$`)

// Request/response DTOs (mobile-first)
type registerReq struct {
	MobileNumber string `json:"mobile_number"`
	Password     string `json:"password"`
}

type loginReq struct {
	MobileNumber string `json:"mobile_number"`
	Password     string `json:"password"`
	DeviceInfo   string `json:"device_info,omitempty"`
}

type tokenResp struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AuthHandler wraps AuthService
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// set a per-request timeout for slow DB calls
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

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
		// log the error server-side (assumes a logger available); return safe message
		ErrorJSON(w, http.StatusBadRequest, "register failed")
		return
	}
	JSON(w, http.StatusCreated, map[string]string{"id": id})
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.MobileNumber == "" || req.Password == "" {
		http.Error(w, "mobile_number and password required", http.StatusBadRequest)
		return
	}
	if !mobileRe.MatchString(req.MobileNumber) {
		http.Error(w, "invalid mobile_number format", http.StatusBadRequest)
		return
	}
	access, accessExp, refreshRaw, _, err := h.svc.Login(r.Context(), req.MobileNumber, req.Password, req.DeviceInfo)
	if err != nil {
		http.Error(w, "login failed: "+err.Error(), http.StatusUnauthorized)
		return
	}
	resp := tokenResp{AccessToken: access, RefreshToken: refreshRaw, ExpiresAt: accessExp}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.RefreshToken == "" {
		http.Error(w, "refresh_token required", http.StatusBadRequest)
		return
	}
	newAccess, accessExp, newRefreshRaw, _, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		http.Error(w, "refresh failed: "+err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tokenResp{AccessToken: newAccess, RefreshToken: newRefreshRaw, ExpiresAt: accessExp})
}
