<script lang="ts">
	import { onMount } from 'svelte';
	import pb from '$lib/pb';

	let apiKey = $state('');
	let model = $state('anthropic/claude-sonnet-4');
	let saving = $state(false);
	let saved = $state(false);
	let error = $state('');
	let loading = $state(true);

	// Track record IDs so we can upsert
	let apiKeyRecordId = $state('');
	let modelRecordId = $state('');

	async function loadSettings() {
		loading = true;
		try {
			const result = await pb.collection('app_settings').getList(1, 50);
			for (const record of result.items) {
				if (record.key === 'openrouter_api_key') {
					apiKey = record.value ?? '';
					apiKeyRecordId = record.id;
				} else if (record.key === 'openrouter_model') {
					model = record.value ?? 'anthropic/claude-sonnet-4';
					modelRecordId = record.id;
				}
			}
		} catch {
			// Backend may not be ready
		} finally {
			loading = false;
		}
	}

	async function upsertSetting(key: string, value: string, recordId: string): Promise<string> {
		if (recordId) {
			await pb.collection('app_settings').update(recordId, { value });
			return recordId;
		} else {
			const record = await pb.collection('app_settings').create({ key, value });
			return record.id;
		}
	}

	async function handleSave() {
		saving = true;
		saved = false;
		error = '';
		try {
			apiKeyRecordId = await upsertSetting('openrouter_api_key', apiKey, apiKeyRecordId);
			modelRecordId = await upsertSetting('openrouter_model', model, modelRecordId);
			saved = true;
			setTimeout(() => (saved = false), 3000);
		} catch (err: unknown) {
			error = err instanceof Error ? err.message : 'Failed to save settings.';
		} finally {
			saving = false;
		}
	}

	onMount(loadSettings);
</script>

<div class="mx-auto max-w-lg space-y-6">
	<h1 class="text-xl font-bold text-slate-900">Settings</h1>

	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400">Loading settings…</div>
	{:else}
		<div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm">
			<h2 class="mb-4 text-sm font-semibold text-slate-700">AI Configuration</h2>

			<div class="space-y-4">
				<div>
					<label for="api-key" class="mb-1 block text-sm font-medium text-slate-700">
						OpenRouter API Key
					</label>
					<input
						id="api-key"
						type="password"
						bind:value={apiKey}
						placeholder="sk-or-v1-..."
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
					<p class="mt-1 text-xs text-slate-500">
						Get your key from
						<a
							href="https://openrouter.ai/keys"
							target="_blank"
							rel="noopener"
							class="text-blue-600 hover:underline"
						>
							openrouter.ai/keys
						</a>
					</p>
				</div>

				<div>
					<label for="model" class="mb-1 block text-sm font-medium text-slate-700">
						Model
					</label>
					<input
						id="model"
						type="text"
						bind:value={model}
						placeholder="anthropic/claude-sonnet-4"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
					/>
					<p class="mt-1 text-xs text-slate-500">
						OpenRouter model ID (e.g., anthropic/claude-sonnet-4, openai/gpt-4o)
					</p>
				</div>

				{#if error}
					<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
						{error}
					</div>
				{/if}

				{#if saved}
					<div
						class="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700"
					>
						Settings saved successfully.
					</div>
				{/if}

				<button
					onclick={handleSave}
					disabled={saving}
					class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
				>
					{saving ? 'Saving...' : 'Save Settings'}
				</button>
			</div>
		</div>
	{/if}
</div>
