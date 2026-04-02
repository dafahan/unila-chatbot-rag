# Context Relevance Check

Fitur ini menambahkan langkah *relevance check* sebelum LLM menghasilkan jawaban. Tujuannya adalah mencegah LLM menjawab di luar domain regulasi resmi UNILA ketika hasil retrieval Qdrant tidak relevan dengan pertanyaan.

## Prinsip Desain

UNILA AI adalah sistem **domain-locked** — bukan general chatbot. Seluruh jawaban harus bersumber dari dokumen regulasi resmi kampus. Mengizinkan LLM menjawab dari *pre-trained knowledge* ketika konteks tidak tersedia akan menghasilkan informasi generik yang **tidak mencerminkan aturan UNILA** dan berpotensi menyesatkan mahasiswa.

Konsekuensinya: ketika konteks Qdrant tidak relevan, sistem **wajib short-circuit** dan mengembalikan pesan statis — bukan memanggil LLM untuk menebak jawaban.

## Alur Pipeline

```
User Query
    │
    ▼
Retrieval (Qdrant hybrid: dense + BM25)
    │
    ▼
checkRelevance() ── LLM menilai: apakah context relevan? (YA / TIDAK)
    │
    ├── TIDAK ──► SHORT-CIRCUIT
    │             Kembalikan pesan statis tanpa memanggil LLM generator
    │             "Informasi ini tidak tersedia pada dokumen regulasi UNILA.
    │              Silakan hubungi Admin UPT."
    │
    └── YA ──► buildPrompt dengan context dokumen
               │
               ▼
           Generate jawaban (stream / non-stream)
```

## Detail Implementasi

### `checkRelevance()` — `usecase/chat.go`

Melakukan satu LLM call non-streaming untuk menilai relevansi:

```
Kamu adalah penilai relevansi. Tugasmu HANYA menentukan apakah konteks
berikut relevan untuk menjawab pertanyaan.

Pertanyaan: <query>
Konteks: [chunk 1] [chunk 2] ...

Apakah konteks di atas RELEVAN? Jawab HANYA dengan satu kata: YA atau TIDAK.
```

- Parsing dengan `strings.HasPrefix` — toleran terhadap whitespace dan variasi kapitalisasi
- Jika LLM call gagal (timeout, error): default `true` agar hasil retrieval tidak dibuang

### Short-circuit pada `Answer()` dan `AnswerStream()`

```go
if !uc.checkRelevance(ctx, req.Query, chunks) {
    // Langsung kembalikan pesan statis, tidak ada LLM generation call
    return &domain.ChatResponse{Answer: noInfoMessage(req.Language)}, nil
}
```

Untuk streaming, pesan statis dikirim melalui `onToken` tanpa membuka stream LLM.

### `buildPrompt()` — satu mode saja

Karena LLM generation hanya dipanggil ketika konteks relevan, `buildPrompt` tidak lagi memiliki mode "tanpa context". Context selalu dimasukkan ke prompt, dan fallback rule hanya untuk edge case ketika jawaban spesifik tidak ditemukan di dalam context yang sudah divalidasi relevan.

## Trade-off

| | Nilai |
|---|---|
| LLM call tambahan per request | +1 (relevance check) |
| LLM generation call saat tidak relevan | 0 (di-skip) |
| Risiko halusinasi dari pre-trained knowledge | Dieliminasi |
| Konsistensi domain | ✓ Selalu bersumber dari regulasi resmi |

Dibanding pendekatan sebelumnya (LLM menjawab dari pengetahuan sendiri), short-circuit lebih aman dan lebih efisien: tidak ada generation call sia-sia, dan tidak ada risiko mahasiswa menerima informasi generik yang tidak berlaku di UNILA.
