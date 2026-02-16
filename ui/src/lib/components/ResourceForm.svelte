<script lang="ts">
	import pb from '$lib/pb';
	import type { RecordModel } from 'pocketbase';

	let {
		initialName = '',
		initialUrl = '',
		initialType = 'rss',
		initialArticleSelector = '',
		initialContentSelector = '',
		resourceId = '',
		onSave,
		onCancel
	}: {
		initialName?: string;
		initialUrl?: string;
		initialType?: string;
		initialArticleSelector?: string;
		initialContentSelector?: string;
		resourceId?: string;
		onSave: () => void;
		onCancel?: () => void;
	} = $props();

	let name = $state(initialName);
	let url = $state(initialUrl);
	let type = $state<string>(initialType);
	let articleSelector = $state(initialArticleSelector);
	let contentSelector = $state(initialContentSelector);
	let saving = $state(false);
	let error = $state('');

	let isEdit = $derived(!!resourceId);

	async function handleSubmit() {
		if (!name.trim() || !url.trim()) {
			error = 'Name and URL are required.';
			return;
		}
		saving = true;
		error = '';
		try {
			const data: Record<string, unknown> = {
				name: name.trim(),
				url: url.trim(),
				type,
				article_selector: type === 'watchlist' ? articleSelector.trim() : '',
				content_selector: type === 'watchlist' ? contentSelector.trim() : ''
			};

			if (isEdit) {
				await pb.collection('resources').update(resourceId, data);
			} else {
				await pb.collection('resources').create({
					...data,
					status: 'healthy',
					consecutive_failures: 0,
					active: true,
					check_interval: 30
				});
			}

			// Reset form if adding
			if (!isEdit) {
				name = '';
				url = '';
				type = 'rss';
				articleSelector = '';
				contentSelector = '';
			}

			onSave();
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to save resource.';
		} finally {
			saving = false;
		}
	}
</script>

<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-4">
	{#if error}
		<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
			{error}
		</div>
	{/if}

	<div class="flex flex-col gap-4 md:flex-row">
		<div class="flex-1">
			<label for="res-name" class="mb-1 block text-sm font-medium text-slate-700">Name</label>
			<input
				id="res-name"
				type="text"
				bind:value={name}
				placeholder="My RSS Feed"
				class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
			/>
		</div>
		<div class="flex-1">
			<label for="res-url" class="mb-1 block text-sm font-medium text-slate-700">URL</label>
			<input
				id="res-url"
				type="url"
				bind:value={url}
				placeholder="https://example.com/feed.xml"
				class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
			/>
		</div>
		<div class="w-full md:w-36">
			<label for="res-type" class="mb-1 block text-sm font-medium text-slate-700">Type</label>
			<select
				id="res-type"
				bind:value={type}
				class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
			>
				<option value="rss">RSS</option>
				<option value="watchlist">Watchlist</option>
			</select>
		</div>
	</div>

	{#if type === 'watchlist'}
		<div class="flex flex-col gap-4 md:flex-row">
			<div class="flex-1">
				<label for="res-article-sel" class="mb-1 block text-sm font-medium text-slate-700">
					Article Selector
				</label>
				<input
					id="res-article-sel"
					type="text"
					bind:value={articleSelector}
					placeholder="CSS selector for articles"
					class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				/>
			</div>
			<div class="flex-1">
				<label for="res-content-sel" class="mb-1 block text-sm font-medium text-slate-700">
					Content Selector
				</label>
				<input
					id="res-content-sel"
					type="text"
					bind:value={contentSelector}
					placeholder="CSS selector for content"
					class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				/>
			</div>
		</div>
	{/if}

	<div class="flex items-center gap-2">
		<button
			type="submit"
			disabled={saving}
			class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
		>
			{saving ? 'Saving...' : isEdit ? 'Update Resource' : 'Add Resource'}
		</button>
		{#if onCancel}
			<button
				type="button"
				onclick={onCancel}
				class="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50"
			>
				Cancel
			</button>
		{/if}
	</div>
</form>
