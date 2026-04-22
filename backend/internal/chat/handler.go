// Package chat contains HTTP handlers for the Zarvis API.
package chat

import (
	"encoding/json"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/zarvis/internal/badges"
	"github.com/zarvis/internal/mcp"
	"github.com/zarvis/internal/prompt"
	"github.com/zarvis/internal/search"
	"github.com/zarvis/internal/state"
	"github.com/zarvis/internal/tools"
)

// Handler holds all dependencies shared across API endpoints.
type Handler struct {
	Anthropic *anthropic.Client
	Store     state.Store
	Registry  *mcp.Registry
	Search    *search.Engine
	Prompt    *prompt.Builder
	Badges    *badges.Engine
	Tools     *tools.Executor
	JWTSecret string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
