package search

import "testing"

func TestChunkText_SmallText(t *testing.T) {
	chunks := ChunkText("hello world", 500)
	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestChunkText_SplitsParagraphs(t *testing.T) {
	text := "First paragraph with some content.\n\nSecond paragraph with different content.\n\nThird paragraph here."
	chunks := ChunkText(text, 50)
	if len(chunks) < 2 {
		t.Errorf("expected multiple chunks, got %d: %v", len(chunks), chunks)
	}
}

func TestChunkText_Empty(t *testing.T) {
	chunks := ChunkText("", 500)
	if len(chunks) != 0 {
		t.Errorf("expected 0 chunks for empty, got %d", len(chunks))
	}
}

func TestTokenize(t *testing.T) {
	tokens := Tokenize("Hello, World! This is a TEST.")
	expected := []string{"hello", "world", "this", "is", "a", "test"}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d: %v", len(expected), len(tokens), tokens)
	}
	for i, tok := range tokens {
		if tok != expected[i] {
			t.Errorf("token %d: got %q, want %q", i, tok, expected[i])
		}
	}
}

func TestRankChunks_ReturnsRelevant(t *testing.T) {
	chunks := []Chunk{
		{ID: 1, Content: "Go is a programming language designed at Google"},
		{ID: 2, Content: "Python is popular for data science and machine learning"},
		{ID: 3, Content: "Go has goroutines for concurrent programming"},
	}
	results := RankChunks(chunks, "Go programming", 2)
	if len(results) == 0 {
		t.Fatal("expected results")
	}
	// Chunk 1 and 3 mention Go and programming, should rank higher
	if results[0].ChunkID != 1 && results[0].ChunkID != 3 {
		t.Errorf("top result should be about Go, got chunk %d", results[0].ChunkID)
	}
}

func TestRankChunks_EmptyQuery(t *testing.T) {
	chunks := []Chunk{{ID: 1, Content: "some content"}}
	results := RankChunks(chunks, "", 5)
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestRankChunks_NoMatch(t *testing.T) {
	chunks := []Chunk{{ID: 1, Content: "cats and dogs"}}
	results := RankChunks(chunks, "quantum physics", 5)
	if len(results) != 0 {
		t.Errorf("expected no results for unrelated query, got %d", len(results))
	}
}

func TestRankChunks_TopK(t *testing.T) {
	chunks := []Chunk{
		{ID: 1, Content: "apple banana cherry"},
		{ID: 2, Content: "apple banana"},
		{ID: 3, Content: "apple"},
	}
	results := RankChunks(chunks, "apple banana cherry", 2)
	if len(results) > 2 {
		t.Errorf("expected at most 2 results, got %d", len(results))
	}
}
