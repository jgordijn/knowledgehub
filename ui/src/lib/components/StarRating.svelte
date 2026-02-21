<script lang="ts">
	let {
		aiStars = 0,
		userStars = 0,
		onRate
	}: {
		aiStars?: number;
		userStars?: number;
		onRate?: (stars: number) => void;
	} = $props();

	let hovered = $state(0);

	let effectiveStars = $derived(userStars || aiStars || 0);
	let isUserRated = $derived(userStars > 0);
</script>

<div class="flex items-center gap-0.5" role="group" aria-label="Star rating">
	{#each [1, 2, 3, 4, 5] as star}
		<button
			class="h-6 w-6 min-w-[24px] text-lg leading-none transition-colors
				focus:outline-none focus:ring-2 focus:ring-amber-400 focus:ring-offset-1 rounded
				{hovered >= star
				? 'text-amber-400'
				: (hovered > 0 ? 0 : effectiveStars) >= star
					? isUserRated
						? 'text-amber-400'
						: 'text-amber-400/70'
					: 'text-slate-200 dark:text-slate-600'}"
			onmouseenter={() => (hovered = star)}
			onmouseleave={() => (hovered = 0)}
			onclick={() => onRate?.(star)}
			aria-label="Rate {star} star{star > 1 ? 's' : ''}"
		>
			{#if (hovered >= star) || ((hovered > 0 ? 0 : effectiveStars) >= star)}
				★
			{:else}
				☆
			{/if}
		</button>
	{/each}
</div>
