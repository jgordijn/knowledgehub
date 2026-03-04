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
			<div class="flex items-center gap-2">
				<a href="/" class="text-lg font-bold text-slate-900 dark:text-slate-100">KnowledgeHub</a>
				{#if version}
					<span class="text-xs text-slate-400 dark:text-slate-500">{version}</span>
				{/if}
				<a
					href="https://github.com/jgordijn/knowledgehub"
					target="_blank"
					rel="noopener noreferrer"
					class="text-slate-400 hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
					aria-label="GitHub repository"
				>
					<svg class="h-4 w-4" viewBox="0 0 16 16" fill="currentColor">
						<path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
					</svg>
				</a>
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
