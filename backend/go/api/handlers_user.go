package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	repo *repository.UserRepo
}

func NewUserHandler(repo *repository.UserRepo) *UserHandler {
	return &UserHandler{repo: repo}
}

func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok || userID == "" {
		ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	u, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	if u == nil {
		ErrorJSON(w, http.StatusNotFound, "user not found")
		return
	}
	resp := map[string]interface{}{
		"id":            u.ID,
		"mobile_number": u.MobileNumber,
		"email":         u.Email,
		"username":      u.Username,
		"created_at":    u.CreatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	if userID == "" {
		ErrorJSON(w, http.StatusBadRequest, "invalid user id")
		return
	}

	profile, err := h.repo.GetProfileByID(r.Context(), userID)
	if err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to fetch profile")
		return
	}
	if profile == nil {
		ErrorJSON(w, http.StatusNotFound, "user not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		DisplayName *string `json:"display_name"`
		Username    *string `json:"username"`
		Bio         *string `json:"bio"`
		Email       *string `json:"email"`
		Gender      *string `json:"gender"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid payload")
		return
	}

	repoReq := repository.UpdateProfileRequest{
		DisplayName: req.DisplayName,
		Username:    req.Username,
		Bio:         req.Bio,
		Email:       req.Email,
		Gender:      req.Gender,
	}

	if err := h.repo.UpdateProfile(r.Context(), userID, repoReq); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to update profile")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *UserHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		ErrorJSON(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		AvatarURL string `json:"avatar_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AvatarURL == "" {
		ErrorJSON(w, http.StatusBadRequest, "invalid avatar url")
		return
	}

	if err := h.repo.UpdateAvatar(r.Context(), userID, req.AvatarURL); err != nil {
		ErrorJSON(w, http.StatusInternalServerError, "failed to update avatar")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
