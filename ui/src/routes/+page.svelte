<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { RecordModel, UnsubscribeFunc } from 'pocketbase';
	import pb from '$lib/pb';
	import EntryCard from '$lib/components/EntryCard.svelte';
	import ChatPanel from '$lib/components/ChatPanel.svelte';
	import LinkPanel from '$lib/components/LinkPanel.svelte';
	import QuarantineBanner from '$lib/components/QuarantineBanner.svelte';

	let entries = $state<RecordModel[]>([]);
	let loading = $state(true);
	let readFilter = $state<'unread' | 'all' | 'bookmarked'>('unread');
	let starFilter = $state<number>(0); // 0 = all, 3/4/5 = minimum
	let resourceFilter = $state<string>(''); // '' = all, or resource ID
	let resourcesOpen = $state(false);
	let markReadOpen = $state(false);
	let unreadCount = $state(0);
	let bookmarkedCount = $state(0);
	let undoEntries = $state<RecordModel[]>([]);
	let undoTimeout: ReturnType<typeof setTimeout> | undefined;

	// Chat state
	let chatEntry = $state<RecordModel | null>(null);

	// Link panel state
	let linkEntry = $state<RecordModel | null>(null);
	let linkUrl = $state<string>('');

	let unsub: UnsubscribeFunc | undefined;

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

	let filteredEntries = $derived(() => {
		let result = entries;
		if (starFilter > 0) {
			result = result.filter((e) => effectiveStars(e) >= starFilter);
		}
		return sortEntries(result);
	});

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
			if (resourceFilter) {
				filters.push(`resource = '${resourceFilter}'`);
			}
			const result = await pb.collection('entries').getList(1, 200, {
				sort: '-published_at,-discovered_at',
				expand: 'resource',
				filter: filters.length > 0 ? filters.join(' && ') : '',
				requestKey: 'loadEntries'
			});
			entries = result.items;
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
			// Entry just marked as read on unread view — remove and offer undo
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
		// Reload to reflect server-side filtering
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

	// Reload entries when read or resource filter changes
	$effect(() => {
		// Track the reactive dependencies
		readFilter;
		resourceFilter;
		if (mounted) {
			loadEntries();
		}
	});

	onMount(async () => {
		await Promise.all([loadEntries(), loadResources(), loadUnreadCount(), loadBookmarkedCount()]);
		mounted = true;

		// Request notification permission
		if ('Notification' in window && Notification.permission === 'default') {
			Notification.requestPermission();
		}

		// Subscribe to realtime updates
		try {
			unsub = await pb.collection('entries').subscribe('*', async (e) => {
				if (e.action === 'create') {
					// Fetch with expanded resource
					try {
						const full = await pb.collection('entries').getOne(e.record.id, {
							expand: 'resource'
						});
						entries = [...entries, full];
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
</script>




<div class="space-y-4">
	<QuarantineBanner />

	<!-- Filters -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<!-- Read/unread tabs -->
		<div class="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-slate-700">
			<button
				onclick={() => (readFilter = 'unread')}
				class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
					{readFilter === 'unread' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				Unread ({unreadCount})
			</button>
			<button
				onclick={() => (readFilter = 'all')}
				class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
					{readFilter === 'all' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				All
			</button>
			<button
				onclick={() => (readFilter = 'bookmarked')}
				class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
					{readFilter === 'bookmarked' ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
			>
				📌 Read Later{bookmarkedCount > 0 ? ` (${bookmarkedCount})` : ''}
			</button>
		</div>

		<!-- Star filter -->
		<div class="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-slate-700">
			{#each [{ label: 'All', value: 0 }, { label: '3+', value: 3 }, { label: '4+', value: 4 }, { label: '5', value: 5 }] as opt}
				<button
					onclick={() => (starFilter = opt.value)}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
						{starFilter === opt.value ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100' : 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
				>
					{opt.label}
				</button>
			{/each}
		</div>

		<!-- Mark as read -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="relative" onfocusout={(e) => { if (!e.currentTarget.contains(e.relatedTarget as Node)) markReadOpen = false; }}>
			<button
				onclick={() => (markReadOpen = !markReadOpen)}
				class="rounded-md px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-100 transition-colors dark:text-slate-400 dark:hover:bg-slate-700"
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

	<!-- Resource filter (collapsible) -->
	{#if resources.length > 1}
		<div>
			<button
				onclick={() => (resourcesOpen = !resourcesOpen)}
				class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors dark:text-slate-400 dark:hover:text-slate-300"
			>
				<svg
					class="h-3 w-3 transition-transform {resourcesOpen ? 'rotate-90' : ''}"
					fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
				Sources{resourceFilter ? ': ' + resources.find((r) => r.id === resourceFilter)?.name : ''}
			</button>
			{#if resourcesOpen}
				<div class="mt-2 flex flex-wrap gap-1.5">
					<button
						onclick={() => (resourceFilter = '')}
						class="rounded-full px-3 py-1 text-xs font-medium transition-colors
							{resourceFilter === '' ? 'bg-slate-800 text-white dark:bg-slate-200 dark:text-slate-900' : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600'}"
					>
						All
					</button>
					{#each resources as res}
						<button
							onclick={() => (resourceFilter = res.id)}
							class="rounded-full px-3 py-1 text-xs font-medium transition-colors
								{resourceFilter === res.id ? 'bg-slate-800 text-white dark:bg-slate-200 dark:text-slate-900' : 'bg-slate-100 text-slate-600 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-400 dark:hover:bg-slate-600'}"
						>
							{res.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Undo banner -->
	{#if undoEntries.length > 0}
		<div class="flex items-center justify-between rounded-lg bg-slate-700 px-4 py-2.5 text-sm text-white shadow-md dark:bg-slate-600">
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

	<!-- Entries -->
	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">Loading entries…</div>
	{:else if filteredEntries().length === 0}
		<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">
			{readFilter === 'unread' ? 'No unread entries. Switch to "All" to see read entries.' : readFilter === 'bookmarked' ? 'No bookmarked entries. Use the bookmark icon to save articles for later.' : 'No entries yet. Add some resources to get started.'}
		</div>
	{:else}
		<div class="grid gap-3">
			{#each filteredEntries() as entry (entry.id)}
				<EntryCard {entry} onOpenChat={openChat} onOpenLink={openLink} onUpdate={handleEntryUpdate} />
			{/each}
		</div>
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
