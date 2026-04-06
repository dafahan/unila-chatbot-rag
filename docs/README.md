# Dokumentasi Sistem RAG UNILA AI

Dokumentasi teknis-akademik sistem chatbot berbasis Retrieval-Augmented Generation (RAG) untuk Universitas Lampung.

## Daftar Dokumen

| No | Dokumen | Isi |
|---|---|---|
| 1 | [Gambaran Sistem](01-gambaran-sistem.md) | Arsitektur umum, tujuan, alur kerja |
| 2 | [Pipeline Ingesti](02-pipeline-ingesti.md) | Ekstraksi PDF → Chunking → Embedding → Qdrant |
| 3 | [Strategi Pencarian](03-strategi-pencarian.md) | Hybrid search BM25 + dense, RRF fusion |
| 4 | [Alur RAG & Prompt](04-rag-flow.md) | Konstruksi prompt, relevance check, query rewriting, multilingual |
| 5 | [Stack Teknologi](05-stack-teknologi.md) | Justifikasi pemilihan teknologi |
| 6 | [Tahapan Implementasi](06-implementasi-tahapan.md) | Status implementasi dan tantangan |
| 7 | [Evaluasi RAGAS](07-evaluasi-ragas.md) | Metodologi dan hasil evaluasi kuantitatif |

## Ringkasan Sistem

```
Dokumen PDF (Admin)
      ↓
  Ekstraksi per-halaman + Deteksi Boilerplate Statistik
      ↓
  Pembersihan Noise PDF (cleanText)
      ↓
  Chunking (300 char, overlap 100)
      ↓
  Deduplikasi Jaccard (threshold 0.75)
      ↓
  Embedding bge-m3 (1024-dim) + BM25 Sparse Vector
      ↓
  Qdrant Vector DB (Dense + Sparse)

      ↓ ← ← ← ← ← ← ← ← ← ← ← ← ←
                                      ↑
Pertanyaan Mahasiswa (EN/ID)          ↑
      ↓                               ↑
  Query Rewriting (keyword retrieval)  ↑
      ↓                               ↑
  Embed Query (bge-m3) + BM25         ↑
      ↓                               ↑
  Hybrid Search RRF Fusion (Dense + Sparse)
      ↓
  Top-8 Chunks → Relevance Check (LLM)
      ↓
  Build Prompt Bilingual (EN/ID) → Llama 3 8B
      ↓
  Jawaban (Markdown) + Link Sumber PDF → Mahasiswa
```

## Fitur Utama

| Fitur | Status |
|---|---|
| Upload & ingesti dokumen PDF | ✅ |
| Pembersihan noise PDF (header/footer/artefak) | ✅ |
| Hybrid semantic search (dense bge-m3 + BM25 sparse) | ✅ |
| RRF fusion ranking | ✅ |
| Deduplikasi chunk Jaccard | ✅ |
| Deteksi boilerplate statistik (header/footer) | ✅ |
| Embedding paralel (worker pool 4) | ✅ |
| Auto-detect dimensi embedding | ✅ |
| Context relevance check (guardrail halusinasi) | ✅ |
| Query rewriting untuk retrieval | ✅ |
| Manajemen dokumen (list & delete) | ✅ |
| Serving PDF statis untuk atribusi sumber | ✅ |
| Antarmuka bilingual EN/ID (toggle) | ✅ |
| Prompt LLM bilingual sesuai bahasa pengguna | ✅ |
| Query translation EN→ID sebelum retrieval | ✅ |
| Streaming response (SSE, token per token) | ✅ |
| Riwayat percakapan multi-turn | ✅ |
| Rendering Markdown pada respons | ✅ |
| Evaluasi RAGAS (faithfulness, precision, recall) | ✅ |

## Hasil Evaluasi RAGAS

| Metrik | Skor | Status |
|---|---|---|
| Faithfulness | 0.9833 | ✅ Excellent |
| Context Recall | 0.8849 | ✅ Good |
| Context Precision | 0.7890 | ✅ Good |
| **Overall Average** | **0.8858** | ✅ |

Evaluasi dilakukan terhadap 22 pertanyaan dari 5 dokumen resmi UNILA menggunakan RAGAS 0.2.6.
