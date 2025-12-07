package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"
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
