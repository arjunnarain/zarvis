package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/zarvis/internal/state"
)

// Upload handles multipart file uploads with PDF extraction support.
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

// ListDocuments returns all documents for a session.
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

// LoadSample creates a document from a built-in sample dataset.
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
Wire to: Chase Bank, Acct# 4455667788, Routing# 021000021`, "invoice_INV-2024-0847.txt"

	case "server_log":
		return `[2024-03-15 08:23:14.332] INFO  server started on :8080 pid=14523
[2024-03-15 08:23:14.335] INFO  connected to postgres://db.internal:5432/prod
[2024-03-15 08:24:01.445] INFO  POST /api/users 201 45ms user_id=usr_8834
[2024-03-15 08:24:02.112] WARN  rate limit approaching for ip=203.0.113.42 (78/100)
[2024-03-15 08:25:33.891] ERROR POST /api/payments 500 234ms err="stripe: card_declined" user_id=usr_7721 amount=49.99
[2024-03-15 08:25:34.002] WARN  payment retry scheduled for user_id=usr_7721 attempt=1
[2024-03-15 08:26:01.123] INFO  GET /api/products 200 8ms count=47
[2024-03-15 08:27:12.334] ERROR GET /api/users/usr_0000 404 3ms err="user not found"
[2024-03-15 08:29:11.445] ERROR POST /api/payments 500 1023ms err="stripe: timeout" user_id=usr_5543 amount=299.00
[2024-03-15 08:29:11.890] ERROR payment gateway timeout, circuit breaker OPEN
[2024-03-15 08:30:02.334] INFO  circuit breaker: CLOSED, payments resumed
[2024-03-15 08:30:15.667] INFO  POST /api/payments 200 89ms user_id=usr_5543 amount=299.00
[2024-03-15 08:31:00.001] INFO  metrics: rps=124 p50=12ms p99=234ms errors=3/1847`, "server_access.log"

	case "api_response":
		return `{
  "status": "success",
  "metadata": {"page": 1, "total": 127, "generated_at": "2024-03-15T10:30:00Z"},
  "data": {
    "company": "TechStartup Inc",
    "departments": [
      {"name": "Engineering", "head": "Sarah Connor", "budget": 2400000, "headcount": 45,
       "teams": [{"name": "Platform", "members": 12}, {"name": "Product", "members": 18}, {"name": "Data", "members": 8}, {"name": "DevOps", "members": 7}]},
      {"name": "Sales", "head": "Mike Ross", "budget": 1800000, "headcount": 32,
       "teams": [{"name": "Enterprise", "members": 14}, {"name": "SMB", "members": 10}, {"name": "Partnerships", "members": 8}]},
      {"name": "Marketing", "head": "Lisa Zhang", "budget": 950000, "headcount": 15,
       "teams": [{"name": "Growth", "members": 6}, {"name": "Content", "members": 5}, {"name": "Brand", "members": 4}]}
    ],
    "financials": {"arr": 12500000, "mrr": 1041667, "growth_rate": 0.23, "burn_rate": 485000, "runway_months": 18}
  }
}`, "company_data.json"

	default:
		return "", ""
	}
}
