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
		messages.push({ role: 'assistant', content: '' });
		const aiIdx = messages.length - 1;

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
				messages[aiIdx].content = 'Error: Failed to get response.';
				streaming = false;
				return;
			}

			const reader = response.body!.getReader();
			const decoder = new TextDecoder();
			let buffer = '';

			while (true) {
				const { done, value } = await reader.read();
				if (done) break;
				buffer += decoder.decode(value, { stream: true });
				const lines = buffer.split('\n');
				buffer = lines.pop() ?? '';
				for (const line of lines) {
					if (line.startsWith('data: ')) {
						const payload = line.slice(6);
						if (payload === '[DONE]') continue;
						try {
							const data = JSON.parse(payload);
							if (data.error) {
								messages[aiIdx].content = 'Error: ' + data.error;
							} else if (data.content) {
								messages[aiIdx].content += data.content;
							}
						} catch {
							// Skip malformed JSON
						}
					}
				}
			}
		} catch {
			messages[aiIdx].content = messages[aiIdx].content || 'Error: Connection failed.';
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
					{#if !msg.content && streaming && msg.role === 'assistant'}
						<span class="typing-dots">
							<span></span>
							<span></span>
							<span></span>
						</span>
					{:else}
						{msg.content}
					{/if}
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


<style>
	.typing-dots {
		display: inline-flex;
		align-items: center;
		gap: 4px;
		padding: 2px 0;
	}

	.typing-dots span {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background-color: #94a3b8;
		animation: typing-bounce 1.4s infinite ease-in-out;
	}

	.typing-dots span:nth-child(1) {
		animation-delay: 0s;
	}

	.typing-dots span:nth-child(2) {
		animation-delay: 0.2s;
	}

	.typing-dots span:nth-child(3) {
		animation-delay: 0.4s;
	}

	@keyframes typing-bounce {
		0%, 60%, 100% {
			transform: translateY(0);
			opacity: 0.4;
		}
		30% {
			transform: translateY(-4px);
			opacity: 1;
		}
	}
</style>