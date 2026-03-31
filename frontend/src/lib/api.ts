const BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080';

export function pdfUrl(filename: string): string {
	return `${BASE}/uploads/${encodeURIComponent(filename)}`;
}

export interface Message {
	role: 'user' | 'assistant';
	content: string;
}

export interface ChatResponse {
	answer: string;
	sources: { filename: string; page_number: number; text: string }[];
}

export async function sendChat(query: string, history: Message[], language = 'id'): Promise<ChatResponse> {
	const res = await fetch(`${BASE}/api/chat`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ query, history, language })
	});
	if (!res.ok) throw new Error(await res.text());
	return res.json();
}

export async function uploadPDF(file: File): Promise<{ filename: string; chunks: number }> {
	const form = new FormData();
	form.append('file', file);
	const res = await fetch(`${BASE}/api/documents/upload`, { method: 'POST', body: form });
	if (!res.ok) throw new Error(await res.text());
	return res.json();
}

export interface DocumentInfo {
	filename: string;
	chunk_count: number;
}

export async function listDocuments(): Promise<DocumentInfo[]> {
	const res = await fetch(`${BASE}/api/documents`);
	if (!res.ok) throw new Error(await res.text());
	return res.json();
}

export async function sendChatStream(
	query: string,
	history: Message[],
	language: string,
	onToken: (token: string) => void,
	onDone: (sources: ChatResponse['sources']) => void,
	onError: (err: string) => void
): Promise<void> {
	const res = await fetch(`${BASE}/api/chat/stream`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ query, history, language })
	});
	if (!res.ok) throw new Error(await res.text());

	const reader = res.body!.getReader();
	const decoder = new TextDecoder();
	let buffer = '';

	while (true) {
		const { done, value } = await reader.read();
		if (done) break;

		buffer += decoder.decode(value, { stream: true });
		const lines = buffer.split('\n');
		buffer = lines.pop() ?? '';

		for (const line of lines) {
			if (!line.startsWith('data: ')) continue;
			const raw = line.slice(6).trim();
			if (!raw) continue;
			const event = JSON.parse(raw);
			if (event.error) { onError(event.error); return; }
			if (event.token) onToken(event.token);
			if (event.done) onDone(event.sources ?? []);
		}
	}
}

export async function deleteDocument(filename: string): Promise<void> {
	const res = await fetch(`${BASE}/api/documents/${encodeURIComponent(filename)}`, {
		method: 'DELETE'
	});
	if (!res.ok) throw new Error(await res.text());
}
