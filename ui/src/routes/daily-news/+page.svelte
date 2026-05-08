<script lang="ts">
	import { onMount } from 'svelte';
	import DailyNewsDigest from '$lib/components/DailyNewsDigest.svelte';
	import { dailyNewsLoadingMessage, type DailyNewsDigestDTO } from '$lib/daily-news-ui';
	import pb from '$lib/pb';

	let latestDigest: DailyNewsDigestDTO | null = null;

	onMount(async () => {
		try {
			const response = await pb.send('/api/daily-news/digests', { method: 'GET' });
			latestDigest = response.latest ?? response.digest ?? response;
		} catch {
			latestDigest = null;
		}
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
	</div>

	{#if latestDigest?.status === 'success'}
		<DailyNewsDigest digest={latestDigest} />
	{/if}
</section>
