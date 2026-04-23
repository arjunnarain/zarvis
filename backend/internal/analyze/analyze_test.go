package analyze

import "testing"

func TestDetectType_CSV(t *testing.T) {
	content := "name,age,city\nAlice,30,NYC\nBob,25,SF"
	if got := DetectType(content); got != DocTypeCSV {
		t.Errorf("DetectType = %q, want csv", got)
	}
}

func TestDetectType_JSON_Object(t *testing.T) {
	if got := DetectType(`{"name": "Alice", "age": 30}`); got != DocTypeJSON {
		t.Errorf("DetectType = %q, want json", got)
	}
}

func TestDetectType_JSON_Array(t *testing.T) {
	if got := DetectType(`[{"a":1},{"a":2}]`); got != DocTypeJSON {
		t.Errorf("DetectType = %q, want json", got)
	}
}

func TestDetectType_LogFile(t *testing.T) {
	content := "[2024-03-15 08:23:14] INFO started\n[2024-03-15 08:23:15] ERROR failed\n[2024-03-15 08:23:16] WARN slow"
	if got := DetectType(content); got != DocTypeLogFile {
		t.Errorf("DetectType = %q, want log", got)
	}
}

func TestDetectType_XML(t *testing.T) {
	if got := DetectType(`<?xml version="1.0"?><root><item>test</item></root>`); got != DocTypeXML {
		t.Errorf("DetectType = %q, want xml", got)
	}
}

func TestDetectType_Markdown(t *testing.T) {
	content := "# Title\n\nSome text\n\n## Section\n\nMore text"
	if got := DetectType(content); got != DocTypeMarkdown {
		t.Errorf("DetectType = %q, want markdown", got)
	}
}

func TestDetectType_Empty(t *testing.T) {
	if got := DetectType(""); got != DocTypeUnknown {
		t.Errorf("DetectType('') = %q, want unknown", got)
	}
}

func TestInferSchema_CSV(t *testing.T) {
	content := "name,age,email\nAlice,30,alice@test.com\nBob,25,bob@test.com\n,35,"
	schema := InferSchema(content)

	if schema.DocType != DocTypeCSV {
		t.Fatalf("DocType = %q, want csv", schema.DocType)
	}
	if schema.RowCount != 3 {
		t.Errorf("RowCount = %d, want 3", schema.RowCount)
	}
	if schema.FieldCount != 3 {
		t.Errorf("FieldCount = %d, want 3", schema.FieldCount)
	}
	if !schema.Fields[0].Nullable {
		t.Error("name should be nullable")
	}
	if schema.Fields[2].Type != TypeEmail {
		t.Errorf("email type = %q, want email", schema.Fields[2].Type)
	}
	if len(schema.Issues) == 0 {
		t.Error("expected data quality issues")
	}
}

func TestInferSchema_JSON_Array(t *testing.T) {
	schema := InferSchema(`[{"name":"Alice","age":30},{"name":"Bob","age":25}]`)
	if schema.DocType != DocTypeJSON {
		t.Fatalf("DocType = %q, want json", schema.DocType)
	}
	if schema.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", schema.RowCount)
	}
}

func TestClassifyValue(t *testing.T) {
	tests := []struct {
		input string
		want  FieldType
	}{
		{"hello", TypeString},
		{"42", TypeNumber},
		{"3.14", TypeNumber},
		{"$85000", TypeNumber},
		{"true", TypeBoolean},
		{"alice@test.com", TypeEmail},
		{"https://example.com", TypeURL},
		{"2024-03-15", TypeDate},
		{"01/15/2023", TypeDate},
	}
	for _, tt := range tests {
		got := classifyValue(tt.input)
		if got != tt.want {
			t.Errorf("classifyValue(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestCleanAnalysis(t *testing.T) {
	content := "name,salary\nAlice,85000\nBob,92000\n,78000"
	schema := InferSchema(content)
	report := CleanAnalysis(content, schema)

	if report.EmptyValues == 0 {
		t.Error("should detect empty values")
	}
	if len(report.Suggestions) == 0 {
		t.Error("should have suggestions")
	}
}
