<p align="center">
  <h1 align="center">Zarvis</h1>
  <p align="center"><em>Document Intelligence Platform with Spirit Animal Guides</em></p>
  <p align="center">
    <a href="#quick-start">Quick Start</a> · <a href="#features">Features</a> · <a href="#architecture">Architecture</a> · <a href="#sample-inputoutput">Samples</a> · <a href="#api-reference">API</a>
  </p>
</p>

---

Upload messy, unstructured documents — CSVs with broken formatting, invoices as plain text, server logs, PDFs, YAML configs, resumes. Zarvis parses them into clean, structured, queryable JSON through six AI-powered views, each guided by a spirit animal with a distinct analytical purpose.

> **Problem Statement #1** — *Turn messy documents into structured, queryable data.*

---

## Why This Exists

Real-world data doesn't arrive in neat schemas. It arrives as:
- A CSV where salaries are `$85,000`, `92000`, and `eighty thousand` in the same column
- An invoice as freeform text with addresses, line items, and tax math jumbled together
- Server logs with circuit breaker events, embedded key-value pairs, and mixed timestamps
- A resume with no structure at all — just paragraphs of career history

Every time, someone manually wrangles this into something usable. Zarvis automates that transformation and makes the result queryable, visualizable, and exportable.

---

## Problem Framing & Scoping

**Interpretation:** Rather than building a batch ETL pipeline, I chose a conversational approach — the AI explains what it finds, flags issues, and lets you interact with the structured output. This makes the tool useful even when you don't know what's in the document.

**Why spirit animals?** Each analysis mode (parse, table, schema, charts, cross-doc) is a genuinely different task. The animal metaphor gives each mode a clear identity and personality, making the UI intuitive without labels or tutorials. It's not decoration — it's information architecture.

**What I deliberately left out:**
| Decision | Reasoning |
|---|---|
| No vector embeddings | BM25 keyword retrieval works well at demo scale. Production path: swap scoring function for ONNX sentence embeddings (Hugot library) — storage layer stays identical. |
| No multi-user sharing | Documents scoped to authenticated sessions. Would add org/team model for production. |
| No batch processing | All parsing is on-demand and streaming. Would add a job queue for 100+ page PDFs. |
| No custom model fine-tuning | Claude's tool-use capabilities handle all document types out of the box. |

---

## Features

### Six Spirit Animal Views

| | Tab | Spirit | What it does |
|---|---|---|---|
| 🦊 | **Explorer** | Fox | Parse any document into structured JSON. Auto-detects type (CSV, JSON, log, invoice, resume, YAML). Flags data quality issues. |
| 🦉 | **Table** | Owl | View structured data as interactive tables. Natural language queries: *"show rows where salary > 90000"* |
| 🐉 | **Schema** | Dragon | Infer field types, nullability, enums, relationships. Reports data quality issues with actionable suggestions. |
| 🦦 | **Summary** | Otter | Human-readable reports — key facts, patterns, overview. Copy-paste ready. |
| 📈 | **Graphs** | Phoenix | Bar charts, pie charts, trend lines — rendered as inline SVG with legends and axes. Only enabled when numeric data exists. |
| 🌲 | **Oracle** | Forest Spirit | Cross-document queries via BM25 search. Compare documents, find patterns across a Forest (collection). |

### Smart Tab System
Tabs auto-enable based on document state. Upload a CSV → Table + Graphs unlock. Upload an invoice → only Schema + Summary available. Oracle requires 1+ documents in a Forest. Disabled tabs show a tooltip explaining why.

### Document Forests
Group documents into named collections. Add multiple documents to a Forest, then query across all of them in the Oracle tab. Forests support reset (clear all documents) and re-indexing.

### Data Quality Scoring
After parsing, an **Output Data Quality** score (0–100, A–F grade) appears with a 4-dimension breakdown:
- **Completeness** — how few null/empty values in the output
- **Consistency** — field name and type uniformity across records  
- **Validity** — how many significant tokens from the raw input were captured
- **Structure** — depth and organization of the JSON output

### Before / After Diff View
Split-pane modal showing raw messy input (red) alongside clean structured output (green). Visually tells the transformation story.

### Export
Download structured data in **JSON**, **CSV**, **TSV**, or **Markdown table** format. Handles nested structures by flattening objects to dot-notation or finding the deepest array.

### Inline Search
🔍 Search panel in the Explorer tab — real-time text search across raw and structured content with highlighted matches. Click a result to query about it.

### 8 Built-in Sample Documents
Try instantly without uploading a file:

| Sample | Format | What makes it messy |
|---|---|---|
| Messy Employee CSV | CSV | 5 date formats, mixed currency, `eighty thousand` as salary |
| Cloud Invoice | Plain text | No schema — addresses, line items, tax math as freeform text |
| Server Crash Log | Log file | Circuit breaker events, error traces, latency metrics |
| Org Chart JSON | Nested JSON | 3 levels deep — departments → teams → projects |
| Support Tickets | Email dump | Semi-structured with priorities, agents, status changes |
| Bank Statement | TSV | Tab-separated transactions with running balance |
| Resume / CV | Free text | Zero structure — pure entity extraction challenge |
| K8s Config | YAML | Multi-document with nested specs, env vars, resource limits |

### Authentication
Email/password auth with JWT tokens (HS256, 7-day expiry, bcrypt password hashing). All API routes protected. Sessions scoped to authenticated users.

### Premium UI
Shader.se-inspired landing page with scroll animations, 3D spirit orb (Three.js), serif typography (DM Serif Display), and a "Dark Observatory" app theme with grain texture, warm amber accents, and editorial tab navigation.

---

## Architecture

```
                        ┌─────────────────────────────────┐
                        │     Frontend (React + Three.js)  │
                        │     Landing Page → Auth → App    │
                        └──────────┬──────────────────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                     │
         POST /api/auth    POST /api/upload    POST /api/chat (SSE)
              │                    │                     │
              ▼                    ▼                     ▼
     ┌────────────┐      ┌──────────────┐      ┌──────────────┐
     │  JWT Auth   │      │  Upload +    │      │  Anthropic   │
     │  bcrypt     │      │  PDF Extract │      │  Claude API  │
     └────────────┘      │  + Analyze   │      └──────┬───────┘
                          └──────┬───────┘             │ tool_use
                                 │                     ▼
                          ┌──────┴───────┐    ┌──────────────────┐
                          │  Go-side     │    │  Tool Executor   │
                          │  Analysis    │    ├── DB CRUD        │
                          │  • Type det. │    ├── BM25 search    │
                          │  • Schema    │    ├── Forest queries │
                          │  • Quality   │    ├── PDF extraction │
                          │  • Tab recs  │    └── 11 tools total │
                          └──────────────┘    └────────┬─────────┘
                                                       │
                                              ┌────────┴────────┐
                                              │     SQLite      │
                                              │  users sessions │
                                              │  documents      │
                                              │  forests chunks │
                                              └─────────────────┘
```

### What Makes This More Than a Wrapper

The backend does **real computation** before Claude ever sees the data:

| Package | What it does | LLM involved? |
|---|---|---|
| `analyze/detect.go` | Document type detection (CSV, JSON, XML, log, markdown, key-value) using regex, CSV parsing, structural analysis | No |
| `analyze/schema.go` | Field type inference (string, number, date, email, URL, boolean), nullability, uniqueness, sample values | No |
| `analyze/quality.go` | Output quality scoring — completeness, consistency, validity (token capture), structure depth | No |
| `analyze/tabs.go` | Smart tab recommendations based on document type, numeric fields, forest size | No |
| `analyze/clean.go` | Data quality report — empty values, currency detection, mixed types, suggestions | No |
| `search/bm25.go` | Full BM25 scoring (k1=1.2, b=0.75) with IDF weighting and length normalization | No |
| `search/chunker.go` | Paragraph/sentence-aware text chunking (~500 chars) | No |
| `search/engine.go` | Index orchestration — auto-indexes on forest add, re-indexes after parse | No |
| `chat/chat_handler.go` | Multi-turn tool execution loop — up to 5 rounds of tool calls per request | Orchestration |
| `chat/export_handler.go` | JSON → CSV/TSV/Markdown conversion with nested object flattening | No |

---

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| **Backend** | Go 1.25 + Chi | Single binary, fast, clean middleware chain |
| **Auth** | JWT (HS256) + bcrypt | Stateless, no session store needed |
| **Storage** | SQLite (pure Go, no CGO) | Zero-dependency, embedded, ACID compliant |
| **Search** | BM25 (pure Go) | No external service, sub-millisecond, upgradeable to ONNX |
| **LLM** | Claude via `anthropic-sdk-go` | Native tool-use + streaming support |
| **PDF** | pdftotext (poppler) | Layout-preserving text extraction |
| **Frontend** | React 19 + Vite + TypeScript | Fast HMR, type safety |
| **3D** | Three.js + React Three Fiber | Spirit orb particle animation |
| **Styling** | Tailwind v4 + Framer Motion | Utility CSS + spring animations |
| **Deploy** | Docker (multi-stage) | Node build → Go build → Alpine (~35MB) |

---

## Quick Start

### Prerequisites
- Go 1.23+, Node.js 18+
- Anthropic API key ([console.anthropic.com](https://console.anthropic.com))
- Optional: `poppler` for PDF support (`brew install poppler`)

### Option 1: Docker (recommended)
```bash
git clone https://github.com/arjunnarain/zarvis.git && cd zarvis
ZARVIS_API_KEY=sk-ant-... docker compose up --build
# → http://localhost:8080
```

### Option 2: Local development
```bash
# Backend
cd backend && go mod tidy
ZARVIS_API_KEY=sk-ant-... ZARVIS_BASE_URL=https://api.anthropic.com go run .

# Frontend (separate terminal)
cd frontend && npm install && npm run dev
# → http://localhost:5173
```

### Run tests
```bash
cd backend && go test ./...
# 42 tests across 5 packages — all passing
```

---

## Tests

**42 tests** across 5 packages, covering the core backend logic:

| Package | Tests | What's covered |
|---|---|---|
| `auth` | 8 | Password hashing, JWT generation/validation, middleware auth/reject |
| `state` | 10 | Users, sessions, messages, documents, forests, badges, chunks CRUD |
| `search` | 7 | Text chunking, tokenization, BM25 ranking, edge cases, top-K |
| `mcp` | 6 | Registry loading, module filtering, field validation, all-modules check |
| `analyze` | 11 | Type detection (7 formats), schema inference, value classification, quality |

These are not token tests — they catch real problems: duplicate emails, foreign key consistency, BM25 ranking accuracy, schema inference for mixed-type columns.

---

## Sample Input/Output

The [`samples/`](samples/) directory contains messy input documents and their expected structured outputs:

| Input | Messiness | Output |
|---|---|---|
| [`messy_employees.csv`](samples/input/messy_employees.csv) | 5 date formats, `$85,000` vs `eighty thousand`, missing fields, city abbreviations | [`structured JSON`](samples/output/messy_employees_structured.json) with normalized data + per-row issues + quality report |
| [`invoice_INV-2024-0847.txt`](samples/input/invoice_INV-2024-0847.txt) | Plain text — no schema, addresses, line items, tax math all freeform | [`structured JSON`](samples/output/invoice_structured.json) with nested vendor/customer/lineItems/totals |
| [`server_access.log`](samples/input/server_access.log) | Mixed log levels, embedded key-values, circuit breaker state machine | [`structured JSON`](samples/output/server_log_structured.json) with typed entries + computed summary stats |

See [`samples/README.md`](samples/README.md) for detailed analysis of each sample.

---

## Project Structure

```
zarvis/
├── backend/
│   ├── main.go                            # Server, routes, auth middleware
│   └── internal/
│       ├── auth/auth.go                   # JWT + bcrypt (+8 tests)
│       ├── analyze/
│       │   ├── detect.go                  # Document type detection
│       │   ├── schema.go                  # Server-side schema inference
│       │   ├── quality.go                 # Output quality scoring
│       │   ├── tabs.go                    # Smart tab recommendations
│       │   └── clean.go                   # Data quality report (+11 tests)
│       ├── chat/
│       │   ├── handler.go                 # Shared Handler struct
│       │   ├── auth_handler.go            # Register, Login, GetMe
│       │   ├── session_handler.go         # Sessions, badges, tabs, search, quality
│       │   ├── forest_handler.go          # Forest CRUD + BM25 auto-indexing
│       │   ├── upload_handler.go          # Upload, PDF extraction, 8 samples
│       │   ├── chat_handler.go            # SSE streaming + tool execution loop
│       │   └── export_handler.go          # JSON/CSV/TSV/Markdown export
│       ├── mcp/registry.go               # Tool registry by module (+6 tests)
│       ├── prompt/builder.go             # Module-specific prompt loader
│       ├── search/
│       │   ├── chunker.go                # Paragraph/sentence text chunking
│       │   ├── bm25.go                   # BM25 scoring algorithm
│       │   └── engine.go                 # Index + search orchestrator (+7 tests)
│       ├── state/store.go                # SQLite store (+10 tests)
│       └── tools/executor.go             # 11 tool implementations
├── frontend/src/
│   ├── App.tsx                            # Landing → Auth → Main app
│   ├── lib/api.ts                        # JWT-authenticated fetch wrapper
│   └── components/
│       ├── LandingPage.tsx               # Scroll-animated marketing page
│       ├── AuthScreen.tsx                # Login / register
│       ├── Chat.tsx                      # SSE client, charts, tools, search
│       ├── ModuleTabs.tsx                # 6 smart tabs with lock states
│       ├── DocumentUpload.tsx            # Drag-drop + 8 sample documents
│       ├── ForestManager.tsx             # Forest CRUD dropdown
│       ├── DocumentList.tsx              # Document selector
│       ├── QualityBadge.tsx              # Output quality score card
│       ├── DiffView.tsx                  # Before/after split pane
│       ├── ExportModal.tsx               # 4-format export dialog
│       ├── SearchPanel.tsx               # Real-time document search
│       ├── RawViewer.tsx                 # Raw document viewer
│       └── SpiritOrb.tsx                # Three.js particle animation
├── docs/
│   ├── mcp_tools.json                    # 11 tools across 6 modules
│   └── prompts/                          # 6 module-specific system prompts
├── samples/                              # 3 input/output pairs for evaluation
├── Dockerfile                            # Multi-stage: Node → Go → Alpine
└── docker-compose.yml
```

---

## API Reference

### Auth (public)
| Endpoint | Method | Body | Returns |
|---|---|---|---|
| `/api/auth/register` | POST | `{email, password, name}` | `{token, user}` |
| `/api/auth/login` | POST | `{email, password}` | `{token, user}` |

### Protected (requires `Authorization: Bearer <token>`)
| Endpoint | Method | Description |
|---|---|---|
| `/api/auth/me` | GET | Current user profile |
| `/api/session` | POST | Create session |
| `/api/session/{id}` | GET / PATCH | Get / update session |
| `/api/session/{id}/tabs` | GET | Smart tab recommendations |
| `/api/session/{id}/quality` | GET | Output data quality score |
| `/api/session/{id}/search?q=` | GET | Text search across document |
| `/api/session/{id}/documents` | GET | List uploaded documents |
| `/api/session/{id}/export?format=` | GET | Export (json / csv / tsv / markdown) |
| `/api/session/{id}/export-schema` | GET | Export inferred schema |
| `/api/upload` | POST | Upload document (multipart, 10MB) |
| `/api/sample` | POST | Load built-in sample document |
| `/api/forest` | POST | Create Forest collection |
| `/api/session/{id}/forests` | GET | List Forests |
| `/api/forest/{id}/documents` | GET / POST / DELETE | Forest documents / clear |
| `/api/chat` | POST | Stream chat `{session_id, module, message}` |

### SSE Event Types
| Event | Payload | When |
|---|---|---|
| `delta` | `{text}` | Streaming text from Claude |
| `tool_use` | `{tool, display_name, id}` | Tool invoked |
| `tool_result` | `{tool, output, error}` | Tool execution complete |
| `badge` | `{badge_key}` | Badge earned |
| `done` | `{status}` | Response complete |
| `error` | `{message}` | Error occurred |

---

## Production Upgrade Path

| Current | Production | Effort |
|---|---|---|
| BM25 keyword search | ONNX sentence embeddings (all-MiniLM-L6-v2) via Hugot | Swap scoring function only |
| SQLite | PostgreSQL | Change connection string + minor query tweaks |
| Single instance | Kubernetes with HPA | Dockerfile already Alpine-optimized |
| JWT in localStorage | HTTP-only cookies + refresh tokens | Auth middleware change |
| On-demand parsing | Background job queue (Asynq/River) | Add worker process |

---

<p align="center">
  <sub>Built by <a href="https://github.com/arjunnarain">Arjun Narain</a> for the Razorpay engineering challenge</sub>
</p>
