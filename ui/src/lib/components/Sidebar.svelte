<script lang="ts">
	import { page } from '$app/state';
	import { onMount } from 'svelte';

	let {
		onLogout,
		unreadCount = 0,
		bookmarkedCount = 0,
		resources = [],
		selectedSources = new Set<string>(),
		sourceCounts = new Map<string, number>(),
		onToggleSource,
		onClearSources,
		onCloseMobile
	}: {
		onLogout?: () => void;
		unreadCount?: number;
		bookmarkedCount?: number;
		resources?: { id: string; name: string }[];
		selectedSources?: Set<string>;
		sourceCounts?: Map<string, number>;
		onToggleSource?: (id: string) => void;
		onClearSources?: () => void;
		onCloseMobile?: () => void;
	} = $props();

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

	const avatarColors = [
		'#f97316', '#8b5cf6', '#06b6d4', '#f472b6', '#34d399',
		'#a78bfa', '#fb923c', '#38bdf8', '#f87171', '#4ade80'
	];

	function getAvatarColor(index: number): string {
		return avatarColors[index % avatarColors.length];
	}

	function getInitials(name: string): string {
		const words = name.trim().split(/\s+/);
		if (words.length >= 2) {
			return (words[0][0] + words[1][0]).toUpperCase();
		}
		return name.slice(0, 2).toUpperCase();
	}

	function handleNavClick() {
		onCloseMobile?.();
	}
</script>

<aside class="flex h-full flex-col border-r border-slate-200 bg-white dark:border-slate-700 dark:bg-slate-800">
	<!-- Logo area -->
	<div class="flex items-center gap-2 px-4 pb-3 pt-5">
		<div>
			<span class="text-[17px] font-bold text-slate-900 dark:text-slate-50">Knowledge<span class="text-amber-500 dark:text-amber-400">Hub</span></span>
			{#if version}
				<span class="ml-1 text-[11px] text-slate-400 dark:text-slate-500">{version}</span>
			{/if}
		</div>
		<a
			href="https://github.com/jgordijn/knowledgehub"
			target="_blank"
			rel="noopener noreferrer"
			class="ml-auto inline-flex flex-shrink-0 items-center text-slate-400 transition-colors hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
			aria-label="GitHub repository"
			title="View on GitHub"
		>
			<svg class="h-[15px] w-[15px]" viewBox="0 0 24 24" fill="currentColor">
				<path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12"/>
			</svg>
		</a>
	</div>

	<!-- Nav links -->
	<nav class="px-2">
		<a
			href="/"
			onclick={handleNavClick}
			class="mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] transition-colors
				{page.url.pathname === '/'
				? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-700 dark:text-slate-50'
				: 'text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-100'}"
		>
			<span>📡</span>Feed
			{#if unreadCount > 0}
				<span class="ml-auto rounded-full bg-amber-500 px-[7px] py-px text-[11px] font-bold text-slate-900">{unreadCount}</span>
			{/if}
		</a>
		<a
			href="/"
			onclick={(e) => { handleNavClick(); }}
			data-tab="bookmarked"
			class="mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-100 transition-colors"
		>
			<span>📌</span>Saved
			{#if bookmarkedCount > 0}
				<span class="ml-auto rounded-full bg-slate-200 px-[7px] py-px text-[11px] font-bold text-slate-500 dark:bg-slate-600 dark:text-slate-400">{bookmarkedCount}</span>
			{/if}
		</a>

		<div class="mx-2 my-2.5 h-px bg-slate-200 dark:bg-slate-700"></div>

		<a
			href="/resources"
			onclick={handleNavClick}
			class="mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] transition-colors
				{page.url.pathname === '/resources'
				? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-700 dark:text-slate-50'
				: 'text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-100'}"
		>
			<span>📦</span>Sources
		</a>
		<a
			href="/settings"
			onclick={handleNavClick}
			class="mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-[13px] transition-colors
				{page.url.pathname === '/settings'
				? 'bg-slate-100 font-medium text-slate-900 dark:bg-slate-700 dark:text-slate-50'
				: 'text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-100'}"
		>
			<span>⚙️</span>Settings
		</a>

		{#if onLogout}
			<button
				onclick={() => { handleNavClick(); onLogout?.(); }}
				class="mb-0.5 flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] text-slate-400 transition-colors hover:bg-slate-50 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700/50 dark:hover:text-slate-300"
			>
				Logout
			</button>
		{/if}
	</nav>

	<!-- Source filter section -->
	{#if resources.length > 0}
		<div class="mx-2 my-2.5 h-px bg-slate-200 dark:bg-slate-700"></div>

		<div class="flex-1 overflow-y-auto px-2 pb-2">
			<div class="flex items-center justify-between px-3 pb-1 pt-1.5">
				<span class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-slate-500">Filter by Source</span>
				{#if selectedSources.size > 0}
					<button
						onclick={() => onClearSources?.()}
						class="border-none bg-transparent p-0 text-[10px] font-medium text-blue-500 hover:underline dark:text-blue-400"
					>
						Clear
					</button>
				{/if}
			</div>
			<ul class="list-none">
				{#each resources as res, i}
					{@const isActive = selectedSources.has(res.id)}
					{@const count = sourceCounts.get(res.id) ?? 0}
					<li>
						<button
							onclick={() => onToggleSource?.(res.id)}
							class="mb-px flex w-full select-none items-center gap-2 rounded-md border border-transparent px-3 py-1.5 text-left text-[12px] transition-colors
								{isActive
								? 'border-blue-500/30 bg-blue-50 font-semibold text-slate-900 dark:border-blue-500/30 dark:bg-blue-900/20 dark:text-slate-50'
								: 'text-slate-500 hover:bg-slate-50 hover:text-slate-900 dark:text-slate-400 dark:hover:bg-slate-700/50 dark:hover:text-slate-50'}"
						>
							<span
								class="inline-flex h-[14px] w-[14px] flex-shrink-0 items-center justify-center rounded-sm border-[1.5px] text-[9px]
									{isActive
									? 'border-blue-500 bg-blue-500 text-white'
									: 'border-slate-300 text-transparent dark:border-slate-600'}"
							>✓</span>
							<span
								class="inline-flex h-4 w-4 flex-shrink-0 items-center justify-center rounded-sm text-[8px] font-bold text-slate-900"
								style="background: {getAvatarColor(i)}"
							>{getInitials(res.name)}</span>
							<span class="flex-1 truncate">{res.name}</span>
							<span class="ml-auto text-[10px] font-normal text-slate-400 dark:text-slate-500">{count}</span>
						</button>
					</li>
				{/each}
			</ul>
		</div>
	{/if}

	<!-- Footer -->
	<div class="mt-auto border-t border-slate-200 px-3.5 py-3 text-[11px] leading-relaxed text-slate-400 dark:border-slate-700 dark:text-slate-500">
		<b>?</b> shortcuts · <b>j/k</b> navigate · <b>o</b> open
	</div>
</aside>
