<script lang="ts">
	import { goto } from '$app/navigation';
	import pb from '$lib/pb';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin() {
		if (!email.trim() || !password.trim()) {
			error = 'Email and password are required.';
			return;
		}
		loading = true;
		error = '';
		try {
			await pb.collection('_superusers').authWithPassword(email.trim(), password.trim());
			goto('/');
		} catch {
			error = 'Invalid email or password.';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<div class="w-full max-w-sm space-y-6">
		<div class="text-center">
			<h1 class="text-2xl font-bold text-slate-900">KnowledgeHub</h1>
			<p class="mt-1 text-sm text-slate-500">Sign in to continue</p>
		</div>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleLogin();
			}}
			class="space-y-4 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
		>
			{#if error}
				<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
					{error}
				</div>
			{/if}

			<div>
				<label for="email" class="mb-1 block text-sm font-medium text-slate-700">Email</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					placeholder="you@example.com"
					class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				/>
			</div>

			<div>
				<label for="password" class="mb-1 block text-sm font-medium text-slate-700">
					Password
				</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					placeholder="••••••••"
					class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none"
				/>
			</div>

			<button
				type="submit"
				disabled={loading}
				class="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
			>
				{loading ? 'Signing in...' : 'Sign In'}
			</button>
		</form>

		<p class="text-center text-xs text-slate-400">
			Create your account first with:<br />
			<code class="rounded bg-slate-100 px-1.5 py-0.5 font-mono text-slate-600">
				./knowledgehub superuser upsert EMAIL PASS
			</code>
		</p>
	</div>
</div>
