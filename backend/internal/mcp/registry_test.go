package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestRegistry(t *testing.T) *Registry {
	t.Helper()
	root := filepath.Join("..", "..", "..", "docs", "mcp_tools.json")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("mcp_tools.json not found at %s: %v", root, err)
	}
	r, err := LoadRegistry(root)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	return r
}

func TestLoadRegistry_HasTools(t *testing.T) {
	r := setupTestRegistry(t)
	if len(r.Tools) == 0 {
		t.Fatal("expected tools, got none")
	}
}

func TestToolsForModule_Explorer(t *testing.T) {
	r := setupTestRegistry(t)
	tools := r.ToolsForModule("explorer")
	if len(tools) == 0 {
		t.Fatal("explorer module should have tools")
	}
	for _, tool := range tools {
		found := false
		for _, m := range tool.Modules {
			if m == "explorer" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %s should not be in explorer module", tool.Name)
		}
	}
}

func TestToolsForModule_Oracle(t *testing.T) {
	r := setupTestRegistry(t)
	tools := r.ToolsForModule("oracle")
	if len(tools) == 0 {
		t.Fatal("oracle module should have tools")
	}
	hasForest := false
	for _, tool := range tools {
		if tool.Name == "get_forest_documents" {
			hasForest = true
		}
	}
	if !hasForest {
		t.Error("oracle module should include get_forest_documents tool")
	}
}

func TestToolsForModule_Unknown_ReturnsEmpty(t *testing.T) {
	r := setupTestRegistry(t)
	tools := r.ToolsForModule("nonexistent")
	if len(tools) != 0 {
		t.Errorf("unknown module should return 0 tools, got %d", len(tools))
	}
}

func TestToolsHaveRequiredFields(t *testing.T) {
	r := setupTestRegistry(t)
	for _, tool := range r.Tools {
		if tool.Name == "" {
			t.Error("tool has empty name")
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %s has empty display_name", tool.Name)
		}
		if tool.Description == "" {
			t.Errorf("tool %s has empty description", tool.Name)
		}
		if len(tool.Modules) == 0 {
			t.Errorf("tool %s has no modules", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %s has empty input_schema", tool.Name)
		}
	}
}

func TestAllModulesHaveTools(t *testing.T) {
	r := setupTestRegistry(t)
	modules := []string{"explorer", "table", "schema", "summary", "oracle"}
	for _, mod := range modules {
		tools := r.ToolsForModule(mod)
		if len(tools) == 0 {
			t.Errorf("module %s should have at least one tool", mod)
		}
	}
}
