package analyze

import (
	"encoding/json"
	"math"
)

// QualityScore represents a data quality assessment from 0-100.
type QualityScore struct {
	Score       int              `json:"score"`
	Grade       string           `json:"grade"` // A, B, C, D, F
	Breakdown   QualityBreakdown `json:"breakdown"`
	Suggestions []string         `json:"suggestions"`
}

type QualityBreakdown struct {
	Completeness int `json:"completeness"` // 0-100: how few missing values
	Consistency  int `json:"consistency"`  // 0-100: how uniform are types
	Validity     int `json:"validity"`     // 0-100: how many values parse correctly
	Structure    int `json:"structure"`    // 0-100: how well-structured is the data
}

// ComputeQualityScore analyzes raw content and produces a quality score.
func ComputeQualityScore(rawContent, structuredJSON string) *QualityScore {
	schema := InferSchema(rawContent)
	qs := &QualityScore{}

	if schema == nil || len(schema.Fields) == 0 {
		qs.Score = 20
		qs.Grade = "F"
		qs.Breakdown = QualityBreakdown{Completeness: 20, Consistency: 20, Validity: 20, Structure: 20}
		qs.Suggestions = []string{"Could not detect structure — document may be unstructured text"}
		return qs
	}

	totalCells := schema.RowCount * schema.FieldCount
	if totalCells == 0 {
		totalCells = 1
	}

	// Completeness: % of non-empty cells
	totalEmpty := 0
	for _, f := range schema.Fields {
		totalEmpty += f.EmptyCount
	}
	completeness := 100 - int(math.Min(100, float64(totalEmpty*100)/float64(totalCells)))

	// Consistency: % of fields with a single type (not mixed)
	mixedCount := 0
	for _, f := range schema.Fields {
		if f.Type == TypeMixed {
			mixedCount++
		}
	}
	consistency := 100
	if schema.FieldCount > 0 {
		consistency = 100 - (mixedCount * 100 / schema.FieldCount)
	}

	// Validity: penalize for known issues
	issueCount := len(schema.Issues)
	validity := int(math.Max(0, float64(100-issueCount*8)))

	// Structure: based on document type
	structure := 50
	switch schema.DocType {
	case DocTypeCSV, DocTypeJSON:
		structure = 90
	case DocTypeXML:
		structure = 80
	case DocTypeLogFile:
		structure = 60
	case DocTypeKeyValue:
		structure = 55
	case DocTypeMarkdown:
		structure = 40
	case DocTypeUnknown:
		structure = 20
	}

	// If structured JSON exists, bonus for structure
	if structuredJSON != "" {
		structure = int(math.Min(100, float64(structure+20)))

		// Check if the structured output is actually good JSON
		var parsed any
		if json.Unmarshal([]byte(structuredJSON), &parsed) == nil {
			validity = int(math.Min(100, float64(validity+10)))
		}
	}

	qs.Breakdown = QualityBreakdown{
		Completeness: completeness,
		Consistency:  consistency,
		Validity:     validity,
		Structure:    structure,
	}

	// Weighted average
	qs.Score = (completeness*30 + consistency*25 + validity*20 + structure*25) / 100

	// Grade
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

	// Suggestions
	if completeness < 80 {
		qs.Suggestions = append(qs.Suggestions, "High number of missing values — consider data imputation")
	}
	if consistency < 80 {
		qs.Suggestions = append(qs.Suggestions, "Mixed data types detected — normalize columns to consistent types")
	}
	if validity < 70 {
		qs.Suggestions = append(qs.Suggestions, "Multiple data quality issues found — review flagged problems")
	}
	if structure < 60 {
		qs.Suggestions = append(qs.Suggestions, "Weakly structured document — parsing may require manual review")
	}

	return qs
}
