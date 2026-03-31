# UNILA AI — Academic Chatbot

A self-hosted RAG (Retrieval-Augmented Generation) chatbot for Universitas Lampung (UNILA). Students can ask questions about academic regulations, administrative procedures, and campus services — answered based on official university documents uploaded by admins.

Built with Go, SvelteKit, Qdrant, and Ollama (Llama 3 8B). Supports bilingual responses (English / Indonesian).

## Features

- **Upload PDFs** — admin uploads official university documents
- **Semantic search** — hybrid vector + keyword boost retrieval
- **Accurate answers** — grounded in uploaded documents, no hallucination
- **Source links** — each answer links back to the original PDF
- **Bilingual** — full EN/ID toggle; English queries are translated to Indonesian before retrieval
- **Streaming** — responses stream token by token via SSE (no waiting for full response)
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
| Embedding | nomic-embed-text (768-dim) |

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
ollama pull nomic-embed-text
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
OLLAMA_EMBED_MODEL=nomic-embed-text

QDRANT_HOST=localhost
QDRANT_PORT=6334
QDRANT_COLLECTION=unila_docs

CHUNK_SIZE=512
CHUNK_OVERLAP=64
TOP_K=5
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
│       └/pdf/            # PDF extraction and cleaning
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
