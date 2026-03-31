# Bab II — Pipeline Ingesti Dokumen

Pipeline ingesti adalah proses mengubah dokumen PDF menjadi representasi vektor yang dapat dicari secara semantik. Terdapat lima tahap utama.

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

## 2.2 Tahap 2: Pembersihan Teks — Deteksi Boilerplate Statistik

**Implementasi:** `backend/pkg/pdf/extract.go` — fungsi `detectBoilerplate()` dan `cleanText()`

### 2.2.1 Pendekatan Berbasis Statistik

Alih-alih menggunakan ekspresi reguler (*regex*) yang spesifik untuk dokumen tertentu, sistem menggunakan pendekatan **statistik lintas halaman** yang generik dan berlaku untuk dokumen PDF apapun.

**Prinsip:** Baris teks yang muncul di lebih dari 30% halaman dokumen dianggap sebagai boilerplate (header, footer, penanda bagian berulang) dan dihapus secara otomatis.

**Algoritma:**

```
Untuk setiap halaman dalam dokumen:
  Catat baris-baris unik yang muncul di halaman tersebut

Hitung frekuensi kemunculan setiap baris lintas halaman

Threshold = max(2, 30% × jumlah halaman)

Jika frekuensi baris ≥ threshold → tandai sebagai boilerplate
```

**Implementasi Go:**
```go
func detectBoilerplate(pages []string, threshold float64) map[string]bool {
    linePageCount := make(map[string]int)
    for _, page := range pages {
        seen := make(map[string]bool)
        for _, line := range strings.Split(page, "\n") {
            normalized := strings.TrimSpace(line)
            if len(normalized) < 5 { continue }
            if !seen[normalized] {
                linePageCount[normalized]++
                seen[normalized] = true
            }
        }
    }
    minPages := int(float64(len(pages)) * threshold)
    if minPages < 2 { minPages = 2 }
    boilerplate := make(map[string]bool)
    for line, count := range linePageCount {
        if count >= minPages { boilerplate[line] = true }
    }
    return boilerplate
}
```

### 2.2.2 Pembersihan Tambahan via Regex

Setelah boilerplate dihapus, regex minimal diterapkan untuk membersihkan artefak yang tersisa:

| Pola | Aksi |
|---|---|
| Baris titik-titik dari daftar isi (`\.{4,}\s*\d*`) | Dihapus |
| Baris kosong berlebihan (`\n{3,}`) | Dinormalisasi menjadi dua baris |

### 2.2.3 Keunggulan Pendekatan Statistik vs Regex Hardcoded

| Aspek | Regex Hardcoded | Statistik (digunakan) |
|---|---|---|
| Berlaku untuk dokumen lain | ❌ Perlu ditulis ulang | ✅ Otomatis |
| Ketergantungan pada format spesifik | ❌ Tinggi | ✅ Tidak ada |
| Maintenance | ❌ Perlu update tiap dokumen baru | ✅ Tidak perlu |
| Risiko menghapus konten valid | ⚠️ Ada jika pola terlalu luas | ✅ Minimum |

## 2.3 Tahap 3: Pemecahan Teks (*Chunking*)

**Implementasi:** `backend/internal/usecase/ingestion.go` — fungsi `splitIntoChunks()`

Teks dibagi menjadi potongan (*chunk*) berukuran tetap dengan *overlap* untuk menjaga konteks antar-potongan.

**Parameter (dapat dikonfigurasi via `.env`):**

| Parameter | Nilai Default | Keterangan |
|---|---|---|
| `CHUNK_SIZE` | 512 karakter | Target ukuran setiap *chunk* |
| `CHUNK_OVERLAP` | 64 karakter | Karakter yang diulang antar-*chunk* bersebelahan |

**Mekanisme *overlap*:**
```
Chunk 1: [============================= ... ==]
Chunk 2:                          [====== ... ============]
                                  ↑ Overlap zona
```

*Overlap* mencegah informasi penting yang berada di batas antar-*chunk* menjadi terpotong dan tidak terjawab.

## 2.4 Tahap 4: Deduplikasi *Chunk*

**Implementasi:** `backend/internal/usecase/ingestion.go` — fungsi `deduplicateChunks()`

Dokumen akademik seringkali mengandung teks yang hampir identik di beberapa lokasi (contoh: prakata dari berbagai edisi revisi yang mengulang kalimat yang sama). *Chunk* duplikat menyebabkan hasil pencarian didominasi oleh konten yang serupa.

**Metode:** Kemiripan Jaccard (*Jaccard Similarity*) pada himpunan kata:

$$J(A, B) = \frac{|A \cap B|}{|A \cup B|}$$

Di mana $A$ dan $B$ adalah himpunan kata dari dua *chunk*. Jika $J > 0.75$, *chunk* dianggap duplikat dan dibuang.

**Contoh:** Prakata edisi ke-2 dan ke-3 yang sama-sama menyebut "format umum yang dapat memayungi seluruh bidang ilmu" memiliki Jaccard > 0.75 sehingga hanya satu yang disimpan.

## 2.5 Tahap 5: Pembuatan Embedding dan Penyimpanan

**Implementasi:** `backend/internal/usecase/ingestion.go` (paralel) + `backend/internal/repository/qdrant.go`

Setiap *chunk* dikonversi menjadi vektor numerik berdimensi 768 menggunakan model embedding `nomic-embed-text` via Ollama.

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
| `vector` | float32[768] | Representasi semantik *chunk* |
| `payload.text` | string | Teks asli *chunk* |
| `payload.filename` | string | Nama file PDF sumber |
| `payload.document_id` | UUID | Identifier dokumen |
| `payload.page` | int | Nomor halaman |

**Metrik jarak:** Cosine Similarity — dipilih karena efektif untuk mengukur kemiripan semantik antar vektor teks.

## 2.6 Penyimpanan File PDF

**Implementasi:** `backend/internal/handler/document.go`

Selain data vektor di Qdrant, file PDF asli disimpan ke disk untuk keperluan atribusi sumber. Saat mahasiswa menerima jawaban, tautan ke halaman PDF asli disertakan.

```
Upload PDF
    │
    ├── Simpan file ke disk → ./uploads/{filename}
    │
    └── Proses ingesti → Qdrant

Serving statis: GET /uploads/{filename} → file PDF
```

Direktori upload dikonfigurasi via environment variable `UPLOAD_DIR` (default: `./uploads`).
