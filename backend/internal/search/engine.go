package search

import (
	"fmt"
	"strings"

	"github.com/zarvis/internal/state"
)

// ChunkStore is the minimal interface needed by the search engine.
type ChunkStore interface {
	SaveChunks(chunks []state.ChunkRecord) error
	GetForestChunks(forestID int64) ([]state.ChunkRecord, error)
	DeleteDocumentChunks(docID, forestID int64) error
}

// Engine orchestrates document indexing and search.
type Engine struct {
	store ChunkStore
}

func NewEngine(store ChunkStore) *Engine {
	return &Engine{store: store}
}

// IndexDocument chunks a document's content and stores the chunks.
func (e *Engine) IndexDocument(docID, forestID int64, content string) error {
	_ = e.store.DeleteDocumentChunks(docID, forestID)

	texts := ChunkText(content, 500)
	records := make([]state.ChunkRecord, len(texts))
	for i, t := range texts {
		records[i] = state.ChunkRecord{
			DocumentID: docID,
			ForestID:   forestID,
			Content:    t,
			Position:   i,
		}
	}
	return e.store.SaveChunks(records)
}

// Search finds the top-K most relevant chunks in a forest for a query.
func (e *Engine) Search(forestID int64, query string, topK int) ([]SearchResult, error) {
	records, err := e.store.GetForestChunks(forestID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	// Convert to Chunk for BM25
	chunks := make([]Chunk, len(records))
	for i, r := range records {
		chunks[i] = Chunk{ID: r.ID, DocumentID: r.DocumentID, ForestID: r.ForestID, Content: r.Content, Position: r.Position}
	}
	return RankChunks(chunks, query, topK), nil
}

// FormatResults formats search results for Claude's context.
func FormatResults(results []SearchResult, docNames map[int64]string) string {
	if len(results) == 0 {
		return "No relevant chunks found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant chunks:\n\n", len(results)))
	for i, r := range results {
		docName := docNames[r.DocID]
		if docName == "" {
			docName = fmt.Sprintf("doc#%d", r.DocID)
		}
		sb.WriteString(fmt.Sprintf("--- [%d] %s (relevance: %.2f) ---\n%s\n\n", i+1, docName, r.Score, r.Content))
	}
	return sb.String()
}
