<script lang="ts">
	import pb from '$lib/pb';
	import ResourceForm from './ResourceForm.svelte';

	let {
		onClose,
		onEntryAdded
	}: {
		onClose: () => void;
		onEntryAdded?: () => void;
	} = $props();

	type State = 'input' | 'loading' | 'result' | 'edit' | 'error';

	interface RSSArticle {
		title: string;
		url: string;
		published_at?: string;
	}

	interface RSSInfo {
		feed_url: string;
		site_name: string;
		articles: RSSArticle[];
	}

	interface QuickAddResult {
		entry: { id: string; title: string; url: string };
		rss?: RSSInfo | null;
		message: string;
	}

	let state = $state<State>('input');
	let url = $state('');
	let result = $state<QuickAddResult | null>(null);
	let errorMessage = $state('');
	let subscribing = $state(false);

	async function handleSubmit() {
		const trimmed = url.trim();
		if (!trimmed) return;

		state = 'loading';
		errorMessage = '';

		try {
			const resp = await pb.send('/api/quick-add', {
				method: 'POST',
				body: JSON.stringify({ url: trimmed }),
				headers: { 'Content-Type': 'application/json' }
			});
			result = resp as QuickAddResult;
			state = 'result';
			onEntryAdded?.();
		} catch (err: unknown) {
			if (err && typeof err === 'object' && 'response' in err) {
				const pbErr = err as { response?: { error?: string; data?: Record<string, unknown> } };
				errorMessage = pbErr.response?.error || 'Failed to add article.';
			} else if (err instanceof Error) {
				errorMessage = err.message;
			} else {
				errorMessage = 'An unexpected error occurred.';
			}
			state = 'error';
		}
	}

	async function handleAddRSS() {
		if (!result?.rss) return;
		subscribing = true;

		try {
			await pb.send('/api/quick-add/subscribe', {
				method: 'POST',
				body: JSON.stringify({
					feed_url: result.rss.feed_url,
					name: result.rss.site_name
				}),
				headers: { 'Content-Type': 'application/json' }
			});
			onClose();
		} catch (err: unknown) {
			errorMessage = err instanceof Error ? err.message : 'Failed to subscribe.';
		} finally {
			subscribing = false;
		}
	}

	function handleEdit() {
		state = 'edit';
	}

	function handleNoThanks() {
		onClose();
	}

	function handleResourceSaved() {
		onClose();
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onClose();
		}
	}

	function formatDate(dateStr?: string): string {
		if (!dateStr) return '';
		try {
			const d = new Date(dateStr);
			return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
		} catch {
			return dateStr;
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Backdrop -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
	onclick={(e) => { if (e.target === e.currentTarget) onClose(); }}
>
	<!-- Modal -->
	<div class="w-full max-w-lg rounded-xl bg-white shadow-2xl dark:bg-slate-800" onclick={(e) => e.stopPropagation()}>
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-slate-200 px-5 py-3 dark:border-slate-700">
			<h2 class="text-lg font-semibold text-slate-900 dark:text-slate-100">Quick Add Article</h2>
			<button
				onclick={onClose}
				class="rounded-md p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300"
				aria-label="Close"
			>
				<svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
					<path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
				</svg>
			</button>
		</div>

		<!-- Body -->
		<div class="px-5 py-4">
			{#if state === 'input' || state === 'error'}
				<form onsubmit={(e) => { e.preventDefault(); handleSubmit(); }} class="space-y-3">
					<div>
						<label for="quick-add-url" class="mb-1 block text-sm font-medium text-slate-700 dark:text-slate-300">
							Article URL
						</label>
						<input
							id="quick-add-url"
							type="url"
							bind:value={url}
							placeholder="https://example.com/interesting-article"
							required
							autofocus
							class="w-full rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
						/>
					</div>
					{#if state === 'error' && errorMessage}
						<div class="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-800 dark:bg-red-900/30 dark:text-red-300">
							{errorMessage}
						</div>
					{/if}
					<div class="flex justify-end">
						<button
							type="submit"
							disabled={!url.trim()}
							class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
						>
							Add Article
						</button>
					</div>
				</form>

			{:else if state === 'loading'}
				<div class="flex flex-col items-center gap-3 py-8">
					<div class="h-8 w-8 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"></div>
					<p class="text-sm text-slate-500 dark:text-slate-400">Fetching article…</p>
				</div>

			{:else if state === 'result' && result}
				<div class="space-y-4">
					<!-- Added article confirmation -->
					<div class="rounded-md border border-green-200 bg-green-50 px-3 py-2 dark:border-green-800 dark:bg-green-900/30">
						<p class="text-sm font-medium text-green-800 dark:text-green-300">✓ Article added</p>
						<p class="mt-0.5 text-sm text-green-700 dark:text-green-400">{result.entry.title}</p>
					</div>

					<!-- RSS Discovery -->
					{#if result.rss}
						<div class="space-y-3">
							<div class="flex items-start gap-2">
								<svg class="mt-0.5 h-5 w-5 shrink-0 text-orange-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
									<path stroke-linecap="round" stroke-linejoin="round" d="M6.503 20.752c0 1.794-1.456 3.248-3.251 3.248-1.796 0-3.252-1.454-3.252-3.248 0-1.794 1.456-3.248 3.252-3.248 1.795 0 3.251 1.454 3.251 3.248zm2.502-1.752h-2.502c0-2.006-.794-3.874-2.249-5.33-1.455-1.455-3.324-2.249-5.254-2.249v-2.5c5.725 0 10.005 4.494 10.005 10.079zm5.003 0h-2.502c0-6.94-5.626-12.579-12.504-12.579v-2.5c8.312 0 15.006 6.793 15.006 15.079z" />
								</svg>
								<div>
									<p class="text-sm font-medium text-slate-900 dark:text-slate-100">RSS feed found!</p>
									<a
										href={result.rss.feed_url}
										target="_blank"
										rel="noopener noreferrer"
										class="text-xs text-blue-600 hover:underline break-all dark:text-blue-400"
									>
										{result.rss.feed_url}
									</a>
								</div>
							</div>

							<!-- Article previews -->
							{#if result.rss.articles.length > 0}
								<div class="rounded-md border border-slate-200 dark:border-slate-700">
									<p class="border-b border-slate-200 px-3 py-1.5 text-xs font-medium text-slate-500 dark:border-slate-700 dark:text-slate-400">
										Recent articles
									</p>
									<ul class="divide-y divide-slate-100 dark:divide-slate-700">
										{#each result.rss.articles as article}
											<li class="px-3 py-2">
												<a
													href={article.url}
													target="_blank"
													rel="noopener noreferrer"
													class="text-sm text-blue-600 hover:underline dark:text-blue-400 line-clamp-1"
												>
													{article.title}
												</a>
												{#if article.published_at}
													<span class="ml-2 text-xs text-slate-400 dark:text-slate-500">
														{formatDate(article.published_at)}
													</span>
												{/if}
											</li>
										{/each}
									</ul>
								</div>
							{/if}

							<!-- Action buttons -->
							<p class="text-sm text-slate-600 dark:text-slate-400">
								Subscribe to this RSS feed?
							</p>
							<div class="flex flex-wrap gap-2">
								<button
									onclick={handleAddRSS}
									disabled={subscribing}
									class="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
								>
									{subscribing ? 'Adding…' : 'Add RSS'}
								</button>
								<button
									onclick={handleEdit}
									class="rounded-md border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-slate-600 dark:text-slate-300 dark:hover:bg-slate-700"
								>
									Edit
								</button>
								<button
									onclick={handleNoThanks}
									class="rounded-md px-4 py-2 text-sm font-medium text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300"
								>
									No thanks
								</button>
							</div>
						</div>
					{:else}
						<!-- No RSS found — just show close button -->
						<div class="flex justify-end pt-2">
							<button
								onclick={onClose}
								class="rounded-md bg-slate-100 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-200 dark:bg-slate-700 dark:text-slate-300 dark:hover:bg-slate-600"
							>
								Done
							</button>
						</div>
					{/if}
				</div>

			{:else if state === 'edit' && result?.rss}
				<div class="space-y-3">
					<p class="text-sm text-slate-600 dark:text-slate-400">
						Customize the RSS resource before adding:
					</p>
					<ResourceForm
						initialName={result.rss.site_name}
						initialUrl={result.rss.feed_url}
						initialType="rss"
						onSave={handleResourceSaved}
						onCancel={() => { state = 'result'; }}
					/>
				</div>
			{/if}
		</div>
	</div>
</div>
