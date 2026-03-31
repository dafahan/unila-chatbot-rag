# Bab IV — Alur RAG dan Rekayasa Prompt

## 4.1 Alur Lengkap RAG (*Retrieval-Augmented Generation*)

RAG mengatasi keterbatasan utama LLM murni: model tidak mengetahui dokumen internal yang tidak ada dalam data pelatihannya. Dengan RAG, dokumen yang relevan disertakan langsung dalam konteks prompt saat menjawab.

```
┌─────────────────────────────────────────────────────────────────┐
│                        CHAT USE CASE                            │
│                                                                 │
│  1. Embed Query          2. Hybrid Search      3. Build Prompt  │
│  ─────────────           ──────────────        ───────────────  │
│  "Syarat KRS?"           Qdrant →              System Prompt    │
│       ↓                  Top-8 Chunks          (EN atau ID)     │
│  [768-dim vector]        (vector + kw)         + Context[1..8]  │
│                                                + History        │
│                                                + Query          │
│                          4. Generate                            │
│                          ──────────                             │
│                          LLM → Jawaban (bahasa sesuai pilihan)  │
└─────────────────────────────────────────────────────────────────┘
```

## 4.2 Konstruksi Prompt Bilingual

**Implementasi:** `backend/internal/usecase/chat.go` — fungsi `buildPrompt()`

Sistem mendukung dua bahasa instruksi: **Bahasa Inggris (EN)** dan **Bahasa Indonesia (ID)**. Bahasa dipilih berdasarkan field `language` pada request dari frontend (default: `"en"`).

```go
var promptLang = map[string][6]string{
    "en": {
        "You are an academic assistant for Universitas Lampung (UNILA).",
        "Answer DIRECTLY and COMPLETELY based on the document context below.",
        "- Go straight to the answer, NO opening remarks whatsoever.",
        "- NEVER mention file names, document names, or example numbers.",
        "- NEVER write 'According to the document...' or similar phrases.",
        "- If information is unavailable, answer ONLY: 'This information is not available...'",
    },
    "id": {
        "Kamu adalah asisten akademik Universitas Lampung (UNILA).",
        "Jawab LANGSUNG dan LENGKAP berdasarkan konteks dokumen di bawah.",
        "- Langsung ke isi jawaban, TANPA kalimat pembuka apapun.",
        "- DILARANG menyebut nama file, nama dokumen, atau nomor contoh.",
        "- DILARANG menulis 'Menurut panduan...', 'Berdasarkan dokumen...'",
        "- Jika informasi tidak tersedia, jawab HANYA: 'Informasi ini tidak tersedia...'",
    },
}
```

### Struktur Lengkap Prompt

```
[SYSTEM INSTRUCTION — dalam bahasa terpilih]
You are an academic assistant... / Kamu adalah asisten akademik...
STRICT RULES:
- No opening remarks
- No file name mentions
- No hedging phrases
- Unavailable → direct to Admin UPT
- Use bullet points if listing
- ALWAYS respond in [selected language]

=== CONTEXT ===
[1] (Source: Panduan-KTI.pdf, Page 15)
{teks chunk 1}

[2] (Source: Panduan-KTI.pdf, Page 16)
{teks chunk 2}
...
[8] (Source: ...)
{teks chunk 8}
=== END CONTEXT ===

=== CONVERSATION HISTORY === (jika ada)
USER: pertanyaan sebelumnya
ASSISTANT: jawaban sebelumnya
=== END HISTORY ===

STUDENT QUESTION: {pertanyaan mahasiswa}

ANSWER:
```

### Prinsip Rekayasa Prompt yang Diterapkan

| Prinsip | Implementasi |
|---|---|
| **Role assignment** | "You are an academic assistant" / "Kamu adalah asisten akademik UNILA" |
| **Language enforcement** | "ALWAYS respond in [en/id]" — mencegah LLM berganti bahasa |
| **Grounding** | Jawab HANYA berdasarkan konteks yang diberikan |
| **Format instruction** | Gunakan bullet point jika ada daftar |
| **Anti-attribution** | Larang menyebut nama file/dokumen dalam jawaban |
| **Anti-hedging** | Larang kalimat pembuka seperti "Berdasarkan dokumen..." |
| **Hallucination prevention** | Jika tidak ada → arahkan ke Admin UPT |

## 4.3 Atribusi Sumber PDF

Meskipun LLM dilarang menyebut nama file dalam jawaban, sistem tetap menampilkan sumber di antarmuka frontend sebagai tautan yang dapat diklik langsung ke file PDF.

```
Backend response:
{
  "answer": "Syarat cuti akademik adalah...",
  "sources": [
    { "filename": "Panduan-Akademik.pdf", "page_number": 23 },
    { "filename": "Panduan-Akademik.pdf", "page_number": 24 }
  ]
}

Frontend:
┌─────────────────────────────────────┐
│ Jawaban LLM (Markdown rendered)     │
│                                     │
│ 📄 Panduan-Akademik.pdf             │← Link ke /uploads/Panduan-Akademik.pdf
└─────────────────────────────────────┘
```

Sumber yang sama dideduplikasi di frontend (berdasarkan filename) sehingga tidak muncul tautan ganda.

## 4.4 Manajemen Riwayat Percakapan

Sistem mendukung percakapan multi-gilir (*multi-turn conversation*). Riwayat percakapan sebelumnya disisipkan ke dalam prompt agar LLM memiliki konteks pertanyaan yang berkaitan.

Riwayat disimpan di sisi frontend (state Svelte) dan dikirim bersama setiap request. Backend bersifat *stateless*.

```
Request ke /api/chat:
{
  "query": "Apakah boleh ambil lebih dari 24 SKS?",
  "language": "id",
  "history": [
    { "role": "user",      "content": "Berapa SKS maksimal per semester?" },
    { "role": "assistant", "content": "SKS maksimal adalah 24 SKS..." }
  ]
}
```

## 4.5 Fallback dan Batasan Sistem

Jika tidak ada *chunk* yang relevan ditemukan, sistem mengembalikan respons standar sesuai bahasa:

- **EN:** *"This information is not available. Please contact the UPT Admin."*
- **ID:** *"Informasi ini tidak tersedia. Silakan hubungi Admin UPT."*

Ini mencegah model **mengarang jawaban** (*hallucination*) yang dapat menyesatkan mahasiswa.

## 4.6 Dukungan Dua LLM Provider

Sistem mengimplementasikan antarmuka `LLMProvider` yang memungkinkan pertukaran provider tanpa mengubah logika bisnis (*Strategy Pattern*):

```go
type LLMProvider interface {
    GenerateCompletion(ctx, prompt) (string, error)
    GenerateEmbedding(ctx, text)   ([]float32, error)
    EmbeddingDimension()            int
}
```

| Adapter | Model Completion | Model Embedding | Dimensi |
|---|---|---|---|
| `OllamaAdapter` | llama3:8b-instruct-q4_K_M | nomic-embed-text | 768 |
| `GeminiAdapter` | gemini-1.5-flash | text-embedding-004 | 768 |

Pemilihan dilakukan via environment variable `LLM_ENGINE=ollama` atau `LLM_ENGINE=gemini`.

Parameter generasi yang dikonfigurasi pada `OllamaAdapter`: `temperature: 0.3`, `top_p: 0.9` — dipilih untuk keseimbangan antara konsistensi jawaban dan variasi ekspresi yang wajar.
