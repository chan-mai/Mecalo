import type { Schemas } from '~/composables/api/client';

export function gramsOf(rev: Schemas['FoodRevision'], q: Schemas['Quantity']): number | undefined {
	if (q.unit === 'g') return q.amount;
	const p = rev.portions.find((x) => x.unit === q.unit);
	return p ? p.grams * q.amount : undefined;
}

export function defaultQuantity(rev: Schemas['FoodRevision']): Schemas['Quantity'] {
	const first = rev.portions[0];
	return first ? { amount: 1, unit: first.unit } : { amount: 100, unit: 'g' };
}

// 100gあたりの値から指定量分を算出
export function scaleProfile(profile: Schemas['NutrientProfile'], grams: number): Schemas['NutrientProfile'] {
	const out: Schemas['NutrientProfile'] = {};
	for (const [code, v] of Object.entries(profile)) {
		out[code] = { ...v, amount: v.amount * grams / 100 };
	}
	return out;
}
