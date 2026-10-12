import { ApiError, unwrap, useApi } from './api/client';

// プロフィール未登録時オンボーディング
export function useOnboarding() {
	const state = useState<{ uid: string; needed: boolean } | null>('onboarding', () => null);
	const api = useApi();

	async function needed(uid: string): Promise<boolean> {
		if (state.value?.uid === uid) return state.value.needed;
		let result = false;
		try {
			await unwrap(api.GET('/settings'));
		} catch (err) {
			// 通信不能時は表示なし
			result = err instanceof ApiError && err.status === 404;
		}
		state.value = { uid, needed: result };
		return result;
	}

	function complete(uid: string) {
		state.value = { uid, needed: false };
	}

	return { needed, complete };
}
