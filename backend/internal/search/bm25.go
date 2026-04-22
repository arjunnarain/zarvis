package search

import (
	"math"
	"regexp"
	"sort"
	"strings"
)

var wordRe = regexp.MustCompile(`[a-z0-9]+`)

// Tokenize normalizes and splits text into lowercase tokens.
func Tokenize(text string) []string {
	return wordRe.FindAllString(strings.ToLower(text), -1)
}

// SearchResult holds a chunk ID and its relevance score.
type SearchResult struct {
	ChunkID  int64
	Score    float64
	Content  string
	DocID    int64
	Position int
}

// BM25 parameters
const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// RankChunks scores chunks against a query using BM25.
func RankChunks(chunks []Chunk, query string, topK int) []SearchResult {
	if len(chunks) == 0 || query == "" {
		return nil
	}

	queryTokens := Tokenize(query)
	if len(queryTokens) == 0 {
		return nil
	}

	// Build corpus: tokenize each chunk
	corpus := make([][]string, len(chunks))
	totalLen := 0
	for i, c := range chunks {
		corpus[i] = Tokenize(c.Content)
		totalLen += len(corpus[i])
	}
	avgDL := float64(totalLen) / float64(len(chunks))
	n := float64(len(chunks))

	// Document frequency for each query term
	df := make(map[string]int)
	for _, qt := range queryTokens {
		for _, doc := range corpus {
			for _, t := range doc {
				if t == qt {
					df[qt]++
					break
				}
			}
		}
	}

	// Score each chunk
	results := make([]SearchResult, 0, len(chunks))
	for i, doc := range corpus {
		score := 0.0
		dl := float64(len(doc))

		// Term frequency in this doc
		tf := make(map[string]int)
		for _, t := range doc {
			tf[t]++
		}

		for _, qt := range queryTokens {
			f := float64(tf[qt])
			if f == 0 {
				continue
			}
			dfi := float64(df[qt])
			// IDF component
			idf := math.Log((n-dfi+0.5)/(dfi+0.5) + 1)
			// TF component with length normalization
			tfNorm := (f * (bm25K1 + 1)) / (f + bm25K1*(1-bm25B+bm25B*dl/avgDL))
			score += idf * tfNorm
		}

		if score > 0 {
			results = append(results, SearchResult{
				ChunkID:  chunks[i].ID,
				Score:    score,
				Content:  chunks[i].Content,
				DocID:    chunks[i].DocumentID,
				Position: chunks[i].Position,
			})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}
	return results
}
