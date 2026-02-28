import { AsyncAuthStore } from 'pocketbase';

const AUTH_STORAGE_KEY = 'pocketbase_auth';
const REMEMBER_ME_KEY = 'kh_remember_me';

/**
 * Reads the remember-me preference from localStorage.
 * Defaults to true (checked) if not set.
 */
export function getRememberMe(): boolean {
	const val = localStorage.getItem(REMEMBER_ME_KEY);
	return val === null ? true : val === 'true';
}

/**
 * Persists the remember-me preference to localStorage.
 */
export function setRememberMe(value: boolean): void {
	localStorage.setItem(REMEMBER_ME_KEY, String(value));
}

/**
 * Returns the active storage backend based on remember-me preference.
 */
function activeStorage(): Storage {
	return getRememberMe() ? localStorage : sessionStorage;
}

/**
 * Loads the initial auth payload from whichever storage has data.
 * Checks localStorage first, then sessionStorage.
 */
function loadInitial(): string {
	return localStorage.getItem(AUTH_STORAGE_KEY) || sessionStorage.getItem(AUTH_STORAGE_KEY) || '';
}

/**
 * Switches the auth data to the target storage backend and clears the other.
 * Called at login time after the remember-me preference has been saved.
 */
export function switchStorageBackend(): void {
	const remember = getRememberMe();
	const data = localStorage.getItem(AUTH_STORAGE_KEY) || sessionStorage.getItem(AUTH_STORAGE_KEY) || '';

	if (remember) {
		// Move to localStorage, clear sessionStorage
		if (data) localStorage.setItem(AUTH_STORAGE_KEY, data);
		sessionStorage.removeItem(AUTH_STORAGE_KEY);
	} else {
		// Move to sessionStorage, clear localStorage
		if (data) sessionStorage.setItem(AUTH_STORAGE_KEY, data);
		localStorage.removeItem(AUTH_STORAGE_KEY);
	}
}

/**
 * Custom auth store that delegates to localStorage or sessionStorage
 * based on the kh_remember_me preference.
 */
export const authStore = new AsyncAuthStore({
	save: async (serialized) => {
		activeStorage().setItem(AUTH_STORAGE_KEY, serialized);
	},
	clear: async () => {
		localStorage.removeItem(AUTH_STORAGE_KEY);
		sessionStorage.removeItem(AUTH_STORAGE_KEY);
	},
	initial: loadInitial()
});
