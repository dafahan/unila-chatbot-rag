# Bab VI — Tahapan Implementasi

## 6.1 Tahap 1: Perancangan Arsitektur

**Status: Selesai**

- Penentuan pola Clean Architecture untuk backend Go
- Perancangan antarmuka `LLMProvider` dan `DocumentRepository` (Strategy Pattern)
- Pemilihan Qdrant sebagai vector database
- Perancangan API endpoint

**Keluaran:**
- Struktur direktori proyek
- Definisi interface domain (`llm.go`, `document.go`, `chat.go`)
- Konfigurasi berbasis environment variable (`.env`)

## 6.2 Tahap 2: Implementasi Adapter LLM

**Status: Selesai**

Dua adapter diimplementasikan yang keduanya memenuhi interface `LLMProvider`:

- `OllamaAdapter` — komunikasi HTTP ke Ollama API lokal, auto-detect dimensi embedding
- `GeminiAdapter` — komunikasi via SDK resmi Google ke Gemini API

## 6.3 Tahap 3: Pipeline Ingesti Dokumen

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Ekstraksi teks PDF per-halaman | `pkg/pdf/extract.go` | ✅ |
| Deteksi boilerplate statistik (>30% halaman) | `pkg/pdf/extract.go` | ✅ |
| Pembersihan daftar isi (regex) | `pkg/pdf/extract.go` | ✅ |
| Pembersihan noise PDF (cleanText — "BUKU –") | `usecase/ingestion.go` | ✅ |
| Chunking berbasis kata (300 char, overlap 100) | `usecase/ingestion.go` | ✅ |
| Deduplikasi Jaccard (threshold 0.75) | `usecase/ingestion.go` | ✅ |
| Embedding paralel bge-m3 (4 worker pool) | `usecase/ingestion.go` | ✅ |
| BM25 sparse vector per chunk | `usecase/ingestion.go` | ✅ |
| Simpan ke Qdrant via gRPC (dense + sparse) | `repository/qdrant.go` | ✅ |
| Simpan file PDF ke disk | `handler/document.go` | ✅ |

## 6.4 Tahap 4: Pencarian Hybrid

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Dense vector search (bge-m3, Cosine) | `repository/qdrant.go` | ✅ |
| BM25 sparse vector search | `repository/qdrant.go` | ✅ |
| RRF (Reciprocal Rank Fusion) fusion | `repository/qdrant.go` | ✅ |
| Score threshold filtering (0.06) | `repository/qdrant.go` | ✅ |
| Query translation EN→ID via LLM | `usecase/chat.go` | ✅ |
| Query rewriting untuk retrieval | `usecase/chat.go` | ✅ |

## 6.5 Tahap 5: Context Relevance Check

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| `checkRelevance()` — validasi topik chunks vs query | `usecase/chat.go` | ✅ |
| Mode prompt dengan konteks (`contextRelevant=true`) | `usecase/chat.go` | ✅ |
| Mode prompt tanpa konteks (`contextRelevant=false`) | `usecase/chat.go` | ✅ |

## 6.6 Tahap 6: Rekayasa Prompt dan Generasi

**Status: Selesai**

| Sub-tahap | Status |
|---|---|
| System prompt bilingual (EN/ID) via `langRules` map | ✅ |
| Penyertaan konteks bernomor dengan atribusi sumber | ✅ |
| Dukungan riwayat percakapan multi-turn | ✅ |
| Anti-hallucination: DILARANG mengarang angka/nama/prosedur | ✅ |
| Anti-paraphrase: SALIN PERSIS nilai spesifik dari konteks | ✅ |
| Anti-hedging: larang kalimat pembuka basa-basi | ✅ |
| Anti-attribution: larang sebut nama file | ✅ |
| Language enforcement: jawab sesuai bahasa pilihan | ✅ |

## 6.7 Tahap 7: API dan Manajemen Dokumen

**Status: Selesai**

| Endpoint | Metode | Keterangan | Status |
|---|---|---|---|
| `/api/chat` | POST | RAG query + language field | ✅ |
| `/api/chat/stream` | POST | RAG query dengan SSE streaming | ✅ |
| `/api/documents/upload` | POST | Upload PDF + ingest | ✅ |
| `/api/documents` | GET | Daftar dokumen di Qdrant | ✅ |
| `/api/documents/{filename}` | DELETE | Hapus dari Qdrant + disk | ✅ |
| `/uploads/{filename}` | GET | Serving file PDF statis | ✅ |

## 6.8 Tahap 8: Antarmuka Pengguna

**Status: Selesai**

| Fitur | Rute/Komponen | Status |
|---|---|---|
| Landing page | `/` | ✅ |
| Chat mahasiswa | `/chat` | ✅ |
| Panel admin (upload, list, delete) | `/admin` | ✅ |
| Rendering Markdown respons LLM | `marked` | ✅ |
| Link sumber PDF yang dapat diklik | Chat page | ✅ |
| Saran pertanyaan (suggestion chips) | Chat page | ✅ |
| Riwayat percakapan dengan clear | Chat page | ✅ |
| Toggle bahasa EN/ID | Semua halaman | ✅ |

## 6.9 Tahap 9: Evaluasi RAGAS

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Dataset evaluasi 22 pertanyaan (5 dokumen) | `eval/eval_dataset.json` | ✅ |
| Script otomasi query ke API + cache response | `eval/run_ragas.py` | ✅ |
| Evaluasi faithfulness via RAGAS 0.2.6 | `eval/run_ragas.py` | ✅ |
| Evaluasi context_precision via RAGAS 0.2.6 | `eval/run_ragas.py` | ✅ |
| Evaluasi context_recall via RAGAS 0.2.6 | `eval/run_ragas.py` | ✅ |
| Scores cache (skip re-eval NaN secara bertahap) | `eval/scores_cache.json` | ✅ |
| Report per-pertanyaan + per-dokumen + aggregate | `eval/run_ragas.py` | ✅ |

**Hasil Evaluasi:**

| Metrik | Skor |
|---|---|
| Faithfulness | 0.9833 |
| Context Recall | 0.8849 |
| Context Precision | 0.7890 |
| **Overall** | **0.8858** |

## 6.10 Tantangan dan Solusi

| Tantangan | Solusi yang Diterapkan |
|---|---|
| Retrieval gagal untuk query semantik Bahasa Indonesia | Ganti embedding nomic-embed-text → bge-m3 (1024d, multibahasa) |
| Gap semantik antara query dan teks dokumen | Query rewriting via LLM sebelum embedding |
| LLM jawab "tidak tersedia" padahal konteks relevan | Longgarkan threshold `checkRelevance` — hanya tolak jika topik benar-benar berbeda |
| LLM paraphrase nilai spesifik (angka/URL/warna) | Tambah rule "SALIN PERSIS nilai dari konteks" di prompt |
| Noise "BUKU –" di chunks SIAKAD | `cleanText()` strip prefiks artefak sebelum chunking |
| Chunk prakata mendominasi hasil pencarian | Hybrid search BM25 + dense dengan RRF fusion |
| Duplikasi konten dari multi-edisi dokumen | Deduplikasi Jaccard similarity (threshold 0.75) |
| Noise header/footer dari PDF | Deteksi boilerplate statistik lintas halaman (>30%) |
| Embedding lambat (sequential) | Worker pool paralel (4 goroutine) |
| LLM menjawab dalam bahasa yang salah | Instruksi `ALWAYS respond in [language]` dalam prompt |
| Dimensi embedding hardcoded saat ganti model | Auto-detect dimensi via probe embedding saat startup |

## 6.11 Yang Belum Diimplementasikan (Pengembangan Lanjutan)

| Fitur | Keterangan |
|---|---|
| Autentikasi admin | Panel admin saat ini terbuka tanpa login |
| Cross-encoder reranker | Reranking berbasis model ML setelah retrieval awal |
| Rate limiting | Pembatasan request per pengguna |
| Tracking nomor halaman PDF akurat | Saat ini nomor halaman dari metadata extraction |
