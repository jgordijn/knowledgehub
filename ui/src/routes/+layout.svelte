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
		// Re-evaluate auth state on every navigation (not just initial mount)
		const checkAuth = () => {
			if (page.url.pathname === '/login') {
				ready = true;
				return;
			}
			if (!pb.authStore.isValid) {
				ready = false;
				goto('/login');
				return;
			}
			ready = true;
		};

		checkAuth();

		// Listen for auth changes (login/logout) to update ready state
		const unsub = pb.authStore.onChange(() => {
			checkAuth();
		});

		return () => unsub();
	});

	// Also react to client-side navigation (goto) after login
	$effect(() => {
		// Track pathname changes
		page.url.pathname;
		if (page.url.pathname !== '/login' && pb.authStore.isValid) {
			ready = true;
		}
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
		<a href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-white focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-blue-600 focus:shadow-md">
			Skip to main content
		</a>
		<Nav onLogout={handleLogout} />
		<main id="main-content" class="mx-auto w-full max-w-5xl flex-1 px-4 py-6">
			{@render children()}
		</main>
	</div>
{/if}
