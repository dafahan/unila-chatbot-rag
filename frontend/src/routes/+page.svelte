<script lang="ts">
	import { t, lang, toggleLang } from '$lib/i18n';
</script>

<svelte:head>
	<title>UNILA AI — Asisten Akademik Virtual</title>
</svelte:head>

<div class="min-h-screen flex flex-col" style="background-color: #f8f7f4;">

	<!-- Nav -->
	<nav class="px-8 py-5 flex items-center justify-between border-b border-gray-200 bg-white/80 backdrop-blur-sm sticky top-0 z-10">
		<div class="flex items-center gap-3">
			<div class="w-8 h-8 rounded-lg flex items-center justify-center" style="background-color: #1a3557;">
				<span class="text-white text-xs font-bold">U</span>
			</div>
			<span class="font-semibold text-sm tracking-wide" style="color: #1a3557;">UNILA AI</span>
		</div>
		<div class="flex items-center gap-6">
			<button
				onclick={toggleLang}
				class="text-xs px-2.5 py-1 rounded-md border font-mono transition-colors hover:bg-gray-50"
				style="color: #6b7280; border-color: #e5e7eb;">
				{$lang === 'en' ? 'EN' : 'ID'}
			</button>
			<a href="/chat" class="text-sm text-gray-500 hover:text-gray-800 transition-colors">{$t.startChat}</a>
			<a href="/admin" class="text-sm px-4 py-2 rounded-lg border transition-colors hover:bg-gray-50"
				style="color: #1a3557; border-color: #1a3557;">
				{$t.admin}
			</a>
		</div>
	</nav>

	<!-- Hero -->
	<main class="flex-1 flex flex-col items-center justify-center px-6 py-24 text-center">
		<div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-medium mb-8 border"
			style="background-color: #f0f4f9; border-color: #c9d9ec; color: #1a3557;">
			{$t.tagline}
		</div>

		<h1 class="text-5xl font-light leading-tight mb-5 max-w-2xl"
			style="font-family: 'Lora', serif; color: #1a3557;">
			{$t.heroTitle}<br />
			<span class="italic font-normal" style="color: #c9a84c;">{$t.heroAccent}</span>
		</h1>

		<p class="text-base leading-relaxed max-w-md mb-10" style="color: #6b7280;">
			{$t.heroDesc}
		</p>

		<div class="flex items-center gap-3">
			<a href="/chat"
				class="px-6 py-3 rounded-xl text-sm font-medium text-white transition-all hover:opacity-90 active:scale-95"
				style="background-color: #1a3557;">
				{$t.btnAsk}
			</a>
			<a href="/admin"
				class="px-6 py-3 rounded-xl text-sm font-medium transition-all hover:bg-gray-100 border border-gray-200"
				style="color: #1a3557;">
				{$t.btnUpload}
			</a>
		</div>
	</main>

	<!-- Features -->
	<section class="px-6 pb-24 grid grid-cols-1 sm:grid-cols-3 gap-6 max-w-4xl mx-auto w-full">
		{#each [
			{ icon: '📄', title: $t.feat1Title, desc: $t.feat1Desc },
			{ icon: '🔍', title: $t.feat2Title, desc: $t.feat2Desc },
			{ icon: '⚡', title: $t.feat3Title, desc: $t.feat3Desc },
		] as f}
			<div class="bg-white rounded-2xl px-6 py-7 border border-gray-100 shadow-sm">
				<div class="text-2xl mb-3">{f.icon}</div>
				<h3 class="text-sm font-semibold mb-2" style="color: #1a3557;">{f.title}</h3>
				<p class="text-sm leading-relaxed" style="color: #6b7280;">{f.desc}</p>
			</div>
		{/each}
	</section>

	<!-- Footer -->
	<footer class="border-t border-gray-200 py-6 text-center text-xs" style="color: #9ca3af;">
		&copy; {new Date().getFullYear()} {$t.footer}
	</footer>
</div>
