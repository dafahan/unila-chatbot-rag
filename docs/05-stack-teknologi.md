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
- Mendukung cosine similarity, dot product, dan Euclidean distance
- Payload filtering — memungkinkan filter berdasarkan metadata (nama file, dll.) untuk operasi delete per-dokumen
- Web UI bawaan di `/dashboard` untuk monitoring tanpa tools tambahan
- Mendukung sparse vector untuk hybrid search BM25-native di masa depan

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

**Model Embedding:** `nomic-embed-text`
- Ukuran: 274 MB
- Dimensi: 768
- Dioptimalkan untuk pencarian semantik dokumen

## 5.4 LLM Cadangan: Google Gemini API

**Model:** `gemini-1.5-flash` (completion), `text-embedding-004` (embedding)

**Justifikasi:**
- Sebagai fallback jika Ollama tidak tersedia atau untuk evaluasi perbandingan
- `gemini-1.5-flash` memiliki kemampuan Bahasa Indonesia dan Bahasa Inggris yang sangat baik
- Dimensi embedding sama (768) — koleksi Qdrant tidak perlu dibuat ulang saat berganti provider

## 5.5 Frontend: SvelteKit

**Versi:** SvelteKit 2.x, Svelte 5.x, Vite 7.x
**Runtime:** Bun
**Styling:** Tailwind CSS 4.x

**Justifikasi:**
- Svelte 5 dengan *runes* (`$state`, `$derived`) — reaktivitas yang lebih efisien dan eksplisit
- SvelteKit mendukung SSR dan SPA dalam satu framework
- Bundle size minimal dibanding React/Next.js — penting untuk server dengan bandwidth terbatas
- Tailwind CSS untuk styling cepat tanpa CSS custom yang berlebihan

**Sistem i18n (Internasionalisasi):**

Sistem bahasa diimplementasikan menggunakan Svelte stores tanpa pustaka eksternal:

```typescript
// src/lib/i18n.ts
export const lang = writable<Lang>(stored ?? 'en');  // persisten via localStorage
export const t = derived(lang, $lang => translations[$lang]);
export function toggleLang() { lang.update(l => l === 'en' ? 'id' : 'en'); }
```

- `lang` — writable store, nilai aktif `'en'` atau `'id'`, disimpan ke `localStorage`
- `t` — derived store berisi seluruh string terjemahan yang aktif (reaktif)
- Toggle berlaku seketika di semua komponen tanpa *reload*
- Pilihan bahasa juga dikirim ke backend (`language` field di request) sehingga respons LLM menggunakan bahasa yang sama

**Pustaka Tambahan:**
- `marked` — rendering Markdown pada respons LLM (bullet point, penomoran, bold)

## 5.6 Infrastruktur

| Komponen | Teknologi | Keterangan |
|---|---|---|
| Containerisasi | Docker + Docker Compose | Hanya untuk Qdrant |
| Ollama | Native install (systemd/daemon) | Langsung di host OS |
| Backend | Go binary via `air` (dev) | Live reload saat development |
| Environment | `.env` file + `godotenv` | Konfigurasi terpusat |
| File Storage | Disk lokal + Go static server | PDF di `./uploads/`, serving di `/uploads/` |

## 5.7 Ringkasan Dependensi Backend (Go)

| Pustaka | Versi | Fungsi |
|---|---|---|
| `github.com/qdrant/go-client` | v1.17.1 | Klien gRPC Qdrant |
| `github.com/google/generative-ai-go` | v0.20.1 | SDK Google Gemini |
| `github.com/ledongthuc/pdf` | latest | Ekstraksi teks PDF |
| `github.com/joho/godotenv` | v1.5.1 | Load file .env |
| `github.com/google/uuid` | v1.6.0 | Generate UUID |
| `google.golang.org/grpc` | v1.79.3 | Komunikasi gRPC ke Qdrant |
