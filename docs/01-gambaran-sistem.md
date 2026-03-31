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
| Vector Database | Qdrant | Penyimpanan dan pencarian vektor |
| LLM Utama | Ollama + Llama 3 8B (Q4_K_M) | Generasi teks dan embedding |
| LLM Cadangan | Google Gemini API | Alternatif LLM berbasis cloud |
| File Storage | Disk lokal + static server | Penyimpanan dan serving PDF |

## 1.4 Alur Kerja Utama

Sistem memiliki dua alur kerja utama:

### Alur Ingesti Dokumen (Admin)
```
Upload PDF → Ekstraksi Teks → Deteksi Boilerplate Statistik →
Chunking → Deduplikasi Jaccard → Embedding Paralel → Simpan ke Qdrant
```

### Alur Chat RAG (Mahasiswa)
```
Pertanyaan (+ pilihan bahasa EN/ID) → Embedding → Pencarian Hybrid →
Keyword Boost Reranking → Konstruksi Prompt Bilingual →
LLM → Jawaban (Markdown) + Link Sumber PDF
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
