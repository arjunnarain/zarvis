# Zarvis — Document Intelligence with Spirit Animal Guides

Upload messy documents. Get clean, structured, queryable data — viewed through five spirit-animal lenses, with cross-document search powered by BM25.

**Problem statement:** #1 — Turn messy documents into structured, queryable data.

## The Problem

Real-world data is messy. CSVs with inconsistent formatting, PDFs with no schema, JSON with deep nesting, logs with mixed formats. Getting from raw document to clean, structured, queryable data is tedious manual work every time.

Zarvis takes a conversational approach: upload documents, organize them into **Forests** (collections), and interact with your data through five different lenses — each guided by a spirit animal with a distinct analytical perspective.

## The Five Views

| Animal | Tab | Purpose |
|--------|-----|---------|
| 🦊 Fox | **Explorer** | Parse raw documents into structured JSON. Detects document type, extracts entities, flags data quality issues |
| 🦉 Owl | **Table** | View structured data in tables. Filter, sort, aggregate via natural language |
| 🐉 Dragon | **Schema** | Infer field types, detect patterns, map relationships, report data quality |
| 🦦 Otter | **Summary** | Generate human-readable reports — key findings, patterns, overview |
| 🌲 Oracle | **Oracle** | Cross-document queries via BM25 search. Compare, find patterns across a Forest |

## Architecture

```
Frontend (React + Three.js)
    │
    ├── Auth ──────────► POST /api/auth/register, /api/auth/login
    │                       └── JWT (HS256, 7-day, bcrypt passwords)
    │
    ├── Upload ────────► POST /api/upload (multipart, PDF extraction via pdftotext)
    │                       └── auto-indexes into BM25 search
    │
    └── Chat ──────────► POST /api/chat (SSE streaming)
         │                   │
         │              ┌────┴────┐
         │              │  Claude  │ ←── module-specific system prompt
         │              └────┬────┘
         │                   │ tool_use
         │              ┌────┴──────────────┐
         │              │  Tool Executor    │
         │              ├── get_raw_document │ ← SQLite read
         │              ├── save_structured  │ ← SQLite write + re-index BM25
         │              ├── save_schema      │ ← SQLite write
         │              ├── query_forest     │ ← BM25 search across chunks
         │              └── 9 total tools    │
         │              └───────────────────┘
         │
    SQLite (users, sessions, documents, forests, chunks, badges)
```

### Key Design Decisions

1. **Tool execution loop** — Claude calls tools, the Go backend executes them (DB queries, BM25 search, PDF extraction), feeds results back, Claude responds with real data. This is not a proxy — the backend does real work.

2. **BM25 search engine** — Pure Go implementation. Documents are chunked (~500 chars), tokenized, and scored with BM25 (k1=1.2, b=0.75) for cross-document retrieval. Upgrade path: swap scoring function for ONNX embeddings without changing storage layer.

3. **Forests** — Document collections with many-to-many relationships. A document can belong to multiple forests. The Oracle tab queries across all documents in the active forest.

4. **Server-side state** — All intelligence decisions (what tools to provide, what data to inject) happen in Go. The frontend is display-only.

5. **Module-specific prompts** — Each tab has a distinct system prompt that constrains Claude's behavior and available tools. The prompt builder loads from markdown files at startup.

## Tech Stack

| Layer | Choice | Why |
|---|---|---|
| Backend | Go + Chi | Fast, single binary, clean middleware |
| Auth | JWT (HS256) + bcrypt | Stateless, no session store needed |
| Storage | SQLite (pure Go) | Zero-dependency, embedded, ACID |
| Search | BM25 (pure Go) | No external service, fast, upgradeable |
| LLM | Claude via anthropic-sdk-go | Native tool-use + streaming |
| PDF | pdftotext (poppler) | Robust layout-preserving extraction |
| Frontend | React 19 + Vite + TypeScript | Fast dev |
| 3D | Three.js + React Three Fiber | Spirit orb animation |
| Styling | Tailwind v4 + Framer Motion | Rapid iteration |

## Running Locally

### Prerequisites
- Go 1.23+, Node.js 18+, Anthropic API key
- Optional: `poppler` for PDF support (`brew install poppler`)

### Backend
```bash
cd backend
go mod tidy
ZARVIS_API_KEY=sk-ant-... ZARVIS_BASE_URL=https://api.anthropic.com go run .
# → :8080
```

### Frontend (dev mode)
```bash
cd frontend
npm install
npm run dev
# → http://localhost:5173 (proxies /api → :8080)
```

### Docker (production)
```bash
ZARVIS_API_KEY=sk-ant-... docker compose up --build
# → http://localhost:8080 (single port)
```

### Tests
```bash
cd backend && go test ./...
# 31 tests across auth, store, search, registry
```

## Project Structure

```
zarvis/
├── backend/
│   ├── main.go                          # Server bootstrap, routes, middleware
│   └── internal/
│       ├── auth/auth.go                 # JWT + bcrypt (+ 8 tests)
│       ├── badges/engine.go             # Gamification badges
│       ├── chat/
│       │   ├── handler.go              # Handler struct + shared helpers
│       │   ├── auth_handler.go         # Register, Login, GetMe
│       │   ├── session_handler.go      # Session CRUD, badges
│       │   ├── forest_handler.go       # Forest CRUD, doc linking, BM25 indexing
│       │   ├── upload_handler.go       # File upload, PDF extraction, samples
│       │   └── chat_handler.go         # SSE streaming + tool execution loop
│       ├── mcp/registry.go             # Tool registry by module (+ 6 tests)
│       ├── prompt/builder.go           # Module-specific prompt loader
│       ├── search/
│       │   ├── chunker.go             # Text chunking (paragraph/sentence)
│       │   ├── bm25.go               # BM25 scoring algorithm
│       │   └── engine.go             # Index + search orchestrator (+ 7 tests)
│       ├── state/store.go             # SQLite: users, sessions, docs, forests, chunks (+ 10 tests)
│       └── tools/executor.go          # Tool execution: DB CRUD, BM25 search, forest queries
├── frontend/src/
│   ├── App.tsx                         # Auth gate, module tabs, state management
│   ├── lib/api.ts                     # Authenticated fetch wrapper
│   └── components/
│       ├── AuthScreen.tsx             # Login/register
│       ├── Chat.tsx                   # SSE client, tool progress, @ mentions
│       ├── ModuleTabs.tsx             # 5 spirit animal tabs
│       ├── DocumentUpload.tsx         # Drag-drop + sample documents
│       ├── ForestManager.tsx          # Forest CRUD dropdown
│       ├── DocumentList.tsx           # Document selector
│       ├── BadgeShelf.tsx             # Badge display with tooltips
│       └── SpiritOrb.tsx             # Three.js particle animation
├── docs/
│   ├── mcp_tools.json                 # Tool registry (9 tools across 5 modules)
│   └── prompts/                       # Module-specific system prompts
│       ├── explorer.md, table.md, schema.md, summary.md, oracle.md
├── Dockerfile                         # Multi-stage: Node build → Go build → Alpine
└── docker-compose.yml
```

## API Reference

### Auth (public)
| Endpoint | Method | Body | Returns |
|---|---|---|---|
| `/api/auth/register` | POST | `{email, password, name}` | `{token, user}` |
| `/api/auth/login` | POST | `{email, password}` | `{token, user}` |

### Protected (requires `Authorization: Bearer <token>`)
| Endpoint | Method | Description |
|---|---|---|
| `/api/auth/me` | GET | Current user |
| `/api/session` | POST | Create session |
| `/api/session/{id}` | GET/PATCH | Get/update session |
| `/api/session/{id}/badges` | GET | Earned badges |
| `/api/session/{id}/documents` | GET | List documents |
| `/api/session/{id}/forests` | GET | List forests |
| `/api/upload` | POST | Upload document (multipart) |
| `/api/sample` | POST | Load sample document |
| `/api/forest` | POST | Create forest |
| `/api/forest/{id}/documents` | GET/POST/DELETE | Forest documents / clear |
| `/api/chat` | POST | Stream chat `{session_id, module, message}` |

### SSE Events
`delta` · `tool_use` · `tool_result` · `badge` · `done` · `error`

## What I Deliberately Left Out

- **Vector embeddings** — BM25 handles keyword retrieval well at demo scale. Production path: ONNX sentence embeddings via the Hugot library, swapping only the scoring function.
- **Multi-user document sharing** — Documents are scoped to sessions. Would add team/org model.
- **Export** — Structured data is JSON in SQLite. Would add CSV/Excel/API export.
- **Scheduled processing** — All parsing is on-demand. Would add background job queue for large documents.
