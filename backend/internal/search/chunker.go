// Package search provides BM25-based text search over document chunks.
package search

import "strings"

// Chunk represents a segment of a document.
type Chunk struct {
	ID         int64  `json:"id"`
	DocumentID int64  `json:"document_id"`
	ForestID   int64  `json:"forest_id"`
	Content    string `json:"content"`
	Position   int    `json:"position"`
}

// ChunkText splits text into chunks of approximately targetSize characters.
// It splits on paragraph boundaries first, then sentence boundaries.
func ChunkText(text string, targetSize int) []string {
	if targetSize <= 0 {
		targetSize = 500
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= targetSize {
		return []string{text}
	}

	// Split into paragraphs first
	paragraphs := strings.Split(text, "\n\n")
	var chunks []string
	var current strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// If adding this paragraph would exceed target, flush current
		if current.Len() > 0 && current.Len()+len(para)+2 > targetSize {
			chunks = append(chunks, strings.TrimSpace(current.String()))
			current.Reset()
		}

		// If a single paragraph exceeds target, split it on sentences
		if len(para) > targetSize {
			sentences := splitSentences(para)
			for _, sent := range sentences {
				if current.Len() > 0 && current.Len()+len(sent)+1 > targetSize {
					chunks = append(chunks, strings.TrimSpace(current.String()))
					current.Reset()
				}
				if current.Len() > 0 {
					current.WriteString(" ")
				}
				current.WriteString(sent)
			}
		} else {
			if current.Len() > 0 {
				current.WriteString("\n\n")
			}
			current.WriteString(para)
		}
	}

	if current.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(current.String()))
	}

	return chunks
}

func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	for i, r := range text {
		current.WriteRune(r)
		if (r == '.' || r == '!' || r == '?' || r == '\n') && i+1 < len(text) && text[i+1] == ' ' {
			sentences = append(sentences, strings.TrimSpace(current.String()))
			current.Reset()
		}
	}
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}
	return sentences
}
