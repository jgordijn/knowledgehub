<script lang="ts">
	import {
		dailyNewsSubsetMessage,
		renderDailyNewsMarkdown,
		type DailyNewsDigestDTO
	} from '$lib/daily-news-ui';

	let { digest }: { digest: DailyNewsDigestDTO } = $props();

	let renderedBody = $derived(renderDailyNewsMarkdown(digest.body_markdown));
	let subsetMessage = $derived(dailyNewsSubsetMessage(digest));
</script>

<article class="overflow-hidden rounded-3xl border border-slate-200 bg-stone-50 shadow-sm dark:border-slate-700 dark:bg-slate-900">
	<header class="border-b border-slate-200 bg-white px-6 py-5 dark:border-slate-700 dark:bg-slate-800">
		<p class="font-serif text-xs font-semibold uppercase tracking-[0.28em] text-amber-600 dark:text-amber-400">Daily News</p>
		<h2 class="mt-2 font-serif text-3xl font-bold text-slate-950 dark:text-slate-50">
			{digest.title || 'Daily News'}
		</h2>
		{#if subsetMessage}
			<p class="mt-3 rounded-full bg-amber-50 px-3 py-1 text-sm text-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
				{subsetMessage}
			</p>
		{/if}
	</header>

	<div class="daily-news-body px-6 py-6 font-serif text-slate-900 dark:text-slate-100">
		{@html renderedBody}
	</div>
</article>

<style>
	.daily-news-body :global(h1),
	.daily-news-body :global(h2),
	.daily-news-body :global(h3) {
		margin-top: 1.4rem;
		margin-bottom: 0.65rem;
		font-weight: 800;
		line-height: 1.1;
	}

	.daily-news-body :global(h1) { font-size: 2rem; }
	.daily-news-body :global(h2) { font-size: 1.5rem; }
	.daily-news-body :global(h3) { font-size: 1.2rem; }
	.daily-news-body :global(p),
	.daily-news-body :global(ul),
	.daily-news-body :global(ol),
	.daily-news-body :global(blockquote),
	.daily-news-body :global(table),
	.daily-news-body :global(pre) { margin: 0.8rem 0; }
	.daily-news-body :global(ul) { list-style: disc; padding-left: 1.4rem; }
	.daily-news-body :global(ol) { list-style: decimal; padding-left: 1.4rem; }
	.daily-news-body :global(a) { color: #b45309; text-decoration: underline; }
	.daily-news-body :global(blockquote) { border-left: 4px solid #f59e0b; padding-left: 1rem; font-style: italic; }
	.daily-news-body :global(table) { width: 100%; border-collapse: collapse; }
	.daily-news-body :global(th),
	.daily-news-body :global(td) { border: 1px solid currentColor; padding: 0.35rem 0.5rem; }
	.daily-news-body :global(code) { border-radius: 0.25rem; background: rgb(15 23 42 / 0.08); padding: 0.1rem 0.25rem; }
	.daily-news-body :global(pre code) { display: block; overflow-x: auto; padding: 0.75rem; }
</style>
