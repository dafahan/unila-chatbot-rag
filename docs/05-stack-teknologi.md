# Bab V — Stack Teknologi dan Justifikasi

## 5.1 Backend: Go (Golang)

**Versi:** Go 1.22+

**Justifikasi:**
- Performa tinggi dengan *concurrency* native via goroutine — krusial untuk worker pool embedding paralel
- Konsumsi memori rendah dibanding runtime JVM atau Python, sesuai batasan 16GB RAM
- Kompilasi ke binary tunggal, mempermudah deployment di server universitas
- Standard library HTTP yang kuat tanpa framework eksternal berat

**Pola Arsitektur:** Clean Architecture (Handler → UseCase → Repository/Adapter)

**Pola Tambahan:**
- *Strategy Pattern* — `LLMProvider` interface memungkinkan pertukaran Ollama/Gemini tanpa mengubah use case
- *Worker Pool* — 4 goroutine paralel untuk embedding batch

## 5.2 Vector Database: Qdrant

**Versi:** latest (Docker image)
**Protokol:** gRPC (port 6334) untuk performa operasi, HTTP REST (port 6333) untuk monitoring

**Justifikasi:**
- Open-source, dapat di-*self-host* tanpa biaya lisensi
- Mendukung **hybrid search** — dense vector + sparse vector dalam satu query dengan RRF fusion
- Payload filtering — memungkinkan filter berdasarkan metadata (nama file, dll.) untuk operasi delete per-dokumen
- Web UI bawaan di `/dashboard` untuk monitoring tanpa tools tambahan
- Sparse vector API (BM25) sudah aktif digunakan dalam sistem

**Koleksi Qdrant yang digunakan:**
- Dense vector: `bge-m3` (1024-dim, Cosine Similarity)
- Sparse vector: `bm25` (variable-dim, dot product)

**Alternatif yang dipertimbangkan:** Milvus (lebih kompleks), Weaviate (konsumsi RAM lebih tinggi), pgvector (kurang optimal untuk pencarian murni vektor).

## 5.3 LLM Utama: Ollama + Llama 3 8B

**Model:** `llama3:8b-instruct-q4_K_M`
**Ukuran:** ~4.9 GB (format GGUF, kuantisasi Q4_K_M)
**Runtime:** Ollama (port 11434, native install)

**Justifikasi:**
- Berjalan sepenuhnya lokal (*self-hosted*) — tidak ada data mahasiswa yang dikirim ke luar
- Q4_K_M: kuantisasi 4-bit dengan kualitas lebih baik dari Q4_0, masih muat dalam RAM 16GB
- Llama 3 8B memiliki kemampuan *instruction following* yang baik untuk Bahasa Indonesia maupun Bahasa Inggris
- Ollama menyediakan REST API standar yang mudah diintegrasikan

**Parameter Generasi:**
| Parameter | Nilai | Alasan |
|---|---|---|
| `temperature` | 0.3 | Cukup deterministik untuk informasi akademik, tidak kaku |
| `top_p` | 0.9 | Mempertahankan variasi ekspresi yang wajar |

## 5.4 Model Embedding: bge-m3

**Model:** `bge-m3` via Ollama
**Dimensi:** 1024 (auto-detect saat startup)
**Ukuran:** ~1.2 GB

**Justifikasi:**
- Didesain untuk teks multibahasa — performa retrieval Bahasa Indonesia jauh lebih baik dibanding `nomic-embed-text`
- Dimensi lebih tinggi (1024 vs 768) menghasilkan representasi semantik yang lebih kaya
- State-of-the-art untuk tugas retrieval multibahasa

**Perbandingan dengan alternatif:**

| Model | Dimensi | Bahasa Indonesia | Ukuran |
|---|---|---|---|
| nomic-embed-text (sebelumnya) | 768 | ⚠️ Terbatas | 274 MB |
| **bge-m3 (digunakan)** | **1024** | **✅ Baik** | **1.2 GB** |
| multilingual-e5-large | 1024 | ✅ Baik | ~1.2 GB |

**Auto-detect dimensi:** `OllamaAdapter` melakukan probe embedding saat startup dan menggunakan dimensi aktual dari model, sehingga tidak perlu hardcode — model embedding dapat diganti hanya dengan mengubah `OLLAMA_EMBED_MODEL` di `.env` dan meng-recreate koleksi Qdrant.

## 5.5 LLM Cadangan: Google Gemini API

**Model:** `gemini-2.0-flash` (completion), `text-embedding-004` (embedding)

**Justifikasi:**
- Sebagai fallback jika Ollama tidak tersedia atau untuk evaluasi perbandingan
- `gemini-2.0-flash` memiliki kemampuan Bahasa Indonesia dan Bahasa Inggris yang sangat baik
- Aktivasi cukup dengan mengubah `LLM_ENGINE=gemini` di `.env`

## 5.6 Frontend: SvelteKit

**Versi:** SvelteKit 2.x, Svelte 5.x, Vite 7.x
**Runtime:** Bun
**Styling:** Tailwind CSS 4.x

**Justifikasi:**
- Svelte 5 dengan *runes* (`$state`, `$derived`) — reaktivitas yang lebih efisien dan eksplisit
- SvelteKit mendukung SSR dan SPA dalam satu framework
- Bundle size minimal dibanding React/Next.js
- Tailwind CSS untuk styling cepat

**Sistem i18n:**
```typescript
export const lang = writable<Lang>(stored ?? 'en');
export const t = derived(lang, $lang => translations[$lang]);
export function toggleLang() { lang.update(l => l === 'en' ? 'id' : 'en'); }
```

**Pustaka Tambahan:**
- `marked` — rendering Markdown pada respons LLM

## 5.7 Evaluasi: RAGAS

**Versi:** ragas 0.2.6
**Evaluator LLM:** Groq API (llama-3.1-8b-instant) via OpenAI-compatible endpoint
**Embedding evaluasi:** Jina AI API (jina-embeddings-v3)

**Metrik yang diukur:**
- `faithfulness` — seberapa setia jawaban terhadap konteks yang diberikan
- `context_precision` — seberapa presisi chunk yang di-retrieve
- `context_recall` — seberapa lengkap konteks yang ditemukan

## 5.8 Infrastruktur

| Komponen | Teknologi | Keterangan |
|---|---|---|
| Containerisasi | Docker + Docker Compose | Hanya untuk Qdrant |
| Ollama | Native install (systemd/daemon) | Langsung di host OS |
| Backend | Go binary via `air` (dev) | Live reload saat development |
| Environment | `.env` file + `godotenv` | Konfigurasi terpusat |
| File Storage | Disk lokal + Go static server | PDF di `./uploads/` |

## 5.9 Ringkasan Dependensi Backend (Go)

| Pustaka | Fungsi |
|---|---|
| `github.com/qdrant/go-client` | Klien gRPC Qdrant |
| `github.com/google/generative-ai-go` | SDK Google Gemini |
| `github.com/ledongthuc/pdf` | Ekstraksi teks PDF |
| `github.com/joho/godotenv` | Load file .env |
| `github.com/google/uuid` | Generate UUID |
| `google.golang.org/grpc` | Komunikasi gRPC ke Qdrant |

## 5.10 Konfigurasi RAG (`.env`)

| Variable | Nilai | Keterangan |
|---|---|---|
| `OLLAMA_EMBED_MODEL` | `bge-m3` | Model embedding |
| `OLLAMA_MODEL` | `llama3:8b-instruct-q4_K_M` | Model LLM |
| `CHUNK_SIZE` | `300` | Ukuran chunk dalam karakter |
| `CHUNK_OVERLAP` | `100` | Overlap antar chunk |
| `TOP_K` | `8` | Jumlah chunk yang diambil per query |
| `SCORE_THRESHOLD` | `0.06` | Skor minimum untuk chunk yang dikembalikan |
