<script lang="ts">
	import { onMount } from 'svelte';
	import pb from '$lib/pb';

	let quarantinedCount = $state(0);

	onMount(async () => {
		try {
			const result = await pb.collection('resources').getList(1, 1, {
				filter: 'status="quarantined"'
			});
			quarantinedCount = result.totalItems;
		} catch {
			// Silently ignore — backend may not be ready
		}
	});
</script>

{#if quarantinedCount > 0}
	<div class="rounded-lg border border-red-200 bg-red-50 px-4 py-3">
		<div class="flex items-center gap-2">
			<span class="text-red-600">🔴</span>
			<p class="text-sm text-red-800">
				<strong>{quarantinedCount}</strong> resource{quarantinedCount > 1 ? 's' : ''} quarantined.
				<a href="/resources" class="font-medium text-red-700 underline hover:text-red-900">
					View resources →
				</a>
			</p>
		</div>
	</div>
{/if}
