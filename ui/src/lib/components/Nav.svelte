<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	let { onLogout }: { onLogout?: () => void } = $props();
	let mobileOpen = $state(false);
	let version = $state('');

	onMount(async () => {
		try {
			const res = await fetch('/api/version');
			const data = await res.json();
			version = data.version ?? '';
		} catch {
			// ignore
		}
	});

	function closeMobile() {
		mobileOpen = false;
	}
</script>

<nav class="border-b border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800">
	<div class="mx-auto max-w-5xl px-4">
		<div class="flex h-14 items-center justify-between">
			<div class="flex items-baseline gap-2">
				<a href="/" class="text-lg font-bold text-slate-900 dark:text-slate-100">KnowledgeHub</a>
				{#if version}
					<span class="text-xs text-slate-400 dark:text-slate-500">{version}</span>
				{/if}
			</div>

			<!-- Desktop nav -->
			<div class="hidden items-center gap-6 md:flex">
				<a
					href="/"
					class="text-sm font-medium {page.url.pathname === '/'
						? 'text-blue-600 dark:text-blue-400'
						: 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
				>
					Feed
				</a>
				<a
					href="/resources"
					class="text-sm font-medium {page.url.pathname === '/resources'
						? 'text-blue-600 dark:text-blue-400'
						: 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
				>
					Resources
				</a>
				<a
					href="/settings"
					class="text-sm font-medium {page.url.pathname === '/settings'
						? 'text-blue-600 dark:text-blue-400'
						: 'text-slate-600 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'}"
				>
					Settings
				</a>
				{#if onLogout}
					<button
						onclick={onLogout}
						class="text-sm font-medium text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
					>
						Logout
					</button>
				{/if}
			</div>

			<!-- Mobile hamburger -->
			<button
				class="flex h-10 w-10 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 md:hidden dark:text-slate-400 dark:hover:bg-slate-700"
				onclick={() => (mobileOpen = !mobileOpen)}
				aria-label="Toggle menu"
			>
				{#if mobileOpen}
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M6 18L18 6M6 6l12 12"
						/>
					</svg>
				{:else}
					<svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path
							stroke-linecap="round"
							stroke-linejoin="round"
							stroke-width="2"
							d="M4 6h16M4 12h16M4 18h16"
						/>
					</svg>
				{/if}
			</button>
		</div>

		<!-- Mobile menu -->
		{#if mobileOpen}
			<div class="border-t border-slate-200 pb-3 pt-2 md:hidden dark:border-slate-700">
				<a
					href="/"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/'
						? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
						: 'text-slate-600 hover:bg-slate-50 dark:text-slate-400 dark:hover:bg-slate-700'}"
					onclick={closeMobile}
				>
					Feed
				</a>
				<a
					href="/resources"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/resources'
						? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
						: 'text-slate-600 hover:bg-slate-50 dark:text-slate-400 dark:hover:bg-slate-700'}"
					onclick={closeMobile}
				>
					Resources
				</a>
				<a
					href="/settings"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/settings'
						? 'bg-blue-50 text-blue-600 dark:bg-blue-900/30 dark:text-blue-400'
						: 'text-slate-600 hover:bg-slate-50 dark:text-slate-400 dark:hover:bg-slate-700'}"
					onclick={closeMobile}
				>
					Settings
				</a>
				{#if onLogout}
					<button
						onclick={() => {
							closeMobile();
							onLogout?.();
						}}
						class="block w-full rounded-md px-3 py-2 text-left text-sm font-medium text-slate-400 hover:bg-slate-50 dark:text-slate-500 dark:hover:bg-slate-700"
					>
						Logout
					</button>
				{/if}
			</div>
		{/if}
	</div>
</nav>
