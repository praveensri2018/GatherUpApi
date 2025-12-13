package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gatherup/repository"
	"gatherup/storage"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type PostHandler struct {
	repo *repository.SocialRepo
	r2   *storage.R2Client
}

func NewPostHandler(repo *repository.SocialRepo, r2 *storage.R2Client) *PostHandler {
	return &PostHandler{repo: repo, r2: r2}
}

/* ---------------- POSTS ---------------- */

func (h *PostHandler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	_, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")

	// Max 50MB
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		http.Error(w, "invalid multipart data", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	mediaType := r.FormValue("media_type")
	if mediaType == "" {
		mediaType = "image"
	}

	sortOrder := 0
	if v := r.FormValue("sort_order"); v != "" {
		fmt.Sscan(v, &sortOrder)
	}

	objectKey := fmt.Sprintf(
		"posts/%s/%s_%s",
		postID,
		uuid.New().String(),
		header.Filename,
	)

	mediaURL, err := h.r2.Upload(
		r.Context(),
		objectKey,
		file,
		header.Size,
		header.Header.Get("Content-Type"),
	)
	if err != nil {
		http.Error(w, "upload failed", http.StatusInternalServerError)
		return
	}

	err = h.repo.AddPostMedia(
		r.Context(),
		postID,
		mediaURL,
		mediaType,
		header.Size,
		sortOrder,
	)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"media_url":  mediaURL,
		"media_type": mediaType,
		"file_size":  header.Size,
	})
}

// POST /posts
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Kind  string `json:"kind"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	postID, err := h.repo.CreatePost(r.Context(), userID, req.Title, req.Body, req.Kind)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"id": postID})
}

// GET /posts
func (h *PostHandler) List(w http.ResponseWriter, r *http.Request) {
	posts, err := h.repo.ListPosts(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(posts)
}

// GET /posts/{id}
func (h *PostHandler) Get(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	post, err := h.repo.GetPostByID(r.Context(), postID)
	if err != nil {
		http.Error(w, "post not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(post)
}

// PATCH /posts/{id}
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	if err := h.repo.UpdatePost(r.Context(), postID, req.Title, req.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /posts/{id}
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	postID := chi.URLParam(r, "id")

	if err := h.repo.DeletePost(r.Context(), postID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- POST REACTIONS ---------------- */

func (h *PostHandler) React(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")
	h.repo.ReactToPost(r.Context(), postID, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *PostHandler) Unreact(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")
	h.repo.UnreactPost(r.Context(), postID, userID)
	w.WriteHeader(http.StatusNoContent)
}

/* ---------------- POST REPORT ---------------- */

func (h *PostHandler) Report(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	postID := chi.URLParam(r, "id")

	var req struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	h.repo.Report(r.Context(), userID, "post", postID, req.Reason)
	w.WriteHeader(http.StatusNoContent)
}

func (h *PostHandler) ListMyPosts(w http.ResponseWriter, r *http.Request) {
	userID, ok := FromContextUserID(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	posts, err := h.repo.ListPostsByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(posts)
}
