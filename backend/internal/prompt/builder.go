// Package prompt loads module-specific system prompts.
package prompt

import (
	"os"
	"strings"
)

type Builder struct {
	prompts map[string]string
}

func NewBuilder(promptDir string) (*Builder, error) {
	modules := []string{"explorer", "table", "schema", "summary", "graphs", "oracle"}
	b := &Builder{prompts: make(map[string]string)}

	for _, mod := range modules {
		path := promptDir + "/" + mod + ".md"
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		text := string(raw)
		if idx := strings.Index(text, "\n---\n"); idx != -1 {
			text = text[idx+5:]
		}
		b.prompts[mod] = text
	}
	return b, nil
}

func (b *Builder) Build(module string) string {
	if p, ok := b.prompts[module]; ok {
		return p
	}
	return b.prompts["explorer"]
}
