<script lang="ts">
	import { onMount } from 'svelte';
	import pb from '$lib/pb';

	const DISMISS_KEY = 'quarantine-banner-dismissed';

	let quarantinedCount = $state(0);
	let dismissed = $state(typeof sessionStorage !== 'undefined' && sessionStorage.getItem(DISMISS_KEY) === 'true');

	onMount(async () => {
		try {
			const result = await pb.collection('resources').getList(1, 1, {
				filter: 'status="quarantined"',
				requestKey: 'quarantineCheck'
			});
			quarantinedCount = result.totalItems;
		} catch {
			// Silently ignore — backend may not be ready
		}
	});
</script>

{#if quarantinedCount > 0 && !dismissed}
	<div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 dark:border-red-800 dark:bg-red-900/30">
		<div class="flex items-center gap-2">
			<span class="text-red-600 dark:text-red-400">🔴</span>
			<p class="flex-1 text-sm text-red-800 dark:text-red-300">
				<strong>{quarantinedCount}</strong> resource{quarantinedCount > 1 ? 's' : ''} quarantined.
				<a href="/resources" class="font-medium text-red-700 underline hover:text-red-900 dark:text-red-400 dark:hover:text-red-200">
					View resources →
				</a>
			</p>
			<button
				onclick={() => { dismissed = true; sessionStorage.setItem(DISMISS_KEY, 'true'); }}
				class="flex h-6 w-6 items-center justify-center rounded text-red-400 hover:bg-red-100 hover:text-red-600 dark:text-red-500 dark:hover:bg-red-900/50 dark:hover:text-red-300"
				aria-label="Dismiss"
			>
				✕
			</button>
		</div>
	</div>
{/if}
