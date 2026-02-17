<script lang="ts">
	import { onMount } from 'svelte';
	import type { RecordModel } from 'pocketbase';
	import pb from '$lib/pb';
	import ResourceForm from '$lib/components/ResourceForm.svelte';

	let resources = $state<RecordModel[]>([]);
	let loading = $state(true);
	let editingResource = $state<RecordModel | null>(null);
	let showAddForm = $state(false);
	let deleteTarget = $state<RecordModel | null>(null);

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

	async function loadResources() {
		loading = true;
		try {
			const result = await pb.collection('resources').getList(1, 200, {
				sort: '-created'
			});
			resources = result.items;
		} catch {
			// Backend may not be ready
		} finally {
			loading = false;
		}
	}

	async function toggleActive(resource: RecordModel) {
		try {
			const updated = await pb.collection('resources').update(resource.id, {
				active: !resource.active
			});
			resources = resources.map((r) => (r.id === resource.id ? { ...r, ...updated } : r));
		} catch {
			// Silently ignore
		}
	}

	async function retryQuarantined(resource: RecordModel) {
		try {
			const updated = await pb.collection('resources').update(resource.id, {
				status: 'healthy',
				consecutive_failures: 0,
				quarantined_at: null,
				last_error: null
			});
			resources = resources.map((r) => (r.id === resource.id ? { ...r, ...updated } : r));
		} catch {
			// Silently ignore
		}
	}

	async function confirmDelete() {
		if (!deleteTarget) return;
		try {
			await pb.collection('resources').delete(deleteTarget.id);
			resources = resources.filter((r) => r.id !== deleteTarget!.id);
		} catch {
			// Silently ignore
		} finally {
			deleteTarget = null;
		}
	}

	function handleSaved() {
		showAddForm = false;
		editingResource = null;
		loadResources();
	}


	let fetchingAll = $state(false);
	let fetchingId = $state('');
	let fetchMessage = $state('');

	async function fetchAllResources() {
		fetchingAll = true;
		fetchMessage = '';
		try {
			const res = await fetch('/api/trigger/all', {
				method: 'POST',
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			const data = await res.json();
			fetchMessage = data.message || data.error || 'Done';
			setTimeout(() => (fetchMessage = ''), 4000);
		} catch {
			fetchMessage = 'Failed to trigger fetch.';
		} finally {
			fetchingAll = false;
		}
	}

	async function fetchResource(resource: RecordModel) {
		fetchingId = resource.id;
		fetchMessage = '';
		try {
			const res = await fetch(`/api/trigger/${resource.id}`, {
				method: 'POST',
				headers: { Authorization: `Bearer ${pb.authStore.token}` }
			});
			const data = await res.json();
			fetchMessage = data.message || data.error || 'Done';
			setTimeout(() => {
				fetchMessage = '';
				loadResources();
			}, 2000);
		} catch {
			fetchMessage = 'Failed to trigger fetch.';
		} finally {
			fetchingId = '';
		}
	}

	onMount(loadResources);
</script>

<div class="space-y-6">
	<div class="flex items-center justify-between gap-2">
		<h1 class="text-xl font-bold text-slate-900">Resources</h1>
		<div class="flex items-center gap-2">
			<button
				onclick={fetchAllResources}
				disabled={fetchingAll}
				class="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50"
			>
				{fetchingAll ? 'Fetching…' : '↻ Fetch All'}
			</button>
			<button
				onclick={() => {
					showAddForm = !showAddForm;
					editingResource = null;
				}}
				class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
			>
				{showAddForm ? 'Cancel' : '+ Add Resource'}
			</button>
		</div>
	</div>

	{#if fetchMessage}
		<div class="rounded-md border border-blue-200 bg-blue-50 px-3 py-2 text-sm text-blue-700">
			{fetchMessage}
		</div>
	{/if}

	<!-- Add form -->
	{#if showAddForm}
		<div class="rounded-lg border border-slate-200 bg-white p-4">
			<h2 class="mb-3 text-sm font-semibold text-slate-700">Add New Resource</h2>
			<ResourceForm onSave={handleSaved} onCancel={() => (showAddForm = false)} />
		</div>
	{/if}

	<!-- Resource list -->
	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400">Loading resources…</div>
	{:else if resources.length === 0}
		<div class="py-12 text-center text-sm text-slate-400">
			No resources yet. Click "+ Add Resource" to get started.
		</div>
	{:else}
		<div class="space-y-3">
			{#each resources as resource (resource.id)}
				{#if editingResource?.id === resource.id}
					<!-- Edit form -->
					<div class="rounded-lg border-2 border-blue-200 bg-white p-4">
						<h2 class="mb-3 text-sm font-semibold text-slate-700">Edit Resource</h2>
						<ResourceForm
							resourceId={resource.id}
							initialName={resource.name}
							initialUrl={resource.url}
							initialType={resource.type}
							initialArticleSelector={resource.article_selector}
							initialContentSelector={resource.content_selector}
							onSave={handleSaved}
							onCancel={() => (editingResource = null)}
						/>
					</div>
				{:else}
					<div class="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
						<div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
							<!-- Info -->
							<div class="min-w-0 flex-1">
								<div class="flex items-center gap-2">
									<!-- Status dot -->
									{#if resource.status === 'healthy'}
										<span class="h-2.5 w-2.5 shrink-0 rounded-full bg-green-500" title="Healthy"></span>
									{:else if resource.status === 'failing'}
										<span class="h-2.5 w-2.5 shrink-0 rounded-full bg-yellow-500" title="Failing"></span>
									{:else if resource.status === 'quarantined'}
										<span class="h-2.5 w-2.5 shrink-0 rounded-full bg-red-500" title="Quarantined"></span>
									{/if}

									<h3 class="text-sm font-semibold text-slate-900">{resource.name}</h3>
									<span
										class="rounded-full px-2 py-0.5 text-xs font-medium
											{resource.type === 'rss' ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700'}"
									>
										{resource.type}
									</span>
									{#if !resource.active}
										<span class="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-500">
											Inactive
										</span>
									{/if}
								</div>

								<p class="mt-1 truncate text-xs text-slate-500">{resource.url}</p>

								<!-- Status detail -->
								{#if resource.status === 'failing'}
									<p class="mt-1 text-xs text-yellow-700">
										{resource.consecutive_failures}/5 failures — {resource.last_error || 'Unknown error'}
									</p>
								{:else if resource.status === 'quarantined'}
									<p class="mt-1 text-xs text-red-700">
										Quarantined {relativeTime(resource.quarantined_at)} — {resource.last_error || 'Unknown error'}
									</p>
								{/if}
							</div>

							<!-- Actions -->
							<div class="flex items-center gap-2 shrink-0">
								<!-- Active toggle -->
								<button
									onclick={() => toggleActive(resource)}
									class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors min-w-[44px]
										{resource.active ? 'bg-blue-600' : 'bg-slate-200'}"
									title={resource.active ? 'Deactivate' : 'Activate'}
								>
									<span
										class="inline-block h-4 w-4 rounded-full bg-white shadow transition-transform
											{resource.active ? 'translate-x-6' : 'translate-x-1'}"
									></span>
								</button>

								{#if resource.status === 'quarantined'}
									<button
										onclick={() => retryQuarantined(resource)}
										class="rounded-md border border-amber-300 bg-amber-50 px-3 py-1.5 text-xs font-medium text-amber-700 hover:bg-amber-100 min-h-[44px] sm:min-h-0"
									>
										Retry Now
									</button>
								{/if}

								<button
									onclick={() => fetchResource(resource)}
									disabled={fetchingId === resource.id}
									class="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 min-h-[44px] sm:min-h-0"
									title="Fetch now"
								>
									{fetchingId === resource.id ? '↻…' : '↻ Fetch'}
								</button>

								<button
									onclick={() => {
										editingResource = resource;
										showAddForm = false;
									}}
									class="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50 min-h-[44px] sm:min-h-0"
								>
									Edit
								</button>

								<button
									onclick={() => (deleteTarget = resource)}
									class="rounded-md border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50 min-h-[44px] sm:min-h-0"
								>
									Delete
								</button>
							</div>
						</div>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>

<!-- Delete confirmation dialog -->
{#if deleteTarget}
	<div class="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
		<div class="w-full max-w-sm rounded-lg bg-white p-6 shadow-xl">
			<h2 class="text-lg font-semibold text-slate-900">Delete Resource</h2>
			<p class="mt-2 text-sm text-slate-600">
				Delete <strong>{deleteTarget.name}</strong> and all its entries? This cannot be undone.
			</p>
			<div class="mt-4 flex justify-end gap-2">
				<button
					onclick={() => (deleteTarget = null)}
					class="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
				>
					Cancel
				</button>
				<button
					onclick={confirmDelete}
					class="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700"
				>
					Delete
				</button>
			</div>
		</div>
	</div>
{/if}
