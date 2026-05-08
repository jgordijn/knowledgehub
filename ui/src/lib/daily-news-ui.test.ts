import { describe, expect, it } from 'vitest';
import {
	dailyNewsNavItem,
	dailyNewsLoadingMessage,
	renderDailyNewsMarkdown,
	dailyNewsSubsetMessage,
	dailyNewsStateMessage,
	type DailyNewsDigestDTO
} from './daily-news-ui';

describe('daily news UI helpers', () => {
	it('exposes the authenticated navigation item', () => {
		expect(dailyNewsNavItem()).toEqual({ href: '/daily-news', label: 'Daily News', icon: '🗞️' });
	});

	it('provides a clear page loading state', () => {
		expect(dailyNewsLoadingMessage()).toBe('Loading Daily News…');
	});

	it('renders daily news markdown through a strict allowlist sanitizer', () => {
		const html = renderDailyNewsMarkdown(`
# Morning Edition

Top **story** with [safe link](https://example.com).

<img src="https://example.com/tracker.png" onerror="alert(1)">
<script>alert('xss')</script>
[bad](javascript:alert(1)) [protocol](//evil.example) [data](data:text/html,boom)
<iframe src="https://example.com"></iframe><style>body{display:none}</style>
`);

		expect(html).toContain('<h1');
		expect(html).toContain('<strong>story</strong>');
		expect(html).toContain('href="https://example.com"');
		expect(html).toContain('rel="noopener noreferrer"');
		expect(html).not.toContain('<img');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('<iframe');
		expect(html).not.toContain('<style');
		expect(html).not.toContain('javascript:');
		expect(html).not.toContain('data:text');
		expect(html).not.toContain('href="//evil.example"');
	});

	it('explains when the digest used a subset of candidates', () => {
		const digest: DailyNewsDigestDTO = {
			id: 'digest1',
			status: 'success',
			title: 'Daily Briefing',
			body_markdown: '# Daily Briefing',
			candidate_count: 42,
			included_count: 20,
			used_subset: true
		};

		expect(dailyNewsSubsetMessage(digest)).toBe('This digest is based on 20 of 42 available articles.');
		expect(dailyNewsSubsetMessage({ ...digest, used_subset: false })).toBe('');
	});

	it('describes pending, running, failed, and empty digest states', () => {
		expect(dailyNewsStateMessage({ id: 'p', status: 'pending' })).toEqual({
			tone: 'info',
			title: 'Daily News is queued',
			message: 'Your digest has been queued and will be generated shortly.'
		});
		expect(dailyNewsStateMessage({ id: 'r', status: 'running' }).title).toBe('Daily News is being generated');
		expect(dailyNewsStateMessage({ id: 'f', status: 'failed', error_message: 'OpenRouter unavailable' })).toEqual({
			tone: 'error',
			title: 'Daily News generation failed',
			message: 'OpenRouter unavailable'
		});
		expect(dailyNewsStateMessage({ id: 'e', status: 'success', body_markdown: '', candidate_count: 0 })).toEqual({
			tone: 'empty',
			title: 'No articles today',
			message: 'No new articles matched this digest window.'
		});
	});
});
