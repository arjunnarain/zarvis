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

	"github.com/zarvis/internal/analyze"
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

	// Server-side analysis: detect type, infer schema, run data quality check
	preAnalysis := analyze.InferSchema(content)
	cleanReport := analyze.CleanAnalysis(content, preAnalysis)
	analysisJSON, _ := json.Marshal(map[string]any{
		"doc_type":     preAnalysis.DocType,
		"row_count":    preAnalysis.RowCount,
		"field_count":  preAnalysis.FieldCount,
		"fields":       preAnalysis.Fields,
		"issues":       preAnalysis.Issues,
		"clean_report": cleanReport,
	})
	// Store pre-analysis as initial schema
	_ = h.Store.UpdateDocumentStructured(doc.ID, "", string(analysisJSON), "")

	h.Badges.CheckAndAward(sessionID, "upload")

	// Compute quality score
	quality := analyze.ComputeQualityScore(content, "")

	response := map[string]any{
		"id":           doc.ID,
		"session_id":   doc.SessionID,
		"filename":     doc.Filename,
		"created_at":   doc.CreatedAt,
		"pre_analysis": preAnalysis,
		"quality":      quality,
	}
	writeJSON(w, http.StatusCreated, response)
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

	case "support_tickets":
		return `From: sarah.chen@globalcorp.com
Date: 2024-03-12 09:15 AM
Subject: URGENT - Payment gateway down
Priority: P1
Assigned: DevOps Team

Our payment processing is completely down since 8:45 AM. Customers getting 500 errors on checkout. Revenue impact ~$12,000/hour. Need immediate fix.

Status: RESOLVED (10:02 AM) - Root cause: expired SSL cert on payment-gateway-prod-3

---

From: mike.j@globalcorp.com
Date: 2024-03-12 11:30 AM
Subject: Dashboard loading slowly
Priority: P3
Assigned: Frontend Team

The analytics dashboard takes 15+ seconds to load. Started after yesterday's deploy (v2.4.1). Only affects pages with >1000 data points.

Status: IN PROGRESS - Identified N+1 query in chart renderer

---

From: lisa.wang@globalcorp.com
Date: 2024-03-13 02:20 PM
Subject: User can't reset password
Priority: P2
Assigned: Auth Team

Customer (acct #88412) reports password reset emails not arriving. Checked - emails ARE being sent but caught by their spam filter. Same issue reported by 3 other enterprise accounts this week.

Status: OPEN - Need to investigate DKIM/SPF records

---

From: raj.patel@globalcorp.com
Date: 2024-03-13 04:45 PM
Subject: API rate limiting too aggressive
Priority: P2
Assigned: Platform Team

Enterprise customer "DataFlow Inc" hitting rate limits at 50 req/s. Their contract allows 200 req/s. Config shows default tier applied instead of enterprise tier. Affecting their production pipeline.

Status: RESOLVED - Updated rate limit config, applied enterprise tier

---

From: emma.wilson@globalcorp.com
Date: 2024-03-14 08:00 AM
Subject: Data export missing columns
Priority: P3
Assigned: Backend Team

CSV export from the reports page is missing the "region" and "channel" columns. These were added in the Jan release but never added to the export query. 5 customers have reported this.

Status: OPEN - Backlog, scheduled for sprint 24.`, "support_tickets_march2024.txt"

	case "bank_statement":
		return `ACME CORP - BUSINESS CHECKING
Account: ****4521
Period: March 1 - March 31, 2024
Opening Balance: $45,231.88

Date	Description	Debit	Credit	Balance
03/01	AWS Monthly Invoice	$2,847.33		$42,384.55
03/01	Stripe Payout		$8,412.00	$50,796.55
03/03	Gusto Payroll - Biweekly	$28,450.00		$22,346.55
03/04	WeWork Office Rent	$4,200.00		$18,146.55
03/05	Google Cloud Platform	$1,233.67		$16,912.88
03/05	Customer Payment - Invoice #1847		$15,000.00	$31,912.88
03/07	Stripe Payout		$6,830.50	$38,743.38
03/08	Adobe Creative Cloud	$599.88		$38,143.50
03/10	Slack Enterprise	$1,250.00		$36,893.50
03/11	Customer Payment - Invoice #1852		$22,500.00	$59,393.50
03/12	GitHub Enterprise	$441.00		$58,952.50
03/14	Stripe Payout		$9,102.33	$68,054.83
03/15	Insurance - Liability	$1,875.00		$66,179.83
03/17	Gusto Payroll - Biweekly	$28,450.00		$37,729.83
03/18	Customer Payment - Invoice #1860		$18,750.00	$56,479.83
03/19	DataDog Monitoring	$890.00		$55,589.83
03/20	Vercel Pro Plan	$200.00		$55,389.83
03/21	Stripe Payout		$11,445.67	$66,835.50
03/22	Tax Payment - Q1 Estimated	$12,000.00		$54,835.50
03/25	Customer Payment - Invoice #1865		$9,800.00	$64,635.50
03/28	Stripe Payout		$7,233.00	$71,868.50
03/29	Notion Team Plan	$480.00		$71,388.50
03/31	Gusto Payroll - Biweekly	$28,450.00		$42,938.50

Closing Balance: $42,938.50
Total Debits: $111,367.88
Total Credits: $109,073.50`, "bank_statement_march2024.tsv"

	case "resume":
		return `PRIYA SHARMA
Mumbai, India | priya.sharma@email.com | +91-98765-43210 | github.com/priyasharma | linkedin.com/in/priyasharma

SUMMARY
Senior software engineer with 7 years of experience in distributed systems, payment infrastructure, and developer tooling. Led teams of 5-8 engineers. Passionate about reliability engineering and developer experience.

EXPERIENCE

Senior Software Engineer | Razorpay | Jan 2022 - Present
- Led the redesign of the payment routing engine, improving success rates by 4.2% across 50M+ monthly transactions
- Built a real-time fraud detection pipeline processing 3000 TPS using Kafka, Flink, and custom ML models
- Mentored 6 junior engineers; 3 promoted within 18 months
- Reduced P99 latency of payment API from 450ms to 180ms through connection pooling and query optimization
- Technologies: Go, Java, PostgreSQL, Redis, Kafka, Kubernetes, AWS

Software Engineer | Flipkart | Jun 2019 - Dec 2021
- Designed the cart & checkout service handling 200K concurrent users during Big Billion Days sale
- Implemented distributed caching layer reducing database load by 65%
- Built internal CLI tool for service deployment adopted by 40+ teams
- On-call for payments infrastructure (99.99% uptime SLA)
- Technologies: Java, Spring Boot, MySQL, Redis, Docker

Junior Software Engineer | Infosys | Jul 2017 - May 2019
- Developed REST APIs for a banking client's loan management system
- Wrote integration tests covering 89% of critical paths
- Technologies: Java, Oracle DB, Angular

EDUCATION
B.Tech Computer Science | IIT Bombay | 2013 - 2017
GPA: 8.7/10 | Dean's List 2015, 2016

SKILLS
Languages: Go, Java, Python, TypeScript, SQL
Infrastructure: Kubernetes, Docker, AWS (ECS, Lambda, SQS, DynamoDB), Terraform
Databases: PostgreSQL, MySQL, Redis, MongoDB, DynamoDB
Tools: Kafka, gRPC, Prometheus, Grafana, Git, Jenkins

CERTIFICATIONS
AWS Solutions Architect Associate (2023)
Certified Kubernetes Administrator (2022)

PUBLICATIONS
"Optimizing Payment Routing in Multi-Acquirer Systems" - RazorCon 2023
"Building Reliable Distributed Systems at Scale" - GopherCon India 2022`, "resume_priya_sharma.txt"

	case "config_yaml":
		return `apiVersion: apps/v1
kind: Deployment
metadata:
  name: zarvis-api
  namespace: production
  labels:
    app: zarvis
    tier: backend
    version: v2.4.1
  annotations:
    deployment.kubernetes.io/revision: "47"
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: zarvis
  template:
    metadata:
      labels:
        app: zarvis
        tier: backend
    spec:
      serviceAccountName: zarvis-sa
      terminationGracePeriodSeconds: 30
      containers:
        - name: zarvis-api
          image: registry.internal/zarvis:v2.4.1-sha-a3b8c2d
          ports:
            - containerPort: 8080
              name: http
            - containerPort: 9090
              name: metrics
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: zarvis-secrets
                  key: database-url
            - name: REDIS_URL
              value: "redis://redis-cluster.production:6379"
            - name: ANTHROPIC_API_KEY
              valueFrom:
                secretKeyRef:
                  name: zarvis-secrets
                  key: anthropic-key
            - name: LOG_LEVEL
              value: "info"
            - name: MAX_UPLOAD_SIZE
              value: "10485760"
            - name: CORS_ORIGINS
              value: "https://zarvis.io,https://app.zarvis.io"
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 2000m
              memory: 2Gi
          readinessProbe:
            httpGet:
              path: /api/health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /api/health
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 20
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir:
            sizeLimit: 1Gi
---
apiVersion: v1
kind: Service
metadata:
  name: zarvis-api
  namespace: production
spec:
  selector:
    app: zarvis
  ports:
    - port: 80
      targetPort: 8080
      name: http
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: zarvis-api
  namespace: production
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: zarvis-api
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70`, "k8s_deployment.yaml"

	default:
		return "", ""
	}
}
