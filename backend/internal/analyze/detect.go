// Package analyze provides server-side document analysis — type detection,
// schema inference, and data cleaning done in Go before Claude sees the data.
package analyze

import (
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strings"
)

// DocType represents the detected document type.
type DocType string

const (
	DocTypeCSV        DocType = "csv"
	DocTypeJSON       DocType = "json"
	DocTypeLogFile    DocType = "log"
	DocTypeKeyValue   DocType = "key_value"
	DocTypeMarkdown   DocType = "markdown"
	DocTypeXML        DocType = "xml"
	DocTypeUnknown    DocType = "unknown"
)

var (
	logLineRe    = regexp.MustCompile(`^\[?\d{4}[-/]\d{2}[-/]\d{2}[T ]?\d{2}:\d{2}`)
	xmlTagRe     = regexp.MustCompile(`<[a-zA-Z][^>]*>`)
	keyValueRe   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9 _-]*:\s+\S`)
	markdownHeadRe = regexp.MustCompile(`^#{1,6}\s+\S`)
)

// DetectType analyzes raw content and returns the most likely document type.
func DetectType(content string) DocType {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return DocTypeUnknown
	}

	// JSON: starts with { or [
	first := trimmed[0]
	if first == '{' || first == '[' {
		var js json.RawMessage
		if json.Unmarshal([]byte(trimmed), &js) == nil {
			return DocTypeJSON
		}
	}

	// XML: has tags
	if strings.HasPrefix(trimmed, "<?xml") || (strings.HasPrefix(trimmed, "<") && xmlTagRe.MatchString(trimmed)) {
		return DocTypeXML
	}

	lines := strings.Split(trimmed, "\n")

	// CSV: first line has commas/tabs and subsequent lines have similar structure
	if isCSVLike(lines) {
		return DocTypeCSV
	}

	// Log file: multiple lines starting with timestamps
	logCount := 0
	for _, line := range lines[:min(20, len(lines))] {
		if logLineRe.MatchString(strings.TrimSpace(line)) {
			logCount++
		}
	}
	if logCount > len(lines[:min(20, len(lines))])/2 {
		return DocTypeLogFile
	}

	// Markdown: has headers
	mdCount := 0
	for _, line := range lines[:min(20, len(lines))] {
		if markdownHeadRe.MatchString(strings.TrimSpace(line)) {
			mdCount++
		}
	}
	if mdCount >= 2 {
		return DocTypeMarkdown
	}

	// Key-value: multiple "Key: Value" lines
	kvCount := 0
	for _, line := range lines[:min(30, len(lines))] {
		if keyValueRe.MatchString(strings.TrimSpace(line)) {
			kvCount++
		}
	}
	if kvCount > len(lines[:min(30, len(lines))])/3 {
		return DocTypeKeyValue
	}

	return DocTypeUnknown
}

func isCSVLike(lines []string) bool {
	if len(lines) < 2 {
		return false
	}
	header := lines[0]
	// Try comma-separated
	r := csv.NewReader(strings.NewReader(header))
	fields, err := r.Read()
	if err != nil || len(fields) < 2 {
		// Try tab-separated
		if strings.Count(header, "\t") >= 1 {
			return true
		}
		return false
	}
	// Check that at least one more row has similar field count
	r2 := csv.NewReader(strings.NewReader(lines[1]))
	fields2, err := r2.Read()
	if err != nil {
		return false
	}
	return len(fields2) >= len(fields)-1 && len(fields2) <= len(fields)+1
}
