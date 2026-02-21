<script lang="ts">
	import { onMount } from 'svelte';
	import pb from '$lib/pb';
	import { getTheme, setTheme, type ThemeMode } from '$lib/theme';

	let apiKey = $state('');
	let model = $state('anthropic/claude-sonnet-4');
	let saving = $state(false);
	let saved = $state(false);
	let error = $state('');
	let loading = $state(true);

	// Track record IDs so we can upsert
	let apiKeyRecordId = $state('');
	let modelRecordId = $state('');

	// Theme
	let themeMode = $state<ThemeMode>('system');

	// Password change
	let oldPassword = $state('');
	let newPassword = $state('');
	let confirmPassword = $state('');
	let pwSaving = $state(false);
	let pwSaved = $state(false);
	let pwError = $state('');

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

	function handleThemeChange(mode: ThemeMode) {
		themeMode = mode;
		setTheme(mode);
	}

	async function handlePasswordChange() {
		pwError = '';
		pwSaved = false;

		if (!oldPassword || !newPassword || !confirmPassword) {
			pwError = 'All fields are required.';
			return;
		}
		if (newPassword.length < 8) {
			pwError = 'New password must be at least 8 characters.';
			return;
		}
		if (newPassword !== confirmPassword) {
			pwError = 'New passwords do not match.';
			return;
		}

		pwSaving = true;
		try {
			const userId = pb.authStore.record?.id;
			if (!userId) throw new Error('Not authenticated.');
			await pb.collection('_superusers').update(userId, {
				oldPassword,
				password: newPassword,
				passwordConfirm: confirmPassword
			});
			oldPassword = '';
			newPassword = '';
			confirmPassword = '';
			pwSaved = true;
			setTimeout(() => (pwSaved = false), 3000);
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'response' in err) {
				const resp = (err as { response?: { message?: string } }).response;
				pwError = resp?.message || 'Failed to change password.';
			} else {
				pwError = err instanceof Error ? err.message : 'Failed to change password.';
			}
		} finally {
			pwSaving = false;
		}
	}

	onMount(() => {
		themeMode = getTheme();
		loadSettings();
	});
</script>

<div class="mx-auto max-w-lg space-y-6">
	<h1 class="text-xl font-bold text-slate-900 dark:text-slate-100">Settings</h1>

	{#if loading}
		<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">Loading settings…</div>
	{:else}
		<!-- AI Configuration -->
		<div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800">
			<h2 class="mb-4 text-sm font-semibold text-slate-700 dark:text-slate-300">AI Configuration</h2>

			<div class="space-y-4">
				<div>
					<label for="api-key" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						OpenRouter API Key
					</label>
					<input
						id="api-key"
						type="password"
						bind:value={apiKey}
						placeholder="sk-or-v1-..."
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm font-mono focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
					<p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
						Get your key from
						<a
							href="https://openrouter.ai/keys"
							target="_blank"
							rel="noopener"
							class="text-blue-600 hover:underline dark:text-blue-400"
						>
							openrouter.ai/keys
						</a>
					</p>
				</div>

				<div>
					<label for="model" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						Model
					</label>
					<input
						id="model"
						type="text"
						bind:value={model}
						placeholder="anthropic/claude-sonnet-4"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
					<p class="mt-1 text-xs text-slate-500 dark:text-slate-400">
						OpenRouter model ID (e.g., anthropic/claude-sonnet-4, openai/gpt-4o)
					</p>
				</div>

				{#if error}
					<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300">
						{error}
					</div>
				{/if}

				{#if saved}
					<div
						class="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700 dark:border-green-800 dark:bg-green-900/30 dark:text-green-300"
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

		<!-- Appearance -->
		<div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800">
			<h2 class="mb-4 text-sm font-semibold text-slate-700 dark:text-slate-300">Appearance</h2>

			<div class="flex gap-1 rounded-lg bg-slate-100 p-1 dark:bg-slate-700">
				{#each [
					{ label: '☀️ Light', value: 'light' },
					{ label: '🌙 Dark', value: 'dark' },
					{ label: '💻 System', value: 'system' }
				] as opt}
					<button
						onclick={() => handleThemeChange(opt.value as ThemeMode)}
						class="flex-1 rounded-md px-3 py-1.5 text-sm font-medium transition-colors
							{themeMode === opt.value
							? 'bg-white text-slate-900 shadow-sm dark:bg-slate-600 dark:text-slate-100'
							: 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
					>
						{opt.label}
					</button>
				{/each}
			</div>
		</div>

		<!-- Change Password -->
		<div class="rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800">
			<h2 class="mb-4 text-sm font-semibold text-slate-700 dark:text-slate-300">Change Password</h2>

			<div class="space-y-4">
				<div>
					<label for="old-password" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						Current Password
					</label>
					<input
						id="old-password"
						type="password"
						bind:value={oldPassword}
						placeholder="••••••••"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
				</div>

				<div>
					<label for="new-password" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						New Password
					</label>
					<input
						id="new-password"
						type="password"
						bind:value={newPassword}
						placeholder="Min. 8 characters"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
				</div>

				<div>
					<label for="confirm-password" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						Confirm New Password
					</label>
					<input
						id="confirm-password"
						type="password"
						bind:value={confirmPassword}
						placeholder="••••••••"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
				</div>

				{#if pwError}
					<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300">
						{pwError}
					</div>
				{/if}

				{#if pwSaved}
					<div
						class="rounded-md border border-green-200 bg-green-50 px-3 py-2 text-sm text-green-700 dark:border-green-800 dark:bg-green-900/30 dark:text-green-300"
					>
						Password changed successfully.
					</div>
				{/if}

				<button
					onclick={handlePasswordChange}
					disabled={pwSaving}
					class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
				>
					{pwSaving ? 'Changing...' : 'Change Password'}
				</button>
			</div>
		</div>
	{/if}
</div>
