package api

import (
	"encoding/json"
	"net/http"

	"gatherup/repository"

	"github.com/go-chi/chi/v5"
)

type CommentHandler struct {
	repo *repository.SocialRepo
}

func NewCommentHandler(repo *repository.SocialRepo) *CommentHandler {
	return &CommentHandler{repo: repo}
}

/* ---------------- COMMENTS ---------------- */

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	comments, err := h.repo.ListComments(r.Context(), postID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(comments)
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")

	var req struct {
		Body string `json:"body"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	h.repo.CreateComment(r.Context(), postID, userID, req.Body)
	w.WriteHeader(http.StatusCreated)
}

func (h *CommentHandler) Update(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")

	var req struct {
		Body string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.repo.UpdateComment(r.Context(), commentID, req.Body)
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "id")

	h.repo.DeleteComment(r.Context(), commentID)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- COMMENT REACTIONS ---------------- */

func (h *CommentHandler) React(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	commentID := chi.URLParam(r, "id")

	h.repo.ReactToComment(r.Context(), commentID, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommentHandler) Unreact(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	commentID := chi.URLParam(r, "id")

	h.repo.UnreactComment(r.Context(), commentID, userID)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- COMMENT REPORT ---------------- */

func (h *CommentHandler) Report(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	commentID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.repo.Report(r.Context(), userID, "comment", commentID, req.Reason)
	w.WriteHeader(http.StatusNoContent)
}
