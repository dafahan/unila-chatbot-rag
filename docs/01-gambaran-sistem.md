# Bab I — Gambaran Umum Sistem

## 1.1 Latar Belakang

Mahasiswa Universitas Lampung (UNILA) seringkali membutuhkan informasi seputar peraturan akademik, prosedur administrasi, dan layanan kampus yang tersebar dalam berbagai dokumen resmi universitas. Keterbatasan akses terhadap informasi yang cepat dan akurat mendorong pengembangan sistem asisten virtual berbasis kecerdasan buatan.

Sistem ini mengimplementasikan pendekatan **Retrieval-Augmented Generation (RAG)**, yaitu teknik yang menggabungkan kemampuan pencarian semantik (*retrieval*) dengan kemampuan generasi teks model bahasa besar (*large language model*). Pendekatan RAG dipilih karena memungkinkan model untuk menjawab berdasarkan dokumen yang spesifik dan dapat diperbarui, tanpa perlu melatih ulang model dari awal.

## 1.2 Tujuan Sistem

1. Menyediakan antarmuka tanya-jawab **bilingual (Bahasa Indonesia dan Bahasa Inggris)** untuk mahasiswa UNILA.
2. Menjawab pertanyaan berdasarkan dokumen resmi universitas yang diunggah administrator.
3. Memberikan atribusi sumber jawaban sehingga mahasiswa dapat merujuk dan membuka dokumen asli (PDF).
4. Berjalan secara mandiri (*self-hosted*) di infrastruktur lokal universitas tanpa ketergantungan penuh pada layanan cloud.

## 1.3 Arsitektur Sistem

Sistem terdiri dari tiga lapisan utama yang mengikuti prinsip *Clean Architecture*:

```
┌─────────────────────────────────────────────────────┐
│                   Frontend (SvelteKit)               │
│  Halaman Beranda · Halaman Chat · Panel Admin        │
│  Toggle Bahasa EN/ID · Link Sumber PDF               │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP REST API
┌──────────────────────▼──────────────────────────────┐
│              Backend (Go — REST API)                 │
│                                                      │
│  Handler → Use Case → Repository / Adapter          │
│  ├── ChatUseCase      (RAG Flow + Bilingual Prompt) │
│  └── IngestionUseCase (Pipeline Dokumen)            │
│                                                      │
│  Static file server: /uploads/{filename}            │
└──────┬───────────────────────────┬──────────────────┘
       │ gRPC                      │ HTTP
┌──────▼──────┐           ┌────────▼────────┐
│   Qdrant    │           │     Ollama      │
│ (Vector DB) │           │   (LLM Lokal)   │
└─────────────┘           └─────────────────┘
```

### Komponen Utama

| Komponen | Teknologi | Fungsi |
|---|---|---|
| Frontend | SvelteKit + Tailwind CSS | Antarmuka pengguna bilingual |
| Backend API | Go 1.22+ | Logika bisnis dan orkestrasi |
| Vector Database | Qdrant | Penyimpanan dan pencarian vektor (dense + sparse) |
| LLM Utama | Ollama + Llama 3 8B (Q4_K_M) | Generasi teks |
| Model Embedding | Ollama + bge-m3 | Embedding teks multibahasa (1024-dim) |
| LLM Cadangan | Google Gemini API | Alternatif LLM berbasis cloud |
| File Storage | Disk lokal + static server | Penyimpanan dan serving PDF |

## 1.4 Alur Kerja Utama

Sistem memiliki dua alur kerja utama:

### Alur Ingesti Dokumen (Admin)
```
Upload PDF
    ↓
Ekstraksi Teks per-halaman
    ↓
Deteksi Boilerplate Statistik (header/footer berulang)
    ↓
Pembersihan Noise PDF (cleanText — artefak PDF seperti "BUKU –")
    ↓
Chunking berbasis kata (300 char, overlap 100)
    ↓
Deduplikasi Jaccard (threshold 0.75)
    ↓
Embedding paralel bge-m3 (dense, 1024-dim) + BM25 (sparse)
    ↓
Simpan ke Qdrant (dense vector + sparse BM25 vector + metadata)
```

### Alur Chat RAG (Mahasiswa)
```
Pertanyaan (+ pilihan bahasa EN/ID)
        ↓ [jika EN] translate query ke ID via LLM
  Query Rewriting → frasa kata kunci retrieval
        ↓
  Embed Query bge-m3 (dense) + BM25 vectorize (sparse)
        ↓
  Qdrant Hybrid Search — RRF Fusion (dense + sparse)
        ↓
  Top-8 Chunks
        ↓
  Context Relevance Check (LLM)
        ↓ relevant          ↓ tidak relevan
  Build Prompt           Build Prompt
  dengan konteks         tanpa konteks (noContextNote)
        ↓                        ↓
  LLM Generate (Llama 3 8B)
        ↓ streaming token per token (SSE)
  Jawaban muncul bertahap di UI + Link Sumber PDF
```

## 1.5 Dukungan Bilingual

Sistem mendukung dua bahasa secara penuh:

| Aspek | Bahasa Inggris (EN) | Bahasa Indonesia (ID) |
|---|---|---|
| Teks antarmuka | Semua label, tombol, hint | Semua label, tombol, hint |
| Saran pertanyaan | Dalam Bahasa Inggris | Dalam Bahasa Indonesia |
| System prompt LLM | EN instruction set | ID instruction set |
| Respons LLM | Selalu Bahasa Inggris | Selalu Bahasa Indonesia |

Pilihan bahasa disimpan di `localStorage` browser sehingga persisten antar sesi. Pergantian bahasa berlaku seketika tanpa *reload* halaman, memanfaatkan reaktivitas Svelte derived store.

## 1.6 Dokumen yang Diindeks

| Dokumen | Keterangan |
|---|---|
| Panduan Penulisan Karya Ilmiah 2020 | Format skripsi/tesis/disertasi |
| Peraturan Akademik 2025 | Peraturan Rektor tentang penyelenggaraan pendidikan |
| PANDUAN SIAKAD MAHASISWA | Panduan penggunaan sistem informasi akademik |
| SOP 2020 | Standar Operasional Prosedur akademik |
| Profil Program Studi 2024 | Data program studi UNILA |
| Statistik Akreditasi UNILA 2024 | Data akreditasi program studi |
| Panduan KKN UNILA | Panduan Kuliah Kerja Nyata |
