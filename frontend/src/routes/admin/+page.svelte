<script lang="ts">
	import { uploadPDF, listDocuments, deleteDocument, type DocumentInfo } from '$lib/api';
	import { t, lang, toggleLang } from '$lib/i18n';
	import { onMount } from 'svelte';

	let files: FileList | null = $state(null);
	let uploading = $state(false);
	let documents: DocumentInfo[] = $state([]);
	let error = $state('');
	let dragOver = $state(false);
	let deletingFile = $state('');
	let inputEl: HTMLInputElement;

	onMount(async () => {
		await loadDocuments();
	});

	async function loadDocuments() {
		try {
			documents = await listDocuments();
		} catch {
			// collection not yet created, skip
		}
	}

	async function upload() {
		if (!files || files.length === 0 || uploading) return;
		uploading = true;
		error = '';

		for (const file of Array.from(files)) {
			try {
				await uploadPDF(file);
			} catch (e) {
				error = `${$t.uploadError} ${file.name}: ${e instanceof Error ? e.message : 'Unknown error'}`;
			}
		}

		files = null;
		uploading = false;
		await loadDocuments();
	}

	async function remove(filename: string) {
		if (deletingFile) return;
		deletingFile = filename;
		error = '';
		try {
			await deleteDocument(filename);
			await loadDocuments();
		} catch (e) {
			error = `${$t.uploadError}: ${e instanceof Error ? e.message : 'Unknown error'}`;
		} finally {
			deletingFile = '';
		}
	}

	function onDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const dropped = e.dataTransfer?.files;
		if (dropped && dropped.length > 0) files = dropped;
	}
</script>

<svelte:head>
	<title>Admin — UNILA AI</title>
</svelte:head>

<div class="min-h-screen" style="background-color: #f8f7f4;">

	<header class="bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
		<div class="flex items-center gap-4">
			<a href="/" class="text-gray-400 hover:text-gray-600 transition-colors text-sm">{$t.home}</a>
			<div class="w-px h-4 bg-gray-200"></div>
			<div>
				<h1 class="text-sm font-semibold" style="color: #1a3557;">{$t.adminTitle}</h1>
				<p class="text-xs" style="color: #9ca3af;">{$t.adminSubtitle}</p>
			</div>
		</div>
		<button
			onclick={toggleLang}
			class="text-xs px-2.5 py-1 rounded-md border font-mono transition-colors hover:bg-gray-50"
			style="color: #6b7280; border-color: #e5e7eb;">
			{$lang === 'en' ? 'EN' : 'ID'}
		</button>
	</header>

	<div class="max-w-2xl mx-auto px-6 py-12">

		<div class="mb-8">
			<h2 class="text-2xl font-light mb-1" style="font-family: 'Lora', serif; color: #1a3557;">
				{$t.uploadTitle}
			</h2>
			<p class="text-sm" style="color: #6b7280;">
				{$t.uploadDesc}
			</p>
		</div>

		<!-- Drop zone -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			onclick={() => inputEl.click()}
			onkeydown={(e) => e.key === 'Enter' && inputEl.click()}
			ondragover={(e) => { e.preventDefault(); dragOver = true; }}
			ondragleave={() => dragOver = false}
			ondrop={onDrop}
			class="rounded-2xl border-2 border-dashed p-12 text-center cursor-pointer transition-all"
			style="border-color: {dragOver ? '#1a3557' : '#d1d5db'};
				   background-color: {dragOver ? '#f0f4f9' : 'white'};">

			<input bind:this={inputEl} type="file" accept=".pdf" multiple bind:files class="hidden" />

			<div class="w-12 h-12 rounded-xl mx-auto mb-4 flex items-center justify-center"
				style="background-color: #f0f4f9;">
				<svg class="w-6 h-6" style="color: #1a3557;" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m6.75 12-3-3m0 0-3 3m3-3v6m-1.5-15H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
				</svg>
			</div>

			{#if files && files.length > 0}
				<p class="text-sm font-medium mb-1" style="color: #1a3557;">
					{files.length} {$t.filesReady}
				</p>
				<p class="text-xs" style="color: #9ca3af;">
					{Array.from(files).map(f => f.name).join(', ')}
				</p>
			{:else}
				<p class="text-sm font-medium mb-1" style="color: #374151;">
					{$t.dropzone}
				</p>
				<p class="text-xs" style="color: #9ca3af;">{$t.dropzoneHint}</p>
			{/if}
		</div>

		<button
			onclick={upload}
			disabled={!files || files.length === 0 || uploading}
			class="mt-4 w-full rounded-xl py-3 text-sm font-medium text-white transition-all
			       hover:opacity-90 active:scale-95 disabled:opacity-40 disabled:cursor-not-allowed"
			style="background-color: #1a3557;">
			{#if uploading}
				<span class="flex items-center justify-center gap-2">
					<svg class="w-4 h-4 animate-spin" fill="none" viewBox="0 0 24 24">
						<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
						<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8z"></path>
					</svg>
					{$t.uploading}
				</span>
			{:else}
				{$t.uploadBtn}
			{/if}
		</button>

		{#if error}
			<div class="mt-4 px-4 py-3 rounded-xl bg-red-50 border border-red-100 text-sm text-red-600">
				{error}
			</div>
		{/if}

		<!-- Document list -->
		<div class="mt-10">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-xs font-semibold uppercase tracking-widest" style="color: #9ca3af;">
					{$t.knowledgeBase}
				</h3>
				<span class="text-xs" style="color: #9ca3af;">{documents.length} {$t.documents}</span>
			</div>

			{#if documents.length === 0}
				<div class="text-center py-10 rounded-2xl border border-dashed border-gray-200">
					<p class="text-sm" style="color: #9ca3af;">{$t.noDocuments}</p>
				</div>
			{:else}
				<ul class="space-y-2">
					{#each documents as doc}
						<li class="flex items-center justify-between bg-white rounded-xl border border-gray-100 px-5 py-4 shadow-sm">
							<div class="flex items-center gap-3 min-w-0">
								<div class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0"
									style="background-color: #f0f4f9;">
									<svg class="w-4 h-4" style="color: #1a3557;" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
										<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z" />
									</svg>
								</div>
								<div class="min-w-0">
									<p class="text-sm font-medium truncate" style="color: #1f2937;">{doc.filename}</p>
									<p class="text-xs" style="color: #9ca3af;">{doc.chunk_count} {$t.chunks}</p>
								</div>
							</div>

							<button
								onclick={() => remove(doc.filename)}
								disabled={deletingFile === doc.filename}
								class="shrink-0 ml-4 text-xs px-3 py-1.5 rounded-lg border transition-all
								       hover:bg-red-50 hover:border-red-200 hover:text-red-600
								       disabled:opacity-40 disabled:cursor-not-allowed"
								style="color: #9ca3af; border-color: #e5e7eb;">
								{deletingFile === doc.filename ? $t.deleting : $t.deleteBtn}
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
</div>
