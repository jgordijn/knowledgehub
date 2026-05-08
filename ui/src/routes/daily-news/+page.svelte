<script lang="ts">
	import { onMount } from 'svelte';
	import DailyNewsDigest from '$lib/components/DailyNewsDigest.svelte';
	import {
		dailyNewsLoadingMessage,
		dailyNewsStateMessage,
		dailyNewsArchiveLabel,
		dailyNewsGenerateButtonLabel,
		dailyNewsRegenerateButtonLabel,
		dailyNewsCanRegenerate,
		type DailyNewsDigestDTO,
		type DailyNewsDigestListDTO
	} from '$lib/daily-news-ui';
	import pb from '$lib/pb';

	let latestDigest = $state<DailyNewsDigestDTO | null>(null);
	let selectedDigest = $state<DailyNewsDigestDTO | null>(null);
	let archive = $state<DailyNewsDigestDTO[]>([]);
	let hasMoreArchive = $state(false);
	let archiveOffset = 0;
	let archiveLoading = $state(false);
	let archiveError = $state('');
	let generateLoading = $state(false);
	let regenerateLoading = $state(false);
	let actionError = $state('');
	const archiveLimit = 10;
	let displayDigest = $derived(selectedDigest ?? latestDigest);
	let stateMessage = $derived(dailyNewsStateMessage(displayDigest));

	async function loadDigests(selected = '', offset = 0) {
		archiveLoading = true;
		archiveError = '';
		try {
			const params = new URLSearchParams({ limit: String(archiveLimit), offset: String(offset) });
			if (selected) params.set('selected', selected);
			const response = (await pb.send(`/api/daily-news/digests?${params}`, { method: 'GET' })) as DailyNewsDigestListDTO;
			latestDigest = response.latest ?? null;
			selectedDigest = response.selected ?? latestDigest;
			archive = offset === 0 ? (response.archive ?? []) : [...archive, ...(response.archive ?? [])];
			hasMoreArchive = Boolean(response.has_more);
			archiveOffset = offset + (response.archive?.length ?? 0);
		} catch {
			archiveError = 'Could not load Daily News editions.';
			if (offset === 0) {
				latestDigest = null;
				selectedDigest = null;
				archive = [];
			}
		} finally {
			archiveLoading = false;
		}
	}

	function applyReturnedDigest(digest: DailyNewsDigestDTO) {
		if (!latestDigest || latestDigest.id === digest.id) {
			latestDigest = digest;
		}
		selectedDigest = digest;
	}

	async function generateNow() {
		generateLoading = true;
		actionError = '';
		try {
			const digest = (await pb.send('/api/daily-news/generate', { method: 'POST' })) as DailyNewsDigestDTO;
			applyReturnedDigest(digest);
			void loadDigests(digest.id, 0);
		} catch {
			actionError = 'Could not queue Daily News generation.';
		} finally {
			generateLoading = false;
		}
	}

	async function regenerateDigest() {
		if (!displayDigest) return;
		regenerateLoading = true;
		actionError = '';
		try {
			const digest = (await pb.send(`/api/daily-news/digests/${displayDigest.id}/regenerate`, { method: 'POST' })) as DailyNewsDigestDTO;
			applyReturnedDigest(digest);
			void loadDigests(digest.id, 0);
		} catch {
			actionError = 'Could not queue Daily News regeneration.';
		} finally {
			regenerateLoading = false;
		}
	}

	onMount(() => {
		void loadDigests();
	});
</script>

<svelte:head>
	<title>Daily News · KnowledgeHub</title>
</svelte:head>

<section class="mx-auto max-w-4xl space-y-4">
	<div class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800">
		<p class="text-xs font-semibold uppercase tracking-[0.2em] text-amber-500 dark:text-amber-400">Daily News</p>
		<h1 class="mt-2 text-3xl font-bold text-slate-950 dark:text-slate-50">Daily News</h1>
		{#if latestDigest?.status === 'success'}
			<p class="mt-3 text-sm text-slate-500 dark:text-slate-400">Latest edition</p>
		{:else}
			<p class="mt-3 text-sm text-slate-500 dark:text-slate-400">{dailyNewsLoadingMessage()}</p>
		{/if}
		<div class="mt-5 flex flex-wrap gap-3">
			<button type="button" class="rounded-full bg-amber-500 px-4 py-2 text-sm font-semibold text-white hover:bg-amber-600 disabled:cursor-not-allowed disabled:opacity-60" disabled={generateLoading || regenerateLoading} onclick={generateNow}>
				{dailyNewsGenerateButtonLabel(generateLoading)}
			</button>
			<button type="button" class="rounded-full border border-slate-300 px-4 py-2 text-sm font-semibold text-slate-700 hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700" disabled={!dailyNewsCanRegenerate(displayDigest) || generateLoading || regenerateLoading} onclick={regenerateDigest}>
				{dailyNewsRegenerateButtonLabel(regenerateLoading)}
			</button>
		</div>
		{#if actionError}
			<p class="mt-3 text-sm text-red-600 dark:text-red-300">{actionError}</p>
		{/if}
	</div>

	{#if stateMessage}
		<div class="rounded-2xl border p-5 shadow-sm {stateMessage.tone === 'error' ? 'border-red-200 bg-red-50 text-red-900 dark:border-red-900 dark:bg-red-950/40 dark:text-red-100' : stateMessage.tone === 'empty' ? 'border-slate-200 bg-white text-slate-700 dark:border-slate-700 dark:bg-slate-800 dark:text-slate-200' : 'border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-900 dark:bg-blue-950/40 dark:text-blue-100'}">
			<h2 class="text-lg font-semibold">{stateMessage.title}</h2>
			<p class="mt-1 text-sm opacity-80">{stateMessage.message}</p>
		</div>
	{:else if displayDigest?.status === 'success'}
		<DailyNewsDigest digest={displayDigest} />
	{/if}

	<div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800">
		<h2 class="text-lg font-semibold text-slate-950 dark:text-slate-50">Previous editions</h2>
		{#if archiveError}
			<p class="mt-2 text-sm text-red-600 dark:text-red-300">{archiveError}</p>
		{:else if archive.length === 0 && !archiveLoading}
			<p class="mt-2 text-sm text-slate-500 dark:text-slate-400">No previous Daily News editions yet.</p>
		{/if}
		{#if archive.length > 0}
			<ul class="mt-3 divide-y divide-slate-100 dark:divide-slate-700">
				{#each archive as digest (digest.id)}
					<li>
						<button type="button" class="w-full py-3 text-left text-sm text-slate-700 hover:text-amber-600 dark:text-slate-200 dark:hover:text-amber-300" onclick={() => loadDigests(digest.id, 0)}>
							{dailyNewsArchiveLabel(digest)}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
		{#if hasMoreArchive}
			<button type="button" class="mt-4 rounded-full border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-60 dark:border-slate-600 dark:text-slate-200 dark:hover:bg-slate-700" disabled={archiveLoading} onclick={() => loadDigests(selectedDigest?.id ?? '', archiveOffset)}>
				{archiveLoading ? 'Loading…' : 'Load more editions'}
			</button>
		{/if}
	</div>
</section>
