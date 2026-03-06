import PocketBase, { isTokenExpired } from 'pocketbase';
import { authStore } from './auth-store';

const SUPERUSER_COLLECTION = '_superusers';
const AUTH_REFRESH_PATH = `/api/collections/${SUPERUSER_COLLECTION}/auth-refresh`;
const AUTO_REFRESH_THRESHOLD_SECONDS = 60 * 60;

const pb = new PocketBase('/', authStore);

let refreshPromise: Promise<void> | null = null;

function isSuperuserSession(): boolean {
	return pb.authStore.record?.collectionName === SUPERUSER_COLLECTION;
}

function shouldClearAuth(error: unknown): boolean {
	if (typeof error !== 'object' || error === null || !('status' in error)) {
		return false;
	}

	const status = error.status;
	return status === 401 || status === 403;
}

async function refreshSuperuserAuth(): Promise<void> {
	if (!pb.authStore.token || !isSuperuserSession()) {
		return;
	}

	if (!refreshPromise) {
		refreshPromise = pb
			.collection(SUPERUSER_COLLECTION)
			.authRefresh()
			.then(() => undefined)
			.catch((error) => {
				if (shouldClearAuth(error)) {
					pb.authStore.clear();
				}
				throw error;
			})
			.finally(() => {
				refreshPromise = null;
			});
	}

	await refreshPromise;
}

async function maybeRefreshAuth(url: string): Promise<void> {
	if (url.includes(AUTH_REFRESH_PATH)) {
		return;
	}

	if (!pb.authStore.isValid || !isSuperuserSession()) {
		return;
	}

	if (!isTokenExpired(pb.authStore.token, AUTO_REFRESH_THRESHOLD_SECONDS)) {
		return;
	}

	await refreshSuperuserAuth();
}

function syncAuthorizationHeader(options: Record<string, any>, previousToken: string): void {
	if (!pb.authStore.token) {
		return;
	}

	const headers = { ...(options.headers || {}) };
	const headerKey = Object.keys(headers).find((key) => key.toLowerCase() === 'authorization');

	if (!headerKey) {
		headers.Authorization = pb.authStore.token;
		options.headers = headers;
		return;
	}

	if (!previousToken || headers[headerKey] === previousToken) {
		headers[headerKey] = pb.authStore.token;
		options.headers = headers;
	}
}

export async function refreshAuthSession(): Promise<boolean> {
	if (!pb.authStore.isValid || !isSuperuserSession()) {
		return pb.authStore.isValid;
	}

	try {
		await refreshSuperuserAuth();
		return true;
	} catch {
		return false;
	}
}

pb.beforeSend = async (url, options) => {
	const previousToken = pb.authStore.token;

	try {
		await maybeRefreshAuth(url);
	} catch (error) {
		if (!pb.authStore.isValid) {
			throw error;
		}
	}

	syncAuthorizationHeader(options, previousToken);

	return { url, options };
};

export default pb;
