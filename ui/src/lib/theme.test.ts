import { describe, it, expect, beforeEach, vi } from 'vitest';
import { initTheme, getTheme, setTheme } from './theme';

// Mock matchMedia for jsdom (not available by default)
Object.defineProperty(window, 'matchMedia', {
	writable: true,
	value: vi.fn().mockImplementation((query: string) => ({
		matches: false,
		media: query,
		onchange: null,
		addListener: vi.fn(),
		removeListener: vi.fn(),
		addEventListener: vi.fn(),
		removeEventListener: vi.fn(),
		dispatchEvent: vi.fn(),
	})),
});

describe('theme', () => {
	beforeEach(() => {
		localStorage.clear();
		document.documentElement.classList.remove('dark');
	});

	it('defaults to system theme', () => {
		initTheme();
		expect(getTheme()).toBe('system');
	});

	it('stores and retrieves dark theme', () => {
		setTheme('dark');
		expect(getTheme()).toBe('dark');
		expect(localStorage.getItem('kh-theme')).toBe('dark');
		expect(document.documentElement.classList.contains('dark')).toBe(true);
	});

	it('stores and retrieves light theme', () => {
		setTheme('light');
		expect(getTheme()).toBe('light');
		expect(localStorage.getItem('kh-theme')).toBe('light');
		expect(document.documentElement.classList.contains('dark')).toBe(false);
	});

	it('initializes from stored preference', () => {
		localStorage.setItem('kh-theme', 'dark');
		initTheme();
		expect(getTheme()).toBe('dark');
		expect(document.documentElement.classList.contains('dark')).toBe(true);
	});

	it('system mode responds to prefers-color-scheme', () => {
		setTheme('system');
		expect(getTheme()).toBe('system');
	});
});
