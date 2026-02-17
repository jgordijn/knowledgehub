<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import Nav from '$lib/components/Nav.svelte';
	import pb from '$lib/pb';

	let { children } = $props();
	let ready = $state(false);

	onMount(() => {
		// Allow login page without auth
		if (page.url.pathname === '/login') {
			ready = true;
			return;
		}

		// Redirect to login if not authenticated
		if (!pb.authStore.isValid) {
			goto('/login');
			return;
		}

		ready = true;
	});

	function handleLogout() {
		pb.authStore.clear();
		goto('/login');
	}
</script>

{#if page.url.pathname === '/login'}
	<div class="flex min-h-screen flex-col bg-slate-50">
		<main class="mx-auto w-full max-w-5xl flex-1 px-4 py-6">
			{@render children()}
		</main>
	</div>
{:else if ready}
	<div class="flex min-h-screen flex-col bg-slate-50">
		<Nav onLogout={handleLogout} />
		<main class="mx-auto w-full max-w-5xl flex-1 px-4 py-6">
			{@render children()}
		</main>
	</div>
{/if}
