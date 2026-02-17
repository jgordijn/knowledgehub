<script lang="ts">
	import { page } from '$app/state';

	let { onLogout }: { onLogout?: () => void } = $props();
	let mobileOpen = $state(false);

	function closeMobile() {
		mobileOpen = false;
	}
</script>

<nav class="border-b border-slate-200 bg-white">
	<div class="mx-auto max-w-5xl px-4">
		<div class="flex h-14 items-center justify-between">
			<a href="/" class="text-lg font-bold text-slate-900">KnowledgeHub</a>

			<!-- Desktop nav -->
			<div class="hidden items-center gap-6 md:flex">
				<a
					href="/"
					class="text-sm font-medium {page.url.pathname === '/'
						? 'text-blue-600'
						: 'text-slate-600 hover:text-slate-900'}"
				>
					Feed
				</a>
				<a
					href="/resources"
					class="text-sm font-medium {page.url.pathname === '/resources'
						? 'text-blue-600'
						: 'text-slate-600 hover:text-slate-900'}"
				>
					Resources
				</a>
				<a
					href="/settings"
					class="text-sm font-medium {page.url.pathname === '/settings'
						? 'text-blue-600'
						: 'text-slate-600 hover:text-slate-900'}"
				>
					Settings
				</a>
				{#if onLogout}
					<button
						onclick={onLogout}
						class="text-sm font-medium text-slate-400 hover:text-slate-600"
					>
						Logout
					</button>
				{/if}
			</div>

			<!-- Mobile hamburger -->
			<button
				class="flex h-10 w-10 items-center justify-center rounded-md text-slate-600 hover:bg-slate-100 md:hidden"
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
			<div class="border-t border-slate-200 pb-3 pt-2 md:hidden">
				<a
					href="/"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/'
						? 'bg-blue-50 text-blue-600'
						: 'text-slate-600 hover:bg-slate-50'}"
					onclick={closeMobile}
				>
					Feed
				</a>
				<a
					href="/resources"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/resources'
						? 'bg-blue-50 text-blue-600'
						: 'text-slate-600 hover:bg-slate-50'}"
					onclick={closeMobile}
				>
					Resources
				</a>
				<a
					href="/settings"
					class="block rounded-md px-3 py-2 text-sm font-medium {page.url.pathname === '/settings'
						? 'bg-blue-50 text-blue-600'
						: 'text-slate-600 hover:bg-slate-50'}"
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
						class="block w-full rounded-md px-3 py-2 text-left text-sm font-medium text-slate-400 hover:bg-slate-50"
					>
						Logout
					</button>
				{/if}
			</div>
		{/if}
	</div>
</nav>
