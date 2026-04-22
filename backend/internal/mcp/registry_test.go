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

func TestToolsForModule_Devlog(t *testing.T) {
	r := setupTestRegistry(t)
	tools := r.ToolsForModule("devlog")
	if len(tools) == 0 {
		t.Fatal("devlog module should have tools")
	}
	for _, tool := range tools {
		found := false
		for _, m := range tool.Modules {
			if m == "devlog" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %s should not be in devlog module", tool.Name)
		}
	}
}

func TestToolsForModule_Github(t *testing.T) {
	r := setupTestRegistry(t)
	tools := r.ToolsForModule("github")
	if len(tools) == 0 {
		t.Fatal("github module should have tools")
	}
	hasGHPRs := false
	for _, tool := range tools {
		if tool.Name == "github_prs" {
			hasGHPRs = true
		}
	}
	if !hasGHPRs {
		t.Error("github module should include github_prs tool")
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
