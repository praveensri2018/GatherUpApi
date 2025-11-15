/* Place: backend/go/api/handlers_user.go */
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

// GET /api/me
func (h *UserHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok || userID == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	u, err := h.repo.GetByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to fetch user", http.StatusInternalServerError)
		return
	}
	if u == nil {
		http.Error(w, "user not found", http.StatusNotFound)
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
