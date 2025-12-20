package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"
	"strconv"
)

type FeedHandler struct {
	repo *repository.SocialRepo
}

func NewFeedHandler(repo *repository.SocialRepo) *FeedHandler {
	return &FeedHandler{repo: repo}
}

// GET /posts/feed/personalized
func (h *FeedHandler) PersonalizedFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, _ := FromContextUserID(ctx)

	cursorStr := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	limit := 10
	if limitStr != "" {
		limit, _ = strconv.Atoi(limitStr)
	}

	var cursor *float64
	if cursorStr != "" {
		val, _ := strconv.ParseFloat(cursorStr, 64)
		cursor = &val
	}

	items, nextCursor, err :=
		h.repo.GetPersonalizedFeed(ctx, userID, cursor, limit)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"items":       items,
		"next_cursor": nextCursor,
	})
}
