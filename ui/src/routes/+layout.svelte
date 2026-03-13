<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import pb, { refreshAuthSession } from '$lib/pb';
	import { initTheme, getTheme, setTheme } from '$lib/theme';
	import type { ThemeMode } from '$lib/theme';
	import { sidebarData } from '$lib/stores/sidebar';

	let { children } = $props();
	let ready = $state(false);
	let mobileOpen = $state(false);
	let themeMode = $state<ThemeMode>('system');

	// Sidebar store data — reactive
	let sbData = $state({
		unreadCount: 0,
		bookmarkedCount: 0,
		resources: [] as { id: string; name: string }[],
		selectedSources: new Set<string>(),
		sourceCounts: new Map<string, number>(),
		tierCounts: [] as { label: string; id: string; count: number; icon: string }[],
		readFilter: 'unread' as 'unread' | 'all' | 'bookmarked',
		onToggleSource: (_id: string) => {},
		onClearSources: () => {},
		onSetReadFilter: (_f: 'unread' | 'all' | 'bookmarked') => {}
	});

	onMount(() => {
		initTheme();
		themeMode = getTheme();

		const unsubStore = sidebarData.subscribe((data) => {
			sbData = data;
		});

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

		const initialize = async () => {
			await refreshAuthSession();
			checkAuth();
		};
		void initialize();

		const unsub = pb.authStore.onChange(() => {
			checkAuth();
		});

		return () => { unsub(); unsubStore(); };
	});

	$effect(() => {
		page.url.pathname;
		if (page.url.pathname !== '/login' && pb.authStore.isValid) {
			ready = true;
		}
	});

	function handleLogout() {
		pb.authStore.clear();
		goto('/login');
	}

	function closeMobileSidebar() {
		mobileOpen = false;
	}

	function toggleTheme() {
		const modes: ThemeMode[] = ['light', 'dark', 'system'];
		const idx = modes.indexOf(themeMode);
		const next = modes[(idx + 1) % modes.length];
		setTheme(next);
		themeMode = next;
	}

	const themeIcon = $derived(themeMode === 'dark' ? '🌙' : themeMode === 'light' ? '☀️' : '🖥️');
	const themeLabel = $derived(themeMode === 'dark' ? 'Dark' : themeMode === 'light' ? 'Light' : 'System');
</script>

{#if page.url.pathname === '/login'}
	<div class="flex min-h-screen flex-col bg-slate-50 dark:bg-slate-900">
		<main class="mx-auto w-full max-w-5xl flex-1 px-4 py-6">
			{@render children()}
		</main>
	</div>
{:else if ready}
	<div class="flex h-screen overflow-hidden bg-slate-50 dark:bg-slate-900">
		<a href="#main-content" class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:bg-white focus:px-4 focus:py-2 focus:text-sm focus:font-medium focus:text-blue-600 focus:shadow-md">
			Skip to main content
		</a>

		<!-- Desktop sidebar (hidden on mobile) -->
		<div class="hidden w-[220px] flex-shrink-0 md:block">
			<Sidebar
				onLogout={handleLogout}
				unreadCount={sbData.unreadCount}
				bookmarkedCount={sbData.bookmarkedCount}
				resources={sbData.resources}
				selectedSources={sbData.selectedSources}
				sourceCounts={sbData.sourceCounts}
				tierCounts={sbData.tierCounts}
				readFilter={sbData.readFilter}
				onToggleSource={sbData.onToggleSource}
				onClearSources={sbData.onClearSources}
				onSetReadFilter={sbData.onSetReadFilter}
			/>
		</div>

		<!-- Mobile sidebar overlay -->
		{#if mobileOpen}
			<!-- svelte-ignore a11y_no_static_element_interactions a11y_click_events_have_key_events -->
			<div
				class="fixed inset-0 z-[90] bg-black/50 md:hidden"
				onclick={closeMobileSidebar}
			></div>
		{/if}

		<!-- Mobile sidebar drawer -->
		<div
			class="fixed bottom-0 top-0 z-[100] w-[220px] transition-[left] duration-250 ease-in-out md:hidden
				{mobileOpen ? 'left-0' : '-left-[230px]'}"
		>
			<Sidebar
				onLogout={handleLogout}
				onCloseMobile={closeMobileSidebar}
				unreadCount={sbData.unreadCount}
				bookmarkedCount={sbData.bookmarkedCount}
				resources={sbData.resources}
				selectedSources={sbData.selectedSources}
				sourceCounts={sbData.sourceCounts}
				tierCounts={sbData.tierCounts}
				readFilter={sbData.readFilter}
				onToggleSource={sbData.onToggleSource}
				onClearSources={sbData.onClearSources}
				onSetReadFilter={sbData.onSetReadFilter}
			/>
		</div>

		<!-- Main content area -->
		<div class="flex min-w-0 flex-1 flex-col overflow-hidden">
			<!-- Mobile topbar with hamburger + theme toggle -->
			<div class="flex items-center gap-2 border-b border-slate-200 bg-white px-3 py-2 md:hidden dark:border-slate-700 dark:bg-slate-800">
				<button
					onclick={() => (mobileOpen = !mobileOpen)}
					class="flex-shrink-0 text-[22px] text-slate-900 dark:text-slate-50"
					aria-label="Menu"
				>☰</button>
				<div class="flex-1"></div>
				<button
					onclick={toggleTheme}
					class="flex flex-shrink-0 select-none items-center gap-1.5 rounded-lg border border-slate-300 bg-slate-50 px-2.5 py-1 text-[12px] text-slate-500 transition-colors hover:border-amber-500 hover:text-slate-900 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400 dark:hover:border-amber-400 dark:hover:text-slate-50"
					aria-label="Toggle theme"
				>
					<span class="text-[15px] leading-none">{themeIcon}</span>
				</button>
			</div>

			<!-- Desktop topbar with theme toggle -->
			<div class="hidden items-center justify-end gap-2 border-b border-slate-200 bg-white px-5 py-2 md:flex dark:border-slate-700 dark:bg-slate-800">
				<button
					onclick={toggleTheme}
					class="flex flex-shrink-0 select-none items-center gap-1.5 rounded-lg border border-slate-300 bg-slate-50 px-2.5 py-1 text-[12px] text-slate-500 transition-colors hover:border-amber-500 hover:text-slate-900 dark:border-slate-600 dark:bg-slate-900 dark:text-slate-400 dark:hover:border-amber-400 dark:hover:text-slate-50"
					aria-label="Toggle theme"
				>
					<span class="text-[15px] leading-none">{themeIcon}</span>
					<span>{themeLabel}</span>
				</button>
			</div>

			<main id="main-content" class="flex-1 overflow-y-auto px-4 py-6 md:px-6">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
