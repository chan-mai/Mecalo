// ログイン後, 初回セットアップ後 演出
export type CurtainKind = 'first' | 'return' | 'setup' | 'onboarded';

export interface CurtainPlayer {
	cover: (kind: CurtainKind, origin?: { x: number; y: number }) => Promise<void>;
	reveal: () => Promise<void>;
}

let player: CurtainPlayer | undefined;
const pendingKey = 'mecalo-curtain-pending';

export function registerCurtain(p: CurtainPlayer | undefined) {
	player = p;
}

function centerOf(el: Element | null | undefined) {
	if (!el) return undefined;
	const r = el.getBoundingClientRect();
	return { x: r.left + r.width / 2, y: r.top + r.height / 2 };
}

export function useEntryCurtain() {
	const { me } = useAuth();

	async function play(kind: CurtainKind, go: () => unknown, from?: Element | null) {
		const p = player;
		if (!p) {
			await go();
			return;
		}
		// 非表示タブ対策
		await Promise.race([p.cover(kind, centerOf(from)), new Promise((r) => setTimeout(r, 2500))]);
		try {
			await go();
		} finally {
			// 遷移先ページ表示切替待ち
			await new Promise((r) => setTimeout(r, 350));
			await p.reveal();
		}
	}

	function loginKind(): CurtainKind {
		return me.value?.onboarded === false ? 'first' : 'return';
	}

	// ログイン完了後遷移
	async function enter(go: () => unknown, from?: Element | null) {
		await play(loginKind(), go, from);
	}

	function markPending() {
		try {
			sessionStorage.setItem(pendingKey, '1');
		} catch {
			// 保存不能時は演出なし
		}
	}

	function takePending(): boolean {
		try {
			const v = sessionStorage.getItem(pendingKey) === '1';
			sessionStorage.removeItem(pendingKey);
			return v;
		} catch {
			return false;
		}
	}

	return { play, enter, loginKind, markPending, takePending };
}
