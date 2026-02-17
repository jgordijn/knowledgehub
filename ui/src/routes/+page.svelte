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
		return [...list].sort((a, b) => {
			const starsA = effectiveStars(a);
			const starsB = effectiveStars(b);
			if (starsB !== starsA) return starsB - starsA;
			return entryTime(b) - entryTime(a);
		});
	}

	let filteredEntries = $derived(() => {
		let result = entries;
		if (readFilter === 'unread') {
			result = result.filter((e) => !e.is_read);
		}
		if (starFilter > 0) {
			result = result.filter((e) => effectiveStars(e) >= starFilter);
		}
		return sortEntries(result);
	});

	async function loadEntries() {
		loading = true;
		try {
			const result = await pb.collection('entries').getList(1, 200, {
				sort: '-user_stars,-ai_stars,-discovered_at',
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
	</div>

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
