<script lang="ts">
	import { onMount } from 'svelte';

	let {
		entryId,
		entryTitle,
		onClose
	}: {
		entryId: string;
		entryTitle: string;
		onClose: () => void;
	} = $props();

	interface ChatMessage {
		role: 'user' | 'assistant';
		content: string;
	}

	let messages = $state<ChatMessage[]>([]);
	let input = $state('');
	let streaming = $state(false);
	let messagesEl: HTMLDivElement | undefined = $state();

	function scrollToBottom() {
		if (messagesEl) {
			messagesEl.scrollTop = messagesEl.scrollHeight;
		}
	}

	$effect(() => {
		// Scroll when messages change
		messages.length;
		// Use tick-like delay
		setTimeout(scrollToBottom, 0);
	});

	async function sendMessage() {
		const text = input.trim();
		if (!text || streaming) return;

		const userMsg: ChatMessage = { role: 'user', content: text };
		messages.push(userMsg);
		input = '';
		streaming = true;

		// Add placeholder for AI response
		const aiMsg: ChatMessage = { role: 'assistant', content: '' };
		messages.push(aiMsg);

		try {
			const response = await fetch('/api/chat', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					entry_id: entryId,
					messages: messages.slice(0, -1).map((m) => ({ role: m.role, content: m.content }))
				})
			});

			if (!response.ok) {
				aiMsg.content = 'Error: Failed to get response.';
				streaming = false;
				return;
			}

			const reader = response.body!.getReader();
			const decoder = new TextDecoder();

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;
				const text = decoder.decode(value);
				for (const line of text.split('\n')) {
					if (line.startsWith('data: ')) {
						try {
							const data = JSON.parse(line.slice(6));
							if (data.content) {
								aiMsg.content += data.content;
							}
						} catch {
							// Skip malformed JSON
						}
					}
				}
			}
		} catch {
			aiMsg.content = aiMsg.content || 'Error: Connection failed.';
		} finally {
			streaming = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && !e.shiftKey) {
			e.preventDefault();
			sendMessage();
		}
	}

	onMount(() => {
		// Focus trap: close on Escape
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') onClose();
		}
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});
</script>

<!-- Backdrop on mobile -->
<div class="fixed inset-0 z-40 bg-black/30 md:hidden" onclick={onClose} role="presentation"></div>

<!-- Panel -->
<div
	class="fixed inset-0 z-50 flex flex-col bg-white md:inset-auto md:top-0 md:right-0 md:bottom-0 md:w-[400px] md:border-l md:border-slate-200 md:shadow-xl"
>
	<!-- Header -->
	<div class="flex items-center gap-3 border-b border-slate-200 px-4 py-3">
		<div class="min-w-0 flex-1">
			<h2 class="truncate text-sm font-semibold text-slate-900">{entryTitle}</h2>
			<p class="text-xs text-slate-500">Chat about this article</p>
		</div>
		<button
			onclick={onClose}
			class="flex h-8 w-8 items-center justify-center rounded-md text-slate-400 hover:bg-slate-100 hover:text-slate-600"
			aria-label="Close chat"
		>
			✕
		</button>
	</div>

	<!-- Messages -->
	<div bind:this={messagesEl} class="flex-1 overflow-y-auto px-4 py-4 space-y-3">
		{#if messages.length === 0}
			<div class="py-12 text-center text-sm text-slate-400">
				Ask anything about this article…
			</div>
		{/if}
		{#each messages as msg}
			<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
				<div
					class="max-w-[85%] rounded-lg px-3 py-2 text-sm whitespace-pre-wrap
						{msg.role === 'user'
						? 'bg-blue-600 text-white'
						: 'bg-slate-100 text-slate-800'}"
				>
					{msg.content || (streaming ? '…' : '')}
				</div>
			</div>
		{/each}
	</div>

	<!-- Input -->
	<div class="border-t border-slate-200 px-4 py-3">
		<div class="flex gap-2">
			<input
				type="text"
				bind:value={input}
				onkeydown={handleKeydown}
				placeholder="Type a message…"
				disabled={streaming}
				class="flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none disabled:opacity-50"
			/>
			<button
				onclick={sendMessage}
				disabled={streaming || !input.trim()}
				class="flex h-10 w-10 min-w-[44px] items-center justify-center rounded-md bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50"
				aria-label="Send message"
			>
				<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path
						stroke-linecap="round"
						stroke-linejoin="round"
						stroke-width="2"
						d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
					/>
				</svg>
			</button>
		</div>
	</div>
</div>
