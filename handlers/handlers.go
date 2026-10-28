// Package handlers serves the bookmarks HTTP API.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"bookmarks/store"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxTitleLength  = 200
)

// Handler serves the API from a store.
type Handler struct {
	store *store.Store
}

// New returns a Handler backed by s.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Routes returns the API's routes.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", h.health)
	mux.HandleFunc("GET /bookmarks", h.list)
	mux.HandleFunc("POST /bookmarks", h.create)
	mux.HandleFunc("GET /bookmarks/{id}", h.get)
	mux.HandleFunc("PUT /bookmarks/{id}", h.update)
	mux.HandleFunc("DELETE /bookmarks/{id}", h.delete)
	return mux
}

type bookmarkRequest struct {
	URL   string   `json:"url"`
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "bookmarks": h.store.Count()})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if tag := q.Get("tag"); tag != "" {
		writeJSON(w, http.StatusOK, h.store.ByTag(tag))
		return
	}

	page, err := intParam(q, "page", 1)
	if err != nil || page < 1 {
		writeError(w, http.StatusBadRequest, "page must be a positive integer")
		return
	}
	size, err := intParam(q, "size", defaultPageSize)
	if err != nil || size < 1 || size > maxPageSize {
		writeError(w, http.StatusBadRequest, "size must be between 1 and 100")
		return
	}

	writeJSON(w, http.StatusOK, h.store.List(page, size))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBookmark(w, r)
	if !ok {
		return
	}

	b, err := h.store.Create(store.Bookmark{URL: req.URL, Title: req.Title, Tags: req.Tags})
	if err != nil {
		log.Printf("create bookmark: %v", err)
		writeError(w, http.StatusInternalServerError, "could not save bookmark")
		return
	}
	w.Header().Set("Location", "/bookmarks/"+strconv.Itoa(b.ID))
	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	b, err := h.store.Get(id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	req, ok := decodeBookmark(w, r)
	if !ok {
		return
	}

	b, err := h.store.Update(id, store.Bookmark{URL: req.URL, Title: req.Title, Tags: req.Tags})
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "bookmark not found")
	case err != nil:
		log.Printf("update bookmark %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "could not save bookmark")
	default:
		writeJSON(w, http.StatusOK, b)
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	if err := h.store.Delete(id); errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "bookmark not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeBookmark reads and validates a bookmark from the request body. On failure it writes the
// error response and returns false.
func decodeBookmark(w http.ResponseWriter, r *http.Request) (bookmarkRequest, bool) {
	var req bookmarkRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return req, false
	}

	req.Title = strings.TrimSpace(req.Title)
	if msg := validate(req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return req, false
	}
	return req, true
}

func validate(req bookmarkRequest) string {
	u, err := url.Parse(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "url must be an absolute http or https URL"
	}
	if req.Title == "" {
		return "title is required"
	}
	if len(req.Title) > maxTitleLength {
		return "title must be at most 200 characters"
	}
	return ""
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func intParam(q url.Values, name string, def int) (int, error) {
	v := q.Get(name)
	if v == "" {
		return def, nil
	}
	return strconv.Atoi(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
