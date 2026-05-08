import { describe, expect, it } from 'vitest';
import { dailyNewsNavItem, dailyNewsLoadingMessage } from './daily-news-ui';

describe('daily news UI helpers', () => {
	it('exposes the authenticated navigation item', () => {
		expect(dailyNewsNavItem()).toEqual({ href: '/daily-news', label: 'Daily News', icon: '🗞️' });
	});

	it('provides a clear page loading state', () => {
		expect(dailyNewsLoadingMessage()).toBe('Loading Daily News…');
	});
});
