package analyze

import (
	"encoding/json"
	"math"
	"strings"
)

type QualityScore struct {
	Score       int              `json:"score"`
	Grade       string           `json:"grade"`
	Status      string           `json:"status"`
	Breakdown   QualityBreakdown `json:"breakdown"`
	Suggestions []string         `json:"suggestions"`
}

type QualityBreakdown struct {
	Completeness int `json:"completeness"`
	Consistency  int `json:"consistency"`
	Validity     int `json:"validity"`
	Structure    int `json:"structure"`
}

func ComputeQualityScore(rawContent, structuredJSON string) *QualityScore {
	if structuredJSON == "" {
		return &QualityScore{
			Score: 0, Grade: "-", Status: "pending",
			Suggestions: []string{"Parse the document in Explorer to get a quality score"},
		}
	}

	qs := &QualityScore{Status: "scored"}

	var parsed any
	if json.Unmarshal([]byte(structuredJSON), &parsed) != nil {
		qs.Score = 25
		qs.Grade = "F"
		qs.Breakdown = QualityBreakdown{25, 25, 25, 25}
		qs.Suggestions = []string{"Structured output is not valid JSON — re-parse"}
		return qs
	}

	structure := scoreStructure(parsed)
	completeness := scoreCompleteness(parsed)
	consistency := scoreConsistency(structuredJSON)
	validity := scoreValidity(rawContent, structuredJSON, parsed)

	qs.Breakdown = QualityBreakdown{completeness, consistency, validity, structure}
	qs.Score = (completeness*30 + consistency*25 + validity*20 + structure*25) / 100

	switch {
	case qs.Score >= 90:
		qs.Grade = "A"
	case qs.Score >= 75:
		qs.Grade = "B"
	case qs.Score >= 60:
		qs.Grade = "C"
	case qs.Score >= 40:
		qs.Grade = "D"
	default:
		qs.Grade = "F"
	}

	if completeness < 80 {
		qs.Suggestions = append(qs.Suggestions, "Some fields are null — consider re-parsing with more detail")
	}
	if consistency < 70 {
		qs.Suggestions = append(qs.Suggestions, "Inconsistent field names across records — normalize schema")
	}
	if validity < 70 {
		qs.Suggestions = append(qs.Suggestions, "Some raw data may not have been captured — review output")
	}
	if structure < 70 {
		qs.Suggestions = append(qs.Suggestions, "Output structure is flat — consider nested grouping")
	}
	if qs.Score >= 85 && len(qs.Suggestions) == 0 {
		qs.Suggestions = append(qs.Suggestions, "Good quality — ready for export and analysis")
	}

	return qs
}

func scoreStructure(parsed any) int {
	switch v := parsed.(type) {
	case []any:
		if len(v) == 0 {
			return 30
		}
		if obj, ok := v[0].(map[string]any); ok {
			// Array of objects — check depth
			depth := maxDepth(obj)
			if depth >= 2 {
				return 95
			}
			return 85
		}
		return 55 // array of primitives
	case map[string]any:
		if len(v) == 0 {
			return 20
		}
		depth := maxDepth(v)
		hasArrays := false
		for _, val := range v {
			if _, ok := val.([]any); ok {
				hasArrays = true
			}
		}
		if hasArrays && depth >= 3 {
			return 95 // well-structured nested with arrays
		}
		if hasArrays {
			return 85
		}
		if depth >= 2 {
			return 75 // nested object without arrays
		}
		return 60 // flat object
	default:
		return 30
	}
}

func maxDepth(obj map[string]any) int {
	d := 1
	for _, v := range obj {
		switch val := v.(type) {
		case map[string]any:
			sub := maxDepth(val) + 1
			if sub > d {
				d = sub
			}
		case []any:
			if len(val) > 0 {
				if inner, ok := val[0].(map[string]any); ok {
					sub := maxDepth(inner) + 2
					if sub > d {
						d = sub
					}
				} else {
					if 2 > d {
						d = 2
					}
				}
			}
		}
	}
	return d
}

func scoreCompleteness(parsed any) int {
	total := 0
	nulls := 0
	empties := 0

	var walk func(any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			for _, fv := range val {
				total++
				if fv == nil {
					nulls++
				} else if s, ok := fv.(string); ok && s == "" {
					empties++
				} else {
					walk(fv)
				}
			}
		case []any:
			for _, item := range val {
				walk(item)
			}
		}
	}
	walk(parsed)

	if total == 0 {
		return 50
	}
	missing := nulls + empties
	return clamp(100 - (missing * 100 / total))
}

func scoreConsistency(structuredJSON string) int {
	// Check arrays of objects for key consistency
	var arr []map[string]any
	if json.Unmarshal([]byte(structuredJSON), &arr) == nil && len(arr) > 1 {
		return scoreArrayConsistency(arr)
	}

	// Check nested arrays
	var obj map[string]any
	if json.Unmarshal([]byte(structuredJSON), &obj) == nil {
		for _, v := range obj {
			data, _ := json.Marshal(v)
			var nested []map[string]any
			if json.Unmarshal(data, &nested) == nil && len(nested) > 1 {
				return scoreArrayConsistency(nested)
			}
		}
		// Single object — check if field names are consistent style (camelCase, snake_case, etc.)
		return scoreNamingConsistency(obj)
	}

	return 60
}

func scoreArrayConsistency(arr []map[string]any) int {
	if len(arr) < 2 {
		return 80
	}
	// Count how many rows have the same keys as row 0
	refKeys := make(map[string]bool)
	for k := range arr[0] {
		refKeys[k] = true
	}

	totalMatch := 0
	for _, row := range arr[1:] {
		rowKeys := make(map[string]bool)
		for k := range row {
			rowKeys[k] = true
		}
		match := 0
		for k := range refKeys {
			if rowKeys[k] {
				match++
			}
		}
		if len(refKeys) > 0 {
			totalMatch += match * 100 / len(refKeys)
		}
	}
	avg := totalMatch / (len(arr) - 1)
	return clamp(avg)
}

func scoreNamingConsistency(obj map[string]any) int {
	camel := 0
	snake := 0
	other := 0
	for k := range obj {
		if strings.Contains(k, "_") {
			snake++
		} else if len(k) > 0 && k[0] >= 'a' && k[0] <= 'z' {
			camel++
		} else {
			other++
		}
	}
	total := camel + snake + other
	if total == 0 {
		return 70
	}
	dominant := camel
	if snake > dominant {
		dominant = snake
	}
	return clamp(50 + dominant*50/total)
}

func scoreValidity(rawContent, structuredJSON string, parsed any) int {
	if rawContent == "" {
		return 60
	}

	// Count how many "significant" tokens from raw appear in structured
	rawTokens := extractSignificantTokens(rawContent)
	structTokens := extractSignificantTokens(structuredJSON)

	if len(rawTokens) == 0 {
		return 60
	}

	found := 0
	for _, rt := range rawTokens {
		for _, st := range structTokens {
			if rt == st {
				found++
				break
			}
		}
	}

	captureRate := found * 100 / len(rawTokens)
	return clamp(captureRate)
}

// extractSignificantTokens gets numbers, emails, dates — things that matter for data capture
func extractSignificantTokens(text string) []string {
	var tokens []string
	seen := make(map[string]bool)
	words := strings.Fields(text)
	for _, w := range words {
		w = strings.Trim(w, `"',;:()[]{}`)
		if len(w) < 2 {
			continue
		}
		// Numbers, emails, dates are significant
		isSignificant := false
		if len(w) >= 3 && (w[0] >= '0' && w[0] <= '9' || w[0] == '$') {
			isSignificant = true
		}
		if strings.Contains(w, "@") {
			isSignificant = true
		}
		if isSignificant && !seen[w] {
			seen[w] = true
			tokens = append(tokens, w)
		}
	}
	return tokens
}

func clamp(v int) int {
	return int(math.Max(0, math.Min(100, float64(v))))
}
