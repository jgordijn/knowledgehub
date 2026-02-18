<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import type { RecordModel, UnsubscribeFunc } from 'pocketbase';
	import pb from '$lib/pb';
	import EntryCard from '$lib/components/EntryCard.svelte';
	import ChatPanel from '$lib/components/ChatPanel.svelte';
	import QuarantineBanner from '$lib/components/QuarantineBanner.svelte';

	let entries = $state<RecordModel[]>([]);
	let loading = $state(true);
	let readFilter = $state<'unread' | 'all'>('unread');
	let starFilter = $state<number>(0); // 0 = all, 3/4/5 = minimum
	let resourceFilter = $state<string>(''); // '' = all, or resource ID
	let resourcesOpen = $state(false);
	let markReadOpen = $state(false);
	let unreadCount = $state(0);

	// Chat state
	let chatEntry = $state<RecordModel | null>(null);

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

	let resources = $derived(() => {
		const map = new Map<string, string>();
		for (const e of entries) {
			const id = e.resource;
			const name = e.expand?.resource?.name;
			if (id && name && !map.has(id)) map.set(id, name);
		}
		return [...map.entries()]
			.map(([id, name]) => ({ id, name }))
			.sort((a, b) => a.name.localeCompare(b.name));
	});

	let filteredEntries = $derived(() => {
		let result = entries;
		if (readFilter === 'unread') {
			result = result.filter((e) => !e.is_read);
		}
		if (starFilter > 0) {
			result = result.filter((e) => effectiveStars(e) >= starFilter);
		}
		if (resourceFilter) {
			result = result.filter((e) => e.resource === resourceFilter);
		}
		return sortEntries(result);
	});

	async function loadEntries() {
		loading = true;
		try {
			const result = await pb.collection('entries').getList(1, 200, {
				sort: '-published_at,-discovered_at',
				expand: 'resource'
			});
			entries = result.items;
			unreadCount = entries.filter((e) => !e.is_read).length;
		} catch {
			// Backend may not be ready
		} finally {
			loading = false;
		}
	}

	function handleEntryUpdate(updated: RecordModel) {
		entries = entries.map((e) => (e.id === updated.id ? updated : e));
		unreadCount = entries.filter((e) => !e.is_read).length;
	}


	async function markAsRead(olderThanDays?: number) {
		markReadOpen = false;
		let toMark = entries.filter((e) => !e.is_read);
		if (olderThanDays !== undefined) {
			const cutoff = Date.now() - olderThanDays * 86400000;
			toMark = toMark.filter((e) => entryTime(e) < cutoff);
		}
		if (toMark.length === 0) return;
		const ids = new Set(toMark.map((e) => e.id));
		await Promise.all(
			toMark.map((e) => pb.collection('entries').update(e.id, { is_read: true }).catch(() => {}))
		);
		entries = entries.map((e) => (ids.has(e.id) ? { ...e, is_read: true } : e));
		unreadCount = entries.filter((e) => !e.is_read).length;
	}


	function openChat(entry: RecordModel) {
		chatEntry = entry;
	}

	function closeChat() {
		chatEntry = null;
	}

	function tryNotify(entry: RecordModel) {
		if (Notification.permission === 'granted' && effectiveStars(entry) === 5) {
			new Notification('KnowledgeHub ★★★★★', { body: entry.title });
		}
	}

	onMount(async () => {
		await loadEntries();

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
						unreadCount = entries.filter((en) => !en.is_read).length;
						tryNotify(full);
					} catch {
						entries = [...entries, e.record];
					}
				} else if (e.action === 'update') {
					entries = entries.map((en) =>
						en.id === e.record.id ? { ...en, ...e.record } : en
					);
					unreadCount = entries.filter((en) => !en.is_read).length;
				} else if (e.action === 'delete') {
					entries = entries.filter((en) => en.id !== e.record.id);
					unreadCount = entries.filter((en) => !en.is_read).length;
				}
			});
		} catch {
			// Realtime may not be available
		}
	});

	onDestroy(() => {
		unsub?.();
	});
</script>

<svelte:window onclick={() => (markReadOpen = false)} />


<div class="space-y-4">
	<QuarantineBanner />

	<!-- Filters -->
	<div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
		<!-- Read/unread tabs -->
		<div class="flex gap-1 rounded-lg bg-slate-100 p-1">
			<button
				onclick={() => (readFilter = 'unread')}
				class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
					{readFilter === 'unread' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600 hover:text-slate-900'}"
			>
				Unread ({unreadCount})
			</button>
			<button
				onclick={() => (readFilter = 'all')}
				class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
					{readFilter === 'all' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600 hover:text-slate-900'}"
			>
				All
			</button>
		</div>

		<!-- Star filter -->
		<div class="flex gap-1 rounded-lg bg-slate-100 p-1">
			{#each [{ label: 'All', value: 0 }, { label: '3+', value: 3 }, { label: '4+', value: 4 }, { label: '5', value: 5 }] as opt}
				<button
					onclick={() => (starFilter = opt.value)}
					class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors
						{starFilter === opt.value ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600 hover:text-slate-900'}"
				>
					{opt.label}
				</button>
			{/each}
		</div>

		<!-- Mark as read -->
		<div class="relative">
			<button
				onclick={(e) => { e.stopPropagation(); markReadOpen = !markReadOpen; }}
				class="rounded-md px-3 py-1.5 text-sm font-medium text-slate-600 hover:bg-slate-100 transition-colors"
			>
				Mark read ▾
			</button>
			{#if markReadOpen}
				<div class="absolute right-0 z-10 mt-1 w-44 rounded-lg border border-slate-200 bg-white py-1 shadow-lg">
					{#each [
						{ label: 'All entries', days: undefined },
						{ label: 'Older than 1 day', days: 1 },
						{ label: 'Older than 3 days', days: 3 },
						{ label: 'Older than 7 days', days: 7 },
						{ label: 'Older than 30 days', days: 30 }
					] as opt}
						<button
							onclick={() => markAsRead(opt.days)}
							class="block w-full px-3 py-1.5 text-left text-sm text-slate-700 hover:bg-slate-50"
						>
							{opt.label}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	</div>

	<!-- Resource filter (collapsible) -->
	{#if resources().length > 1}
		<div>
			<button
				onclick={() => (resourcesOpen = !resourcesOpen)}
				class="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700 transition-colors"
			>
				<svg
					class="h-3 w-3 transition-transform {resourcesOpen ? 'rotate-90' : ''}"
					fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"
				>
					<path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
				</svg>
				Sources{resourceFilter ? ': ' + resources().find((r) => r.id === resourceFilter)?.name : ''}
			</button>
			{#if resourcesOpen}
				<div class="mt-2 flex flex-wrap gap-1.5">
					<button
						onclick={() => (resourceFilter = '')}
						class="rounded-full px-3 py-1 text-xs font-medium transition-colors
							{resourceFilter === '' ? 'bg-slate-800 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'}"
					>
						All
					</button>
					{#each resources() as res}
						<button
							onclick={() => (resourceFilter = res.id)}
							class="rounded-full px-3 py-1 text-xs font-medium transition-colors
								{resourceFilter === res.id ? 'bg-slate-800 text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'}"
						>
							{res.name}
						</button>
					{/each}
				</div>
			{/if}
		</div>
	{/if}

	<!-- Entries -->
	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400">Loading entries…</div>
	{:else if filteredEntries().length === 0}
		<div class="py-12 text-center text-sm text-slate-400">
			{readFilter === 'unread' ? 'No unread entries. Switch to "All" to see read entries.' : 'No entries yet. Add some resources to get started.'}
		</div>
	{:else}
		<div class="grid gap-3">
			{#each filteredEntries() as entry (entry.id)}
				<EntryCard {entry} onOpenChat={openChat} onUpdate={handleEntryUpdate} />
			{/each}
		</div>
	{/if}
</div>

<!-- Chat panel -->
{#if chatEntry}
	<ChatPanel entryId={chatEntry.id} entryTitle={chatEntry.title ?? 'Untitled'} onClose={closeChat} />
{/if}
