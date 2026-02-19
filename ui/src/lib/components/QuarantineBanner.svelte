<script lang="ts">
	import { onMount } from 'svelte';
	import pb from '$lib/pb';

	let quarantinedCount = $state(0);
	let dismissed = $state(false);

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
	<div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
		<div class="flex items-center gap-2">
			<span class="text-red-600">🔴</span>
			<p class="flex-1 text-sm text-red-800">
				<strong>{quarantinedCount}</strong> resource{quarantinedCount > 1 ? 's' : ''} quarantined.
				<a href="/resources" class="font-medium text-red-700 underline hover:text-red-900">
					View resources →
				</a>
			</p>
			<button
				onclick={() => (dismissed = true)}
				class="flex h-6 w-6 items-center justify-center rounded text-red-400 hover:bg-red-100 hover:text-red-600"
				aria-label="Dismiss"
			>
				✕
			</button>
		</div>
	</div>
{/if}
