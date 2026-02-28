<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import pb from '$lib/pb';
	import { getRememberMe, setRememberMe, switchStorageBackend } from '$lib/auth-store';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);
	let needsSetup = $state(false);
	let checking = $state(true);
	let setupDone = $state(false);
	let rememberMe = $state(getRememberMe());

	onMount(async () => {
		// Already logged in? Go to feed.
		if (pb.authStore.isValid) {
			goto('/');
			return;
		}

		try {
			const res = await fetch('/api/setup-status');
			const data = await res.json();
			needsSetup = data.needsSetup === true;
		} catch {
			// Assume login mode if check fails
		}
		checking = false;
	});

	async function handleSetup() {
		if (!email.trim() || !password.trim()) {
			error = 'Email and password are required.';
			return;
		}
		if (password.length < 8) {
			error = 'Password must be at least 8 characters.';
			return;
		}
		loading = true;
		error = '';
		try {
			const res = await fetch('/api/setup', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email: email.trim(), password })
			});
			const data = await res.json();
			if (!res.ok) {
				error = data.error || 'Setup failed.';
				return;
			}
			setupDone = true;
			needsSetup = false;
		} catch {
			error = 'Setup failed. Check the server logs.';
		} finally {
			loading = false;
		}
	}

	async function handleLogin() {
		if (!email.trim() || !password.trim()) {
			error = 'Email and password are required.';
			return;
		}
		loading = true;
		error = '';
		setRememberMe(rememberMe);
		try {
			await pb.collection('_superusers').authWithPassword(email.trim(), password.trim());
			switchStorageBackend();
			goto('/');
		} catch {
			error = 'Invalid email or password.';
		} finally {
			loading = false;
		}
	}
</script>

{#if checking}
	<div class="flex min-h-[60vh] items-center justify-center">
		<p class="text-sm text-slate-400 dark:text-slate-500">Loading…</p>
	</div>
{:else}
	<div class="flex min-h-[60vh] items-center justify-center">
		<div class="w-full max-w-sm space-y-6">
			<div class="text-center">
				<h1 class="text-2xl font-bold text-slate-900 dark:text-slate-100">KnowledgeHub</h1>
				{#if needsSetup}
					<p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Create your account to get started</p>
				{:else}
					<p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Sign in to continue</p>
				{/if}
			</div>

			{#if setupDone}
				<div
					class="rounded-md border border-green-200 bg-green-50 px-4 py-3 text-sm text-green-700 dark:border-green-800 dark:bg-green-900/30 dark:text-green-300"
				>
					Account created! Sign in below.
				</div>
			{/if}

			<form
				onsubmit={(e) => {
					e.preventDefault();
					needsSetup ? handleSetup() : handleLogin();
				}}
				class="space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-800"
			>
				{#if error}
					<div
						class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300"
					>
						{error}
					</div>
				{/if}

				<div>
					<label for="email" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						Email
					</label>
					<input
						id="email"
						type="email"
						bind:value={email}
						placeholder="you@example.com"
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
				</div>

				<div>
					<label for="password" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
						Password
					</label>
					<input
						id="password"
						type="password"
						bind:value={password}
						placeholder={needsSetup ? 'Min. 8 characters' : '••••••••'}
						class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
					/>
				</div>

				{#if !needsSetup}
					<label class="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
						<input
							type="checkbox"
							bind:checked={rememberMe}
							class="rounded border-slate-300 text-blue-600 focus:ring-blue-500 dark:border-slate-600 dark:bg-slate-700"
						/>
						Remember me
					</label>
				{/if}

				<button
					type="submit"
					disabled={loading}
					class="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
				>
					{#if loading}
						{needsSetup ? 'Creating account…' : 'Signing in…'}
					{:else}
						{needsSetup ? 'Create Account' : 'Sign In'}
					{/if}
				</button>
			</form>
		</div>
	</div>
{/if}
