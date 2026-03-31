# Dokumentasi Sistem RAG UNILA AI

Dokumentasi teknis-akademik sistem chatbot berbasis Retrieval-Augmented Generation (RAG) untuk Universitas Lampung.

## Daftar Dokumen

| No | Dokumen | Isi |
|---|---|---|
| 1 | [Gambaran Sistem](01-gambaran-sistem.md) | Arsitektur umum, tujuan, alur kerja |
| 2 | [Pipeline Ingesti](02-pipeline-ingesti.md) | Ekstraksi PDF → Chunking → Embedding → Qdrant |
| 3 | [Strategi Pencarian](03-strategi-pencarian.md) | Hybrid search, keyword boost, deduplikasi |
| 4 | [Alur RAG & Prompt](04-rag-flow.md) | Konstruksi prompt, multilingual, multi-turn, fallback |
| 5 | [Stack Teknologi](05-stack-teknologi.md) | Justifikasi pemilihan teknologi |
| 6 | [Tahapan Implementasi](06-implementasi-tahapan.md) | Status implementasi dan tantangan |

## Ringkasan Sistem

```
Dokumen PDF (Admin)
      ↓
  Ekstraksi + Deteksi Boilerplate Statistik
      ↓
  Chunking (512 char, overlap 64)
      ↓
  Deduplikasi Jaccard (threshold 0.75)
      ↓
  Embedding nomic-embed-text (768-dim, 4 worker paralel)
      ↓
  Qdrant Vector DB (Cosine Similarity)

      ↓ ← ← ← ← ← ← ← ← ← ← ← ← ←
                                      ↑
Pertanyaan Mahasiswa (EN/ID)          ↑
      ↓                               ↑
  Embed Query                         ↑
      ↓                               ↑
  Ekstrak Keyword                     ↑
      ↓                               ↑
  Hybrid Search (Vector + Keyword Boost)
      ↓
  Top-8 Chunks → Prompt Bilingual (EN/ID) → Llama 3 8B
      ↓
  Jawaban (Markdown) + Link Sumber PDF → Mahasiswa
```

## Fitur Utama

| Fitur | Status |
|---|---|
| Upload & ingesti dokumen PDF | ✅ |
| Hybrid semantic search (vector + keyword) | ✅ |
| Deduplikasi chunk Jaccard | ✅ |
| Deteksi boilerplate statistik (header/footer) | ✅ |
| Embedding paralel (worker pool) | ✅ |
| Manajemen dokumen (list & delete) | ✅ |
| Serving PDF statis untuk atribusi sumber | ✅ |
| Antarmuka bilingual EN/ID (toggle) | ✅ |
| Prompt LLM bilingual sesuai bahasa pengguna | ✅ |
| Query translation EN→ID sebelum retrieval | ✅ |
| Streaming response (SSE, token per token) | ✅ |
| Riwayat percakapan multi-turn | ✅ |
| Rendering Markdown pada respons | ✅ |
| Fallback ke Gemini API | ✅ |
