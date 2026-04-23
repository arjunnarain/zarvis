package chat

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"

	"github.com/zarvis/internal/mcp"
)

type chatReq struct {
	SessionID string `json:"session_id"`
	Module    string `json:"module"`
	Message   string `json:"message"`
}

// Chat handles streaming conversation with Claude, including tool execution.
func (h *Handler) Chat(w http.ResponseWriter, r *http.Request) {
	var req chatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.Module == "" {
		req.Module = "explorer"
	}

	sess, err := h.Store.GetSession(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	emit := func(eventType string, payload any) {
		data, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data)
		flusher.Flush()
	}

	_ = h.Store.AppendMessage(sess.ID, req.Module, "user", req.Message)

	systemPrompt := h.Prompt.Build(req.Module)
	messages, err := h.buildMessageHistory(sess.ID, req.Module)
	if err != nil {
		emit("error", map[string]string{"message": err.Error()})
		return
	}

	moduleTools := h.Registry.ToolsForModule(req.Module)
	anthropicTools := toAnthropicToolParams(moduleTools)

	// Multi-turn tool execution loop — up to 5 rounds of tool calls
	for attempt := range 5 {
		_ = attempt
		var assistantText string
		var toolCalls []toolCall

		stream := h.Anthropic.Messages.NewStreaming(r.Context(), anthropic.MessageNewParams{
			Model:     "claude-haiku-4-5-20251001",
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  messages,
			Tools:     anthropicTools,
		})

		msg := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			_ = msg.Accumulate(event)

			switch evt := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				if delta, ok := evt.Delta.AsAny().(anthropic.TextDelta); ok {
					assistantText += delta.Text
					emit("delta", map[string]string{"text": delta.Text})
				}
			case anthropic.ContentBlockStartEvent:
				if tu, ok := evt.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
					displayName := tu.Name
					for _, t := range moduleTools {
						if t.Name == tu.Name {
							displayName = t.DisplayName
							break
						}
					}
					emit("tool_use", map[string]any{"tool": tu.Name, "display_name": displayName, "id": tu.ID})
				}
			}
		}
		if err := stream.Err(); err != nil {
			log.Printf("anthropic stream error: %v", err)
			emit("error", map[string]string{"message": err.Error()})
			return
		}

		// Collect tool calls from the accumulated message
		for _, block := range msg.Content {
			if tu, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				toolCalls = append(toolCalls, toolCall{ID: tu.ID, Name: tu.Name, Input: tu.Input})
			}
		}

		if assistantText != "" {
			_ = h.Store.AppendMessage(sess.ID, req.Module, "assistant", assistantText)
		}

		// No tool calls = conversation complete
		if len(toolCalls) == 0 {
			break
		}

		// Execute tools and build continuation message
		messages = append(messages, msg.ToParam())
		var toolResults []anthropic.ContentBlockParamUnion
		for _, tc := range toolCalls {
			result := h.Tools.Execute(r.Context(), sess.ID, tc.Name, tc.Input)

			newBadges := h.Badges.CheckAndAward(sess.ID, tc.Name)
			for _, bk := range newBadges {
				emit("badge", map[string]string{"badge_key": bk})
			}

			output := result.Output
			isError := false
			if result.Error != "" {
				output = result.Error
				isError = true
			}
			emit("tool_result", map[string]any{"tool": tc.Name, "output": output, "error": isError})
			toolResults = append(toolResults, anthropic.NewToolResultBlock(tc.ID, output, isError))
		}
		messages = append(messages, anthropic.NewUserMessage(toolResults...))
	}

	emit("done", map[string]string{"status": "ok"})
}

type toolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

func (h *Handler) buildMessageHistory(sessionID, module string) ([]anthropic.MessageParam, error) {
	msgs, err := h.Store.RecentMessages(sessionID, module, 20)
	if err != nil {
		return nil, err
	}
	out := make([]anthropic.MessageParam, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			out = append(out, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			out = append(out, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
		}
	}
	return out, nil
}

func toAnthropicToolParams(tools []mcp.Tool) []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var raw map[string]any
		_ = json.Unmarshal(t.InputSchema, &raw)
		schema := anthropic.ToolInputSchemaParam{Properties: raw["properties"]}
		if req, ok := raw["required"]; ok {
			schema.ExtraFields = map[string]any{"required": req}
		}
		out = append(out, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: schema,
			},
		})
	}
	return out
}
