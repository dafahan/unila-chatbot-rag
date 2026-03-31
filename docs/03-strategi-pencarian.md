# Bab III — Strategi Pencarian (Retrieval)

Komponen *retrieval* adalah inti dari sistem RAG. Kualitas jawaban yang dihasilkan sangat bergantung pada relevansi *chunk* yang berhasil ditemukan.

## 3.1 Pencarian Vektor (*Dense Retrieval*)

**Implementasi:** `backend/internal/repository/qdrant.go`

Pencarian vektor mengubah pertanyaan pengguna menjadi vektor menggunakan model embedding yang sama dengan yang digunakan saat ingesti, kemudian mencari *chunk* dengan vektor paling mirip menggunakan **Cosine Similarity**:

$$\text{sim}(q, d) = \frac{q \cdot d}{\|q\| \cdot \|d\|}$$

Di mana $q$ adalah vektor pertanyaan dan $d$ adalah vektor *chunk* dokumen.

**Kelemahan pencarian vektor murni:**
Pencarian semantik bekerja berdasarkan *makna*, bukan kata kunci. Ini menyebabkan *chunk* yang mengandung kata-kata "format penulisan karya ilmiah UNILA" (dari bagian prakata) mendapat skor tinggi untuk hampir semua pertanyaan tentang karya ilmiah — meski bukan konten yang relevan.

## 3.2 Pencarian Hybrid (*Hybrid Retrieval*)

Untuk mengatasi kelemahan di atas, sistem mengimplementasikan **pencarian hybrid** yang menggabungkan pencarian vektor dengan penguatan berbasis kata kunci (*keyword boost*).

### Alur Pencarian Hybrid

```
Pertanyaan Pengguna
       │
       ├──→ [Embedding Model] ──→ Query Vector
       │                                │
       └──→ [Keyword Extractor]         │
                    │                   ▼
                    │         [Qdrant Vector Search]
                    │         Ambil 4×TopK kandidat
                    │                   │
                    └──────────→ [Keyword Boost Reranker]
                                 skor_final = vektor_sim + (0.1 × jumlah_keyword_match)
                                        │
                                        ▼
                                 [Urutkan & Ambil Top-K]
                                        │
                                        ▼
                                 Chunk Relevan → LLM
```

### 3.2.1 Ekstraksi Kata Kunci

**Implementasi:** `backend/internal/usecase/chat.go` — fungsi `extractKeywords()`

Kata kunci diekstraksi dari pertanyaan dengan membuang *stopwords* Bahasa Indonesia dan kata pendek:

**Daftar stopwords yang difilter:**
`apa, yang, di, ke, dari, dan, untuk, dengan, ini, itu, adalah, bagaimana, cara, tentang, pada, dalam, atau, juga, ada, tidak, bisa, saya, kamu, gua`

**Aturan:** Kata dengan panjang ≤ 3 karakter juga dibuang.

**Contoh:**
```
Input:  "Bagaimana format halaman judul skripsi di UNILA?"
Output: ["format", "halaman", "judul", "skripsi", "unila"]
```

### 3.2.2 Keyword Boost Reranking

Setiap kandidat hasil vector search dinilai ulang:

$$\text{skor\_final}_i = \text{cosine\_sim}_i + \sum_{k \in K} \mathbb{1}[\text{keyword}_k \in \text{chunk}_i] \times 0.1$$

Di mana $K$ adalah himpunan kata kunci dari pertanyaan.

**Efek:** *Chunk* yang mengandung kata "halaman" dan "judul" mendapat boost +0.2, sehingga lebih diprioritaskan di atas *chunk* prakata yang hanya mirip secara semantik.

### 3.2.3 Ekspansi Kandidat

Sistem mengambil **4×TopK kandidat** dari Qdrant (minimum 20) sebelum reranking, untuk memastikan ada cukup ruang bagi *chunk* relevan yang mungkin tersembunyi di peringkat lebih rendah.

| Konfigurasi | Nilai |
|---|---|
| `TOP_K` (hasil akhir) | 8 |
| Kandidat awal (sebelum rerank) | 32 (4×8) |

## 3.3 Perbandingan Strategi Pencarian

| Strategi | Kelebihan | Kekurangan |
|---|---|---|
| **BM25 (Keyword only)** | Tepat untuk kata kunci eksak | Tidak memahami sinonim/makna |
| **Dense Vector only** | Memahami makna/sinonim | Dapat salah pada konten berulang |
| **Hybrid (digunakan)** | Menggabungkan kelebihan keduanya | Lebih kompleks |
| **Cross-encoder reranker** | Akurasi tertinggi | Lambat, memerlukan model terpisah |

Sistem menggunakan **Hybrid dengan Keyword Boost** sebagai keseimbangan antara akurasi dan efisiensi komputasi, mengingat batasan hardware 16GB RAM.

## 3.4 Konfigurasi Pencarian

| Parameter | Nilai | Keterangan |
|---|---|---|
| Model Embedding | `nomic-embed-text` | 768 dimensi, berjalan lokal via Ollama |
| Dimensi Vektor | 768 | Fixed oleh model |
| Metrik Jarak | Cosine Similarity | Standar untuk teks semantik |
| TopK Hasil Akhir | 8 | Dikonfigurasi via `TOP_K` |
| Kandidat Awal | max(20, 4×TopK) | Untuk ruang reranking |
| Keyword Boost | +0.1 per keyword | Konstanta empiris |
| Threshold Duplikasi | Jaccard > 0.75 | Saat ingesti |
