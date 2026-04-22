// Package tools executes document processing tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zarvis/internal/search"
	"github.com/zarvis/internal/state"
)

type Result struct {
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

type Executor struct {
	Store  state.Store
	Search *search.Engine
}

func NewExecutor(store state.Store, searchEngine *search.Engine) *Executor {
	return &Executor{Store: store, Search: searchEngine}
}

func (e *Executor) Execute(_ context.Context, sessionID, toolName string, input json.RawMessage) Result {
	switch toolName {
	case "get_raw_document":
		return e.getRawDocument(sessionID)
	case "get_structured_data":
		return e.getStructuredData(sessionID)
	case "save_structured_data":
		return e.saveStructuredData(sessionID, input)
	case "get_schema":
		return e.getSchema(sessionID)
	case "save_schema":
		return e.saveSchema(sessionID, input)
	case "save_summary":
		return e.saveSummary(sessionID, input)
	case "get_summary":
		return e.getSummary(sessionID)
	case "query_structured_data":
		return e.queryStructuredData(sessionID, input)
	case "list_documents":
		return e.listDocuments(sessionID)
	case "get_forest_documents":
		return e.getForestDocuments(input)
	case "query_forest":
		return e.queryForest(input)
	default:
		return Result{Error: fmt.Sprintf("unknown tool: %s", toolName)}
	}
}

func (e *Executor) getRawDocument(sessionID string) Result {
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document uploaded yet. Ask the user to upload a document first."}
	}
	content := doc.RawContent
	if len(content) > 8000 {
		content = content[:8000] + "\n\n... [truncated, " + fmt.Sprintf("%d", len(doc.RawContent)) + " chars total]"
	}
	return Result{Output: fmt.Sprintf("Document: %s\n\n%s", doc.Filename, content)}
}

func (e *Executor) getStructuredData(sessionID string) Result {
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document uploaded yet."}
	}
	if doc.StructuredJSON == "" {
		return Result{Error: "Document hasn't been parsed into structured data yet. Use save_structured_data first."}
	}
	return Result{Output: doc.StructuredJSON}
}

func (e *Executor) saveStructuredData(sessionID string, input json.RawMessage) Result {
	var params struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Data == "" {
		return Result{Error: "data (JSON string) is required"}
	}
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document to attach structured data to."}
	}
	if err := e.Store.UpdateDocumentStructured(doc.ID, params.Data, doc.SchemaJSON, doc.Summary); err != nil {
		return Result{Error: err.Error()}
	}
	// Re-index chunks in all forests containing this document
	e.reindexDocInForests(doc.ID, params.Data)
	return Result{Output: "Structured data saved successfully."}
}

func (e *Executor) getSchema(sessionID string) Result {
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document uploaded yet."}
	}
	if doc.SchemaJSON == "" {
		return Result{Error: "Schema hasn't been inferred yet. Use save_schema first."}
	}
	return Result{Output: doc.SchemaJSON}
}

func (e *Executor) saveSchema(sessionID string, input json.RawMessage) Result {
	var params struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Schema == "" {
		return Result{Error: "schema (JSON string) is required"}
	}
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document to attach schema to."}
	}
	if err := e.Store.UpdateDocumentStructured(doc.ID, doc.StructuredJSON, params.Schema, doc.Summary); err != nil {
		return Result{Error: err.Error()}
	}
	return Result{Output: "Schema saved successfully."}
}

func (e *Executor) saveSummary(sessionID string, input json.RawMessage) Result {
	var params struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Summary == "" {
		return Result{Error: "summary is required"}
	}
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document to attach summary to."}
	}
	if err := e.Store.UpdateDocumentStructured(doc.ID, doc.StructuredJSON, doc.SchemaJSON, params.Summary); err != nil {
		return Result{Error: err.Error()}
	}
	return Result{Output: "Summary saved successfully."}
}

func (e *Executor) getSummary(sessionID string) Result {
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document uploaded yet."}
	}
	if doc.Summary == "" {
		return Result{Error: "No summary generated yet. Use save_summary first."}
	}
	return Result{Output: doc.Summary}
}

func (e *Executor) queryStructuredData(sessionID string, input json.RawMessage) Result {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.Query == "" {
		return Result{Error: "query is required"}
	}
	doc, err := e.Store.GetLatestDocument(sessionID)
	if err != nil {
		return Result{Error: "No document uploaded yet."}
	}
	if doc.StructuredJSON == "" {
		return Result{Error: "Document hasn't been parsed yet. Parse it first."}
	}
	// Return the structured data so Claude can query it in-context
	return Result{Output: fmt.Sprintf("Structured data to answer query \"%s\":\n\n%s", params.Query, doc.StructuredJSON)}
}

func (e *Executor) listDocuments(sessionID string) Result {
	docs, err := e.Store.ListDocuments(sessionID)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if len(docs) == 0 {
		return Result{Output: "No documents uploaded yet."}
	}
	out := ""
	for _, d := range docs {
		out += fmt.Sprintf("#%d: %s (%s)\n", d.ID, d.Filename, d.CreatedAt.Format("2006-01-02 15:04"))
	}
	return Result{Output: out}
}

// reindexDocInForests re-chunks and re-indexes a document in all forests it belongs to.
func (e *Executor) reindexDocInForests(docID int64, content string) {
	if e.Search == nil {
		return
	}
	forestIDs, err := e.Store.GetForestsForDocument(docID)
	if err != nil {
		return
	}
	for _, fid := range forestIDs {
		_ = e.Search.IndexDocument(docID, fid, content)
	}
}

func (e *Executor) getForestDocuments(input json.RawMessage) Result {
	var params struct {
		ForestID int64  `json:"forest_id"`
		Query    string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.ForestID == 0 {
		return Result{Error: "forest_id is required"}
	}

	// If there's a query, use BM25 search for relevant chunks
	if params.Query != "" && e.Search != nil {
		results, err := e.Search.Search(params.ForestID, params.Query, 10)
		if err != nil {
			return Result{Error: err.Error()}
		}
		if len(results) == 0 {
			return Result{Output: "No relevant content found for that query."}
		}
		// Get doc names for attribution
		docs, _ := e.Store.GetForestDocuments(params.ForestID)
		docNames := make(map[int64]string)
		for _, d := range docs {
			docNames[d.ID] = d.Filename
		}
		return Result{Output: search.FormatResults(results, docNames)}
	}

	// Fallback: load all docs (for general queries like "compare everything")
	docs, err := e.Store.GetForestDocuments(params.ForestID)
	if err != nil {
		return Result{Error: err.Error()}
	}
	if len(docs) == 0 {
		return Result{Output: "No documents in this forest yet."}
	}
	var sb strings.Builder
	for _, d := range docs {
		sb.WriteString(fmt.Sprintf("=== Document #%d: %s ===\n", d.ID, d.Filename))
		if d.StructuredJSON != "" {
			data := d.StructuredJSON
			if len(data) > 3000 {
				data = data[:3000] + "\n... [truncated]"
			}
			sb.WriteString("Structured data:\n" + data + "\n\n")
		} else if d.RawContent != "" {
			raw := d.RawContent
			if len(raw) > 2000 {
				raw = raw[:2000] + "\n... [truncated]"
			}
			sb.WriteString("Raw content:\n" + raw + "\n\n")
		}
	}
	return Result{Output: sb.String()}
}

func (e *Executor) queryForest(input json.RawMessage) Result {
	var params struct {
		ForestID int64  `json:"forest_id"`
		Query    string `json:"query"`
	}
	if err := json.Unmarshal(input, &params); err != nil || params.ForestID == 0 {
		return Result{Error: "forest_id and query are required"}
	}

	// Use BM25 search to find relevant chunks
	if e.Search != nil && params.Query != "" {
		results, err := e.Search.Search(params.ForestID, params.Query, 10)
		if err != nil {
			return Result{Error: err.Error()}
		}
		docs, _ := e.Store.GetForestDocuments(params.ForestID)
		docNames := make(map[int64]string)
		for _, d := range docs {
			docNames[d.ID] = d.Filename
		}
		if len(results) == 0 {
			return Result{Output: "No relevant content found for: " + params.Query}
		}
		return Result{Output: fmt.Sprintf("BM25 search results for \"%s\":\n\n%s", params.Query, search.FormatResults(results, docNames))}
	}

	return Result{Error: "query is required"}
}
