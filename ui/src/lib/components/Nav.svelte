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
					<svg class="h-4 w-4" viewBox="0 0 24 24" fill="currentColor">
						<path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
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
