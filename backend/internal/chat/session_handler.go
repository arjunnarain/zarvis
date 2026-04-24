package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/zarvis/internal/analyze"
	"github.com/zarvis/internal/auth"
)

type createSessionReq struct {
	PrimaryAnimal string `json:"primary_animal"`
}

// GetUserSessions returns all sessions for the authenticated user, most recent first.
func (h *Handler) GetUserSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	sessions, err := h.Store.ListSessionsByUser(userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if sessions == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	writeJSON(w, http.StatusOK, sessions)
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

// GetTabs returns which tabs should be enabled based on document state.
func (h *Handler) GetTabs(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	hasDoc := false
	rawContent := ""
	structuredJSON := ""
	schemaJSON := ""

	doc, err := h.Store.GetLatestDocumentByUser(userID)
	if err == nil {
		hasDoc = true
		rawContent = doc.RawContent
		structuredJSON = doc.StructuredJSON
		schemaJSON = doc.SchemaJSON
	}

	forestDocCount := 0
	forests, _ := h.Store.ListForestsByUser(userID)
	for _, f := range forests {
		if f.DocCount > forestDocCount {
			forestDocCount = f.DocCount
		}
	}

	tabs := analyze.TabRecommendations(hasDoc, rawContent, structuredJSON, schemaJSON, forestDocCount)
	writeJSON(w, http.StatusOK, tabs)
}

// GetQuality returns the data quality score for the latest document.
func (h *Handler) GetQuality(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	doc, err := h.Store.GetLatestDocumentByUser(userID)
	if err != nil {
		http.Error(w, "no document", http.StatusNotFound)
		return
	}
	quality := analyze.ComputeQualityScore(doc.RawContent, doc.StructuredJSON)
	writeJSON(w, http.StatusOK, quality)
}

// SearchDocument does a text search across raw and structured document content.
func (h *Handler) SearchDocument(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	doc, err := h.Store.GetLatestDocumentByUser(userID)
	if err != nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	// Search both raw and structured content
	var results []map[string]string
	queryLower := strings.ToLower(query)

	// Search raw content line by line
	for i, line := range strings.Split(doc.RawContent, "\n") {
		if strings.Contains(strings.ToLower(line), queryLower) {
			results = append(results, map[string]string{
				"source": "raw",
				"line":   fmt.Sprintf("%d", i+1),
				"text":   strings.TrimSpace(line),
			})
		}
	}

	// Search structured JSON
	if doc.StructuredJSON != "" {
		for i, line := range strings.Split(doc.StructuredJSON, "\n") {
			if strings.Contains(strings.ToLower(line), queryLower) {
				results = append(results, map[string]string{
					"source": "structured",
					"line":   fmt.Sprintf("%d", i+1),
					"text":   strings.TrimSpace(line),
				})
			}
		}
	}

	// Limit results
	if len(results) > 20 {
		results = results[:20]
	}
	writeJSON(w, http.StatusOK, results)
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
