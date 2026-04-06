# Bab IV — Alur RAG dan Rekayasa Prompt

## 4.1 Alur Lengkap RAG (*Retrieval-Augmented Generation*)

RAG mengatasi keterbatasan utama LLM murni: model tidak mengetahui dokumen internal yang tidak ada dalam data pelatihannya. Dengan RAG, dokumen yang relevan disertakan langsung dalam konteks prompt saat menjawab.

```
┌─────────────────────────────────────────────────────────────────┐
│                        CHAT USE CASE                            │
│                                                                 │
│  1. Query Rewriting      2. Embed + BM25   3. Hybrid Search    │
│  ──────────────────      ──────────────    ────────────────     │
│  "Syarat KRS?"           dense (bge-m3)    Qdrant RRF Fusion   │
│       ↓                  + sparse (BM25)   → Top-8 Chunks      │
│  "syarat KRS SKS         (1024-dim)                            │
│   pengambilan mata                                              │
│   kuliah semester"                                              │
│                                                                 │
│  4. Relevance Check      5. Build Prompt   6. Generate         │
│  ──────────────────      ─────────────     ──────────          │
│  LLM: "konteks ini       Mode A: + ctx     LLM → Jawaban      │
│  relevan?" → YA/TIDAK    Mode B: noCtx     (bahasa sesuai     │
│                                            pilihan)            │
└─────────────────────────────────────────────────────────────────┘
```

## 4.2 Query Rewriting untuk Retrieval

**Implementasi:** `backend/internal/usecase/chat.go` — fungsi `rewriteForRetrieval()`

Sebelum embedding, query pengguna ditulis ulang menjadi frasa kata kunci bergaya dokumen. Ini menjembatani gap semantik antara cara pengguna bertanya dengan cara dokumen ditulis.

**Masalah tanpa query rewriting:**
> Query: *"Apa saja jenis dosen yang diakui di UNILA?"*
> Dokumen: *"Dosen terdiri atas dosen tetap, dosen tidak tetap, dan dosen tamu."*
>
> Kata *"diakui"* tidak muncul di dokumen. Dense embedding harus menjembatani gap ini sepenuhnya.

**Dengan query rewriting:**
```
Query asli: "Apa saja jenis dosen yang diakui di UNILA?"
     ↓ LLM rewrite
Kata kunci: "jenis dosen dosen tetap tidak tetap tamu"
     ↓ Gabungkan
Query retrieval: "Apa saja jenis dosen yang diakui di UNILA? jenis dosen dosen tetap tidak tetap tamu"
```

Gabungan query asli + kata kunci meningkatkan probabilitas kecocokan baik untuk BM25 (lexical) maupun dense (semantic).

## 4.3 Context Relevance Check (Guardrail Halusinasi)

**Implementasi:** `backend/internal/usecase/chat.go` — fungsi `checkRelevance()`

Setelah retrieval, sistem memvalidasi apakah chunk yang dikembalikan Qdrant memang **berkaitan topik** dengan pertanyaan. Ini mencegah LLM menjawab berdasarkan konteks yang sama sekali tidak relevan.

```go
func (uc *ChatUseCase) checkRelevance(ctx, query, chunks) bool {
    // Kirim pertanyaan + preview chunk ke LLM
    // Tanya: "Apakah konteks berkaitan dengan topik pertanyaan?"
    // Jawab TIDAK hanya jika topik berbeda total
    // → return true/false
}
```

**Dua mode buildPrompt berdasarkan hasil relevance check:**

| Mode | Kondisi | Perilaku |
|---|---|---|
| `contextRelevant = true` | Chunks topik berkaitan | Prompt menyertakan konteks dokumen, LLM wajib menjawab dari konteks |
| `contextRelevant = false` | Chunks sama sekali tidak relevan | Prompt tanpa konteks (noContextNote), LLM jawab dari pengetahuannya atau nyatakan tidak tersedia |

**Threshold relevance check yang longgar:** Sistem hanya menolak konteks yang *sama sekali* tidak berhubungan topik — bukan menolak hanya karena fakta spesifik tidak ada. Ini mencegah false negative yang menyebabkan LLM salah fallback ke "tidak tersedia".

## 4.4 Konstruksi Prompt Bilingual

**Implementasi:** `backend/internal/usecase/chat.go` — fungsi `buildPrompt()`

Sistem mendukung dua bahasa instruksi: **Bahasa Inggris (EN)** dan **Bahasa Indonesia (ID)**.

### Rules yang Diterapkan

| Rule | Tujuan |
|---|---|
| Langsung ke isi jawaban | Anti-hedging (larang kalimat pembuka basa-basi) |
| DILARANG sebut nama file/dokumen | Anti-attribution |
| DILARANG sebut gambar/tabel | Mencegah referensi elemen visual yang tidak bisa ditampilkan |
| Sebutkan lokasi UI saat menjelaskan aksi klik | Kontekstual untuk pertanyaan SIAKAD |
| DILARANG mengarang angka/nama/prosedur | Anti-hallucination |
| SALIN PERSIS nilai dari konteks | Mencegah paraphrase nilai spesifik (angka, URL, warna) |

### Struktur Lengkap Prompt (mode contextRelevant)

```
[SYSTEM INSTRUCTION — dalam bahasa terpilih]
Kamu adalah asisten akademik Universitas Lampung (UNILA).
STRICT RULES:
- Langsung ke isi jawaban
- DILARANG sebut nama file/dokumen
- DILARANG mengarang angka/nama/prosedur yang tidak ada di konteks
- SALIN PERSIS nilai spesifik dari konteks: angka, satuan, URL, warna
- Jika tidak ada di konteks → "Informasi ini tidak tersedia..."
- Gunakan bullet point jika ada daftar
- ALWAYS respond in [id/en]

=== CONTEXT ===
[1] (Source: Panduan-KTI.pdf, Page 15)
{teks chunk 1}

[2] (Source: Peraturan-Akademik-2025.pdf, Page 8)
{teks chunk 2}
...
=== END CONTEXT ===

=== CONVERSATION HISTORY === (jika ada)
USER: pertanyaan sebelumnya
ASSISTANT: jawaban sebelumnya
=== END HISTORY ===

STUDENT QUESTION: {pertanyaan mahasiswa}

ANSWER:
```

## 4.5 Atribusi Sumber PDF

Meskipun LLM dilarang menyebut nama file dalam jawaban, sistem menampilkan sumber di antarmuka frontend sebagai tautan yang dapat diklik.

```
Backend response:
{
  "answer": "Syarat cuti akademik adalah...",
  "sources": [
    { "filename": "Peraturan-Akademik-2025.pdf", "page_number": 23 },
    { "filename": "Peraturan-Akademik-2025.pdf", "page_number": 24 }
  ]
}

Frontend:
┌─────────────────────────────────────┐
│ Jawaban LLM (Markdown rendered)     │
│                                     │
│ 📄 Peraturan-Akademik-2025.pdf      │← Link ke /uploads/...
└─────────────────────────────────────┘
```

## 4.6 Manajemen Riwayat Percakapan

Sistem mendukung percakapan multi-gilir (*multi-turn conversation*). Riwayat disisipkan ke dalam prompt agar LLM memiliki konteks pertanyaan yang berkaitan.

Riwayat disimpan di sisi frontend (state Svelte) dan dikirim bersama setiap request. Backend bersifat *stateless*.

## 4.7 Streaming Response (Server-Sent Events)

Sistem mendukung streaming respons LLM token per token menggunakan **Server-Sent Events (SSE)**:

```
Frontend                          Backend (SSE)
   │                                    │
   ├──POST /api/chat/stream ──────────→ │
   │                                    │ translate + rewrite query
   │                                    │ embed + BM25 retrieval
   │                                    │ relevance check
   │                                    │ build prompt
   │                                    │ LLM generate (stream=true)
   │ ←── data: {"token":"Syarat"} ───── │
   │ ←── data: {"token":" cuti"} ────── │
   │ ←── data: {"done":true,"sources":[...]} ─ │
```

| Event | Format |
|---|---|
| Token | `data: {"token": "teks"}` |
| Selesai + sumber | `data: {"done": true, "sources": [...]}` |
| Error | `data: {"error": "pesan error"}` |

## 4.8 Dukungan Dua LLM Provider

Sistem mengimplementasikan antarmuka `LLMProvider` (*Strategy Pattern*):

```go
type LLMProvider interface {
    GenerateCompletion(ctx, prompt) (string, error)
    GenerateCompletionStream(ctx, prompt, onToken) error
    GenerateEmbedding(ctx, text)   ([]float32, error)
    EmbeddingDimension()            int
}
```

| Adapter | Model Completion | Model Embedding | Dimensi |
|---|---|---|---|
| `OllamaAdapter` | llama3:8b-instruct-q4_K_M | bge-m3 | 1024 (auto-detect) |
| `GeminiAdapter` | gemini-2.0-flash | text-embedding-004 | 768 |

Dimensi embedding pada `OllamaAdapter` dideteksi otomatis saat startup — tidak hardcoded sehingga model embedding dapat diganti tanpa mengubah kode.

Parameter generasi `OllamaAdapter`: `temperature: 0.3`, `top_p: 0.9`.
