import { writable, derived } from 'svelte/store';

export type Lang = 'en' | 'id';

const stored = (typeof localStorage !== 'undefined' ? localStorage.getItem('lang') : null) as Lang | null;
export const lang = writable<Lang>(stored ?? 'en');

lang.subscribe(v => {
	if (typeof localStorage !== 'undefined') localStorage.setItem('lang', v);
});

export function toggleLang() {
	lang.update(l => (l === 'en' ? 'id' : 'en'));
}

const translations = {
	en: {
		// Nav
		startChat: 'Start Chat',
		admin: 'Admin',

		// Landing
		tagline: 'Universitas Lampung — Academic Information System',
		heroTitle: 'Your Virtual',
		heroAccent: 'Academic Assistant',
		heroDesc: 'Get quick and accurate answers about academic regulations, administrative procedures, and campus services — powered by official UNILA documents.',
		btnAsk: 'Ask a Question',
		btnUpload: 'Upload Document',
		feat1Title: 'Based on Official Documents',
		feat1Desc: 'Answers are drawn directly from official university documents and regulations.',
		feat2Title: 'Semantic Search',
		feat2Desc: 'Understands the intent of your question, not just keyword matching.',
		feat3Title: 'Fast Response',
		feat3Desc: 'Powered by a local AI model running on campus infrastructure.',
		footer: 'Universitas Lampung. Local AI-powered RAG System.',

		// Chat
		chatTitle: 'UNILA Academic Assistant',
		chatSubtitle: 'Based on official university documents',
		clearChat: 'Clear conversation',
		emptyTitle: 'How can I help you?',
		emptyDesc: 'Ask about academic regulations, course registration, KKN, graduation, and other campus services.',
		suggestions: [
			'What are the requirements for academic leave?',
			'How do I apply for KKN?',
			'What is the maximum credit load per semester?',
			'What documents are needed for graduation?',
		],
		placeholder: 'Type your question...',
		send: 'Send',
		inputHint: 'Enter to send · Shift+Enter for new line',
		serverError: 'Failed to reach server. Make sure the backend is running.',

		// Admin
		adminTitle: 'Admin Panel',
		adminSubtitle: 'Knowledge Base Management',
		uploadTitle: 'Upload Document',
		uploadDesc: 'Add official university PDF documents to the chatbot knowledge base.',
		dropzone: 'Drag file here or click to select',
		dropzoneHint: 'PDF files only · Multiple files allowed',
		filesReady: 'file(s) ready to upload',
		uploadBtn: 'Start Upload',
		uploading: 'Processing document...',
		uploadError: 'Upload failed',
		knowledgeBase: 'Knowledge Base',
		documents: 'documents',
		noDocuments: 'No documents uploaded yet.',
		chunks: 'chunks',
		deleteBtn: 'Delete',
		deleting: 'Deleting...',
		home: '← Home',
	},
	id: {
		// Nav
		startChat: 'Mulai Chat',
		admin: 'Admin',

		// Landing
		tagline: 'Universitas Lampung — Sistem Informasi Akademik',
		heroTitle: 'Asisten Akademik',
		heroAccent: 'Virtual Anda',
		heroDesc: 'Dapatkan jawaban seputar peraturan akademik, prosedur administrasi, dan layanan kampus secara cepat dan akurat berbasis dokumen resmi UNILA.',
		btnAsk: 'Mulai Bertanya',
		btnUpload: 'Upload Dokumen',
		feat1Title: 'Berbasis Dokumen Resmi',
		feat1Desc: 'Jawaban diambil langsung dari dokumen dan peraturan resmi universitas.',
		feat2Title: 'Pencarian Semantik',
		feat2Desc: 'Memahami maksud pertanyaan, bukan sekadar mencocokkan kata kunci.',
		feat3Title: 'Respons Cepat',
		feat3Desc: 'Ditenagai model AI lokal yang berjalan di infrastruktur kampus.',
		footer: 'Universitas Lampung. Sistem RAG berbasis AI lokal.',

		// Chat
		chatTitle: 'Asisten Akademik UNILA',
		chatSubtitle: 'Berbasis dokumen resmi universitas',
		clearChat: 'Bersihkan percakapan',
		emptyTitle: 'Ada yang bisa saya bantu?',
		emptyDesc: 'Tanyakan seputar peraturan akademik, KRS, KKN, wisuda, dan layanan kampus lainnya.',
		suggestions: [
			'Apa syarat untuk mengajukan cuti akademik?',
			'Bagaimana prosedur pengajuan KKN?',
			'Berapa SKS maksimal yang bisa diambil per semester?',
			'Apa saja dokumen yang diperlukan untuk wisuda?',
		],
		placeholder: 'Tulis pertanyaan Anda...',
		send: 'Kirim',
		inputHint: 'Enter untuk kirim · Shift+Enter untuk baris baru',
		serverError: 'Gagal menghubungi server. Pastikan backend berjalan.',

		// Admin
		adminTitle: 'Panel Admin',
		adminSubtitle: 'Manajemen Knowledge Base',
		uploadTitle: 'Upload Dokumen',
		uploadDesc: 'Tambahkan dokumen PDF resmi universitas ke dalam knowledge base chatbot.',
		dropzone: 'Seret file ke sini atau klik untuk memilih',
		dropzoneHint: 'Hanya file PDF · Bisa lebih dari satu',
		filesReady: 'file siap diupload',
		uploadBtn: 'Mulai Upload',
		uploading: 'Memproses dokumen...',
		uploadError: 'Gagal upload',
		knowledgeBase: 'Knowledge Base',
		documents: 'dokumen',
		noDocuments: 'Belum ada dokumen yang diupload.',
		chunks: 'chunks',
		deleteBtn: 'Hapus',
		deleting: 'Menghapus...',
		home: '← Beranda',
	},
} as const;

export type T = typeof translations.en;
export const t = derived(lang, $lang => translations[$lang]);
