package analyze

import (
	"encoding/csv"
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// FieldType represents a detected data type.
type FieldType string

const (
	TypeString  FieldType = "string"
	TypeNumber  FieldType = "number"
	TypeBoolean FieldType = "boolean"
	TypeDate    FieldType = "date"
	TypeEmail   FieldType = "email"
	TypeURL     FieldType = "url"
	TypeNull    FieldType = "null"
	TypeMixed   FieldType = "mixed"
)

// FieldInfo describes a single field in the inferred schema.
type FieldInfo struct {
	Name       string    `json:"name"`
	Type       FieldType `json:"type"`
	Nullable   bool      `json:"nullable"`
	Unique     bool      `json:"unique"`
	EmptyCount int       `json:"empty_count"`
	SampleVals []string  `json:"sample_values,omitempty"`
}

// InferredSchema is the result of server-side schema inference.
type InferredSchema struct {
	DocType    DocType     `json:"doc_type"`
	RowCount   int         `json:"row_count"`
	FieldCount int         `json:"field_count"`
	Fields     []FieldInfo `json:"fields"`
	Issues     []string    `json:"issues"`
}

var (
	emailRe = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	urlRe   = regexp.MustCompile(`^https?://`)
	dateRe  = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}|^\d{2}[-/]\d{2}[-/]\d{4}|^\d{2}-[A-Za-z]{3}-\d{4}|^[A-Za-z]+ \d{1,2},? \d{4}`)
)

// InferSchema analyzes document content and returns a schema without using an LLM.
func InferSchema(content string) *InferredSchema {
	docType := DetectType(content)
	switch docType {
	case DocTypeCSV:
		return inferCSVSchema(content)
	case DocTypeJSON:
		return inferJSONSchema(content)
	default:
		return &InferredSchema{DocType: docType, Issues: []string{"Server-side schema inference not available for this document type — Claude will handle it."}}
	}
}

func inferCSVSchema(content string) *InferredSchema {
	r := csv.NewReader(strings.NewReader(content))
	r.TrimLeadingSpace = true
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil || len(records) < 2 {
		return &InferredSchema{DocType: DocTypeCSV, Issues: []string{"Could not parse CSV"}}
	}

	headers := records[0]
	rows := records[1:]
	schema := &InferredSchema{
		DocType:    DocTypeCSV,
		RowCount:   len(rows),
		FieldCount: len(headers),
		Fields:     make([]FieldInfo, len(headers)),
	}

	for i, h := range headers {
		h = strings.TrimSpace(h)
		if h == "" {
			h = "column_" + strconv.Itoa(i+1)
			schema.Issues = append(schema.Issues, "Column "+strconv.Itoa(i+1)+" has no header name")
		}

		field := FieldInfo{Name: h}
		values := make([]string, 0, len(rows))
		seen := make(map[string]bool)

		for _, row := range rows {
			val := ""
			if i < len(row) {
				val = strings.TrimSpace(row[i])
			}
			values = append(values, val)
			if val == "" {
				field.EmptyCount++
			} else {
				seen[val] = true
			}
		}

		field.Nullable = field.EmptyCount > 0
		field.Unique = len(seen) == len(rows)-field.EmptyCount && len(seen) > 0
		field.Type = detectFieldType(values)

		// Sample values (up to 3 non-empty)
		for _, v := range values {
			if v != "" && len(field.SampleVals) < 3 {
				field.SampleVals = append(field.SampleVals, v)
			}
		}

		// Quality issues
		if field.EmptyCount > 0 {
			pct := field.EmptyCount * 100 / len(rows)
			schema.Issues = append(schema.Issues, field.Name+": "+strconv.Itoa(field.EmptyCount)+" empty values ("+strconv.Itoa(pct)+"%)")
		}
		if field.Type == TypeMixed {
			schema.Issues = append(schema.Issues, field.Name+": mixed types detected — inconsistent data")
		}

		schema.Fields[i] = field
	}

	return schema
}

func inferJSONSchema(content string) *InferredSchema {
	trimmed := strings.TrimSpace(content)
	schema := &InferredSchema{DocType: DocTypeJSON}

	// Try array of objects
	var arr []map[string]any
	if json.Unmarshal([]byte(trimmed), &arr) == nil && len(arr) > 0 {
		schema.RowCount = len(arr)
		fieldMap := make(map[string][]any)
		for _, obj := range arr {
			for k, v := range obj {
				fieldMap[k] = append(fieldMap[k], v)
			}
		}
		schema.FieldCount = len(fieldMap)
		for name, vals := range fieldMap {
			field := FieldInfo{Name: name}
			strVals := make([]string, len(vals))
			for i, v := range vals {
				if v == nil {
					field.EmptyCount++
					strVals[i] = ""
				} else {
					b, _ := json.Marshal(v)
					strVals[i] = string(b)
				}
			}
			field.Nullable = field.EmptyCount > 0
			field.Type = detectJSONFieldType(vals)
			for _, v := range strVals {
				if v != "" && len(field.SampleVals) < 3 {
					field.SampleVals = append(field.SampleVals, v)
				}
			}
			schema.Fields = append(schema.Fields, field)
		}
		return schema
	}

	// Try single object
	var obj map[string]any
	if json.Unmarshal([]byte(trimmed), &obj) == nil {
		schema.RowCount = 1
		schema.FieldCount = len(obj)
		for name, val := range obj {
			field := FieldInfo{Name: name}
			field.Type = detectJSONValueType(val)
			b, _ := json.Marshal(val)
			s := string(b)
			if len(s) > 100 {
				s = s[:100] + "..."
			}
			field.SampleVals = []string{s}
			schema.Fields = append(schema.Fields, field)
		}
		return schema
	}

	schema.Issues = []string{"Could not parse JSON structure"}
	return schema
}

func detectFieldType(values []string) FieldType {
	types := map[FieldType]int{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || strings.EqualFold(v, "null") || strings.EqualFold(v, "n/a") || strings.EqualFold(v, "not_available") {
			continue
		}
		types[classifyValue(v)]++
	}
	if len(types) == 0 {
		return TypeNull
	}
	if len(types) == 1 {
		for t := range types {
			return t
		}
	}
	// Number + String mix where string might be currency ($85,000)
	if types[TypeNumber] > 0 && types[TypeString] > 0 {
		return TypeMixed
	}
	return TypeMixed
}

func classifyValue(v string) FieldType {
	if emailRe.MatchString(v) {
		return TypeEmail
	}
	if urlRe.MatchString(v) {
		return TypeURL
	}
	if dateRe.MatchString(v) {
		return TypeDate
	}
	if strings.EqualFold(v, "true") || strings.EqualFold(v, "false") {
		return TypeBoolean
	}
	// Strip currency symbols and commas for number detection
	cleaned := strings.ReplaceAll(strings.TrimPrefix(strings.TrimPrefix(v, "$"), "€"), ",", "")
	if _, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return TypeNumber
	}
	return TypeString
}

func detectJSONFieldType(vals []any) FieldType {
	types := map[FieldType]int{}
	for _, v := range vals {
		types[detectJSONValueType(v)]++
	}
	if len(types) == 1 {
		for t := range types {
			return t
		}
	}
	return TypeMixed
}

func detectJSONValueType(v any) FieldType {
	switch v.(type) {
	case float64:
		return TypeNumber
	case bool:
		return TypeBoolean
	case string:
		return TypeString
	case nil:
		return TypeNull
	default:
		return TypeString // arrays, objects → treat as string
	}
}
