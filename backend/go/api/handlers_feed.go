package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"
)

type FeedHandler struct {
	repo *repository.SocialRepo
}

func NewFeedHandler(repo *repository.SocialRepo) *FeedHandler {
	return &FeedHandler{repo: repo}
}

// GET /posts/feed/personalized
func (h *FeedHandler) PersonalizedFeed(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	feed, err := h.repo.GetPersonalizedFeed(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(feed)
}
