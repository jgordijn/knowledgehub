<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { RecordModel, UnsubscribeFunc } from 'pocketbase';
	import pb from '$lib/pb';
	import EntryCard from '$lib/components/EntryCard.svelte';
	import ChatPanel from '$lib/components/ChatPanel.svelte';
	import LinkPanel from '$lib/components/LinkPanel.svelte';
	import QuarantineBanner from '$lib/components/QuarantineBanner.svelte';
	import QuickAddModal from '$lib/components/QuickAddModal.svelte';
	import { sidebarData } from '$lib/stores/sidebar';

	let entries = $state<RecordModel[]>([]);
	let loading = $state(true);
	let readFilter = $state<'unread' | 'all' | 'bookmarked'>('unread');
	let starFilter = $state<number>(0); // 0 = all, 3/4/5 = minimum
	let markReadOpen = $state(false);
	let unreadCount = $state(0);
	let bookmarkedCount = $state(0);
	let undoEntries = $state<RecordModel[]>([]);
	let undoTimeout: ReturnType<typeof setTimeout> | undefined;

	// Multi-select source filter (task 2.2)
	let selectedSources = $state<Set<string>>(new Set());

	// Chat state
	let chatEntry = $state<RecordModel | null>(null);

	// Quick Add state
	let quickAddOpen = $state(false);

	// Link panel state
	let linkEntry = $state<RecordModel | null>(null);
	let linkUrl = $state<string>('');

	let unsub: UnsubscribeFunc | undefined;

	// Expand/collapse state (task 4.2)
	let expandedSet = $state<Set<string>>(new Set());

	function effectiveStars(entry: RecordModel): number {
		return entry.user_stars || entry.ai_stars || 0;
	}

	function entryTime(e: RecordModel): number {
		return new Date(e.published_at || e.discovered_at).getTime();
	}

	function sortEntries(list: RecordModel[]): RecordModel[] {
		return [...list].sort((a, b) => entryTime(b) - entryTime(a));
	}

	let resources = $state<{ id: string; name: string }[]>([]);

	// Client-side filtering with multi-select source filter (task 2.3)
	let filteredEntries = $derived(() => {
		let result = entries;
		if (starFilter > 0) {
			result = result.filter((e) => effectiveStars(e) >= starFilter);
		}
		// Multi-select source filter: client-side
		if (selectedSources.size > 0) {
			result = result.filter((e) => selectedSources.has(e.resource));
		}
		return sortEntries(result);
	});

	// Tier grouping (task 4.1)
	let featuredEntries = $derived(filteredEntries().filter((e) => effectiveStars(e) === 5));
	let hpEntries = $derived(filteredEntries().filter((e) => effectiveStars(e) === 4));
	let walEntries = $derived(filteredEntries().filter((e) => effectiveStars(e) === 3));
	let lpEntries = $derived(filteredEntries().filter((e) => {
		const s = effectiveStars(e);
		return s >= 1 && s <= 2;
	}));
	let pendingEntries = $derived(filteredEntries().filter((e) => effectiveStars(e) === 0));

	// Compute per-source entry counts (task 2.6)
	let sourceCounts = $derived(() => {
		const counts = new Map<string, number>();
		for (const e of entries) {
			counts.set(e.resource, (counts.get(e.resource) || 0) + 1);
		}
		return counts;
	});

	// Initialize expanded defaults when entries change (task 4.2)
	function initExpandedDefaults() {
		const newSet = new Set<string>();
		for (const e of entries) {
			const stars = effectiveStars(e);
			if (stars >= 4) {
				newSet.add(e.id);
			}
		}
		expandedSet = newSet;
	}

	function toggleExpand(id: string) {
		const newSet = new Set(expandedSet);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		expandedSet = newSet;
	}

	// Section batch controls (task 5.1-5.3)
	function expandAll(sectionEntries: RecordModel[]) {
		const newSet = new Set(expandedSet);
		for (const e of sectionEntries) {
			newSet.add(e.id);
		}
		expandedSet = newSet;
	}

	function collapseAll(sectionEntries: RecordModel[]) {
		const newSet = new Set(expandedSet);
		for (const e of sectionEntries) {
			newSet.delete(e.id);
		}
		expandedSet = newSet;
	}

	function hasCollapsed(sectionEntries: RecordModel[]): boolean {
		return sectionEntries.some((e) => !expandedSet.has(e.id));
	}

	function hasExpanded(sectionEntries: RecordModel[]): boolean {
		return sectionEntries.some((e) => expandedSet.has(e.id));
	}

	// Source filter callbacks (task 2.2)
	function toggleSource(id: string) {
		const newSet = new Set(selectedSources);
		if (newSet.has(id)) {
			newSet.delete(id);
		} else {
			newSet.add(id);
		}
		selectedSources = newSet;
	}

	function clearSources() {
		selectedSources = new Set();
	}

	async function loadEntries() {
		loading = true;
		undoEntries = [];
		clearTimeout(undoTimeout);
		try {
			const filters: string[] = ['resource.active = true'];
			if (readFilter === 'unread') {
				filters.push('is_read = false');
			} else if (readFilter === 'bookmarked') {
				filters.push('bookmarked = true');
			}
			const result = await pb.collection('entries').getList(1, 200, {
				sort: '-published_at,-discovered_at',
				expand: 'resource',
				filter: filters.length > 0 ? filters.join(' && ') : '',
				requestKey: 'loadEntries'
			});
			entries = result.items;
			initExpandedDefaults();
		} catch (err) {
			console.error('loadEntries failed:', err);
		} finally {
			loading = false;
		}
	}

	async function loadUnreadCount() {
		try {
			const result = await pb.collection('entries').getList(1, 1, {
				filter: 'is_read = false && resource.active = true',
				fields: 'id',
				skipTotal: false,
				requestKey: 'loadUnreadCount'
			});
			unreadCount = result.totalItems;
		} catch {
			// ignore
		}
	}

	async function loadBookmarkedCount() {
		try {
			const result = await pb.collection('entries').getList(1, 1, {
				filter: 'bookmarked = true && resource.active = true',
				fields: 'id',
				skipTotal: false,
				requestKey: 'loadBookmarkedCount'
			});
			bookmarkedCount = result.totalItems;
		} catch {
			// ignore
		}
	}

	async function loadResources() {
		try {
			const result = await pb.collection('resources').getList(1, 200, {
				filter: 'active = true',
				sort: 'name',
				fields: 'id,name'
			});
			resources = result.items.map((r) => ({ id: r.id, name: r.name }));
		} catch {
			// Backend may not be ready
		}
	}

	function handleEntryUpdate(updated: RecordModel) {
		const existing = entries.find((e) => e.id === updated.id);

		if (readFilter === 'unread' && updated.is_read && existing) {
			entries = entries.filter((e) => e.id !== updated.id);
			undoEntries = [...undoEntries, { ...existing, ...updated }];
			clearTimeout(undoTimeout);
			undoTimeout = setTimeout(() => { undoEntries = []; }, 15000);
		} else {
			entries = entries.map((e) => (e.id === updated.id ? updated : e));
		}

		loadUnreadCount();
		loadBookmarkedCount();
	}

	async function undoMarkRead() {
		const toUndo = [...undoEntries];
		undoEntries = [];
		clearTimeout(undoTimeout);

		await Promise.all(
			toUndo.map((e) =>
				pb.collection('entries').update(e.id, { is_read: false }).catch(() => {})
			)
		);

		entries = [...entries, ...toUndo.map((e) => ({ ...e, is_read: false }))];
		loadUnreadCount();
		loadBookmarkedCount();
	}

	async function markAsRead(olderThanDays?: number) {
		markReadOpen = false;
		let toMark = entries.filter((e) => !e.is_read);
		if (olderThanDays !== undefined) {
			const cutoff = Date.now() - olderThanDays * 86400000;
			toMark = toMark.filter((e) => entryTime(e) < cutoff);
		}
		if (toMark.length === 0) return;
		await Promise.all(
			toMark.map((e) => pb.collection('entries').update(e.id, { is_read: true }).catch(() => {}))
		);
		await loadEntries();
		loadUnreadCount();
	}

	function openChat(entry: RecordModel) {
		chatEntry = entry;
	}

	function closeChat() {
		chatEntry = null;
	}

	function openLink(entry: RecordModel, url: string) {
		linkEntry = entry;
		linkUrl = url;
	}

	function closeLink() {
		linkEntry = null;
		linkUrl = '';
	}

	function tryNotify(entry: RecordModel) {
		if (Notification.permission === 'granted' && effectiveStars(entry) === 5) {
			new Notification('KnowledgeHub ★★★★★', { body: entry.title });
		}
	}

	let mounted = false;

	// Reload entries when read filter changes
	$effect(() => {
		readFilter;
		if (mounted) {
			loadEntries();
		}
	});

	// Push sidebar data to store so layout can pass to Sidebar component
	$effect(() => {
		sidebarData.set({
			unreadCount,
			bookmarkedCount,
			resources,
			selectedSources,
			sourceCounts: sourceCounts(),
			onToggleSource: toggleSource,
			onClearSources: clearSources
		});
	});

	onMount(async () => {
		await Promise.all([loadEntries(), loadResources(), loadUnreadCount(), loadBookmarkedCount()]);
		mounted = true;

		if ('Notification' in window && Notification.permission === 'default') {
			Notification.requestPermission();
		}

		try {
			unsub = await pb.collection('entries').subscribe('*', async (e) => {
				if (e.action === 'create') {
					try {
						const full = await pb.collection('entries').getOne(e.record.id, {
							expand: 'resource'
						});
						entries = [...entries, full];
						// Add to expanded set if featured/HP
						const stars = effectiveStars(full);
						if (stars >= 4) {
							expandedSet = new Set([...expandedSet, full.id]);
						}
						loadUnreadCount();
						loadBookmarkedCount();
						tryNotify(full);
					} catch {
						entries = [...entries, e.record];
					}
				} else if (e.action === 'update') {
					entries = entries.map((en) =>
						en.id === e.record.id ? { ...en, ...e.record } : en
					);
					loadUnreadCount();
					loadBookmarkedCount();
				} else if (e.action === 'delete') {
					entries = entries.filter((en) => en.id !== e.record.id);
					loadUnreadCount();
					loadBookmarkedCount();
				}
			});
		} catch {
			// Realtime may not be available
		}
	});

	onDestroy(() => {
		unsub?.();
		clearTimeout(undoTimeout);
	});

	// Helper to get resource name by id
	function getResourceName(id: string): string {
		return resources.find((r) => r.id === id)?.name ?? id;
	}
</script>

<div class="space-y-4">
	<QuarantineBanner />

	<!-- Topbar: tabs + source chips + star filter + mark read (task 6.1-6.3) -->
	<div class="flex flex-wrap items-center gap-2">
		<!-- Read filter tabs (segmented control) -->
		<div class="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-slate-700">
			<button
				onclick={() => (readFilter = 'unread')}
				class="rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors
					{readFilter === 'unread' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				Unread{unreadCount > 0 ? ` ${unreadCount}` : ''}
			</button>
			<button
				onclick={() => (readFilter = 'bookmarked')}
				class="rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors
					{readFilter === 'bookmarked' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				Saved{bookmarkedCount > 0 ? ` ${bookmarkedCount}` : ''}
			</button>
			<button
				onclick={() => (readFilter = 'all')}
				class="rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors
					{readFilter === 'all' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				All
			</button>
		</div>

		<!-- Active source chips (task 2.4) -->
		{#if selectedSources.size > 0}
			<div class="inline-flex flex-wrap items-center gap-1.5">
				{#each [...selectedSources] as srcId}
					<span class="inline-flex items-center gap-1 rounded-md border border-blue-500/30 bg-blue-50 px-2 py-0.5 text-[11px] text-blue-600 dark:border-blue-500/30 dark:bg-blue-900/20 dark:text-blue-400">
						{getResourceName(srcId)}
						<button
							onclick={() => toggleSource(srcId)}
							class="ml-0.5 text-[12px] leading-none text-slate-400 hover:text-slate-900 dark:text-slate-500 dark:hover:text-slate-50"
							title="Remove filter"
						>✕</button>
					</span>
				{/each}
			</div>
		{/if}

		<!-- Star filter -->
		<select
			class="rounded-md border border-slate-200 bg-slate-50 px-2.5 py-1 text-[12px] text-slate-500 transition-colors dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400"
			onchange={(e) => (starFilter = parseInt((e.target as HTMLSelectElement).value))}
		>
			<option value="0" selected={starFilter === 0}>★ All</option>
			<option value="3" selected={starFilter === 3}>★ 3+</option>
			<option value="4" selected={starFilter === 4}>★ 4+</option>
			<option value="5" selected={starFilter === 5}>★ 5</option>
		</select>

		<div class="flex-1"></div>

		<!-- Mark as read dropdown -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="relative" onfocusout={(e) => { if (!e.currentTarget.contains(e.relatedTarget as Node)) markReadOpen = false; }}>
			<button
				onclick={() => (markReadOpen = !markReadOpen)}
				class="rounded-md border border-slate-300 px-2.5 py-1 text-[12px] font-medium text-slate-500 transition-colors hover:text-slate-900 dark:border-slate-600 dark:text-slate-400 dark:hover:text-slate-100"
			>
				Mark read ▾
			</button>
			{#if markReadOpen}
				<div class="absolute right-0 z-10 mt-1 w-44 rounded-lg border border-slate-200 bg-white py-1 shadow-lg dark:border-slate-700 dark:bg-slate-800">
					{#each [
						{ label: 'All entries', days: undefined },
						{ label: 'Older than 1 day', days: 1 },
						{ label: 'Older than 3 days', days: 3 },
						{ label: 'Older than 7 days', days: 7 },
						{ label: 'Older than 30 days', days: 30 }
					] as opt}
						<button
							onclick={() => markAsRead(opt.days)}
							class="block w-full px-3 py-1.5 text-left text-sm text-slate-700 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-700"
						>
							{opt.label}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Undo banner -->
	{#if undoEntries.length > 0}
		<div class="fixed bottom-4 left-4 right-4 z-50 mx-auto max-w-lg flex items-center justify-between rounded-lg bg-slate-700 px-4 py-2.5 text-sm text-white shadow-lg dark:bg-slate-600">
			<span>
				{undoEntries.length === 1 ? '1 article' : `${undoEntries.length} articles`} marked as read
			</span>
			<button
				onclick={undoMarkRead}
				class="rounded-md bg-white/20 px-3 py-1 text-sm font-medium hover:bg-white/30 transition-colors"
			>
				Undo
			</button>
		</div>
	{/if}

	<!-- Entries — tiered layout (tasks 4.1-4.5) -->
	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">Loading entries…</div>
	{:else if filteredEntries().length === 0}
		<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">
			{readFilter === 'unread' ? 'No unread entries. Switch to "All" to see read entries.' : readFilter === 'bookmarked' ? 'No bookmarked entries. Use the bookmark icon to save articles for later.' : 'No entries yet. Add some resources to get started.'}
		</div>
	{:else}
		<!-- Featured (5★) -->
		{#if featuredEntries.length > 0}
			<div class="flex items-center gap-2 px-1 pt-2.5 pb-1">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Featured</span>
				{#if hasCollapsed(featuredEntries)}
					<button onclick={() => expandAll(featuredEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Expand all</button>
				{/if}
				{#if hasExpanded(featuredEntries)}
					<button onclick={() => collapseAll(featuredEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Collapse all</button>
				{/if}
			</div>
			{#each featuredEntries as entry (entry.id)}
				<EntryCard {entry} expanded={expandedSet.has(entry.id)} onToggle={() => toggleExpand(entry.id)} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		{/if}

		<!-- High Priority (4★) -->
		{#if hpEntries.length > 0}
			<div class="flex items-center gap-2 px-1 pt-2.5 pb-1">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">High Priority</span>
				{#if hasCollapsed(hpEntries)}
					<button onclick={() => expandAll(hpEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Expand all</button>
				{/if}
				{#if hasExpanded(hpEntries)}
					<button onclick={() => collapseAll(hpEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Collapse all</button>
				{/if}
			</div>
			{#each hpEntries as entry (entry.id)}
				<EntryCard {entry} expanded={expandedSet.has(entry.id)} onToggle={() => toggleExpand(entry.id)} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		{/if}

		<!-- Worth a Look (3★) -->
		{#if walEntries.length > 0}
			<div class="flex items-center gap-2 px-1 pt-2.5 pb-1">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Worth a Look <span class="normal-case tracking-normal">— click ▸ to expand</span></span>
				{#if hasCollapsed(walEntries)}
					<button onclick={() => expandAll(walEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Expand all</button>
				{/if}
				{#if hasExpanded(walEntries)}
					<button onclick={() => collapseAll(walEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Collapse all</button>
				{/if}
			</div>
			{#each walEntries as entry (entry.id)}
				<EntryCard {entry} expanded={expandedSet.has(entry.id)} onToggle={() => toggleExpand(entry.id)} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		{/if}

		<!-- Low Priority (1-2★) -->
		{#if lpEntries.length > 0}
			<div class="flex items-center gap-2 px-1 pt-2.5 pb-1">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Low Priority <span class="normal-case tracking-normal">— click ▸ to expand</span></span>
				{#if hasCollapsed(lpEntries)}
					<button onclick={() => expandAll(lpEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Expand all</button>
				{/if}
				{#if hasExpanded(lpEntries)}
					<button onclick={() => collapseAll(lpEntries)} class="rounded border border-slate-200 px-2 py-px text-[10px] font-medium text-slate-400 transition-colors hover:border-slate-400 hover:text-slate-900 dark:border-slate-600 dark:text-slate-500 dark:hover:border-slate-400 dark:hover:text-slate-50">Collapse all</button>
				{/if}
			</div>
			{#each lpEntries as entry (entry.id)}
				<EntryCard {entry} expanded={expandedSet.has(entry.id)} onToggle={() => toggleExpand(entry.id)} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		{/if}

		<!-- Pending (0★ / processing) -->
		{#if pendingEntries.length > 0}
			<div class="flex items-center gap-2 px-1 pt-2.5 pb-1">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Processing</span>
			</div>
			{#each pendingEntries as entry (entry.id)}
				<EntryCard {entry} expanded={expandedSet.has(entry.id)} onToggle={() => toggleExpand(entry.id)} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		{/if}
	{/if}
</div>

<!-- Chat panel -->
{#if chatEntry}
	<ChatPanel entryId={chatEntry.id} entryTitle={chatEntry.title ?? 'Untitled'} onClose={closeChat} />
{/if}

<!-- Link summary panel -->
{#if linkEntry && linkUrl}
	<LinkPanel url={linkUrl} entryId={linkEntry.id} entryTitle={linkEntry.title ?? 'Untitled'} onClose={closeLink} />
{/if}

<!-- Quick Add FAB -->
<button
	onclick={() => (quickAddOpen = true)}
	class="fixed bottom-6 right-6 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-blue-600 text-white shadow-lg transition-transform hover:scale-105 hover:bg-blue-700 active:scale-95"
	aria-label="Quick Add Article"
	title="Quick Add Article"
>
	<svg class="h-7 w-7" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
		<path stroke-linecap="round" stroke-linejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
	</svg>
</button>

<!-- Quick Add Modal -->
{#if quickAddOpen}
	<QuickAddModal onClose={() => (quickAddOpen = false)} />
{/if}
