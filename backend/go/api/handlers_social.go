package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"

	"github.com/go-chi/chi/v5"
)

type SocialHandler struct {
	repo *repository.SocialRepo
}

func NewSocialHandler(repo *repository.SocialRepo) *SocialHandler {
	return &SocialHandler{repo: repo}
}

/* ---------------- FOLLOW ---------------- */

func (h *SocialHandler) Follow(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := chi.URLParam(r, "id")
	h.repo.FollowUser(r.Context(), userID, targetID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	targetID := chi.URLParam(r, "id")
	h.repo.UnfollowUser(r.Context(), userID, targetID)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- BOOKMARK ---------------- */

func (h *SocialHandler) Bookmark(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")
	h.repo.BookmarkPost(r.Context(), postID, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Unbookmark(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")
	h.repo.UnbookmarkPost(r.Context(), postID, userID)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- BLOCK ---------------- */

func (h *SocialHandler) Block(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		BlockedUserID string  `json:"blocked_user_id"`
		Reason        *string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.repo.BlockUser(r.Context(), userID, req.BlockedUserID, req.Reason)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		BlockedUserID string `json:"blocked_user_id"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.repo.UnblockUser(r.Context(), userID, req.BlockedUserID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *SocialHandler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	data, _ := h.repo.ListBookmarkedPosts(r.Context(), userID)
	json.NewEncoder(w).Encode(data)
}

func (h *SocialHandler) ListFollowers(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	data, _ := h.repo.ListFollowers(r.Context(), userID)
	json.NewEncoder(w).Encode(data)
}

func (h *SocialHandler) ListFollowing(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")
	data, _ := h.repo.ListFollowing(r.Context(), userID)
	json.NewEncoder(w).Encode(data)
}

func (h *SocialHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	data, _ := h.repo.ListBlockedUsers(r.Context(), userID)
	json.NewEncoder(w).Encode(data)
}
