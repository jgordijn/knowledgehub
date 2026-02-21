<script lang="ts">
	import type { RecordModel } from 'pocketbase';
	import pb from '$lib/pb';
	import { sanitizeHTML } from '$lib/markdown';
	import StarRating from './StarRating.svelte';

	let {
		entry,
		onOpenChat,
		onOpenLink,
		onUpdate
	}: {
		entry: RecordModel;
		onOpenChat: (entry: RecordModel) => void;
		onOpenLink: (entry: RecordModel, url: string) => void;
		onUpdate: (entry: RecordModel) => void;
	} = $props();

	let effectiveStars = $derived(entry.user_stars || entry.ai_stars || 0);
	let isFragment = $derived(!!entry.is_fragment);
	let isPending = $derived(entry.processing_status === 'pending' || (!entry.summary && !isFragment));
	let sourceName = $derived(entry.expand?.resource?.name ?? 'Unknown source');
	let displayTime = $derived(entry.published_at || entry.discovered_at);

	const mediaExtensions = new Set([
		'.png', '.jpg', '.jpeg', '.gif', '.webp', '.svg', '.ico', '.bmp', '.avif',
		'.mp4', '.webm', '.mov', '.avi', '.mkv', '.ogv',
		'.mp3', '.wav', '.ogg', '.flac', '.aac',
		'.pdf', '.zip', '.tar', '.gz', '.rar',
	]);
	const mediaHosts = new Set([
		'i.imgur.com', 'imgur.com', 'pbs.twimg.com',
		'media.giphy.com', 'giphy.com',
		'youtube.com', 'www.youtube.com', 'youtu.be',
		'vimeo.com', 'www.vimeo.com',
		'flickr.com', 'www.flickr.com',
	]);

	function isMediaURL(url: URL): boolean {
		const path = url.pathname.toLowerCase();
		for (const ext of mediaExtensions) {
			if (path.endsWith(ext)) return true;
		}
		if (mediaHosts.has(url.hostname)) return true;
		// Common image/video CDN path patterns
		if (/\/(images?|img|media|assets|static|uploads|photos?|thumbnails?)\//i.test(path)) {
			const lastSegment = path.split('/').pop() || '';
			if (/\.\w{2,4}$/.test(lastSegment) && !lastSegment.endsWith('.html') && !lastSegment.endsWith('.htm')) return true;
		}
		return false;
	}

	let referencedLinks = $derived(() => {
		const content = entry.raw_content || '';
		if (!content.includes('<a ')) return [];
		const parser = new DOMParser();
		const doc = parser.parseFromString(content, 'text/html');
		const anchors = doc.querySelectorAll('a[href]');
		const seen = new Set<string>();
		const links: { href: string; text: string }[] = [];
		for (const a of anchors) {
			const href = a.getAttribute('href') || '';
			if (!href || href.startsWith('#') || href.startsWith('mailto:')) continue;
			try {
				const url = new URL(href, entry.url);
				if (url.protocol !== 'http:' && url.protocol !== 'https:') continue;
				const full = url.href;
				if (seen.has(full)) continue;
				if (full === entry.url) continue;
				if (isMediaURL(url)) continue;
				// Skip if the anchor only wraps an image (no meaningful text)
				if (a.querySelector('img') && !(a.textContent || '').trim()) continue;
				seen.add(full);
				const text = (a.textContent || '').trim() || url.hostname + url.pathname;
				links.push({ href: full, text: text.length > 80 ? text.slice(0, 77) + '…' : text });
			} catch {
				continue;
			}
		}
		return links;
	});
	let showLinks = $state(false);

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

	async function toggleBookmark() {
		try {
			const updated = await pb.collection('entries').update(entry.id, {
				bookmarked: !entry.bookmarked
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
	class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm transition-shadow hover:shadow-md dark:border-slate-700 dark:bg-slate-800
		{entry.is_read ? 'opacity-70' : ''}"
>
	<!-- Top row: stars + source + time -->
	<div class="mb-2 flex items-center justify-between gap-2">
		<div class="flex items-center gap-3">
			{#if isPending}
				<div class="flex items-center gap-1 text-slate-400 dark:text-slate-500">
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
			<span class="text-xs text-slate-500 dark:text-slate-400">{sourceName}</span>
		</div>
		<span class="shrink-0 text-xs text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
	</div>

	<!-- Title -->
	<h3 class="mb-1 text-sm font-semibold text-slate-900 leading-snug dark:text-slate-100">
		<a
			href={entry.url}
			target="_blank"
			rel="noopener"
			onclick={markReadAndOpen}
			class="hover:text-blue-600 dark:hover:text-blue-400"
		>
			{entry.title || 'Untitled'}
		</a>
	</h3>

	<!-- Summary, fragment content, or pending -->
	{#if isPending}
		<div class="flex items-center gap-2 py-2 text-sm text-slate-400 dark:text-slate-500">
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
		<div class="fragment-content mb-3 text-sm text-slate-600 leading-relaxed dark:text-slate-400">
			{@html sanitizeHTML(entry.raw_content)}
		</div>
	{:else}
		<p class="mb-3 text-sm text-slate-600 leading-relaxed dark:text-slate-400">
			{entry.summary}
		</p>
	{/if}

	<!-- Referenced links -->
	{#if referencedLinks().length > 0}
		<div class="mb-2">
			<button
				onclick={() => (showLinks = !showLinks)}
				class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors dark:text-slate-400 dark:hover:text-slate-300"
			>
				<svg
					class="h-3 w-3 transition-transform {showLinks ? 'rotate-90' : ''}"
					fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
				🔗 {referencedLinks().length} referenced {referencedLinks().length === 1 ? 'link' : 'links'}
			</button>
			{#if showLinks}
				<div class="mt-1.5 space-y-1 pl-4">
					{#each referencedLinks() as link}
						<div class="flex items-center gap-1.5">
							<a
								href={link.href}
								target="_blank"
								rel="noopener"
								class="text-xs text-blue-600 hover:text-blue-800 hover:underline text-left truncate max-w-[280px] dark:text-blue-400 dark:hover:text-blue-300"
								title={link.href}
							>
								{link.text}
							</a>
							<button
								onclick={() => onOpenLink(entry, link.href)}
								class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 transition-colors dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50"
								title="Get AI summary"
							>
								Summarize
							</button>
						</div>
					{/each}
				</div>
			{/if}
		</div>
	{/if}


	<!-- Actions -->
	<div class="flex items-center gap-1">
		<!-- Read/unread toggle -->
		<button
			onclick={toggleRead}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm
				{entry.is_read
				? 'text-green-600 hover:bg-green-50 dark:text-green-400 dark:hover:bg-green-900/30'
				: 'text-slate-400 hover:bg-slate-50 dark:text-slate-500 dark:hover:bg-slate-700'}"
			title={entry.is_read ? 'Mark as unread' : 'Mark as read'}
		>
			✓
		</button>

		<!-- Bookmark toggle -->
		<button
			onclick={toggleBookmark}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm
				{entry.bookmarked
				? 'text-amber-500 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/30'
				: 'text-slate-400 hover:bg-slate-50 dark:text-slate-500 dark:hover:bg-slate-700'}"
			title={entry.bookmarked ? 'Remove from Read Later' : 'Read Later'}
		>
			<svg class="h-4 w-4" viewBox="0 0 24 24" fill={entry.bookmarked ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
				<path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
			</svg>
		</button>

		<!-- Chat button -->
		<button
			onclick={() => onOpenChat(entry)}
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm text-slate-400 hover:bg-slate-50 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-300"
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
			class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm text-slate-400 hover:bg-slate-50 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-300"
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

	/* Dark mode overrides for fragment content */
	:global(.dark) .fragment-content :global(a) {
		color: var(--color-blue-400);
	}
	:global(.dark) .fragment-content :global(a:hover) {
		color: var(--color-blue-300);
	}
	:global(.dark) .fragment-content :global(blockquote) {
		border-left-color: var(--color-slate-600);
		color: var(--color-slate-400);
	}
	:global(.dark) .fragment-content :global(code) {
		background: var(--color-slate-700);
	}
	:global(.dark) .fragment-content :global(pre) {
		background: var(--color-slate-700);
	}
</style>
