package chat

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/zarvis/internal/state"
)

// CreateForest creates a new document collection.
func (h *Handler) CreateForest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	f, err := h.Store.CreateForest(req.SessionID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

// ListForests returns all forests for a session.
func (h *Handler) ListForests(w http.ResponseWriter, r *http.Request) {
	sid := chi.URLParam(r, "id")
	forests, err := h.Store.ListForests(sid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if forests == nil {
		forests = []state.Forest{}
	}
	writeJSON(w, http.StatusOK, forests)
}

// AddDocToForest links a document to a forest and auto-indexes for BM25 search.
func (h *Handler) AddDocToForest(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		DocumentID int64 `json:"document_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DocumentID == 0 {
		http.Error(w, "document_id required", http.StatusBadRequest)
		return
	}
	if err := h.Store.AddDocumentToForest(forestID, req.DocumentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-index document content for BM25 search
	if h.Search != nil {
		doc, err := h.Store.GetDocument(req.DocumentID)
		if err == nil {
			content := doc.RawContent
			if doc.StructuredJSON != "" {
				content = doc.StructuredJSON
			}
			_ = h.Search.IndexDocument(req.DocumentID, forestID, content)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

// ClearForest removes all documents and search index from a forest.
func (h *Handler) ClearForest(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.ClearForest(forestID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

// GetForestDocs returns all documents in a forest.
func (h *Handler) GetForestDocs(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	docs, err := h.Store.GetForestDocuments(forestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if docs == nil {
		docs = []state.Document{}
	}
	writeJSON(w, http.StatusOK, docs)
}
