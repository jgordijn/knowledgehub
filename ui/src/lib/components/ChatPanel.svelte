<script lang="ts">
	import { onMount } from 'svelte';
	import { renderMarkdown } from '$lib/markdown';

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

	// Resize state
	let panelWidth = $state(400);
	let dragging = $state(false);

	function onDragStart(e: PointerEvent) {
		e.preventDefault();
		dragging = true;
		const onMove = (ev: PointerEvent) => {
			const newWidth = window.innerWidth - ev.clientX;
			panelWidth = Math.max(300, Math.min(newWidth, window.innerWidth * 0.8));
		};
		const onUp = () => {
			dragging = false;
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
		};
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	}

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
		function onKey(e: KeyboardEvent) {
			if (e.key === 'Escape') onClose();
			// Ctrl+Left/Right to resize panel
			if (e.ctrlKey && e.key === 'ArrowLeft') {
				e.preventDefault();
				panelWidth = Math.min(panelWidth + 100, window.innerWidth * 0.8);
			}
			if (e.ctrlKey && e.key === 'ArrowRight') {
				e.preventDefault();
				panelWidth = Math.max(panelWidth - 100, 300);
			}
		}
		window.addEventListener('keydown', onKey);
		return () => window.removeEventListener('keydown', onKey);
	});
</script>

<!-- Backdrop on mobile -->
<div class="fixed inset-0 z-40 bg-black/30 md:hidden" onclick={onClose} role="presentation"></div>

<!-- Panel -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="fixed inset-0 z-50 flex flex-col bg-white md:inset-auto md:top-0 md:right-0 md:bottom-0 md:border-l md:border-slate-200 md:shadow-xl dark:bg-slate-800 dark:md:border-slate-700 {dragging ? 'select-none' : ''}"
	style:width={`${panelWidth}px`}
>
	<!-- Drag handle (also keyboard-accessible via Ctrl+Arrow) -->
	<div
		class="hidden md:block absolute left-0 top-0 bottom-0 w-1.5 cursor-col-resize hover:bg-blue-400 hover:opacity-40 transition-colors z-10 {dragging ? 'bg-blue-400 opacity-40' : ''}"
		onpointerdown={onDragStart}
		role="separator"
		aria-orientation="vertical"
		aria-label="Resize chat panel (use Ctrl+Left/Right)"
	></div>
	<!-- Header -->
	<div class="flex items-center gap-3 border-b border-slate-200 px-4 py-3 dark:border-slate-700">
		<div class="min-w-0 flex-1">
			<h2 class="truncate text-sm font-semibold text-slate-900 dark:text-slate-100">{entryTitle}</h2>
			<p class="text-xs text-slate-500 dark:text-slate-400">Chat about this article</p>
		</div>
		<button
			onclick={onClose}
			class="flex h-8 w-8 items-center justify-center rounded-md text-slate-400 hover:bg-slate-100 hover:text-slate-600 dark:text-slate-500 dark:hover:bg-slate-700 dark:hover:text-slate-300"
			aria-label="Close chat"
		>
			✕
		</button>
	</div>

	<!-- Messages -->
	<div bind:this={messagesEl} class="flex-1 overflow-y-auto px-4 py-4 space-y-3">
		{#if messages.length === 0}
			<div class="py-12 text-center text-sm text-slate-400 dark:text-slate-500">
				Ask anything about this article…
			</div>
		{/if}
		{#each messages as msg}
			<div class="flex {msg.role === 'user' ? 'justify-end' : 'justify-start'}">
				<div
					class="max-w-[85%] rounded-lg px-3 py-2 text-sm
						{msg.role === 'user'
						? 'bg-blue-600 text-white whitespace-pre-wrap'
						: 'bg-slate-100 text-slate-800 prose prose-sm prose-slate dark:bg-slate-700 dark:text-slate-200 dark:prose-invert'}"
				>
					{#if !msg.content && streaming && msg.role === 'assistant'}
						<span class="typing-dots">
							<span></span>
							<span></span>
							<span></span>
						</span>
					{:else if msg.role === 'assistant'}
						{@html renderMarkdown(msg.content)}
					{:else}
						{msg.content}
					{/if}
				</div>
			</div>
		{/each}
	</div>

	<!-- Input -->
	<div class="border-t border-slate-200 px-4 py-3 dark:border-slate-700">
		<div class="flex gap-2">
			<input
				type="text"
				bind:value={input}
				onkeydown={handleKeydown}
				placeholder="Type a message…"
				disabled={streaming}
				class="flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm focus:border-blue-500 focus:ring-1 focus:ring-blue-500 focus:outline-none disabled:opacity-50 dark:border-slate-600 dark:bg-slate-700 dark:text-slate-100 dark:placeholder-slate-500"
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
	/* Tighten prose inside chat bubbles */
	:global(.prose) {
		max-width: none;
	}
	:global(.prose p:first-child) {
		margin-top: 0;
	}
	:global(.prose p:last-child) {
		margin-bottom: 0;
	}
	:global(.prose pre) {
		overflow-x: auto;
	}

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
