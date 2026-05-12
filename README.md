# UNILA AI — Academic Chatbot

A self-hosted RAG (Retrieval-Augmented Generation) chatbot for Universitas Lampung (UNILA). Students can ask questions about academic regulations, administrative procedures, and campus services — answered based on official university documents uploaded by admins.

Built with Go, SvelteKit, Qdrant, and Ollama (Llama 3 8B). Supports bilingual responses (English / Indonesian).

## Features

- **Upload PDFs** — admin uploads official university documents
- **Hybrid search** — dense vector (bge-m3, 1024-dim) + BM25 sparse vector with RRF fusion
- **Query rewriting** — LLM rewrites query to document-style keywords before retrieval
- **Relevance guardrail** — context relevance check prevents LLM from hallucinating when retrieval misses
- **Accurate answers** — grounded in uploaded documents; specific values copied verbatim from context
- **Source links** — each answer links back to the original PDF page
- **Bilingual** — full EN/ID toggle; English queries translated to Indonesian before retrieval
- **Streaming** — responses stream token by token via SSE
- **Multi-turn chat** — conversation history maintained on the client
- **Self-hosted** — runs entirely on local campus infrastructure

## Tech Stack

| Layer | Technology |
|---|---|
| Frontend | SvelteKit 2 + Svelte 5 + Tailwind CSS 4 |
| Backend | Go (Clean Architecture) |
| Vector DB | Qdrant (Docker) |
| LLM (local) | Ollama + Llama 3 8B Q4_K_M |
| LLM (fallback) | Google Gemini API |
| Embedding | bge-m3 (1024-dim, multilingual) |

## System Architecture

### RAG Pipeline

```
User Query
    │
    ├─ [EN query?] → Translate to Indonesian (Llama 3, silent)
    │
    ├─ Query Rewriting → document-style keywords
    │
    ├─ Embed (dense vector) + BM25 (sparse vector)
    │
    ├─ Qdrant Hybrid Search (RRF fusion, score threshold 0.02)
    │
    ├─ Out-of-Domain Guardrail
    │     ├─ Layer 1: score threshold filters low-similarity chunks
    │     └─ Layer 2: LLM relevance judge (YA/TIDAK)
    │
    ├─ [Relevant] → inject context into prompt
    └─ [Not relevant] → fallback mode (no context injected)
            │
            └─ LLM generates answer (original language)
```

### Knowledge Base

| Document | Chunks |
|---|---|
| Panduan Penulisan Karya Ilmiah 2020 | 332 |
| Peraturan Akademik No. 12 Tahun 2025 | 222 |
| SOP BAK UNILA | 206 |
| Panduan SIAKAD Mahasiswa | 67 |
| Tata Pergaulan Warga (SK 359) | 46 |
| SK 355 Keringanan UKT | 45 |
| KKN UNILA | 42 |
| Seleksi Mandiri 2024 (Perrek No. 1) | 38 |
| UKT & IPI 2020 (Perrek No. 23) | 36 |
| Tarif UKT 2025–2026 | 31 |
| Tarif IPI 2025–2026 | 25 |
| **Total** | **1.090** |

### Multilingual Mechanism

English queries are silently translated to Indonesian before retrieval so BM25 (trained on Indonesian corpus via Sastrawi stemmer) remains accurate. The original English query is still used in the final prompt so the LLM answers in the user's language.

### Out-of-Domain Guardrail

Two-layer approach to prevent hallucination on off-topic questions:
1. **Score threshold** — Qdrant only returns chunks above a minimum similarity score
2. **LLM relevance judge** — before generation, the LLM evaluates whether retrieved chunks actually relate to the query; if not, context is withheld and the model falls back to a constrained response

## Prerequisites

- [Go](https://go.dev/) 1.22+
- [Bun](https://bun.sh/) (frontend runtime)
- [Docker](https://docs.docker.com/get-docker/) (for Qdrant)
- [Ollama](https://ollama.com/) (native install)

## Setup

### 1. Start Qdrant

```bash
docker compose up -d
```

### 2. Pull Ollama models

```bash
ollama pull llama3:8b-instruct-q4_K_M
ollama pull bge-m3
```

### 3. Configure backend

```bash
cp backend/.env.example backend/.env
# Edit backend/.env if needed (defaults work out of the box)
```

Key environment variables:

```env
PORT=8080
LLM_ENGINE=ollama          # or "gemini"

OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3:8b-instruct-q4_K_M
OLLAMA_EMBED_MODEL=bge-m3

QDRANT_HOST=localhost
QDRANT_PORT=6334
QDRANT_COLLECTION=unila_docs

CHUNK_SIZE=300
CHUNK_OVERLAP=100
TOP_K=8
SCORE_THRESHOLD=0.06
```

To use Gemini instead:
```env
LLM_ENGINE=gemini
GEMINI_API_KEY=your_key_here
```

### 4. Run the backend

```bash
cd backend
go run ./cmd/api
```

Or with live reload via [Air](https://github.com/air-verse/air):
```bash
cd backend
air
```

### 5. Run the frontend

```bash
cd frontend
bun install
bun dev
```

The app is available at `http://localhost:5173`. The backend API runs at `http://localhost:8080`.

## Usage

1. Go to `/admin` to upload PDF documents
2. Go to `/chat` to ask questions
3. Toggle **EN / ID** in the header to switch language

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/chat` | Send a question, get full answer (non-streaming) |
| `POST` | `/api/chat/stream` | Send a question, stream answer via SSE |
| `POST` | `/api/documents/upload` | Upload a PDF |
| `GET` | `/api/documents` | List all documents |
| `DELETE` | `/api/documents/{filename}` | Remove a document |
| `GET` | `/uploads/{filename}` | Download/view a PDF |

### Chat request format

```json
{
  "query": "What are the requirements for academic leave?",
  "language": "en",
  "history": [
    { "role": "user",      "content": "..." },
    { "role": "assistant", "content": "..." }
  ]
}
```

`language` accepts `"en"` (default) or `"id"`.

## Project Structure

```
unila-ai/
├── backend/
│   ├── cmd/api/          # Entry point, dependency injection
│   ├── internal/
│   │   ├── adapter/      # Ollama and Gemini adapters
│   │   ├── domain/       # Interfaces and types
│   │   ├── handler/      # HTTP handlers
│   │   ├── repository/   # Qdrant repository
│   │   └── usecase/      # Business logic (chat, ingestion)
│   └── pkg/
│       ├── config/       # Environment configuration
│       ├── bm25/         # BM25 sparse vector implementation
│       └── pdf/          # PDF extraction and cleaning
├── frontend/
│   └── src/
│       ├── lib/          # api.ts, i18n.ts
│       └── routes/       # +page.svelte files
├── docs/                 # Academic documentation (Bahasa Indonesia)
└── docker-compose.yml    # Qdrant only
```

## Technical Documentation

Academic documentation in Indonesian is in the [`docs/`](docs/) folder:

1. [System Overview](docs/01-gambaran-sistem.md)
2. [Ingestion Pipeline](docs/02-pipeline-ingesti.md)
3. [Retrieval Strategy](docs/03-strategi-pencarian.md)
4. [RAG Flow & Prompt Engineering](docs/04-rag-flow.md)
5. [Technology Stack](docs/05-stack-teknologi.md)
6. [Implementation Stages](docs/06-implementasi-tahapan.md)
7. [RAGAS Evaluation](docs/07-evaluasi-ragas.md)

## Evaluation Results (RAGAS)

| Metric | Score |
|---|---|
| Faithfulness | **0.9833** |
| Context Recall | **0.8849** |
| Context Precision | **0.7890** |
| **Overall** | **0.8858** |

Evaluated on 22 questions across 5 official UNILA documents using RAGAS 0.2.6.
