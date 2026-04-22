// Package badges manages the cosmetic gamification system.
package badges

import "github.com/zarvis/internal/state"

type BadgeDef struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

var AllBadges = []BadgeDef{
	{Key: "first_upload", Name: "First Upload", Description: "Upload your first document", Icon: "📄"},
	{Key: "structured", Name: "Structured", Description: "Parse a document into structured data", Icon: "🔧"},
	{Key: "schema_master", Name: "Schema Master", Description: "Infer a schema from a document", Icon: "📐"},
	{Key: "summarizer", Name: "Summarizer", Description: "Generate a document summary", Icon: "📝"},
	{Key: "queried", Name: "Data Explorer", Description: "Query your structured data", Icon: "🔍"},
	{Key: "power_user", Name: "Power User", Description: "Process 5 documents", Icon: "🚀"},
}

type Engine struct{ store state.Store }

func New(store state.Store) *Engine { return &Engine{store: store} }

func (e *Engine) CheckAndAward(sessionID string, action string) []string {
	var earned []string
	switch action {
	case "upload":
		if e.award(sessionID, "first_upload") {
			earned = append(earned, "first_upload")
		}
	case "save_structured_data":
		if e.award(sessionID, "structured") {
			earned = append(earned, "structured")
		}
	case "save_schema":
		if e.award(sessionID, "schema_master") {
			earned = append(earned, "schema_master")
		}
	case "save_summary":
		if e.award(sessionID, "summarizer") {
			earned = append(earned, "summarizer")
		}
	case "query_structured_data":
		if e.award(sessionID, "queried") {
			earned = append(earned, "queried")
		}
	}
	return earned
}

func (e *Engine) award(sessionID, key string) bool {
	existing, _ := e.store.GetBadges(sessionID)
	for _, b := range existing {
		if b.BadgeKey == key {
			return false
		}
	}
	_ = e.store.EarnBadge(sessionID, key)
	return true
}
