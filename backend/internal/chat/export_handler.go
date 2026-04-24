package chat

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/zarvis/internal/auth"
)

// Export converts structured document data to CSV, JSON, or TSV for download.
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	doc, err := h.Store.GetLatestDocumentByUser(userID)
	if err != nil {
		http.Error(w, "no document", http.StatusNotFound)
		return
	}
	if doc.StructuredJSON == "" {
		http.Error(w, "document not yet parsed — parse it in Explorer first", http.StatusBadRequest)
		return
	}

	baseName := strings.TrimSuffix(doc.Filename, "."+fileExt(doc.Filename))

	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_structured.json"`, baseName))
		// Pretty-print the JSON
		var raw any
		if err := json.Unmarshal([]byte(doc.StructuredJSON), &raw); err != nil {
			w.Write([]byte(doc.StructuredJSON))
			return
		}
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(raw)

	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_structured.csv"`, baseName))
		writeCSV(w, doc.StructuredJSON, ',')

	case "tsv":
		w.Header().Set("Content-Type", "text/tab-separated-values")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_structured.tsv"`, baseName))
		writeCSV(w, doc.StructuredJSON, '\t')

	case "markdown":
		w.Header().Set("Content-Type", "text/markdown")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_structured.md"`, baseName))
		writeMarkdownTable(w, doc.StructuredJSON)

	default:
		http.Error(w, "unsupported format: "+format+". Use json, csv, tsv, or markdown", http.StatusBadRequest)
	}
}

// writeCSV converts JSON (array of objects or object with array value) to CSV/TSV.
func writeCSV(w http.ResponseWriter, jsonData string, sep rune) {
	rows, headers := jsonToRows(jsonData)
	if len(rows) == 0 {
		w.Write([]byte("no tabular data found"))
		return
	}

	writer := csv.NewWriter(w)
	writer.Comma = sep
	writer.Write(headers)
	for _, row := range rows {
		record := make([]string, len(headers))
		for i, h := range headers {
			record[i] = fmt.Sprintf("%v", row[h])
		}
		writer.Write(record)
	}
	writer.Flush()
}

// writeMarkdownTable converts JSON to a Markdown table.
func writeMarkdownTable(w http.ResponseWriter, jsonData string) {
	rows, headers := jsonToRows(jsonData)
	if len(rows) == 0 {
		w.Write([]byte("no tabular data found"))
		return
	}

	// Header row
	fmt.Fprintf(w, "| %s |\n", strings.Join(headers, " | "))
	// Separator
	seps := make([]string, len(headers))
	for i := range seps {
		seps[i] = "---"
	}
	fmt.Fprintf(w, "| %s |\n", strings.Join(seps, " | "))
	// Data rows
	for _, row := range rows {
		vals := make([]string, len(headers))
		for i, h := range headers {
			vals[i] = fmt.Sprintf("%v", row[h])
		}
		fmt.Fprintf(w, "| %s |\n", strings.Join(vals, " | "))
	}
}

// jsonToRows extracts rows from JSON. Handles:
// 1. Direct array of objects
// 2. Object containing an array (e.g., {"employees": [...]})
// 3. Nested object → flattened to key-value rows
func jsonToRows(jsonData string) ([]map[string]any, []string) {
	// Try direct array of objects
	var arr []map[string]any
	if json.Unmarshal([]byte(jsonData), &arr) == nil && len(arr) > 0 {
		return flattenRows(arr), extractHeaders(flattenRows(arr))
	}

	// Try object
	var obj map[string]any
	if json.Unmarshal([]byte(jsonData), &obj) != nil {
		return nil, nil
	}

	// Find the largest array of objects inside the object (any depth)
	bestArr := findLargestArray(obj)
	if len(bestArr) > 0 {
		flat := flattenRows(bestArr)
		return flat, extractHeaders(flat)
	}

	// No array found — flatten the entire object to key-value pairs
	flat := flattenObject("", obj)
	rows := make([]map[string]any, 0, len(flat))
	for k, v := range flat {
		rows = append(rows, map[string]any{"field": k, "value": v})
	}
	sort.Slice(rows, func(i, j int) bool {
		return fmt.Sprintf("%v", rows[i]["field"]) < fmt.Sprintf("%v", rows[j]["field"])
	})
	return rows, []string{"field", "value"}
}

// findLargestArray recursively searches for the largest []map[string]any in a structure.
func findLargestArray(obj map[string]any) []map[string]any {
	var best []map[string]any
	for _, v := range obj {
		switch val := v.(type) {
		case []any:
			var asObjs []map[string]any
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					asObjs = append(asObjs, m)
				}
			}
			if len(asObjs) > len(best) {
				best = asObjs
			}
		case map[string]any:
			nested := findLargestArray(val)
			if len(nested) > len(best) {
				best = nested
			}
		}
	}
	return best
}

// flattenObject converts nested objects to dot-notation key-value pairs.
func flattenObject(prefix string, obj map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range obj {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]any:
			for fk, fv := range flattenObject(key, val) {
				result[fk] = fv
			}
		case []any:
			data, _ := json.Marshal(val)
			result[key] = string(data)
		default:
			result[key] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

// flattenRows flattens nested fields in each row to dot-notation.
func flattenRows(rows []map[string]any) []map[string]any {
	var result []map[string]any
	for _, row := range rows {
		flat := make(map[string]any)
		for k, v := range row {
			switch val := v.(type) {
			case map[string]any:
				for fk, fv := range flattenObject(k, val) {
					flat[fk] = fv
				}
			case []any:
				data, _ := json.Marshal(val)
				flat[k] = string(data)
			default:
				flat[k] = v
			}
		}
		result = append(result, flat)
	}
	return result
}

func extractHeaders(rows []map[string]any) []string {
	seen := map[string]bool{}
	var headers []string
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				headers = append(headers, k)
			}
		}
	}
	sort.Strings(headers)
	return headers
}

func fileExt(name string) string {
	parts := strings.Split(name, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return ""
}

// ExportSchema exports the inferred schema as JSON.
func (h *Handler) ExportSchema(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	doc, err := h.Store.GetLatestDocumentByUser(userID)
	if err != nil {
		http.Error(w, "no document", http.StatusNotFound)
		return
	}
	if doc.SchemaJSON == "" {
		http.Error(w, "no schema available", http.StatusBadRequest)
		return
	}
	baseName := strings.TrimSuffix(doc.Filename, "."+fileExt(doc.Filename))
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s_schema.json"`, baseName))
	var raw any
	json.Unmarshal([]byte(doc.SchemaJSON), &raw)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(raw)
}

// Suppress unused import warning
var _ = strconv.Itoa
