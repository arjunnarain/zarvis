// Package mcp manages the developer tool registry.
package mcp

import (
	"encoding/json"
	"os"
)

type Tool struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Description string          `json:"description"`
	Modules     []string        `json:"modules"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type Registry struct {
	Tools []Tool `json:"tools"`
}

func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func (r *Registry) ToolsForModule(module string) []Tool {
	out := make([]Tool, 0)
	for _, t := range r.Tools {
		for _, m := range t.Modules {
			if m == module {
				out = append(out, t)
				break
			}
		}
	}
	return out
}
