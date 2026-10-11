import type { Schemas } from '~/composables/api/client';

export const slots: Schemas['MealSlot'][] = ['breakfast', 'lunch', 'dinner', 'snack'];

export function defaultSlotForNow(now = new Date()): Schemas['MealSlot'] {
	const h = now.getHours();
	if (h < 10) return 'breakfast';
	if (h < 15) return 'lunch';
	if (h < 21) return 'dinner';
	return 'snack';
}

export function formatQuantity(q: Schemas['Quantity']): string {
	const amount = Math.round(q.amount * 100) / 100;
	return q.unit === 'g' ? `${amount}g` : `${amount}${q.unit}`;
}

// 未同期の追加記録に付ける仮ID
export const pendingPrefix = 'pending:';

export function newClientRef(): string {
	return crypto.randomUUID();
}
