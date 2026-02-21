export type ThemeMode = 'light' | 'dark' | 'system';

const STORAGE_KEY = 'kh-theme';

function getStored(): ThemeMode {
	return (localStorage.getItem(STORAGE_KEY) as ThemeMode) || 'system';
}

function isDark(mode: ThemeMode): boolean {
	if (mode === 'dark') return true;
	if (mode === 'light') return false;
	return window.matchMedia('(prefers-color-scheme: dark)').matches;
}

function apply(mode: ThemeMode) {
	document.documentElement.classList.toggle('dark', isDark(mode));
}

let current: ThemeMode = 'system';

export function initTheme() {
	current = getStored();
	apply(current);

	window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
		if (current === 'system') apply('system');
	});
}

export function getTheme(): ThemeMode {
	return current;
}

export function setTheme(mode: ThemeMode) {
	current = mode;
	localStorage.setItem(STORAGE_KEY, mode);
	apply(mode);
}
