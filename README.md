# Zarvis — Document Intelligence with Spirit Animal Guides

Upload messy documents. Get clean, structured, queryable data — viewed through four different lenses, each guided by a spirit animal.

**Problem statement chosen:** #1 — Turn messy documents into structured, queryable data.

## The Problem

Real-world data is messy. CSVs with inconsistent formatting, JSON with nested chaos, logs with no schema, text files that are "sort of tabular." Getting from raw document to clean, structured data that you can actually query is tedious manual work — or a custom script every time.

Zarvis takes a conversational approach: upload a document, and an AI parses it into structured JSON, infers the schema, generates a summary, and lets you query the data through natural language. Four spirit animals provide four different views of the same data.

## The Four Views

| Animal | View | What it does |
|--------|------|-------------|
| 🦊 Fox | **Explorer** | Parses raw documents into clean, structured JSON. The entry point. |
| 🦉 Owl | **Table** | Shows data in tabular form. Supports filtering, sorting, aggregation via natural language queries. |
| 🐉 Dragon | **Schema** | Infers field types, detects patterns, finds data quality issues, maps relationships. |
| 🦦 Otter | **Summary** | Generates human-readable summaries — what the document is, key findings, patterns. |

## Why This Scoping

**What I built:**
- **Real tool execution loop** — Claude calls tools, the backend executes them, feeds results back, Claude responds with real analysis. Not stubs.
- **Schema inference** — the hard sub-problem. Detecting field types, enums, nullability, uniqueness, and relationships from raw data.
- **Four distinct output representations** of the same data — each animal isn't cosmetic, it's a genuinely different lens.
- **Persistent storage** — documents, structured data, schemas, and summaries stored in SQLite. Upload once, query forever.
- **Badge gamification** — cosmetic progression as you use features.
- **3D animated spirit orb** (Three.js) — because developer tools should be delightful.

**What I deliberately left out:**
- **PDF/image parsing** — would need OCR (Tesseract) or a PDF library. Text-based formats (CSV, JSON, TXT, XML, logs) are the focus.
- **Vector search** — querying is done by loading structured data into Claude's context. For large datasets, you'd want embeddings + similarity search.
- **Multi-document joins** — each document is processed independently. A production version would support cross-document queries.
- **Export** — structured data is stored as JSON. Would add CSV/Excel export for production.

## Architecture

```
Frontend (React + Three.js) ──upload──▶ Backend (Go/Chi) ──stream──▶ Anthropic API
         ▲                                │   ▲                         │
         │◀──────── SSE ──────────────────┘   │                         │
         │                                    ▼ tool_use                │
         │                              Tool Executor ◀─────────────────┘
         │                              ├── get/save structured data
         │                              ├── get/save schema
         │                              ├── get/save summary
         │                              ├── query structured data
         │                              └── list documents
         │
         └── SQLite (sessions, documents, structured_json, schema, badges)
```

**The key insight:** Claude does the hard work (parsing, schema inference, summarization), but the backend enforces structure — tools read/write to SQLite, and each module's prompt constrains what Claude focuses on.

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| LLM | Claude via `anthropic-sdk-go` | Native tool-use + streaming |
| Backend | Go 1.22 + Chi | Fast, single binary |
| Streaming | SSE | Simpler than WebSockets for one-way |
| Storage | SQLite (pure Go) | Zero-dependency, embedded |
| Frontend | React 19 + Vite + TypeScript | Fast dev |
| 3D | Three.js + React Three Fiber | Spirit orb animation |
| Styling | Tailwind v4 + Framer Motion | Rapid iteration |

## Running Locally

### Prerequisites
- Go 1.22+, Node.js 18+
- Anthropic API key

### Backend
```bash
cd backend
go mod tidy
ZARVIS_API_KEY=sk-ant-... ZARVIS_BASE_URL=https://api.anthropic.com go run .
# → :8080
```

### Frontend
```bash
cd frontend
npm install
npm run dev
# → http://localhost:5173
```

### Tests
```bash
cd backend && go test ./...
```

## UX Flow

1. **Welcome** — 3D spirit orb greets you
2. **Pick personality** — choose Fox/Owl/Dragon/Otter (shapes the AI's communication style)
3. **Upload** — drag-and-drop your messy document (CSV, JSON, TXT, XML, logs)
4. **Explorer tab** — Fox parses it into structured JSON
5. **Table tab** — Owl shows it as a queryable table. Ask: "show rows where amount > 100"
6. **Schema tab** — Dragon maps field types, finds patterns, flags data quality issues
7. **Summary tab** — Otter writes a human-readable report of what the data contains

## API

| Endpoint | Method | Description |
|---|---|---|
| `/api/session` | POST | Create session |
| `/api/session/{id}` | GET/PATCH | Get/update session |
| `/api/session/{id}/badges` | GET | Get earned badges |
| `/api/session/{id}/document` | GET | Get latest document + structured data |
| `/api/upload` | POST | Upload document (multipart) |
| `/api/chat` | POST | Stream chat (body: `{session_id, module, message}`) |

### SSE Events
`delta` (text), `tool_use` (tool invoked), `tool_result` (tool output), `badge` (badge earned), `done`, `error`
