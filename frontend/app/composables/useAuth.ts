import { ApiError, unwrap, useApi } from './api/client';
import type { Schemas } from './api/client';

export const apiCacheName = 'mecalo-api';

export function useAuth() {
	const me = useState<Schemas['User'] | null>('me', () => null);
	const loaded = useState('me-loaded', () => false);
	const api = useApi();

	async function refresh() {
		try {
			me.value = await unwrap(api.GET('/me'));
		} catch (err) {
			// 通信不能時は現在の状態を維持
			if (err instanceof TypeError) {
				loaded.value = true;
				return me.value;
			}
			if (!(err instanceof ApiError && err.status === 401)) throw err;
			me.value = null;
		}
		loaded.value = true;
		return me.value;
	}

	async function ensureLoaded() {
		if (!loaded.value) await refresh();
		return me.value;
	}

	async function logout() {
		await unwrap(api.POST('/auth/logout'));
		me.value = null;
		// 利用者のAPI応答キャッシュを端末から削除
		if ('caches' in window) await window.caches.delete(apiCacheName);
		await navigateTo('/login');
	}

	return { me, refresh, ensureLoaded, logout };
}
