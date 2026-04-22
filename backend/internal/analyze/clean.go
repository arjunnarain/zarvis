package analyze

import (
	"regexp"
	"strings"
)

// CleanReport contains data quality issues found during server-side analysis.
type CleanReport struct {
	TotalValues      int      `json:"total_values"`
	EmptyValues      int      `json:"empty_values"`
	CurrencyValues   int      `json:"currency_values"`
	DateFormats      int      `json:"date_formats_found"`
	EmailsFound      int      `json:"emails_found"`
	MixedTypeColumns []string `json:"mixed_type_columns"`
	Suggestions      []string `json:"suggestions"`
}

var currencyRe = regexp.MustCompile(`^\$[\d,.]+$|^€[\d,.]+$|^£[\d,.]+$`)

// CleanAnalysis scans raw content and produces a data quality report.
func CleanAnalysis(content string, schema *InferredSchema) *CleanReport {
	report := &CleanReport{}

	if schema == nil {
		return report
	}

	for _, f := range schema.Fields {
		report.TotalValues += schema.RowCount
		report.EmptyValues += f.EmptyCount

		if f.Type == TypeMixed {
			report.MixedTypeColumns = append(report.MixedTypeColumns, f.Name)
		}
		if f.Type == TypeEmail {
			report.EmailsFound += schema.RowCount - f.EmptyCount
		}
		if f.Type == TypeDate {
			report.DateFormats++
		}
	}

	// Count currency values in raw content
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		fields := strings.Split(line, ",")
		for _, f := range fields {
			f = strings.TrimSpace(f)
			if currencyRe.MatchString(f) {
				report.CurrencyValues++
			}
		}
	}

	// Generate suggestions
	if report.EmptyValues > 0 {
		pct := report.EmptyValues * 100 / max(report.TotalValues, 1)
		report.Suggestions = append(report.Suggestions, "Data has "+strings.TrimRight(strings.TrimRight(
			strings.Replace(string(rune('0'+pct/10))+string(rune('0'+pct%10)), "00", "0", 1),
			"0"), ".")+"% missing values — consider imputation or filtering")
	}
	if len(report.MixedTypeColumns) > 0 {
		report.Suggestions = append(report.Suggestions, "Columns with mixed types: "+strings.Join(report.MixedTypeColumns, ", ")+" — needs type normalization")
	}
	if report.CurrencyValues > 0 {
		report.Suggestions = append(report.Suggestions, "Found currency-formatted values ($X,XXX) — strip symbols for numerical analysis")
	}
	if report.DateFormats > 1 {
		report.Suggestions = append(report.Suggestions, "Multiple date columns detected — standardize to ISO 8601 format")
	}

	return report
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
