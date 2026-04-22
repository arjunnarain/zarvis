package chat

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/zarvis/internal/auth"
)

type createSessionReq struct {
	PrimaryAnimal string `json:"primary_animal"`
}

// CreateSession creates a new session for the authenticated user.
func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	userID := auth.GetUserID(r)
	sess, err := h.Store.CreateSession(userID, req.PrimaryAnimal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

// GetSession returns a session by ID.
func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.Store.GetSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

type updateSessionReq struct {
	PrimaryAnimal string `json:"primary_animal"`
}

// UpdateSession modifies session properties.
func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.Store.GetSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var req updateSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.PrimaryAnimal != "" {
		sess.PrimaryAnimal = req.PrimaryAnimal
	}
	if err := h.Store.UpdateSession(sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// GetBadges returns badges earned by a session.
func (h *Handler) GetBadges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.Store.GetBadges(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, b)
}
