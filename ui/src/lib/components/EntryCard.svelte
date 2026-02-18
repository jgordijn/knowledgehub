<script lang="ts">
	import type { RecordModel } from 'pocketbase';
	import pb from '$lib/pb';
	import StarRating from './StarRating.svelte';

	let {
		entry,
		onOpenChat,
		onUpdate
	}: {
		entry: RecordModel;
		onOpenChat: (entry: RecordModel) => void;
		onUpdate: (entry: RecordModel) => void;
	} = $props();

	let effectiveStars = $derived(entry.user_stars || entry.ai_stars || 0);
	let isFragment = $derived(!!entry.is_fragment);
	let isPending = $derived(entry.processing_status === 'pending' || (!entry.summary && !isFragment));
	let sourceName = $derived(entry.expand?.resource?.name ?? 'Unknown source');
	let displayTime = $derived(entry.published_at || entry.discovered_at);

	function relativeTime(dateStr: string): string {
		if (!dateStr) return '';
		const now = Date.now();
		const then = new Date(dateStr).getTime();
		const diffMs = now - then;
		const diffMin = Math.floor(diffMs / 60000);
		if (diffMin < 1) return 'just now';
		if (diffMin < 60) return `${diffMin}m ago`;
		const diffHrs = Math.floor(diffMin / 60);
		if (diffHrs < 24) return `${diffHrs}h ago`;
		const diffDays = Math.floor(diffHrs / 24);
		if (diffDays < 30) return `${diffDays}d ago`;
		return `${Math.floor(diffDays / 30)}mo ago`;
	}

	async function handleRate(stars: number) {
		try {
			const updated = await pb.collection('entries').update(entry.id, { user_stars: stars });
			onUpdate({ ...entry, ...updated });
		} catch {
			// Silently ignore
		}
	}

	async function toggleRead() {
		try {
			const updated = await pb.collection('entries').update(entry.id, {
				is_read: !entry.is_read
			});
			onUpdate({ ...entry, ...updated });
		} catch {
			// Silently ignore
		}
	}

	async function markReadAndOpen() {
		if (!entry.is_read) {
			try {
				const updated = await pb.collection('entries').update(entry.id, { is_read: true });
				onUpdate({ ...entry, ...updated });
			} catch {
				// Still open link
			}
		}
	}
</script>

<div
	class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md
		{entry.is_read ? 'opacity-70' : ''}"
>
	<!-- Top row: stars + source + time -->
	<div class="mb-2 flex items-center justify-between gap-2">
		<div class="flex items-center gap-3">
			{#if isPending}
				<div class="flex items-center gap-1 text-slate-400">
					<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
						<circle
							class="opacity-25"
							cx="12"
							cy="12"
							r="10"
							stroke="currentColor"
							stroke-width="4"
						></circle>
						<path
							class="opacity-75"
							fill="currentColor"
							d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
						></path>
					</svg>
				</div>
			{:else}
				<StarRating
					aiStars={entry.ai_stars}
					userStars={entry.user_stars}
					onRate={handleRate}
				/>
			{/if}
			<span class="text-xs text-slate-500">{sourceName}</span>
		</div>
		<span class="shrink-0 text-xs text-slate-400">{relativeTime(displayTime)}</span>
	</div>

	<!-- Title -->
	<h3 class="mb-1 text-sm font-semibold text-slate-900 leading-snug">
		<a
			href={entry.url}
			target="_blank"
			rel="noopener"
			onclick={markReadAndOpen}
			class="hover:text-blue-600"
		>
			{entry.title || 'Untitled'}
		</a>
	</h3>

	<!-- Summary, fragment content, or pending -->
	{#if isPending}
		<div class="flex items-center gap-2 py-2 text-sm text-slate-400">
			<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
				<circle
					class="opacity-25"
					cx="12"
					cy="12"
					r="10"
					stroke="currentColor"
					stroke-width="4"
				></circle>
				<path
					class="opacity-75"
					fill="currentColor"
					d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
				></path>
			</svg>
			Processing…
		</div>
	{:else if isFragment}
		<div class="fragment-content mb-3 text-sm text-slate-600 leading-relaxed">
			{@html entry.raw_content}
		</div>
	{:else}
		<p class="mb-3 text-sm text-slate-600 leading-relaxed">
			{entry.summary}
		</p>
	{/if}

	<!-- Actions -->
	<div class="flex items-center gap-1">
		<!-- Read/unread toggle -->
		<button
			onclick={toggleRead}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm
				{entry.is_read
				? 'text-green-600 hover:bg-green-50'
				: 'text-slate-400 hover:bg-slate-50'}"
			title={entry.is_read ? 'Mark as unread' : 'Mark as read'}
		>
			✓
		</button>

		<!-- Chat button -->
		<button
			onclick={() => onOpenChat(entry)}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm text-slate-400 hover:bg-slate-50 hover:text-slate-600"
			title="Chat about this article"
		>
			🤖
		</button>

		<!-- Open link -->
		<a
			href={entry.url}
			target="_blank"
			rel="noopener"
			onclick={markReadAndOpen}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm text-slate-400 hover:bg-slate-50 hover:text-slate-600"
			title="Open article"
		>
			↗
		</a>
	</div>
</div>


<style>
	/* Style rendered HTML from fragment feeds */
	.fragment-content :global(a) {
		color: var(--color-blue-600);
		text-decoration: underline;
	}
	.fragment-content :global(a:hover) {
		color: var(--color-blue-800);
	}
	.fragment-content :global(blockquote) {
		border-left: 3px solid var(--color-slate-300);
		padding-left: 0.75rem;
		margin: 0.5rem 0;
		color: var(--color-slate-500);
		font-style: italic;
	}
	.fragment-content :global(code) {
		background: var(--color-slate-100);
		padding: 0.1rem 0.3rem;
		border-radius: 0.25rem;
		font-size: 0.85em;
	}
	.fragment-content :global(pre) {
		background: var(--color-slate-100);
		padding: 0.5rem;
		border-radius: 0.375rem;
		overflow-x: auto;
	}
	.fragment-content :global(ul),
	.fragment-content :global(ol) {
		padding-left: 1.25rem;
		margin: 0.25rem 0;
	}
	.fragment-content :global(ul) {
		list-style-type: disc;
	}
	.fragment-content :global(ol) {
		list-style-type: decimal;
	}
	.fragment-content :global(p) {
		margin: 0.25rem 0;
	}
	.fragment-content :global(p:first-child) {
		margin-top: 0;
	}
	.fragment-content :global(p:last-child) {
		margin-bottom: 0;
	}
</style>