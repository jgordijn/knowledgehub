import { describe, expect, it } from 'vitest';
import {
	dailyNewsNavItem,
	dailyNewsLoadingMessage,
	renderDailyNewsMarkdown,
	dailyNewsSubsetMessage,
	dailyNewsStateMessage,
	dailyNewsArchiveLabel,
	selectDailyNewsDigest,
	dailyNewsGenerateButtonLabel,
	dailyNewsRegenerateButtonLabel,
	dailyNewsCanRegenerate,
	renderDailyNewsReferences,
	validateDailyNewsSettings,
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

[[kh-entry:entry1]] [[kh-entry:missing]]

<img src="https://example.com/tracker.png" onerror="alert(1)">
<script>alert('xss')</script>
[bad](javascript:alert(1)) [protocol](//evil.example) [data](data:text/html,boom)
<iframe src="https://example.com"></iframe><style>body{display:none}</style>
`, ['entry1']);

		expect(html).toContain('<h1');
		expect(html).toContain('<strong>story</strong>');
		expect(html).toContain('href="https://example.com"');
		expect(html).toContain('rel="noopener noreferrer"');
		expect(html).toContain('data-entry-id="entry1"');
		expect(html).not.toContain('data-entry-id="missing"');
		expect(html).not.toContain('<img');
		expect(html).not.toContain('<script');
		expect(html).not.toContain('<iframe');
		expect(html).not.toContain('<style');
		expect(html).not.toContain('javascript:');
		expect(html).not.toContain('data:text');
		expect(html).not.toContain('href="//evil.example"');
	});

	it('strips raw model-supplied HTML controls while preserving validated entry markers', () => {
		const html = renderDailyNewsMarkdown('Raw <button data-entry-id="evil">Open evil</button> and marker [[kh-entry:entry1]].', ['entry1']);

		expect(html).toContain('data-entry-id="entry1"');
		expect(html).not.toContain('data-entry-id="evil"');
		expect(html).not.toContain('<button data-entry-id="evil"');
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

	it('labels archive editions and selects an owned digest by id', () => {
		const latest: DailyNewsDigestDTO = { id: 'latest', status: 'success', local_date: '2026-05-08', title: 'Morning' };
		const older: DailyNewsDigestDTO = { id: 'older', status: 'success', local_date: '2026-05-07' };

		expect(dailyNewsArchiveLabel(latest)).toBe('2026-05-08 · Morning');
		expect(dailyNewsArchiveLabel(older)).toBe('2026-05-07');
		expect(selectDailyNewsDigest([latest, older], 'older')).toEqual(older);
		expect(selectDailyNewsDigest([latest, older], 'missing')).toEqual(latest);
	});

	it('describes Generate now and Regenerate control states', () => {
		expect(dailyNewsGenerateButtonLabel(false)).toBe('Generate now');
		expect(dailyNewsGenerateButtonLabel(true)).toBe('Generating…');
		expect(dailyNewsRegenerateButtonLabel(false)).toBe('Regenerate');
		expect(dailyNewsRegenerateButtonLabel(true)).toBe('Regenerating…');
		expect(dailyNewsCanRegenerate({ id: 's', status: 'success' })).toBe(true);
		expect(dailyNewsCanRegenerate({ id: 'f', status: 'failed' })).toBe(true);
		expect(dailyNewsCanRegenerate({ id: 'p', status: 'pending' })).toBe(false);
		expect(dailyNewsCanRegenerate(null)).toBe(false);
	});

	it('validates Daily News settings before saving', () => {
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '08:00', timezone: 'Europe/Amsterdam', extra_instructions: 'Use bullets\nFocus AI' })).toEqual([]);
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '8:00', timezone: 'Europe/Amsterdam', extra_instructions: '' })).toContain('Use a 24-hour HH:MM generation time.');
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '08:00', timezone: '', extra_instructions: '' })).toContain('Choose a timezone.');
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '08:00', timezone: 'Not/AZone', extra_instructions: '' })).toContain('Choose a valid IANA timezone.');
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '08:00', timezone: 'UTC', extra_instructions: 'x'.repeat(2001) })).toContain('Extra instructions must be 2000 characters or fewer.');
		expect(validateDailyNewsSettings({ enabled: true, generation_time: '08:00', timezone: 'UTC', extra_instructions: 'bad\u0001' })).toContain('Extra instructions contain unsupported control characters.');
	});

	it('renders validated inline KnowledgeHub entry markers as in-app controls only for stored references', () => {
		const html = renderDailyNewsReferences('Read [[kh-entry:entry1]], ignore [[kh-entry:missing]], and repeat [[kh-entry:entry1]].', ['entry1']);

		expect(html).toContain('data-entry-id="entry1"');
		expect(html).toContain('Open referenced entry');
		expect(html).not.toContain('data-entry-id="missing"');
		expect(html).not.toContain('[[kh-entry:entry1]]');
		expect(html).not.toContain('[[kh-entry:missing]]');
	});

	it('describes pending, running, failed, and empty digest states', () => {
		expect(dailyNewsStateMessage({ id: 'p', status: 'pending' })).toEqual({
			tone: 'info',
			title: 'Daily News is queued',
			message: 'Your digest has been queued and will be generated shortly.'
		});
		expect(dailyNewsStateMessage({ id: 'r', status: 'running' })?.title).toBe('Daily News is being generated');
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
