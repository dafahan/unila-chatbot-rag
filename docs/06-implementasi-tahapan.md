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

- `OllamaAdapter` — komunikasi HTTP ke Ollama API lokal (`temperature: 0.3`, `top_p: 0.9`)
- `GeminiAdapter` — komunikasi via SDK resmi Google ke Gemini API

Pemilihan adapter dilakukan saat startup via `LLM_ENGINE` environment variable, tidak mengubah kode use case.

## 6.3 Tahap 3: Pipeline Ingesti Dokumen

**Status: Selesai**

Sub-tahapan yang telah diimplementasikan:

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Ekstraksi teks PDF per-halaman | `pkg/pdf/extract.go` | ✅ |
| Deteksi boilerplate statistik (>30% halaman) | `pkg/pdf/extract.go` | ✅ |
| Pembersihan daftar isi (regex) | `pkg/pdf/extract.go` | ✅ |
| Chunking berbasis kata (512 char, overlap 64) | `usecase/ingestion.go` | ✅ |
| Deduplikasi Jaccard (threshold 0.75) | `usecase/ingestion.go` | ✅ |
| Embedding paralel (4 worker pool) | `usecase/ingestion.go` | ✅ |
| Simpan ke Qdrant via gRPC | `repository/qdrant.go` | ✅ |
| Simpan file PDF ke disk | `handler/document.go` | ✅ |

## 6.4 Tahap 4: Pencarian Hybrid

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Vector search via Qdrant gRPC | `repository/qdrant.go` | ✅ |
| Ekspansi kandidat (max(20, 4×TopK)) | `repository/qdrant.go` | ✅ |
| Ekstraksi keyword (stopword filter ID) | `usecase/chat.go` | ✅ |
| Keyword boost reranking (+0.1/match) | `repository/qdrant.go` | ✅ |

## 6.5 Tahap 5: Rekayasa Prompt dan Generasi

**Status: Selesai**

| Sub-tahap | Status |
|---|---|
| System prompt bilingual (EN/ID) via `promptLang` map | ✅ |
| Penyertaan konteks bernomor dengan atribusi sumber | ✅ |
| Dukungan riwayat percakapan multi-turn | ✅ |
| Instruksi anti-hallucination (fallback ke Admin UPT) | ✅ |
| Instruksi anti-hedging (larang kalimat pembuka basa-basi) | ✅ |
| Instruksi anti-attribution (larang sebut nama file) | ✅ |
| Instruksi language enforcement (jawab sesuai bahasa pilihan) | ✅ |

## 6.6 Tahap 6: API dan Manajemen Dokumen

**Status: Selesai**

| Endpoint | Metode | Keterangan | Status |
|---|---|---|---|
| `/api/chat` | POST | RAG query + language field | ✅ |
| `/api/documents/upload` | POST | Upload PDF + ingest | ✅ |
| `/api/documents` | GET | Daftar dokumen di Qdrant | ✅ |
| `/api/documents/{filename}` | DELETE | Hapus dari Qdrant + disk | ✅ |
| `/uploads/{filename}` | GET | Serving file PDF statis | ✅ |

## 6.7 Tahap 7: Antarmuka Pengguna

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

## 6.8 Tahap 8: Sistem Bilingual (i18n)

**Status: Selesai**

| Sub-tahap | Implementasi | Status |
|---|---|---|
| Terjemahan EN/ID semua string UI | `src/lib/i18n.ts` | ✅ |
| Svelte writable store (`lang`) | `src/lib/i18n.ts` | ✅ |
| Persistensi pilihan bahasa via `localStorage` | `src/lib/i18n.ts` | ✅ |
| Reactive derived store (`t`) | `src/lib/i18n.ts` | ✅ |
| Toggle button EN/ID di semua halaman | Landing, Chat, Admin | ✅ |
| Kirim `language` field ke backend | `src/lib/api.ts` | ✅ |
| Prompt LLM sesuai bahasa (backend) | `usecase/chat.go` | ✅ |

## 6.9 Tantangan dan Solusi

| Tantangan | Solusi yang Diterapkan |
|---|---|
| Chunk prakata mendominasi hasil pencarian | Hybrid search + keyword boost reranking |
| Duplikasi konten dari multi-edisi dokumen | Deduplikasi Jaccard similarity (threshold 0.75) |
| Noise header/footer dari PDF | Deteksi boilerplate statistik lintas halaman (>30%) |
| Regex cleaning terlalu spesifik untuk satu dokumen | Ganti ke pendekatan statistik generik |
| Embedding lambat (sequential) | Worker pool paralel (4 goroutine) |
| LLM menjawab dalam Bahasa Inggris | Instruksi `ALWAYS respond in [language]` dalam prompt |
| LLM menambahkan kalimat pembuka basa-basi | Instruksi anti-hedging eksplisit dalam prompt |
| LLM menyebut nama file/dokumen dalam jawaban | Instruksi anti-attribution dalam prompt |
| LLM menjawab "Contoh 1-2" (nomor artifak) | Instruksi larang sebut nomor contoh |
| Jawaban non-deterministik | Turunkan temperature ke 0.3 (dari 0.1 yang terlalu kaku) |

## 6.10 Yang Belum Diimplementasikan (Pengembangan Lanjutan)

| Fitur | Keterangan |
|---|---|
| Autentikasi admin | Panel admin saat ini terbuka tanpa login |
| Streaming response | LLM saat ini non-streaming (response sekaligus) |
| Tracking nomor halaman PDF akurat | Saat ini nomor halaman bergantung pada data dari model embedding |
| Evaluasi kuantitatif (RAGAS) | Pengukuran precision/recall/faithfulness sistem RAG |
| Cross-encoder reranker | Reranking berbasis model ML untuk akurasi lebih tinggi |
| Sparse vector (BM25 native) | Hybrid search via Qdrant sparse vector API |
| Rate limiting | Pembatasan request per pengguna |
