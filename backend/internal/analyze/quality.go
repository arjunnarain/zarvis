package analyze

import (
	"encoding/json"
	"math"
)

// QualityScore represents a data quality assessment from 0-100.
type QualityScore struct {
	Score       int              `json:"score"`
	Grade       string           `json:"grade"`
	Status      string           `json:"status"` // "pending", "scored"
	Breakdown   QualityBreakdown `json:"breakdown"`
	Suggestions []string         `json:"suggestions"`
}

type QualityBreakdown struct {
	Completeness int `json:"completeness"`
	Consistency  int `json:"consistency"`
	Validity     int `json:"validity"`
	Structure    int `json:"structure"`
}

// ComputeQualityScore assesses the quality of the STRUCTURED OUTPUT, not the raw input.
// If no structured data exists yet, returns a "pending" status.
func ComputeQualityScore(rawContent, structuredJSON string) *QualityScore {
	// No structured data yet — document hasn't been parsed
	if structuredJSON == "" {
		return &QualityScore{
			Score:  0,
			Grade:  "-",
			Status: "pending",
			Suggestions: []string{"Parse the document in Explorer to get a quality score"},
		}
	}

	qs := &QualityScore{Status: "scored"}

	// Validate the structured JSON is actually valid
	var parsed any
	if json.Unmarshal([]byte(structuredJSON), &parsed) != nil {
		qs.Score = 30
		qs.Grade = "F"
		qs.Breakdown = QualityBreakdown{Completeness: 30, Consistency: 30, Validity: 30, Structure: 30}
		qs.Suggestions = []string{"Structured output is not valid JSON — re-parse in Explorer"}
		return qs
	}

	// Structure score: how well-formed is the output?
	structure := scoreStructure(parsed)

	// Completeness: check for null/empty values in the output
	completeness := scoreCompleteness(structuredJSON)

	// Consistency: check field type uniformity
	consistency := scoreConsistency(structuredJSON)

	// Validity: compare raw input analysis to see how much was captured
	validity := scoreValidity(rawContent, structuredJSON)

	qs.Breakdown = QualityBreakdown{
		Completeness: completeness,
		Consistency:  consistency,
		Validity:     validity,
		Structure:    structure,
	}

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
		qs.Suggestions = append(qs.Suggestions, "Some values missing in output — consider re-parsing with more detail")
	}
	if consistency < 80 {
		qs.Suggestions = append(qs.Suggestions, "Mixed types in output — ask Explorer to normalize types")
	}
	if structure < 70 {
		qs.Suggestions = append(qs.Suggestions, "Output structure could be improved — try re-parsing")
	}
	if qs.Score >= 85 {
		qs.Suggestions = append(qs.Suggestions, "Good quality — ready for export and analysis")
	}

	return qs
}

func scoreStructure(parsed any) int {
	switch v := parsed.(type) {
	case []any:
		if len(v) == 0 {
			return 40
		}
		// Array of objects = well structured
		if _, ok := v[0].(map[string]any); ok {
			return 95
		}
		return 70
	case map[string]any:
		if len(v) == 0 {
			return 30
		}
		// Nested object with arrays = well structured
		for _, val := range v {
			if arr, ok := val.([]any); ok && len(arr) > 0 {
				return 90
			}
		}
		return 80
	default:
		return 40
	}
}

func scoreCompleteness(structuredJSON string) int {
	totalFields := 0
	nullFields := 0

	var walk func(any)
	walk = func(v any) {
		switch val := v.(type) {
		case map[string]any:
			for _, fv := range val {
				totalFields++
				if fv == nil {
					nullFields++
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

	var parsed any
	json.Unmarshal([]byte(structuredJSON), &parsed)
	walk(parsed)

	if totalFields == 0 {
		return 50
	}
	return int(math.Max(0, float64(100-nullFields*100/totalFields)))
}

func scoreConsistency(structuredJSON string) int {
	// Check if arrays of objects have consistent keys
	var arr []map[string]any
	if json.Unmarshal([]byte(structuredJSON), &arr) == nil && len(arr) > 1 {
		firstKeys := map[string]bool{}
		for k := range arr[0] {
			firstKeys[k] = true
		}
		matches := 0
		for _, row := range arr[1:] {
			rowKeys := map[string]bool{}
			for k := range row {
				rowKeys[k] = true
			}
			same := true
			for k := range firstKeys {
				if !rowKeys[k] {
					same = false
					break
				}
			}
			if same {
				matches++
			}
		}
		return int(math.Min(100, float64(50+matches*50/len(arr))))
	}

	// Try nested arrays
	var obj map[string]any
	if json.Unmarshal([]byte(structuredJSON), &obj) == nil {
		for _, v := range obj {
			data, _ := json.Marshal(v)
			var nested []map[string]any
			if json.Unmarshal(data, &nested) == nil && len(nested) > 1 {
				return 85 // Has consistent nested arrays
			}
		}
		return 80 // Object structure
	}

	return 70
}

func scoreValidity(rawContent, structuredJSON string) int {
	if rawContent == "" || structuredJSON == "" {
		return 50
	}
	// Rough heuristic: ratio of structured size to raw size
	// Good parsing should produce structured output that's comparable in information density
	ratio := float64(len(structuredJSON)) / float64(len(rawContent))
	if ratio > 0.5 {
		return 90
	}
	if ratio > 0.2 {
		return 75
	}
	return 55
}
