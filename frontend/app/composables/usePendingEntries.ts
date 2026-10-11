import type { Schemas } from './api/client';
import { pendingPrefix } from '~/utils/meals';

// 未同期の操作を画面表示用の記録へ反映
export function usePendingEntries(date: Ref<string>) {
	const { items } = useOutbox();

	const added = computed(() => items.value.flatMap((item): Schemas['MealEntry'][] => {
		const op = item.op;
		if (op.kind !== 'addMeal' || op.body.date !== date.value) return [];
		const nutrients: Schemas['NutrientProfile'] = {};
		if (op.preview.kcal !== undefined) nutrients.energy = { amount: op.preview.kcal, status: 'estimated', coverage: 1 };
		return [{
			id: pendingPrefix + op.body.clientRef,
			date: op.body.date,
			slot: op.body.slot,
			foodId: op.body.foodId,
			foodRevisionId: '',
			foodName: op.preview.foodName,
			brand: '',
			source: { datasetId: '', reference: '' },
			quantity: op.body.quantity,
			grams: op.preview.grams ?? 0,
			nutrients,
			clientRef: op.body.clientRef,
			createdAt: item.createdAt,
			updatedAt: item.createdAt,
		}];
	}));

	const deletedIds = computed(() => new Set(items.value.flatMap((i) => (i.op.kind === 'deleteMeal' ? [i.op.id] : []))));
	const waitingIds = computed(() => new Set(items.value.flatMap((i) => (i.op.kind === 'updateMeal' ? [i.op.id] : []))));

	function merge(entries: Schemas['MealEntry'][]): Schemas['MealEntry'][] {
		return [...entries.filter((entry) => !deletedIds.value.has(entry.id)), ...added.value];
	}

	return { added, waitingIds, merge };
}
