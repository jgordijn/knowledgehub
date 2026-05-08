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
		type DailyNewsDigestListDTO,
		validateDailyNewsSettings,
		type DailyNewsEntryReferenceDTO,
		type DailyNewsSettingsDTO
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
	let referenceModal = $state<DailyNewsEntryReferenceDTO | null>(null);
	let referenceLoading = $state(false);
	let referenceError = $state('');
	let settings = $state<DailyNewsSettingsDTO>({ enabled: true, generation_time: '08:00', timezone: 'Europe/Amsterdam', extra_instructions: '' });
	let settingsError = $state('');
	let settingsSaved = $state('');
	let settingsLoading = $state(false);
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

	async function openEntryReference(entryID: string) {
		if (!displayDigest) return;
		referenceLoading = true;
		referenceError = '';
		referenceModal = null;
		try {
			referenceModal = (await pb.send(`/api/daily-news/digests/${displayDigest.id}/entries/${entryID}`, { method: 'GET' })) as DailyNewsEntryReferenceDTO;
		} catch {
			referenceError = 'Could not open the referenced entry.';
		} finally {
			referenceLoading = false;
		}
	}

	async function loadSettings() {
		try {
			settings = (await pb.send('/api/daily-news/settings', { method: 'GET' })) as DailyNewsSettingsDTO;
		} catch {
			settingsError = 'Could not load Daily News settings.';
		}
	}

	async function saveSettings() {
		settingsError = '';
		settingsSaved = '';
		const errors = validateDailyNewsSettings(settings);
		if (errors.length > 0) {
			settingsError = errors[0];
			return;
		}
		settingsLoading = true;
		try {
			settings = (await pb.send('/api/daily-news/settings', { method: 'PUT', body: settings })) as DailyNewsSettingsDTO;
			settingsSaved = 'Daily News settings saved.';
		} catch {
			settingsError = 'Could not save Daily News settings.';
		} finally {
			settingsLoading = false;
		}
	}

	onMount(() => {
		void loadDigests();
		void loadSettings();
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
	{/if}
	{#if displayDigest?.body_markdown}
		<DailyNewsDigest digest={displayDigest} onOpenEntry={openEntryReference} />
	{/if}

	{#if referenceLoading || referenceError || referenceModal}
		<div class="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/50 p-4">
			<div class="w-full max-w-xl rounded-2xl bg-white p-6 shadow-xl dark:bg-slate-800">
				<div class="flex items-start justify-between gap-4">
					<h2 class="text-xl font-semibold text-slate-950 dark:text-slate-50">Referenced entry</h2>
					<button type="button" class="text-sm text-slate-500 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white" onclick={() => { referenceModal = null; referenceError = ''; }}>Close</button>
				</div>
				{#if referenceLoading}
					<p class="mt-4 text-sm text-slate-500 dark:text-slate-300">Loading referenced entry…</p>
				{:else if referenceError}
					<p class="mt-4 text-sm text-red-600 dark:text-red-300">{referenceError}</p>
				{:else if referenceModal?.available && referenceModal.entry}
					<article class="mt-4 space-y-3 text-slate-700 dark:text-slate-200">
						<p class="text-xs font-semibold uppercase tracking-wide text-amber-600 dark:text-amber-300">{referenceModal.entry.source_name || 'KnowledgeHub entry'} · {referenceModal.entry.effective_stars ?? 0}★</p>
						<h3 class="text-2xl font-bold text-slate-950 dark:text-slate-50">{referenceModal.entry.title}</h3>
						{#if referenceModal.entry.summary}<p>{referenceModal.entry.summary}</p>{/if}
						{#if referenceModal.entry.takeaways?.length}
							<ul class="list-disc pl-5 text-sm">
								{#each referenceModal.entry.takeaways as takeaway}<li>{takeaway}</li>{/each}
							</ul>
						{/if}
						<a class="text-sm font-semibold text-amber-600 underline dark:text-amber-300" href={referenceModal.entry.url} target="_blank" rel="noopener noreferrer">Open original article</a>
					</article>
				{:else}
					<p class="mt-4 text-sm text-slate-600 dark:text-slate-300">{referenceModal?.message || 'Referenced entry is no longer available.'}</p>
				{/if}
			</div>
		</div>
	{/if}

	<div class="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-800">
		<h2 class="text-lg font-semibold text-slate-950 dark:text-slate-50">Daily News settings</h2>
		<div class="mt-4 grid gap-4 sm:grid-cols-2">
			<label class="flex items-center gap-2 text-sm text-slate-700 dark:text-slate-200"><input type="checkbox" bind:checked={settings.enabled} /> Enabled</label>
			<label class="text-sm text-slate-700 dark:text-slate-200">Generation time<input class="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 dark:border-slate-600 dark:bg-slate-900" bind:value={settings.generation_time} placeholder="08:00" /></label>
			<label class="text-sm text-slate-700 dark:text-slate-200">Timezone<input class="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 dark:border-slate-600 dark:bg-slate-900" bind:value={settings.timezone} placeholder="Europe/Amsterdam" /></label>
			<label class="sm:col-span-2 text-sm text-slate-700 dark:text-slate-200">Extra digest instructions<textarea class="mt-1 min-h-28 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 dark:border-slate-600 dark:bg-slate-900" bind:value={settings.extra_instructions} maxlength="2000"></textarea></label>
		</div>
		<button type="button" class="mt-4 rounded-full bg-slate-900 px-4 py-2 text-sm font-semibold text-white disabled:opacity-60 dark:bg-slate-100 dark:text-slate-900" disabled={settingsLoading} onclick={saveSettings}>{settingsLoading ? 'Saving…' : 'Save settings'}</button>
		{#if settingsError}<p class="mt-2 text-sm text-red-600 dark:text-red-300">{settingsError}</p>{/if}
		{#if settingsSaved}<p class="mt-2 text-sm text-green-700 dark:text-green-300">{settingsSaved}</p>{/if}
	</div>

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
