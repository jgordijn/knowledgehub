import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { mount, unmount } from 'svelte';
import { tick } from 'svelte';
import DailyNewsPage from './+page.svelte';
import pb from '$lib/pb';

vi.mock('$lib/pb', () => ({
	default: { send: vi.fn() }
}));

const sendMock = vi.mocked(pb.send);

function digest(overrides: Record<string, unknown> = {}) {
	return {
		id: 'digest-latest',
		status: 'success',
		local_date: '2026-05-09',
		title: 'Morning Briefing',
		body_markdown: 'Read [[kh-entry:entry-1]] today.',
		referenced_entry_ids: ['entry-1'],
		candidate_count: 1,
		included_count: 1,
		used_subset: false,
		...overrides
	};
}

function settings(overrides: Record<string, unknown> = {}) {
	return {
		enabled: true,
		generation_time: '08:00',
		timezone: 'Europe/Amsterdam',
		extra_instructions: '',
		...overrides
	};
}

async function settle() {
	await Promise.resolve();
	await tick();
}

describe('Daily News page', () => {
	let target: HTMLElement;
	let component: Record<string, unknown> | undefined;

	beforeEach(() => {
		target = document.createElement('div');
		document.body.appendChild(target);
		sendMock.mockReset();
	});

	afterEach(() => {
		if (component) unmount(component as never);
		component = undefined;
		document.body.innerHTML = '';
	});

	it('shows the initial loading state and then renders the loaded digest', async () => {
		sendMock.mockImplementation(async (url) => {
			if (String(url).startsWith('/api/daily-news/digests?')) {
				return { latest: digest(), selected: digest(), archive: [], has_more: false };
			}
			if (url === '/api/daily-news/settings') return settings();
			throw new Error(`unexpected request ${url}`);
		});

		component = mount(DailyNewsPage, { target });
		expect(target.textContent).toContain('Loading Daily News…');

		await settle();

		expect(target.textContent).toContain('Latest edition');
		expect(target.textContent).toContain('Morning Briefing');
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/settings', { method: 'GET' });
	});

	it('queues Generate now and Regenerate actions with their loading labels', async () => {
		let resolveGenerate!: (value: unknown) => void;
		let resolveRegenerate!: (value: unknown) => void;
		sendMock.mockImplementation((url) => {
			const request = String(url);
			if (request.includes('selected=generated')) {
				return Promise.resolve({ latest: digest({ id: 'generated', title: 'Generated Briefing' }), selected: digest({ id: 'generated', title: 'Generated Briefing' }), archive: [], has_more: false });
			}
			if (request.includes('selected=regenerated')) {
				return Promise.resolve({ latest: digest({ id: 'regenerated', title: 'Regenerated Briefing' }), selected: digest({ id: 'regenerated', title: 'Regenerated Briefing' }), archive: [], has_more: false });
			}
			if (request.startsWith('/api/daily-news/digests?')) {
				return Promise.resolve({ latest: digest(), selected: digest(), archive: [], has_more: false });
			}
			if (url === '/api/daily-news/settings') return Promise.resolve(settings());
			if (url === '/api/daily-news/generate') return new Promise((resolve) => { resolveGenerate = resolve; });
			if (url === '/api/daily-news/digests/generated/regenerate') return new Promise((resolve) => { resolveRegenerate = resolve; });
			return Promise.reject(new Error(`unexpected request ${url}`));
		});

		component = mount(DailyNewsPage, { target });
		await settle();

		const generateButton = [...target.querySelectorAll('button')].find((button) => button.textContent?.includes('Generate now')) as HTMLButtonElement;
		generateButton.click();
		await tick();
		expect(target.textContent).toContain('Generating…');
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/generate', { method: 'POST' });
		resolveGenerate(digest({ id: 'generated', title: 'Generated Briefing' }));
		await settle();
		expect(target.textContent).toContain('Generated Briefing');

		const regenerateButton = [...target.querySelectorAll('button')].find((button) => button.textContent?.includes('Regenerate')) as HTMLButtonElement;
		regenerateButton.click();
		await tick();
		expect(target.textContent).toContain('Regenerating…');
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/digests/generated/regenerate', { method: 'POST' });
		resolveRegenerate(digest({ id: 'regenerated', title: 'Regenerated Briefing' }));
		await settle();
		expect(target.textContent).toContain('Regenerated Briefing');
	});

	it('validates settings before saving and posts valid settings', async () => {
		sendMock.mockImplementation(async (url, options) => {
			if (String(url).startsWith('/api/daily-news/digests?')) return { latest: null, selected: null, archive: [], has_more: false };
			if (url === '/api/daily-news/settings' && options?.method === 'GET') return settings();
			if (url === '/api/daily-news/settings' && options?.method === 'PUT') return options.body;
			throw new Error(`unexpected request ${url}`);
		});

		component = mount(DailyNewsPage, { target });
		await settle();

		const timeInput = target.querySelector('input[placeholder="08:00"]') as HTMLInputElement;
		timeInput.value = '8:00';
		timeInput.dispatchEvent(new Event('input', { bubbles: true }));
		await tick();
		const saveButton = [...target.querySelectorAll('button')].find((button) => button.textContent?.includes('Save settings')) as HTMLButtonElement;
		saveButton.click();
		await settle();

		expect(target.textContent).toContain('Use a 24-hour HH:MM generation time.');
		expect(sendMock).not.toHaveBeenCalledWith('/api/daily-news/settings', expect.objectContaining({ method: 'PUT' }));

		timeInput.value = '09:30';
		timeInput.dispatchEvent(new Event('input', { bubbles: true }));
		await tick();
		saveButton.click();
		await settle();

		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/settings', expect.objectContaining({ method: 'PUT', body: expect.objectContaining({ generation_time: '09:30' }) }));
		expect(target.textContent).toContain('Daily News settings saved.');
	});

	it('loads more archive editions and selects an archived digest', async () => {
		sendMock.mockImplementation(async (url) => {
			const request = String(url);
			if (request.includes('offset=0') && request.includes('selected=older')) {
				return { latest: digest(), selected: digest({ id: 'older', local_date: '2026-05-08', title: 'Older Edition' }), archive: [], has_more: false };
			}
			if (request.includes('offset=1')) {
				return { latest: digest(), selected: digest(), archive: [digest({ id: 'older', local_date: '2026-05-08', title: 'Older Edition' })], has_more: false };
			}
			if (request.startsWith('/api/daily-news/digests?')) {
				return { latest: digest(), selected: digest(), archive: [digest({ id: 'archived', local_date: '2026-05-07', title: 'Archived Edition' })], has_more: true };
			}
			if (url === '/api/daily-news/settings') return settings();
			throw new Error(`unexpected request ${url}`);
		});

		component = mount(DailyNewsPage, { target });
		await settle();

		const loadMore = [...target.querySelectorAll('button')].find((button) => button.textContent?.includes('Load more editions')) as HTMLButtonElement;
		loadMore.click();
		await settle();
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/digests?limit=10&offset=1&selected=digest-latest', { method: 'GET' });
		expect(target.textContent).toContain('2026-05-08 · Older Edition');

		const older = [...target.querySelectorAll('button')].find((button) => button.textContent?.includes('Older Edition')) as HTMLButtonElement;
		older.click();
		await settle();
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/digests?limit=10&offset=0&selected=older', { method: 'GET' });
		expect(target.textContent).toContain('Older Edition');
	});

	it('opens and closes referenced entry details from digest markers', async () => {
		sendMock.mockImplementation(async (url) => {
			if (String(url).startsWith('/api/daily-news/digests?')) return { latest: digest(), selected: digest(), archive: [], has_more: false };
			if (url === '/api/daily-news/settings') return settings();
			if (url === '/api/daily-news/digests/digest-latest/entries/entry-1') {
				return {
					available: true,
					entry: { title: 'Referenced Article', url: 'https://example.com/article', source_name: 'Example', effective_stars: 4, summary: 'Useful summary', takeaways: ['One'] }
				};
			}
			throw new Error(`unexpected request ${url}`);
		});

		component = mount(DailyNewsPage, { target });
		await settle();
		await settle();

		expect(target.innerHTML).toContain('data-entry-id="entry-1"');
		(target.querySelector('[data-entry-id="entry-1"]') as HTMLButtonElement).click();
		await settle();
		expect(sendMock).toHaveBeenCalledWith('/api/daily-news/digests/digest-latest/entries/entry-1', { method: 'GET' });
		expect(target.textContent).toContain('Referenced entry');
		expect(target.textContent).toContain('Referenced Article');

		([...target.querySelectorAll('button')].find((button) => button.textContent === 'Close') as HTMLButtonElement).click();
		await tick();
		expect(target.textContent).not.toContain('Referenced Article');
	});
});
