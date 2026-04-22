package analyze

import "encoding/json"

// TabRec is a recommendation for whether a tab should be enabled.
type TabRec struct {
	Enabled    bool     `json:"enabled"`
	Reason     string   `json:"reason"`
	ChartTypes []string `json:"chart_types,omitempty"`
}

// TabRecommendations returns which tabs should be active based on document state.
func TabRecommendations(hasDoc bool, rawContent, structuredJSON, schemaJSON string, forestDocCount int) map[string]TabRec {
	tabs := map[string]TabRec{
		"explorer": {Enabled: true, Reason: "Always available"},
	}

	if !hasDoc {
		tabs["table"] = TabRec{Enabled: false, Reason: "Upload a document first"}
		tabs["schema"] = TabRec{Enabled: false, Reason: "Upload a document first"}
		tabs["summary"] = TabRec{Enabled: false, Reason: "Upload a document first"}
		tabs["graphs"] = TabRec{Enabled: false, Reason: "Upload a document first"}
		tabs["oracle"] = TabRec{Enabled: false, Reason: "Upload documents and add to a forest"}
		return tabs
	}

	// Summary: always available if doc exists
	tabs["summary"] = TabRec{Enabled: true, Reason: "Document uploaded"}

	// Schema: available if doc exists
	tabs["schema"] = TabRec{Enabled: true, Reason: "Document uploaded"}

	// Table: only if data is tabular
	docType := DetectType(rawContent)
	isTabular := docType == DocTypeCSV || hasTabularJSON(structuredJSON)
	if isTabular {
		tabs["table"] = TabRec{Enabled: true, Reason: "Tabular data detected"}
	} else {
		tabs["table"] = TabRec{Enabled: false, Reason: "No tabular data — works best with CSV or JSON arrays"}
	}

	// Graphs: only if numeric fields exist in structured data
	chartTypes := detectChartTypes(structuredJSON, schemaJSON)
	if len(chartTypes) > 0 {
		tabs["graphs"] = TabRec{Enabled: true, Reason: "Numeric data found", ChartTypes: chartTypes}
	} else {
		tabs["graphs"] = TabRec{Enabled: false, Reason: "No numeric data for visualization"}
	}

	// Oracle: only if forest has 2+ docs
	if forestDocCount >= 2 {
		tabs["oracle"] = TabRec{Enabled: true, Reason: "Forest has multiple documents"}
	} else if forestDocCount == 1 {
		tabs["oracle"] = TabRec{Enabled: false, Reason: "Add more documents to the forest to compare"}
	} else {
		tabs["oracle"] = TabRec{Enabled: false, Reason: "Add documents to a forest first"}
	}

	return tabs
}

func hasTabularJSON(structuredJSON string) bool {
	if structuredJSON == "" {
		return false
	}
	// Direct array
	var arr []any
	if json.Unmarshal([]byte(structuredJSON), &arr) == nil && len(arr) > 0 {
		return true
	}
	// Object containing an array
	var obj map[string]any
	if json.Unmarshal([]byte(structuredJSON), &obj) == nil {
		for _, v := range obj {
			if data, err := json.Marshal(v); err == nil {
				var nested []any
				if json.Unmarshal(data, &nested) == nil && len(nested) > 1 {
					return true
				}
			}
		}
	}
	return false
}

func detectChartTypes(structuredJSON, schemaJSON string) []string {
	if structuredJSON == "" {
		return nil
	}

	hasNumeric := false
	hasCategorical := false
	hasDate := false

	// Try to use the schema if available
	if schemaJSON != "" {
		var schema struct {
			Fields []struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"fields"`
		}
		if json.Unmarshal([]byte(schemaJSON), &schema) == nil {
			for _, f := range schema.Fields {
				switch f.Type {
				case "number":
					hasNumeric = true
				case "string":
					hasCategorical = true
				case "date":
					hasDate = true
				}
			}
		}
	}

	// Fallback: analyze structured JSON directly
	if !hasNumeric {
		var arr []map[string]any
		if json.Unmarshal([]byte(structuredJSON), &arr) == nil && len(arr) > 0 {
			for _, row := range arr[:1] {
				for _, v := range row {
					switch v.(type) {
					case float64:
						hasNumeric = true
					case string:
						hasCategorical = true
					}
				}
			}
		} else {
			// Try nested array
			var obj map[string]any
			if json.Unmarshal([]byte(structuredJSON), &obj) == nil {
				for _, v := range obj {
					if data, err := json.Marshal(v); err == nil {
						var nested []map[string]any
						if json.Unmarshal(data, &nested) == nil && len(nested) > 0 {
							for _, val := range nested[0] {
								switch val.(type) {
								case float64:
									hasNumeric = true
								case string:
									hasCategorical = true
								}
							}
						}
					}
				}
			}
		}
	}

	if !hasNumeric {
		return nil
	}

	var types []string
	if hasCategorical && hasNumeric {
		types = append(types, "bar", "pie")
	}
	if hasDate && hasNumeric {
		types = append(types, "line")
	}
	if hasNumeric && !hasDate && !hasCategorical {
		types = append(types, "bar") // numeric-only gets bar chart
	}
	if len(types) == 0 && hasNumeric {
		types = append(types, "bar")
	}
	return types
}
