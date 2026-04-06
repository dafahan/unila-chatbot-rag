# Bab II — Pipeline Ingesti Dokumen

Pipeline ingesti adalah proses mengubah dokumen PDF menjadi representasi vektor yang dapat dicari secara semantik. Terdapat enam tahap utama.

## 2.1 Tahap 1: Ekstraksi Teks dari PDF

**Implementasi:** `backend/pkg/pdf/extract.go`

Dokumen PDF diekstraksi menggunakan pustaka `ledongthuc/pdf`. Proses membaca setiap halaman secara berurutan, mengumpulkan teks per-halaman, lalu menggabungkannya setelah pembersihan.

```
PDF (binary) → Ekstraksi per-halaman → []string (teks tiap halaman)
                                             ↓
                              Deteksi Boilerplate Statistik
                                             ↓
                              Teks Bersih → Gabungkan → Raw text
```

**Tantangan:** Dokumen PDF universitas mengandung:
- Header dan footer berulang di setiap halaman (contoh: *"Panduan Penulisan Karya Ilmiah Universitas Lampung 59"*)
- Baris daftar isi yang berisi titik-titik panjang (*".............................. 27"*)
- Nomor halaman berdiri sendiri
- Prefiks artefak dari struktur buku digital (contoh: `BUKU –`)

## 2.2 Tahap 2: Pembersihan Teks

### 2.2.1 Deteksi Boilerplate Statistik

**Implementasi:** `backend/pkg/pdf/extract.go` — fungsi `detectBoilerplate()`

Alih-alih menggunakan ekspresi reguler (*regex*) yang spesifik untuk dokumen tertentu, sistem menggunakan pendekatan **statistik lintas halaman** yang generik.

**Prinsip:** Baris teks yang muncul di lebih dari 30% halaman dokumen dianggap sebagai boilerplate (header, footer, penanda bagian berulang) dan dihapus secara otomatis.

**Algoritma:**

```
Untuk setiap halaman dalam dokumen:
  Catat baris-baris unik yang muncul di halaman tersebut

Hitung frekuensi kemunculan setiap baris lintas halaman

Threshold = max(2, 30% × jumlah halaman)

Jika frekuensi baris ≥ threshold → tandai sebagai boilerplate
```

### 2.2.2 Pembersihan Regex

Setelah boilerplate dihapus, regex minimal diterapkan:

| Pola | Aksi |
|---|---|
| Baris titik-titik dari daftar isi (`\.{4,}\s*\d*`) | Dihapus |
| Baris kosong berlebihan (`\n{3,}`) | Dinormalisasi menjadi dua baris |

### 2.2.3 Pembersihan Noise PDF (cleanText)

**Implementasi:** `backend/internal/usecase/ingestion.go` — fungsi `cleanText()`

Sebelum chunking, setiap halaman dibersihkan dari artefak struktural PDF yang muncul sebagai prefiks baris. Contoh kasus: panduan SIAKAD menghasilkan baris seperti `BUKU – 1.3) Tab Domisili` di mana `BUKU –` adalah artefak navigasi dokumen digital yang tidak bermakna sebagai konten.

```go
func cleanText(text string) string {
    lines := strings.Split(text, "\n")
    for i, line := range lines {
        if idx := strings.Index(line, "BUKU –"); idx != -1 {
            lines[i] = strings.TrimSpace(line[idx+len("BUKU –"):])
        }
    }
    return strings.Join(lines, "\n")
}
```

**Keunggulan pendekatan statistik vs regex hardcoded:**

| Aspek | Regex Hardcoded | Statistik (digunakan) |
|---|---|---|
| Berlaku untuk dokumen lain | ❌ Perlu ditulis ulang | ✅ Otomatis |
| Ketergantungan pada format spesifik | ❌ Tinggi | ✅ Tidak ada |
| Maintenance | ❌ Perlu update tiap dokumen baru | ✅ Tidak perlu |

## 2.3 Tahap 3: Pemecahan Teks (*Chunking*)

**Implementasi:** `backend/internal/usecase/ingestion.go` — fungsi `splitIntoChunks()`

Teks dibagi menjadi potongan (*chunk*) berukuran tetap dengan *overlap* untuk menjaga konteks antar-potongan.

**Parameter (dikonfigurasi via `.env`):**

| Parameter | Nilai | Keterangan |
|---|---|---|
| `CHUNK_SIZE` | 300 karakter | Target ukuran setiap *chunk* |
| `CHUNK_OVERLAP` | 100 karakter | Karakter yang diulang antar-*chunk* bersebelahan |

Chunk yang lebih kecil (300 vs 512 sebelumnya) menghasilkan representasi vektor yang lebih presisi per topik, mengurangi noise semantik dalam satu chunk.

**Mekanisme *overlap*:**
```
Chunk 1: [============================= ... ==]
Chunk 2:                          [====== ... ============]
                                  ↑ Overlap zona
```

## 2.4 Tahap 4: Deduplikasi *Chunk*

**Implementasi:** `backend/internal/usecase/ingestion.go` — fungsi `deduplicateChunks()`

Dokumen akademik seringkali mengandung teks yang hampir identik di beberapa lokasi. *Chunk* duplikat menyebabkan hasil pencarian didominasi oleh konten yang serupa.

**Metode:** Kemiripan Jaccard (*Jaccard Similarity*) pada himpunan kata:

$$J(A, B) = \frac{|A \cap B|}{|A \cup B|}$$

Jika $J > 0.75$, *chunk* dianggap duplikat dan dibuang.

## 2.5 Tahap 5: Pembuatan Embedding dan BM25

**Implementasi:** `backend/internal/usecase/ingestion.go` (paralel) + `backend/internal/repository/qdrant.go`

Setiap *chunk* dikonversi ke dua representasi vektor:

### Dense Vector (Semantic)
Model embedding `bge-m3` via Ollama menghasilkan vektor berdimensi **1024**. Dimensi dideteksi otomatis saat startup dengan melakukan probe embedding — tidak hardcoded sehingga fleksibel saat ganti model.

```go
func NewOllamaAdapter(baseURL, model, embedModel string) (*OllamaAdapter, error) {
    a := &OllamaAdapter{...}
    vec, err := a.GenerateEmbedding(context.Background(), "test")
    a.dimension = len(vec)  // auto-detect, bukan hardcode
    return a, nil
}
```

**Keunggulan bge-m3 vs nomic-embed-text:**
- Dimensi lebih tinggi (1024 vs 768) → representasi lebih kaya
- Dioptimalkan untuk teks multibahasa termasuk Bahasa Indonesia
- Retrieval recall lebih baik untuk query semantik

### Sparse Vector (BM25 Lexical)
Selain dense vector, setiap chunk juga divektorisasi menggunakan **BM25** (Best Match 25) — algoritma pencarian berbasis frekuensi kata yang diimplementasikan secara custom (`backend/pkg/bm25/index.go`).

BM25 menghasilkan sparse vector berisi pasangan `(indeks_kata, bobot_tf_idf)` yang hanya berisi kata-kata yang muncul dalam chunk tersebut.

**Pemrosesan paralel dengan worker pool:**
```
Chunks → [Worker 1] ─┐
         [Worker 2] ─┤→ Qdrant Upsert (gRPC)
         [Worker 3] ─┤
         [Worker 4] ─┘
(4 worker, dibatasi untuk efisiensi memori pada 16GB RAM)
```

**Data yang disimpan per titik di Qdrant:**

| Field | Tipe | Keterangan |
|---|---|---|
| `id` | UUID | Identifier unik titik |
| `vector.dense` | float32[1024] | Representasi semantik bge-m3 |
| `vector.bm25` | sparse float32 | Representasi leksikal BM25 |
| `payload.text` | string | Teks asli *chunk* |
| `payload.filename` | string | Nama file PDF sumber |
| `payload.document_id` | UUID | Identifier dokumen |
| `payload.page` | int | Nomor halaman |

**Metrik jarak:** Cosine Similarity untuk dense vector.

## 2.6 Penyimpanan File PDF

**Implementasi:** `backend/internal/handler/document.go`

File PDF asli disimpan ke disk untuk keperluan atribusi sumber.

```
Upload PDF
    │
    ├── Simpan file ke disk → ./uploads/{filename}
    │
    └── Proses ingesti → Qdrant

Serving statis: GET /uploads/{filename} → file PDF
```
