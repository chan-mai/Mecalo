import { createGlobalState, useLocalStorage, useOnline } from '@vueuse/core';
import { ApiError, unwrap, useApi } from './api/client';
import type { Schemas } from './api/client';

// 未送信の操作, サーバ側で冪等になる形のみ扱う
export type OutboxOp =
	| { kind: 'addMeal'; body: Schemas['AddMealEntry'] & { clientRef: string }; preview: { foodName: string; grams?: number; kcal?: number } }
	| { kind: 'updateMeal'; id: string; body: Schemas['UpdateMealEntry'] }
	| { kind: 'deleteMeal'; id: string }
	| { kind: 'saveWeight'; date: string; body: Schemas['WeightInput'] };

export interface OutboxItem {
	key: string;
	op: OutboxOp;
	createdAt: string;
	// 4xx応答で送信できなかった理由
	error?: string;
}

export function isNetworkError(err: unknown): boolean {
	if (err instanceof ApiError) return err.status === 502 || err.status === 503 || err.status === 504;
	return err instanceof TypeError || (typeof navigator !== 'undefined' && !navigator.onLine);
}

// 利用者IDごとの送信待ち
const useStoredOutbox = createGlobalState(() => useLocalStorage<Record<string, OutboxItem[]>>('mecalo:outbox', {}));

export function useOutbox() {
	const { me } = useAuth();
	const stored = useStoredOutbox();
	const items = computed(() => (me.value ? stored.value[me.value.id] ?? [] : []));
	const online = useOnline();
	const syncing = useState('outbox-syncing', () => false);

	const pending = computed(() => items.value.filter((i) => !i.error));
	const failed = computed(() => items.value.filter((i) => i.error));

	function save(next: OutboxItem[]) {
		if (!me.value) return;
		const id = me.value.id;
		const others = Object.fromEntries(Object.entries(stored.value).filter(([key]) => key !== id));
		stored.value = next.length > 0 ? { ...others, [id]: next } : others;
	}

	function enqueue(op: OutboxOp) {
		save([...items.value, { key: crypto.randomUUID(), op, createdAt: new Date().toISOString() }]);
	}

	function remove(key: string) {
		save(items.value.filter((i) => i.key !== key));
	}

	// 未同期の追加記録を書き換え
	function updatePendingAdd(clientRef: string, fn: (op: Extract<OutboxOp, { kind: 'addMeal' }>) => void) {
		save(items.value.map((i) => {
			if (i.op.kind !== 'addMeal' || i.op.body.clientRef !== clientRef) return i;
			const op = structuredClone(toRaw(i.op));
			fn(op);
			return { ...i, op };
		}));
	}

	async function send(op: OutboxOp): Promise<unknown> {
		const api = useApi();
		switch (op.kind) {
			case 'addMeal':
				return unwrap(api.POST('/meals', { body: op.body }));
			case 'updateMeal':
				return unwrap(api.PATCH('/meals/{id}', { params: { path: { id: op.id } }, body: op.body }));
			case 'deleteMeal':
				try {
					return await unwrap(api.DELETE('/meals/{id}', { params: { path: { id: op.id } } }));
				} catch (err) {
					// 削除済みは成功扱い
					if (err instanceof ApiError && err.status === 404) return undefined;
					throw err;
				}
			case 'saveWeight':
				return unwrap(api.PUT('/weights/{date}', { params: { path: { date: op.date } }, body: op.body }));
		}
	}

	// 送信を試み、通信不能なら端末に保存
	async function run<T>(op: OutboxOp): Promise<{ queued: true } | { queued: false; result: T }> {
		if (online.value) {
			try {
				return { queued: false, result: await send(op) as T };
			} catch (err) {
				if (!isNetworkError(err)) throw err;
			}
		}
		enqueue(op);
		return { queued: true };
	}

	// 先頭から順に送信, 通信不能・5xxで中断
	async function flush(): Promise<number> {
		if (syncing.value || pending.value.length === 0 || !me.value) return 0;
		syncing.value = true;
		let sent = 0;
		try {
			for (const item of [...items.value]) {
				if (item.error) continue;
				try {
					await send(item.op);
					remove(item.key);
					sent++;
				} catch (err) {
					if (!isNetworkError(err) && err instanceof ApiError && err.status >= 400 && err.status < 500 && err.status !== 401) {
						save(items.value.map((i) => (i.key === item.key ? { ...i, error: err.message } : i)));
						continue;
					}
					break;
				}
			}
		} finally {
			syncing.value = false;
		}
		return sent;
	}

	async function retryFailed() {
		save(items.value.map((i) => ({ ...i, error: undefined })));
		return flush();
	}

	return { items, pending, failed, online, syncing, enqueue, remove, updatePendingAdd, run, flush, retryFailed };
}
