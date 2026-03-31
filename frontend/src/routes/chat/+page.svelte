<script lang="ts">
	import { sendChatStream, pdfUrl, type Message } from '$lib/api';
	import { t, lang, toggleLang } from '$lib/i18n';
	import { tick } from 'svelte';
	import { marked } from 'marked';

	function renderMarkdown(text: string): string {
		return marked.parse(text) as string;
	}

	interface ChatEntry {
		role: 'user' | 'assistant';
		content: string;
		sources?: { filename: string; page: number; text: string }[];
		streaming?: boolean;
	}

	let entries: ChatEntry[] = $state([]);
	let input = $state('');
	let loading = $state(false);
	let error = $state('');
	let chatEl: HTMLDivElement;

	function toHistory(): Message[] {
		return entries.map(e => ({ role: e.role, content: e.content }));
	}

	async function submit() {
		const query = input.trim();
		if (!query || loading) return;

		entries.push({ role: 'user', content: query });
		input = '';
		loading = true;
		error = '';

		await tick();
		scrollToBottom();

		// Capture history before adding the streaming assistant entry
		const historySnapshot = toHistory().slice(0, -1);

		// Add streaming assistant entry immediately so the cursor appears
		entries.push({ role: 'assistant', content: '', sources: [], streaming: true });
		const idx = entries.length - 1;

		try {
			await sendChatStream(
				query,
				historySnapshot,
				$lang,
				(token) => {
					entries[idx].content += token;
					scrollToBottom();
				},
				(sources) => {
					entries[idx].sources = sources;
					entries[idx].streaming = false;
				},
				(err) => {
					error = err;
					entries.splice(idx, 1);
				}
			);
		} catch {
			error = $t.serverError;
			entries.splice(idx, 1);
		} finally {
			loading = false;
			await tick();
			scrollToBottom();
		}
	}

	function scrollToBottom() {
		chatEl?.scrollTo({ top: chatEl.scrollHeight, behavior: 'smooth' });
	}

	function onKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			submit();
		}
	}

	function clearChat() {
		entries = [];
		error = '';
	}

	function uniqueSources(sources: ChatEntry['sources']) {
		if (!sources) return [];
		const seen = new Set<string>();
		return sources.filter(s => {
			if (seen.has(s.filename)) return false;
			seen.add(s.filename);
			return true;
		});
	}
</script>

<svelte:head>
	<title>Chat — UNILA AI</title>
</svelte:head>

<div class="h-screen flex flex-col" style="background-color: #f8f7f4;">

	<!-- Header -->
	<header class="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between shrink-0">
		<div class="flex items-center gap-4">
			<a href="/" class="text-gray-400 hover:text-gray-600 transition-colors text-sm">{$t.home}</a>
			<div class="w-px h-4 bg-gray-200"></div>
			<div>
				<h1 class="text-sm font-semibold" style="color: #1a3557;">{$t.chatTitle}</h1>
				<p class="text-xs" style="color: #9ca3af;">{$t.chatSubtitle}</p>
			</div>
		</div>
		<div class="flex items-center gap-3">
			<button
				onclick={toggleLang}
				class="text-xs px-2.5 py-1 rounded-md border font-mono transition-colors hover:bg-gray-50"
				style="color: #6b7280; border-color: #e5e7eb;">
				{$lang === 'en' ? 'EN' : 'ID'}
			</button>
			{#if entries.length > 0}
				<button onclick={clearChat} class="text-xs text-gray-400 hover:text-gray-600 transition-colors">
					{$t.clearChat}
				</button>
			{/if}
		</div>
	</header>

	<!-- Messages -->
	<div bind:this={chatEl} class="flex-1 overflow-y-auto px-4 py-8">
		<div class="max-w-2xl mx-auto space-y-6">

			{#if entries.length === 0}
				<div class="text-center pt-16">
					<div class="w-12 h-12 rounded-2xl mx-auto mb-4 flex items-center justify-center"
						style="background-color: #1a3557;">
						<span class="text-white text-lg">U</span>
					</div>
					<h2 class="text-base font-medium mb-2" style="font-family: 'Lora', serif; color: #1a3557;">
						{$t.emptyTitle}
					</h2>
					<p class="text-sm" style="color: #9ca3af;">
						{$t.emptyDesc}
					</p>
					<div class="mt-8 grid grid-cols-1 sm:grid-cols-2 gap-3 text-left">
						{#each $t.suggestions as suggestion}
							<button
								onclick={() => { input = suggestion; }}
								class="text-left px-4 py-3 rounded-xl border border-gray-200 bg-white text-sm hover:border-gray-300 hover:shadow-sm transition-all"
								style="color: #374151;">
								{suggestion}
							</button>
						{/each}
					</div>
				</div>
			{/if}

			{#each entries as entry}
				<div class="flex {entry.role === 'user' ? 'justify-end' : 'justify-start'} gap-3">
					{#if entry.role === 'assistant'}
						<div class="w-7 h-7 rounded-lg shrink-0 mt-0.5 flex items-center justify-center"
							style="background-color: #1a3557;">
							<span class="text-white text-xs font-bold">U</span>
						</div>
					{/if}

					<div class="max-w-lg">
						<div class="rounded-2xl px-4 py-3 text-sm leading-relaxed
							{entry.role === 'user'
								? 'text-white rounded-br-sm'
								: 'bg-white border border-gray-100 shadow-sm text-gray-800 rounded-bl-sm'}"
							style={entry.role === 'user' ? 'background-color: #1a3557;' : ''}>
							{#if entry.role === 'assistant'}
								{#if entry.streaming}
									<span class="whitespace-pre-wrap">{entry.content}</span><span class="inline-block w-0.5 h-4 ml-0.5 align-middle animate-pulse" style="background-color: #c9a84c;"></span>
								{:else}
									{@html renderMarkdown(entry.content)}
								{/if}
							{:else}
								{entry.content}
							{/if}
						</div>

						<!-- Source links -->
						{#if entry.role === 'assistant' && uniqueSources(entry.sources).length > 0}
							<div class="mt-2 flex flex-wrap gap-2">
								{#each uniqueSources(entry.sources) as src}
									<a
										href={pdfUrl(src.filename)}
										target="_blank"
										rel="noopener noreferrer"
										class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs border transition-all hover:shadow-sm"
										style="color: #1a3557; border-color: #c9d9ec; background-color: #f0f4f9;">
										<svg class="w-3 h-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
										</svg>
										{src.filename}
									</a>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{/each}

			{#if loading && !entries.at(-1)?.streaming}
				<div class="flex justify-start gap-3">
					<div class="w-7 h-7 rounded-lg shrink-0 flex items-center justify-center"
						style="background-color: #1a3557;">
						<span class="text-white text-xs font-bold">U</span>
					</div>
					<div class="bg-white border border-gray-100 shadow-sm rounded-2xl rounded-bl-sm px-4 py-3">
						<div class="flex gap-1 items-center h-4">
							<span class="w-1.5 h-1.5 rounded-full animate-bounce" style="background-color: #c9a84c; animation-delay: 0ms;"></span>
							<span class="w-1.5 h-1.5 rounded-full animate-bounce" style="background-color: #c9a84c; animation-delay: 150ms;"></span>
							<span class="w-1.5 h-1.5 rounded-full animate-bounce" style="background-color: #c9a84c; animation-delay: 300ms;"></span>
						</div>
					</div>
				</div>
			{/if}

			{#if error}
				<p class="text-center text-xs py-2 px-4 rounded-lg bg-red-50 text-red-500 border border-red-100">
					{error}
				</p>
			{/if}
		</div>
	</div>

	<!-- Input -->
	<div class="bg-white border-t border-gray-200 px-4 py-4 shrink-0">
		<div class="max-w-2xl mx-auto">
			<div class="flex gap-3 items-end rounded-2xl border border-gray-200 bg-gray-50 px-4 py-3 focus-within:border-gray-400 focus-within:bg-white transition-all">
				<textarea
					bind:value={input}
					onkeydown={onKeydown}
					placeholder={$t.placeholder}
					rows="1"
					class="flex-1 resize-none bg-transparent text-sm outline-none placeholder:text-gray-400 max-h-32"
					style="color: #1f2937;"
				></textarea>
				<button
					onclick={submit}
					disabled={loading || !input.trim()}
					class="shrink-0 w-8 h-8 rounded-xl flex items-center justify-center transition-all disabled:opacity-30"
					style="background-color: #1a3557;">
					<svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" d="M12 19V5m-7 7 7-7 7 7" />
					</svg>
				</button>
			</div>
			<p class="text-xs text-center mt-2" style="color: #d1d5db;">
				{$t.inputHint}
			</p>
		</div>
	</div>
</div>
