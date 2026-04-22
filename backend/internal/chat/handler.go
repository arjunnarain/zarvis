// Package chat wires HTTP requests to the Anthropic streaming API with tool execution.
package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/go-chi/chi/v5"

	"github.com/zarvis/internal/auth"
	"github.com/zarvis/internal/badges"
	"github.com/zarvis/internal/mcp"
	"github.com/zarvis/internal/prompt"
	"github.com/zarvis/internal/search"
	"github.com/zarvis/internal/state"
	"github.com/zarvis/internal/tools"
)

type Handler struct {
	Anthropic *anthropic.Client
	Store     state.Store
	Registry  *mcp.Registry
	Search    *search.Engine
	Prompt    *prompt.Builder
	Badges    *badges.Engine
	Tools     *tools.Executor
	JWTSecret string
}

// Auth endpoints

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	if len(req.Password) < 6 {
		http.Error(w, `{"error":"password must be at least 6 characters"}`, http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}
	user, err := h.Store.CreateUser(req.Email, hash, req.Name)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, `{"error":"email already registered"}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	token, _ := auth.GenerateToken(user.ID, h.JWTSecret)
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": user})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"email and password required"}`, http.StatusBadRequest)
		return
	}
	user, err := h.Store.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		http.Error(w, `{"error":"invalid email or password"}`, http.StatusUnauthorized)
		return
	}
	token, _ := auth.GenerateToken(user.ID, h.JWTSecret)
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	user, err := h.Store.GetUserByID(userID)
	if err != nil {
		http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Session endpoints

type createSessionReq struct {
	PrimaryAnimal string `json:"primary_animal"`
}

func (h *Handler) CreateSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	userID := auth.GetUserID(r)
	sess, err := h.Store.CreateSession(userID, req.PrimaryAnimal)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.Store.GetSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

type updateSessionReq struct {
	PrimaryAnimal string `json:"primary_animal"`
}

func (h *Handler) UpdateSession(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sess, err := h.Store.GetSession(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	var req updateSessionReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.PrimaryAnimal != "" {
		sess.PrimaryAnimal = req.PrimaryAnimal
	}
	if err := h.Store.UpdateSession(sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handler) GetBadges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	b, err := h.Store.GetBadges(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// Upload handles multipart file uploads.
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	sessionID := r.FormValue("session_id")
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 10*1024*1024)) // 10MB limit
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	content := string(data)

	// PDF extraction via pdftotext
	if strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		extracted, extractErr := extractPDFText(data)
		if extractErr != nil {
			http.Error(w, "PDF extraction failed: "+extractErr.Error(), http.StatusUnprocessableEntity)
			return
		}
		content = extracted
	}

	doc, err := h.Store.SaveDocument(sessionID, header.Filename, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Badges.CheckAndAward(sessionID, "upload")
	writeJSON(w, http.StatusCreated, doc)
}

// GetDocument returns the latest document with structured data.
func (h *Handler) GetDocument(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	doc, err := h.Store.GetLatestDocument(sessionID)
	if err != nil {
		http.Error(w, "no document", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

// ListDocuments returns all documents for a session (without raw content for brevity).
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	docs, err := h.Store.ListDocuments(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if docs == nil {
		docs = []state.Document{}
	}
	writeJSON(w, http.StatusOK, docs)
}

// Forest endpoints

func (h *Handler) CreateForest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	f, err := h.Store.CreateForest(req.SessionID, req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, f)
}

func (h *Handler) ListForests(w http.ResponseWriter, r *http.Request) {
	sid := chi.URLParam(r, "id")
	forests, err := h.Store.ListForests(sid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if forests == nil {
		forests = []state.Forest{}
	}
	writeJSON(w, http.StatusOK, forests)
}

func (h *Handler) AddDocToForest(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		DocumentID int64 `json:"document_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DocumentID == 0 {
		http.Error(w, "document_id required", http.StatusBadRequest)
		return
	}
	if err := h.Store.AddDocumentToForest(forestID, req.DocumentID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Auto-index for BM25 search
	if h.Search != nil {
		doc, err := h.Store.GetDocument(req.DocumentID)
		if err == nil {
			content := doc.RawContent
			if doc.StructuredJSON != "" {
				content = doc.StructuredJSON
			}
			_ = h.Search.IndexDocument(req.DocumentID, forestID, content)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "added"})
}

func (h *Handler) ClearForest(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.Store.ClearForest(forestID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cleared"})
}

func (h *Handler) GetForestDocs(w http.ResponseWriter, r *http.Request) {
	forestID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	docs, err := h.Store.GetForestDocuments(forestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if docs == nil {
		docs = []state.Document{}
	}
	writeJSON(w, http.StatusOK, docs)
}

type chatReq struct {
	SessionID string `json:"session_id"`
	Module    string `json:"module"`
	Message   string `json:"message"`
}

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

		for _, block := range msg.Content {
			if tu, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				toolCalls = append(toolCalls, toolCall{ID: tu.ID, Name: tu.Name, Input: tu.Input})
			}
		}

		if assistantText != "" {
			_ = h.Store.AppendMessage(sess.ID, req.Module, "assistant", assistantText)
		}

		if len(toolCalls) == 0 {
			break
		}

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

// extractPDFText uses pdftotext to convert PDF bytes to plain text.
func extractPDFText(data []byte) (string, error) {
	tmpFile, err := os.CreateTemp("", "zarvis-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	outFile := filepath.Join(os.TempDir(), "zarvis-out.txt")
	defer os.Remove(outFile)

	cmd := exec.Command("pdftotext", "-layout", tmpFile.Name(), outFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pdftotext: %s (%w)", string(out), err)
	}
	text, err := os.ReadFile(outFile)
	if err != nil {
		return "", err
	}
	return string(text), nil
}

// LoadSample creates a document from a built-in sample.
func (h *Handler) LoadSample(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Sample    string `json:"sample"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	content, filename := getSampleData(req.Sample)
	if content == "" {
		http.Error(w, "unknown sample", http.StatusBadRequest)
		return
	}

	doc, err := h.Store.SaveDocument(req.SessionID, filename, content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Badges.CheckAndAward(req.SessionID, "upload")
	writeJSON(w, http.StatusCreated, doc)
}

func getSampleData(name string) (string, string) {
	switch name {
	case "messy_csv":
		return `Name,  Age, City,  Salary, Start Date, Email
Alice Smith, 30, New York, $85,000, 01/15/2023, alice@company.com
Bob Johnson,  25,San Francisco,92000, 2023-03-22, bob@company.com
Charlie Brown, 35, Chicago, $78000,  March 1 2022, charlie@company
Diana Prince, , NYC, 95000.00, 15-Jan-2024, diana@company.com
Eve Wilson, 32, SF, $88,000, 2023/07/10,
Frank Miller, 28, New York, eighty thousand, 01-2023, frank@company.com
, 45, Boston, $102,000, 2022-06-01, grace@company.com
Henry Lee, 31, Chicago, 76000, 04/20/2023, henry@company.com
Ivy Chen, NOT_AVAILABLE, San Francisco, $91,000, 2023-08-15, ivy@company.com
Jack Davis, 29, NYC, $87500, Aug 2023, jack@company.com`, "messy_employees.csv"

	case "invoice":
		return `INVOICE #INV-2024-0847

From: Acme Cloud Services LLC
      123 Tech Boulevard, Suite 400
      San Francisco, CA 94105
      Tax ID: 82-1234567

To:   GlobalCorp Industries
      456 Enterprise Ave
      New York, NY 10001
      Attn: Accounts Payable

Date: March 15, 2024
Due:  April 14, 2024
PO#:  GC-2024-1122

Description                          Qty    Unit Price    Amount
-------------------------------------------------------------
Cloud Compute (m5.xlarge)            720 hrs   $0.192    $138.24
Cloud Compute (c5.2xlarge)           360 hrs   $0.340    $122.40
Object Storage (S3)                  2.4 TB    $23.00/TB  $55.20
Data Transfer (outbound)             850 GB    $0.09/GB   $76.50
Managed Database (db.r5.large)       720 hrs   $0.250    $180.00
Load Balancer                          1 mo    $18.00     $18.00
SSL Certificate                        3       $0.00       FREE
Premium Support                        1 mo   $100.00    $100.00
-------------------------------------------------------------
                                     Subtotal:           $690.34
                                     Tax (8.875%):        $61.27
                                     TOTAL:              $751.61

Payment Terms: Net 30
Wire to: Chase Bank, Acct# 4455667788, Routing# 021000021

Notes: Usage period Feb 1 - Feb 29, 2024. Dispute window: 15 days.
Thank you for your business!`, "invoice_INV-2024-0847.txt"

	case "server_log":
		return `[2024-03-15 08:23:14.332] INFO  server started on :8080 pid=14523
[2024-03-15 08:23:14.335] INFO  connected to postgres://db.internal:5432/prod
[2024-03-15 08:23:15.001] INFO  GET /api/health 200 2ms
[2024-03-15 08:24:01.445] INFO  POST /api/users 201 45ms user_id=usr_8834
[2024-03-15 08:24:02.112] WARN  rate limit approaching for ip=203.0.113.42 (78/100)
[2024-03-15 08:24:15.667] INFO  GET /api/users/usr_8834 200 12ms
[2024-03-15 08:25:33.891] ERROR POST /api/payments 500 234ms err="stripe: card_declined" user_id=usr_7721 amount=49.99
[2024-03-15 08:25:34.002] WARN  payment retry scheduled for user_id=usr_7721 attempt=1
[2024-03-15 08:26:01.123] INFO  GET /api/products 200 8ms count=47
[2024-03-15 08:26:45.556] INFO  POST /api/orders 201 89ms order_id=ord_9912 user_id=usr_8834 total=127.50
[2024-03-15 08:27:00.001] INFO  cron: cleanup expired sessions removed=23
[2024-03-15 08:27:12.334] ERROR GET /api/users/usr_0000 404 3ms err="user not found"
[2024-03-15 08:28:01.778] INFO  POST /api/auth/login 200 156ms user_id=usr_5543
[2024-03-15 08:28:45.223] WARN  disk usage at 82% on /data
[2024-03-15 08:29:11.445] ERROR POST /api/payments 500 1023ms err="stripe: timeout" user_id=usr_5543 amount=299.00
[2024-03-15 08:29:11.890] ERROR payment gateway timeout, circuit breaker OPEN
[2024-03-15 08:29:30.001] WARN  circuit breaker: rejecting payment requests (open for 30s)
[2024-03-15 08:30:01.112] INFO  circuit breaker: HALF-OPEN, testing...
[2024-03-15 08:30:02.334] INFO  circuit breaker: CLOSED, payments resumed
[2024-03-15 08:30:15.667] INFO  POST /api/payments 200 89ms user_id=usr_5543 amount=299.00
[2024-03-15 08:31:00.001] INFO  metrics: rps=124 p50=12ms p99=234ms errors=3/1847`, "server_access.log"

	case "api_response":
		return `{
  "status": "success",
  "metadata": {"page": 1, "per_page": 5, "total": 127, "generated_at": "2024-03-15T10:30:00Z"},
  "data": {
    "company": "TechStartup Inc",
    "departments": [
      {
        "name": "Engineering",
        "head": "Sarah Connor",
        "budget": 2400000,
        "headcount": 45,
        "teams": [
          {"name": "Platform", "members": 12, "projects": ["Infrastructure Migration", "API Gateway v2"]},
          {"name": "Product", "members": 18, "projects": ["Mobile App Redesign", "Checkout Flow", "Search Improvements"]},
          {"name": "Data", "members": 8, "projects": ["ML Pipeline", "Analytics Dashboard"]},
          {"name": "DevOps", "members": 7, "projects": ["K8s Migration", "CI/CD Overhaul", "Monitoring v3"]}
        ]
      },
      {
        "name": "Sales",
        "head": "Mike Ross",
        "budget": 1800000,
        "headcount": 32,
        "teams": [
          {"name": "Enterprise", "members": 14, "projects": ["Q1 Pipeline", "APAC Expansion"]},
          {"name": "SMB", "members": 10, "projects": ["Self-serve Onboarding", "Pricing Revamp"]},
          {"name": "Partnerships", "members": 8, "projects": ["Reseller Program", "API Marketplace"]}
        ]
      },
      {
        "name": "Marketing",
        "head": "Lisa Zhang",
        "budget": 950000,
        "headcount": 15,
        "teams": [
          {"name": "Growth", "members": 6, "projects": ["SEO Overhaul", "Paid Acquisition"]},
          {"name": "Content", "members": 5, "projects": ["Blog Relaunch", "Case Studies"]},
          {"name": "Brand", "members": 4, "projects": ["Rebrand 2024", "Conference Strategy"]}
        ]
      }
    ],
    "financials": {
      "arr": 12500000,
      "mrr": 1041667,
      "growth_rate": 0.23,
      "burn_rate": 485000,
      "runway_months": 18
    }
  }
}`, "company_data.json"

	default:
		return "", ""
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
