<script lang="ts">
	import type { RecordModel } from 'pocketbase';
	import pb from '$lib/pb';
	import { sanitizeHTML } from '$lib/markdown';
	import StarRating from './StarRating.svelte';

	let {
		entry,
		expanded = true,
		onToggle,
		onOpenChat,
		onOpenLink,
		onUpdate
	}: {
		entry: RecordModel;
		expanded?: boolean;
		onToggle?: () => void;
		onOpenChat: (entry: RecordModel) => void;
		onOpenLink: (entry: RecordModel, url: string) => void;
		onUpdate: (entry: RecordModel) => void;
	} = $props();

	let effectiveStars = $derived(entry.user_stars || entry.ai_stars || 0);
	// Tier mapping (task 3.1)
	let tier = $derived<'featured' | 'hp' | 'wal' | 'lp'>(
		effectiveStars >= 5 ? 'featured' :
		effectiveStars === 4 ? 'hp' :
		effectiveStars === 3 ? 'wal' : 'lp'
	);
	let isFragment = $derived(!!entry.is_fragment);
	let isPending = $derived(entry.processing_status === 'pending' || (!entry.summary && !isFragment));
	let sourceName = $derived(entry.expand?.resource?.name ?? 'Unknown source');
	let displayTime = $derived(entry.published_at || entry.discovered_at);

	const avatarColors = [
		'#f97316', '#8b5cf6', '#06b6d4', '#f472b6', '#34d399',
		'#a78bfa', '#fb923c', '#38bdf8', '#f87171', '#4ade80'
	];

	function getSourceColor(): string {
		const name = sourceName;
		let hash = 0;
		for (let i = 0; i < name.length; i++) {
			hash = ((hash << 5) - hash) + name.charCodeAt(i);
			hash |= 0;
		}
		return avatarColors[Math.abs(hash) % avatarColors.length];
	}

	function getSourceInitials(): string {
		const words = sourceName.trim().split(/\s+/);
		if (words.length >= 2) {
			return (words[0][0] + words[1][0]).toUpperCase();
		}
		return sourceName.slice(0, 2).toUpperCase();
	}

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

	// Card-level click handler (task 3.7) — preserved from existing code
	function handleCardClick(event: MouseEvent) {
		let target = event.target as HTMLElement | null;
		while (target && target !== event.currentTarget) {
			const tag = target.tagName?.toLowerCase();
			if (tag === 'a' || tag === 'button' || tag === 'input' || tag === 'select' || tag === 'textarea') {
				return;
			}
			if (target.getAttribute?.('role') === 'button') {
				return;
			}
			target = target.parentElement;
		}
		if (entry.url) {
			markReadAndOpen();
			window.open(entry.url, '_blank', 'noopener');
		}
	}

	function starsDisplay(count: number): string {
		return '★'.repeat(count);
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->

{#if tier === 'featured'}
	<!-- ═══ FEATURED (5★) ═══ -->
	<div
		onclick={handleCardClick}
		class="cursor-pointer rounded-xl border border-slate-200 border-l-4 border-l-amber-500 bg-white p-[18px_20px] shadow-sm transition-shadow hover:shadow-md dark:border-slate-700 dark:border-l-amber-400 dark:bg-slate-800
			{entry.is_read ? 'opacity-70' : ''}"
	>
		<div class="mb-1.5 flex items-start justify-between">
			<span class="text-[11px] font-semibold uppercase tracking-wider text-amber-500 dark:text-amber-400">★★★★★ Featured</span>
			<button
				onclick={(e) => { e.stopPropagation(); onToggle?.(); }}
				class="flex-shrink-0 rounded border border-slate-200 px-1.5 py-px text-[10px] text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-600 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-300"
				title={expanded ? 'Collapse details' : 'Expand details'}
				aria-label={expanded ? 'Collapse' : 'Expand'}
			>{expanded ? '▾' : '▸'}</button>
		</div>

		<h3 class="mb-1.5 text-[19px] font-bold leading-snug text-slate-900 dark:text-slate-50">
			<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline">{entry.title || 'Untitled'}</a>
		</h3>

		{#if expanded}
			<div class="card-detail">
				{#if isPending}
					<div class="flex items-center gap-2 py-2 text-sm text-slate-400 dark:text-slate-500">
						<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
						Processing…
					</div>
				{:else if isFragment}
					<div class="fragment-content mb-3 text-[13px] leading-relaxed text-slate-500 dark:text-slate-400">
						{@html sanitizeHTML(entry.raw_content)}
					</div>
				{:else}
					<p class="mb-2 text-[13px] leading-relaxed text-slate-500 dark:text-slate-400">{entry.summary}</p>
					{#if entry.takeaways?.length}
						<ul class="mb-3">
							{#each entry.takeaways as takeaway}
								<li class="relative py-0.5 pl-3.5 text-[12px] text-slate-600 dark:text-slate-300">
									<span class="absolute left-0 text-amber-500">•</span>{takeaway}
								</li>
							{/each}
						</ul>
					{/if}
				{/if}

				<!-- Meta: source + time -->
				<div class="mb-3 flex flex-wrap items-center gap-2.5">
					<span class="stars text-[13px] tracking-wider text-amber-500 dark:text-amber-400">{starsDisplay(effectiveStars)}</span>
					<span class="inline-flex items-center gap-1.5 text-[11px] text-slate-400 dark:text-slate-500">
						<span class="inline-flex h-[18px] w-[18px] items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
						{sourceName}
					</span>
					<span class="ml-auto text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
				</div>

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
										<a href={link.href} target="_blank" rel="noopener" class="text-xs text-blue-600 hover:text-blue-800 hover:underline text-left truncate max-w-[280px] dark:text-blue-400 dark:hover:text-blue-300" title={link.href}>{link.text}</a>
										<button onclick={() => onOpenLink(entry, link.href)} class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 transition-colors dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50" title="Get AI summary">Summarize</button>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex flex-wrap gap-[7px]">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen}
						class="inline-flex items-center gap-1 rounded-md bg-blue-500 px-3 py-1.5 text-[12px] font-semibold text-white transition-colors hover:bg-blue-600">
						Read Now
					</a>
					<button onclick={toggleBookmark}
						class="inline-flex items-center gap-1 rounded-md border px-3 py-1.5 text-[12px] font-semibold transition-colors
							{entry.bookmarked
							? 'border-amber-500 text-amber-500 dark:border-amber-400 dark:text-amber-400'
							: 'border-amber-500/50 text-amber-500/70 dark:border-amber-400/50 dark:text-amber-400/70'}">
						{entry.bookmarked ? '📌 Saved' : 'Save'}
					</button>
					<button onclick={() => onOpenChat(entry)}
						class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-3 py-1.5 text-[12px] font-semibold text-slate-500 transition-colors hover:text-slate-900 dark:border-slate-600 dark:text-slate-400 dark:hover:text-slate-100">
						🤖 Chat
					</button>
					<button onclick={toggleRead}
						class="inline-flex items-center gap-1 rounded-md border border-slate-200 px-3 py-1.5 text-[12px] text-slate-400 transition-colors hover:text-slate-600 dark:border-slate-700 dark:text-slate-500 dark:hover:text-slate-300"
						title={entry.is_read ? 'Mark as unread' : 'Mark as read'}>
						{entry.is_read ? '✓ Read' : 'Mark read'}
					</button>
				</div>
			</div>
		{/if}
	</div>

{:else if tier === 'hp'}
	<!-- ═══ HIGH PRIORITY (4★) ═══ -->
	<div
		onclick={handleCardClick}
		class="cursor-pointer rounded-lg border border-slate-200 bg-white p-[12px_16px] shadow-sm transition-shadow hover:shadow-md dark:border-slate-700 dark:bg-slate-800
			{entry.is_read ? 'opacity-70' : ''}"
	>
		<div class="flex gap-3">
			<div class="min-w-0 flex-1">
				<!-- Title -->
				<h3 class="mb-1 text-[14px] font-semibold leading-snug text-slate-900 dark:text-slate-50">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline">{entry.title || 'Untitled'}</a>
				</h3>

				{#if expanded}
					<div class="card-detail">
						{#if isPending}
							<div class="flex items-center gap-2 py-2 text-sm text-slate-400 dark:text-slate-500">
								<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
									<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
									<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
								</svg>
								Processing…
							</div>
						{:else if isFragment}
							<div class="fragment-content mb-2 text-[12px] leading-relaxed text-slate-500 line-clamp-2 dark:text-slate-400">
								{@html sanitizeHTML(entry.raw_content)}
							</div>
						{:else}
							<p class="mb-1.5 text-[12px] leading-relaxed text-slate-500 line-clamp-2 dark:text-slate-400">{entry.summary}</p>
							{#if entry.takeaways?.length}
								<ul class="mb-2">
									{#each entry.takeaways as takeaway}
										<li class="relative py-0.5 pl-3.5 text-[12px] text-slate-600 dark:text-slate-300">
											<span class="absolute left-0 text-amber-500">•</span>{takeaway}
										</li>
									{/each}
								</ul>
							{/if}
						{/if}

						<!-- Meta -->
						<div class="flex flex-wrap items-center gap-2">
							{#if !isPending}
								<StarRating aiStars={entry.ai_stars} userStars={entry.user_stars} onRate={handleRate} />
							{/if}
							<span class="inline-flex items-center gap-1.5 text-[11px] text-slate-400 dark:text-slate-500">
								<span class="inline-flex h-[18px] w-[18px] items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
								{sourceName}
							</span>
							<span class="ml-auto text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
						</div>

						<!-- Referenced links -->
						{#if referencedLinks().length > 0}
							<div class="mt-2">
								<button
									onclick={() => (showLinks = !showLinks)}
									class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors dark:text-slate-400 dark:hover:text-slate-300"
								>
									<svg class="h-3 w-3 transition-transform {showLinks ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
										<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
									</svg>
									🔗 {referencedLinks().length} referenced {referencedLinks().length === 1 ? 'link' : 'links'}
								</button>
								{#if showLinks}
									<div class="mt-1.5 space-y-1 pl-4">
										{#each referencedLinks() as link}
											<div class="flex items-center gap-1.5">
												<a href={link.href} target="_blank" rel="noopener" class="text-xs text-blue-600 hover:text-blue-800 hover:underline text-left truncate max-w-[280px] dark:text-blue-400 dark:hover:text-blue-300" title={link.href}>{link.text}</a>
												<button onclick={() => onOpenLink(entry, link.href)} class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 transition-colors dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50" title="Get AI summary">Summarize</button>
											</div>
										{/each}
									</div>
								{/if}
							</div>
						{/if}
					</div>
				{/if}
			</div>

			<!-- Side buttons -->
			{#if expanded}
				<div class="flex flex-shrink-0 flex-col items-end gap-1.5">
					<button
						onclick={(e) => { e.stopPropagation(); onToggle?.(); }}
						class="rounded border border-slate-200 px-1.5 py-px text-[10px] text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-600 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-300"
						title={expanded ? 'Collapse details' : 'Expand details'}
					>{expanded ? '▾' : '▸'}</button>
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen}
						class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm text-slate-400 hover:bg-slate-50 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-300"
						title="Open article">↗</a>
					<button onclick={toggleBookmark}
						class="flex h-8 min-w-[44px] items-center justify-center rounded-md text-sm
							{entry.bookmarked
							? 'text-amber-500 hover:bg-amber-50 dark:text-amber-400 dark:hover:bg-amber-900/30'
							: 'text-slate-400 hover:bg-slate-50 dark:text-slate-500 dark:hover:bg-slate-700'}"
						title={entry.bookmarked ? 'Remove from Saved' : 'Save'}>
						<svg class="h-4 w-4" viewBox="0 0 24 24" fill={entry.bookmarked ? 'currentColor' : 'none'} stroke="currentColor" stroke-width="2">
							<path d="M5 5a2 2 0 012-2h10a2 2 0 012 2v16l-7-3.5L5 21V5z" />
						</svg>
					</button>
				</div>
			{:else}
				<div class="flex flex-shrink-0 items-start">
					<button
						onclick={(e) => { e.stopPropagation(); onToggle?.(); }}
						class="rounded border border-slate-200 px-1.5 py-px text-[10px] text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-600 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-300"
						title="Expand details"
					>▸</button>
				</div>
			{/if}
		</div>
	</div>

{:else if tier === 'wal'}
	<!-- ═══ WORTH A LOOK (3★) ═══ -->
	<div>
		<!-- Compact row -->
		<div
			onclick={handleCardClick}
			class="flex cursor-pointer items-center gap-2.5 rounded px-1 py-2 transition-colors
				{expanded ? 'rounded-b-none bg-slate-50 dark:bg-slate-800' : 'border-b border-slate-100 hover:bg-slate-50 dark:border-slate-800 dark:hover:bg-slate-800'}"
		>
			<span class="text-[13px] tracking-wider text-amber-500 dark:text-amber-400">{starsDisplay(effectiveStars)}</span>
			<span class="flex-1 truncate text-[13px] text-slate-600 dark:text-slate-300">
				<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline">{entry.title || 'Untitled'}</a>
			</span>
			<span class="inline-flex h-[18px] w-[18px] flex-shrink-0 items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
			<span class="flex-shrink-0 text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
			<button
				onclick={(e) => { e.stopPropagation(); onToggle?.(); }}
				class="flex-shrink-0 rounded border border-slate-200 px-1.5 py-px text-[10px] text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-600 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-300"
				title={expanded ? 'Collapse' : 'Expand'}
			>{expanded ? '▾' : '▸'}</button>
		</div>

		<!-- Detail panel (task 3.5) -->
		{#if expanded}
			<div
				onclick={handleCardClick}
				class="animate-slideDown cursor-pointer rounded-lg border border-slate-200 bg-white p-[12px_16px] -mt-0.5 mb-2 dark:border-slate-700 dark:bg-slate-800"
			>
				<h3 class="mb-1 text-[14px] font-semibold text-slate-900 dark:text-slate-50">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline">{entry.title || 'Untitled'}</a>
				</h3>
				{#if isPending}
					<div class="flex items-center gap-2 py-2 text-sm text-slate-400 dark:text-slate-500">
						<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
						Processing…
					</div>
				{:else if isFragment}
					<div class="fragment-content mb-2 text-[12px] leading-relaxed text-slate-500 dark:text-slate-400">
						{@html sanitizeHTML(entry.raw_content)}
					</div>
				{:else}
					<p class="mb-2 text-[12px] leading-relaxed text-slate-500 dark:text-slate-400">{entry.summary}</p>
				{/if}

				<div class="mb-2 flex flex-wrap items-center gap-2">
					{#if !isPending}
						<StarRating aiStars={entry.ai_stars} userStars={entry.user_stars} onRate={handleRate} />
					{/if}
					<span class="inline-flex items-center gap-1.5 text-[11px] text-slate-400 dark:text-slate-500">
						<span class="inline-flex h-[18px] w-[18px] items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
						{sourceName}
					</span>
					<span class="ml-auto text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
				</div>

				<!-- Referenced links -->
				{#if referencedLinks().length > 0}
					<div class="mb-2">
						<button
							onclick={() => (showLinks = !showLinks)}
							class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors dark:text-slate-400 dark:hover:text-slate-300"
						>
							<svg class="h-3 w-3 transition-transform {showLinks ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
							</svg>
							🔗 {referencedLinks().length} referenced {referencedLinks().length === 1 ? 'link' : 'links'}
						</button>
						{#if showLinks}
							<div class="mt-1.5 space-y-1 pl-4">
								{#each referencedLinks() as link}
									<div class="flex items-center gap-1.5">
										<a href={link.href} target="_blank" rel="noopener" class="text-xs text-blue-600 hover:text-blue-800 hover:underline text-left truncate max-w-[280px] dark:text-blue-400 dark:hover:text-blue-300" title={link.href}>{link.text}</a>
										<button onclick={() => onOpenLink(entry, link.href)} class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 transition-colors dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50" title="Get AI summary">Summarize</button>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex flex-wrap gap-[7px]">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen}
						class="inline-flex items-center gap-1 rounded-md bg-blue-500 px-3 py-1.5 text-[12px] font-semibold text-white transition-colors hover:bg-blue-600">Read</a>
					<button onclick={toggleBookmark}
						class="inline-flex items-center gap-1 rounded-md border px-3 py-1.5 text-[12px] font-semibold transition-colors
							{entry.bookmarked
							? 'border-amber-500 text-amber-500 dark:border-amber-400 dark:text-amber-400'
							: 'border-amber-500/50 text-amber-500/70 dark:border-amber-400/50 dark:text-amber-400/70'}">
						{entry.bookmarked ? '📌 Saved' : 'Save'}
					</button>
					<button onclick={() => onOpenChat(entry)}
						class="inline-flex items-center gap-1 rounded-md border border-slate-300 px-3 py-1.5 text-[12px] font-semibold text-slate-500 transition-colors hover:text-slate-900 dark:border-slate-600 dark:text-slate-400 dark:hover:text-slate-100">
						🤖 Chat
					</button>
					<button onclick={toggleRead}
						class="inline-flex items-center gap-1 rounded-md border border-slate-200 px-3 py-1.5 text-[12px] text-slate-400 transition-colors hover:text-slate-600 dark:border-slate-700 dark:text-slate-500 dark:hover:text-slate-300"
						title={entry.is_read ? 'Mark as unread' : 'Mark as read'}>
						{entry.is_read ? '✓ Read' : 'Mark read'}
					</button>
				</div>
			</div>
		{/if}
	</div>

{:else}
	<!-- ═══ LOW PRIORITY (1-2★) ═══ -->
	<div>
		<!-- Minimal muted row (task 3.6) -->
		<div
			onclick={handleCardClick}
			class="flex cursor-pointer items-center gap-2.5 rounded px-1 py-[7px] transition-[opacity,background]
				{expanded ? 'rounded-b-none bg-slate-50 opacity-85 dark:bg-slate-800' : 'border-b border-slate-100 opacity-50 hover:bg-slate-50 hover:opacity-75 dark:border-slate-800 dark:hover:bg-slate-800'}"
		>
			<span class="text-[13px] tracking-wider text-amber-500 dark:text-amber-400">{starsDisplay(effectiveStars)}</span>
			<span class="flex-1 truncate text-[12px] text-slate-400 dark:text-slate-500">
				<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline hover:text-slate-600 dark:hover:text-slate-400">{entry.title || 'Untitled'}</a>
			</span>
			<span class="inline-flex h-[18px] w-[18px] flex-shrink-0 items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
			<span class="flex-shrink-0 text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
			<button
				onclick={(e) => { e.stopPropagation(); onToggle?.(); }}
				class="flex-shrink-0 rounded border border-slate-200 px-1.5 py-px text-[10px] text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-600 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-300"
				title={expanded ? 'Collapse' : 'Expand'}
			>{expanded ? '▾' : '▸'}</button>
		</div>

		<!-- Detail panel -->
		{#if expanded}
			<div
				onclick={handleCardClick}
				class="animate-slideDown cursor-pointer rounded-lg border border-slate-200 bg-white p-[12px_16px] -mt-0.5 mb-2 dark:border-slate-700 dark:bg-slate-800"
			>
				<h3 class="mb-1 text-[14px] font-semibold text-slate-900 dark:text-slate-50">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen} class="hover:underline">{entry.title || 'Untitled'}</a>
				</h3>
				{#if isPending}
					<div class="flex items-center gap-2 py-2 text-sm text-slate-400 dark:text-slate-500">
						<svg class="h-4 w-4 animate-spin" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
						</svg>
						Processing…
					</div>
				{:else if isFragment}
					<div class="fragment-content mb-2 text-[12px] leading-relaxed text-slate-500 dark:text-slate-400">
						{@html sanitizeHTML(entry.raw_content)}
					</div>
				{:else}
					<p class="mb-2 text-[12px] leading-relaxed text-slate-500 dark:text-slate-400">{entry.summary}</p>
				{/if}

				<div class="mb-2 flex flex-wrap items-center gap-2">
					{#if !isPending}
						<StarRating aiStars={entry.ai_stars} userStars={entry.user_stars} onRate={handleRate} />
					{/if}
					<span class="inline-flex items-center gap-1.5 text-[11px] text-slate-400 dark:text-slate-500">
						<span class="inline-flex h-[18px] w-[18px] items-center justify-center rounded text-[9px] font-bold text-slate-900" style="background: {getSourceColor()}">{getSourceInitials()}</span>
						{sourceName}
					</span>
					<span class="ml-auto text-[11px] text-slate-400 dark:text-slate-500">{relativeTime(displayTime)}</span>
				</div>

				<!-- Referenced links -->
				{#if referencedLinks().length > 0}
					<div class="mb-2">
						<button
							onclick={() => (showLinks = !showLinks)}
							class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors dark:text-slate-400 dark:hover:text-slate-300"
						>
							<svg class="h-3 w-3 transition-transform {showLinks ? 'rotate-90' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
								<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
							</svg>
							🔗 {referencedLinks().length} referenced {referencedLinks().length === 1 ? 'link' : 'links'}
						</button>
						{#if showLinks}
							<div class="mt-1.5 space-y-1 pl-4">
								{#each referencedLinks() as link}
									<div class="flex items-center gap-1.5">
										<a href={link.href} target="_blank" rel="noopener" class="text-xs text-blue-600 hover:text-blue-800 hover:underline text-left truncate max-w-[280px] dark:text-blue-400 dark:hover:text-blue-300" title={link.href}>{link.text}</a>
										<button onclick={() => onOpenLink(entry, link.href)} class="shrink-0 rounded px-1.5 py-0.5 text-[10px] font-medium text-blue-600 bg-blue-50 hover:bg-blue-100 transition-colors dark:text-blue-400 dark:bg-blue-900/30 dark:hover:bg-blue-900/50" title="Get AI summary">Summarize</button>
									</div>
								{/each}
							</div>
						{/if}
					</div>
				{/if}

				<!-- Actions -->
				<div class="flex flex-wrap gap-[7px]">
					<a href={entry.url} target="_blank" rel="noopener" onclick={markReadAndOpen}
						class="inline-flex items-center gap-1 rounded-md bg-blue-500 px-3 py-1.5 text-[12px] font-semibold text-white transition-colors hover:bg-blue-600">Read</a>
					<button onclick={toggleBookmark}
						class="inline-flex items-center gap-1 rounded-md border px-3 py-1.5 text-[12px] font-semibold transition-colors
							{entry.bookmarked
							? 'border-amber-500 text-amber-500 dark:border-amber-400 dark:text-amber-400'
							: 'border-amber-500/50 text-amber-500/70 dark:border-amber-400/50 dark:text-amber-400/70'}">
						{entry.bookmarked ? '📌 Saved' : 'Save'}
					</button>
					<button onclick={toggleRead}
						class="inline-flex items-center gap-1 rounded-md border border-slate-200 px-3 py-1.5 text-[12px] text-slate-400 transition-colors hover:text-slate-600 dark:border-slate-700 dark:text-slate-500 dark:hover:text-slate-300"
						title={entry.is_read ? 'Mark as unread' : 'Mark as read'}>
						{entry.is_read ? '✓ Read' : 'Mark read'}
					</button>
				</div>
			</div>
		{/if}
	</div>
{/if}

<style>
	/* slideDown animation for WaL and LP detail panels (task 3.9) */
	@keyframes slideDown {
		from { opacity: 0; transform: translateY(-4px); }
		to { opacity: 1; transform: translateY(0); }
	}
	:global(.animate-slideDown) {
		animation: slideDown 0.15s ease-out;
	}

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
