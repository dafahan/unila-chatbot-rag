# Bab VII — Evaluasi RAGAS

## 7.1 Tujuan Evaluasi

Evaluasi kuantitatif dilakukan untuk mengukur kualitas sistem RAG secara objektif menggunakan kerangka **RAGAS** (*Retrieval-Augmented Generation Assessment*). Evaluasi mencakup tiga dimensi utama:

- **Faithfulness** — seberapa setia jawaban LLM terhadap konteks yang diberikan (anti-halusinasi)
- **Context Precision** — seberapa presisi chunk yang di-*retrieve* (relevansi terhadap *ground truth*)
- **Context Recall** — seberapa lengkap konteks yang ditemukan untuk menjawab pertanyaan

## 7.2 Infrastruktur Evaluasi

| Komponen | Nilai |
|---|---|
| Framework | RAGAS 0.2.6 |
| Evaluator LLM | Groq API — `llama-3.1-8b-instant` |
| Embedding evaluasi | Jina AI API — `jina-embeddings-v3` |
| API Compatibility | OpenAI-compatible endpoint (`api.groq.com/openai/v1`) |
| Script | `eval/run_ragas.py` |
| Dataset | `eval/eval_dataset.json` |
| Cache respons | `eval/responses_cache.json` |
| Cache skor | `eval/scores_cache.json` |

Groq dipilih sebagai *evaluator LLM* karena ketersediaan API gratis dengan latensi rendah, memungkinkan evaluasi 22 pertanyaan selesai dalam hitungan menit tanpa membebani Ollama lokal yang digunakan untuk inferensi RAG.

## 7.3 Dataset Evaluasi

Dataset terdiri dari **22 pertanyaan** yang mencakup **5 dokumen** berbeda yang telah diindeks ke sistem.

| # | Pertanyaan | Ground Truth | Sumber Dokumen |
|---|---|---|---|
| 1 | Berapa ukuran kertas standar untuk penulisan karya ilmiah di UNILA? | Kertas berukuran A4 (21 x 29,7 cm). | Panduan-Penulisan-Karya-Ilmiah-2020.pdf |
| 2 | Apa saja jenis dosen yang diakui di UNILA? | Dosen terdiri atas dosen tetap, dosen tidak tetap, dan dosen tamu. | Peraturan Akademik 2025.pdf |
| 3 | Apa saja subtab yang tersedia di bagian Data Mahasiswa SIAKAD? | Subtab: Ubah Foto, Informasi Umum, Domisili, Orang Tua, Wali, Sekolah. | PANDUAN-SIAKAD-MAHASISWA.pdf |
| 4 | Apa saja fitur yang tersedia di Menu Perkuliahan SIAKAD? | Kurikulum, KRS, jadwal kuliah, kuesioner, nilai/KHS, kemajuan belajar, MK mengulang, berhenti studi, status semester. | PANDUAN-SIAKAD-MAHASISWA.pdf |
| 5 | Bagaimana aturan penulisan judul bab dalam karya ilmiah UNILA? | Judul bab diketik 6 cm dari batas atas kertas, disertai logo UNILA serta tahun pada halaman sampul. | Panduan-Penulisan-Karya-Ilmiah-2020.pdf |
| 6 | Kapan semester ganjil dimulai di UNILA berdasarkan Peraturan Akademik 2025? | Semester Ganjil dimulai pada bulan Agustus. | Peraturan Akademik 2025.pdf |
| 7 | Apa saja bentuk penilaian yang diberikan kepada mahasiswa dalam mata kuliah praktek? | Kuis, UTS, UAS — diberikan paling lambat satu minggu setelah ujian. | SOP-2020.pdf |
| 8 | Berapa kali pertemuan maksimal dan minimal yang harus dipenuhi dosen dalam satu semester? | Maksimal 16 kali, minimal 14 kali pertemuan. | SOP-2020.pdf |
| 9 | Siapa yang menjadi koordinator pembuatan SAP? | Ketua Program Studi membuat surat kepada dosen penanggung jawab. | SOP-2020.pdf |
| 10 | Bagaimana alur pelaksanaan KKN di UNILA? | Pendaftaran → Pembagian kelompok → Orientasi → Proposal → Pelaksanaan → Pengawasan → Laporan → Presentasi → Penutupan. | kkn_unila.ac.pdf |
| 11 | Berapa berat kertas HVS yang digunakan untuk skripsi dan tesis? | 80 gram untuk skripsi/tesis/disertasi; 70 gram untuk laporan kerja mahasiswa. | Panduan-Penulisan-Karya-Ilmiah-2020.pdf |
| 12 | Bagaimana cara mendaftar KKN di UNILA? | Pendaftaran KKN secara online melalui portal Sentra KKN bagi yang memenuhi syarat. | kkn_unila.ac.pdf |
| 13 | Kapan SAP (Satuan Acara Perkuliahan) harus dibuat menurut SOP UNILA? | SAP dibuat sebelum perkuliahan dimulai pada setiap awal semester. | SOP-2020.pdf |
| 14 | Apa yang disampaikan saat orientasi KKN di UNILA? | Informasi mengenai lokasi KKN, etika penghidupan masyarakat, dan hal-hal terkait pelaksanaan KKN. | kkn_unila.ac.pdf |
| 15 | Berapa jumlah semester dalam satu tahun akademik di UNILA? | 2 (dua) semester: semester ganjil dan semester genap. | Peraturan Akademik 2025.pdf |
| 16 | Warna tinta apa yang digunakan untuk tulisan pada sampul skripsi? | Tinta berwarna emas pada sampul depan dan punggung sampul. | Panduan-Penulisan-Karya-Ilmiah-2020.pdf |
| 17 | Apa alamat website untuk mengakses SIAKAD UNILA? | http://siakadu.unila.ac.id/ | PANDUAN-SIAKAD-MAHASISWA.pdf |
| 18 | Ada berapa pilihan filter status pada menu Berita di SIAKAD? | 3 pilihan: Aktif, Prioritas, dan Aktif dan Prioritas. | PANDUAN-SIAKAD-MAHASISWA.pdf |
| 19 | Apa saja yang dibutuhkan untuk login ke SIAKAD? | Akun Pengguna dan Kata Sandi. | PANDUAN-SIAKAD-MAHASISWA.pdf |
| 20 | Berapa jam kegiatan untuk 1 SKS praktikum di UNILA? | 2 jam per minggu di dalam/luar lab, ditambah 1–2 jam terstruktur dan 1–2 jam mandiri. | SOP-2020.pdf |
| 21 | Berapa ukuran huruf dan font yang digunakan untuk teks isi skripsi? | Ukuran 12 dengan font Times New Roman. | Panduan-Penulisan-Karya-Ilmiah-2020.pdf |
| 22 | Kapan semester genap dimulai di UNILA? | Semester Genap dimulai pada bulan Februari. | Peraturan Akademik 2025.pdf |

**Distribusi per dokumen:**

| Dokumen | Jumlah Pertanyaan |
|---|---|
| Panduan-Penulisan-Karya-Ilmiah-2020.pdf | 5 |
| PANDUAN-SIAKAD-MAHASISWA.pdf | 5 |
| SOP-2020.pdf | 5 |
| Peraturan Akademik 2025.pdf | 4 |
| kkn_unila.ac.pdf | 3 |
| **Total** | **22** |

## 7.4 Metodologi Pengumpulan Hasil

### Fase 1: Pengumpulan Respons RAG

Script `eval/run_ragas.py` mengirimkan setiap pertanyaan ke endpoint `/api/chat` dan menyimpan respons (jawaban + chunk konteks) ke `responses_cache.json`. Cache ini mencegah pengulangan query ke sistem RAG jika evaluasi dijalankan berulang.

### Fase 2: Evaluasi RAGAS

Evaluasi dibagi dua fase untuk efisiensi:

- **Fase 1 (context metrics):** `context_precision` dan `context_recall` dievaluasi bersama karena tidak bergantung satu sama lain.
- **Fase 2 (faithfulness):** Dievaluasi terpisah karena memerlukan pemrosesan multi-step yang lebih intensif.

Skor disimpan secara inkremental ke `scores_cache.json`. Pada run berikutnya, hanya pertanyaan dengan skor `NaN` (gagal dievaluasi) yang di-*re-evaluate*, menghindari pemborosan API call.

### Kendala RAGAS 0.2.6

Selama evaluasi ditemukan beberapa *bug* pada RAGAS 0.2.6 yang memerlukan penanganan khusus:

| Bug | Penyebab | Solusi |
|---|---|---|
| `LLMDidNotFinishException` | Groq mengembalikan `finish_reason="eos_token"` alih-alih `"stop"` | `is_finished_parser=lambda _: True` pada `LangchainLLMWrapper` |
| `AttributeError: StringIO has no attribute 'sentences'` | Kegagalan parse output LLM mengembalikan objek `StringIO` alih-alih `None` | Patch `_faithfulness.py`: tambah `hasattr` check |
| `RagasOutputParserException` untuk jawaban berbentuk daftar | Filter kalimat RAGAS membuang baris yang tidak diakhiri titik (`.`) — bullet list terpotong habis | Patch `_faithfulness.py`: longgarkan filter dari *endpoint check* ke *non-empty check* |

Akibat *bug* ini, sebagian pertanyaan menghasilkan `faithfulness = NaN`. Skor agregat dihitung dari pertanyaan yang berhasil dievaluasi.

## 7.5 Hasil Evaluasi Per Pertanyaan

| # | Pertanyaan (disingkat) | Context Precision | Context Recall | Faithfulness |
|---|---|---|---|---|
| 1 | Ukuran kertas standar | 0.833 | 1.000 | 1.000 |
| 2 | Jenis dosen UNILA | 0.698 | 1.000 | 1.000 |
| 3 | Subtab Data Mahasiswa SIAKAD | 1.000 | 1.000 | NaN¹ |
| 4 | Fitur Menu Perkuliahan SIAKAD | 1.000 | 1.000 | NaN¹ |
| 5 | Aturan penulisan judul bab | 1.000 | 0.333 | 1.000 |
| 6 | Semester ganjil dimulai | 1.000 | 0.250 | 1.000 |
| 7 | Bentuk penilaian MK praktek | 0.000 | NaN¹ | NaN¹ |
| 8 | Pertemuan dosen maks/min | 0.833 | 1.000 | NaN¹ |
| 9 | Koordinator pembuatan SAP | 0.250 | 1.000 | 1.000 |
| 10 | Alur pelaksanaan KKN | 0.750 | 1.000 | 1.000 |
| 11 | Berat kertas HVS | 0.833 | 1.000 | NaN¹ |
| 12 | Cara mendaftar KKN | 0.967 | 1.000 | NaN¹ |
| 13 | Waktu pembuatan SAP | 0.722 | 0.000 | NaN¹ |
| 14 | Materi orientasi KKN | 1.000 | 1.000 | NaN¹ |
| 15 | Jumlah semester per tahun | 0.962 | 1.000 | NaN¹ |
| 16 | Warna tinta sampul skripsi | 1.000 | 1.000 | NaN¹ |
| 17 | Alamat website SIAKAD | 1.000 | 1.000 | 1.000 |
| 18 | Filter status berita SIAKAD | 0.333 | 1.000 | NaN¹ |
| 19 | Login SIAKAD | 0.444 | 1.000 | NaN¹ |
| 20 | SKS praktikum (jam) | 0.732 | 1.000 | 0.833 |
| 21 | Ukuran huruf & font skripsi | 1.000 | 1.000 | 1.000 |
| 22 | Semester genap dimulai | 1.000 | 1.000 | 1.000 |

¹ *NaN disebabkan bug RAGAS 0.2.6 pada parsing output berbentuk daftar atau ketidaksesuaian format finish_reason dari Groq API. Nilai dikecualikan dari rata-rata agregat.*

## 7.6 Hasil Evaluasi Agregat

| Metrik | Skor | Pertanyaan Valid |
|---|---|---|
| Faithfulness | **0.9833** | 10 / 22 |
| Context Recall | **0.8849** | 21 / 22 |
| Context Precision | **0.7890** | 22 / 22 |
| **Overall (rata-rata)** | **0.8858** | — |

*Rata-rata dihitung dari pertanyaan yang menghasilkan skor numerik valid (non-NaN).*

### Interpretasi Skor

**Faithfulness (0.9833):** Hampir semua jawaban yang dihasilkan sistem setia terhadap konteks yang diberikan — sistem tidak mengarang informasi di luar dokumen. Skor mendekati sempurna ini mencerminkan efektivitas *prompt engineering* yang diterapkan (aturan "DILARANG mengarang" dan "SALIN PERSIS nilai dari konteks").

**Context Recall (0.8849):** Sistem berhasil menemukan sebagian besar informasi yang relevan. Dua pertanyaan menunjukkan recall rendah (pertanyaan 5: 0.333, pertanyaan 6: 0.250, pertanyaan 13: 0.000) karena *ground truth* sangat spesifik dan teks tepatnya tersebar di bagian dokumen yang sulit dijangkau retrieval — kemungkinan karena boilerplate atau konteks yang terfragmentasi saat chunking.

**Context Precision (0.7890):** Sebagian besar chunk yang diambil relevan dengan pertanyaan. Beberapa pertanyaan (pertanyaan 7: 0.000, pertanyaan 9: 0.250, pertanyaan 18: 0.333) memiliki presisi rendah karena *hybrid search* RRF mengembalikan chunk dari dokumen lain yang sedikit tumpang tindih secara leksikal.

## 7.7 Analisis Per Dokumen

| Dokumen | Avg Context Precision | Avg Context Recall | Avg Faithfulness (valid) |
|---|---|---|---|
| Panduan KI 2020 | 0.933 | 0.867 | 1.000 |
| Peraturan Akademik 2025 | 0.915 | 0.813 | 1.000 |
| PANDUAN SIAKAD | 0.756 | 1.000 | 1.000 |
| SOP 2020 | 0.507 | 0.750 | 0.917 |
| KKN UNILA | 0.906 | 1.000 | 1.000 |

**Pengamatan:**
- **SOP-2020** memiliki presisi dan recall terendah. Dokumen ini mengandung banyak prosedur teknis berurutan yang menyebabkan chunking memotong konteks penting lintas chunk.
- **PANDUAN SIAKAD** recall sempurna (1.000) tetapi presisi lebih rendah karena ada tumpang tindih leksikal BM25 dengan dokumen prosedur lain.
- **KKN UNILA** konsisten tinggi di semua metrik, kemungkinan karena dokumen ini memiliki topik yang lebih terfokus dan tidak ambigu secara leksikal.

## 7.8 Kesimpulan

Sistem RAG UNILA AI mencapai skor keseluruhan **0.8858** dengan keunggulan utama pada faithfulness. Sistem terbukti sangat andal dalam menghindari halusinasi — jawaban yang dihasilkan konsisten bersumber dari dokumen yang diindeks, bukan pengetahuan generik LLM.

Area dengan ruang peningkatan:
- **Context Precision:** Penggunaan *cross-encoder reranker* setelah retrieval awal dapat meningkatkan relevansi chunk yang disertakan dalam prompt.
- **Context Recall untuk pertanyaan faktual spesifik:** Chunk size yang lebih kecil atau segmentasi berbasis paragraf/section dapat membantu menemukan fakta-fakta yang terpencil dalam dokumen.
