import { describe, it, expect, beforeEach } from 'vitest';
import {
	getRememberMe,
	setRememberMe,
	switchStorageBackend
} from './auth-store';

const AUTH_STORAGE_KEY = 'pocketbase_auth';
const REMEMBER_ME_KEY = 'kh_remember_me';

describe('auth-store', () => {
	beforeEach(() => {
		localStorage.clear();
		sessionStorage.clear();
	});

	describe('getRememberMe', () => {
		it('defaults to true when not set', () => {
			expect(getRememberMe()).toBe(true);
		});

		it('returns true when stored as "true"', () => {
			localStorage.setItem(REMEMBER_ME_KEY, 'true');
			expect(getRememberMe()).toBe(true);
		});

		it('returns false when stored as "false"', () => {
			localStorage.setItem(REMEMBER_ME_KEY, 'false');
			expect(getRememberMe()).toBe(false);
		});
	});

	describe('setRememberMe', () => {
		it('stores true value', () => {
			setRememberMe(true);
			expect(localStorage.getItem(REMEMBER_ME_KEY)).toBe('true');
		});

		it('stores false value', () => {
			setRememberMe(false);
			expect(localStorage.getItem(REMEMBER_ME_KEY)).toBe('false');
		});
	});

	describe('switchStorageBackend', () => {
		it('moves data to localStorage when remember=true', () => {
			setRememberMe(true);
			sessionStorage.setItem(AUTH_STORAGE_KEY, 'auth-token-data');

			switchStorageBackend();

			expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBe('auth-token-data');
			expect(sessionStorage.getItem(AUTH_STORAGE_KEY)).toBeNull();
		});

		it('moves data to sessionStorage when remember=false', () => {
			setRememberMe(false);
			localStorage.setItem(AUTH_STORAGE_KEY, 'auth-token-data');

			switchStorageBackend();

			expect(sessionStorage.getItem(AUTH_STORAGE_KEY)).toBe('auth-token-data');
			expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull();
		});

		it('handles no existing data gracefully', () => {
			setRememberMe(true);
			switchStorageBackend();
			// Should not throw or store anything
			expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBeNull();
		});

		it('prefers localStorage data over sessionStorage', () => {
			setRememberMe(true);
			localStorage.setItem(AUTH_STORAGE_KEY, 'local-data');
			sessionStorage.setItem(AUTH_STORAGE_KEY, 'session-data');

			switchStorageBackend();

			expect(localStorage.getItem(AUTH_STORAGE_KEY)).toBe('local-data');
			expect(sessionStorage.getItem(AUTH_STORAGE_KEY)).toBeNull();
		});
	});
});
