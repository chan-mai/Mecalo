import type { Schemas } from '~/composables/api/client';

type Rev = Pick<Schemas['FoodRevision'], 'basis' | 'portions' | 'nutrients'>;

// 重量不明の基準単位とその換算はundefined
export function gramsOf(rev: Rev, q: Schemas['Quantity']): number | undefined {
	const { basis } = rev;
	if (basis && q.unit === basis.unit) return basis.grams === undefined ? undefined : basis.grams * q.amount;
	if (q.unit === 'g') return basis && basis.grams === undefined ? undefined : q.amount;
	const p = rev.portions.find((x) => x.unit === q.unit);
	return p ? p.grams * q.amount : undefined;
}

// 指定数量分の栄養値, 換算できない単位はundefined
export function nutrientsOf(rev: Rev, q: Schemas['Quantity']): Schemas['NutrientProfile'] | undefined {
	const { basis } = rev;
	if (!basis) {
		const grams = gramsOf(rev, q);
		return grams === undefined ? undefined : scaleProfile(rev.nutrients, grams / 100);
	}
	if (q.unit === basis.unit) return scaleProfile(rev.nutrients, q.amount);
	const grams = gramsOf(rev, q);
	return grams === undefined || !basis.grams ? undefined : scaleProfile(rev.nutrients, grams / basis.grams);
}

export function defaultQuantity(rev: Rev): Schemas['Quantity'] {
	if (rev.basis) return { amount: 1, unit: rev.basis.unit };
	const first = rev.portions[0];
	return first ? { amount: 1, unit: first.unit } : { amount: 100, unit: 'g' };
}

// 選択できる単位, 重量不明の基準量ではgを含めない
export function unitOptions(rev: Rev): { unit: string; grams?: number }[] {
	const out: { unit: string; grams?: number }[] = [];
	if (rev.basis) out.push({ unit: rev.basis.unit, grams: rev.basis.grams });
	for (const p of rev.portions) out.push({ unit: p.unit, grams: p.grams });
	if (!rev.basis || rev.basis.grams !== undefined) out.push({ unit: 'g' });
	return out;
}

// 量のみ倍率適用, 状態と充足率は維持
export function scaleProfile(profile: Schemas['NutrientProfile'], factor: number): Schemas['NutrientProfile'] {
	const out: Schemas['NutrientProfile'] = {};
	for (const [code, v] of Object.entries(profile)) {
		out[code] = { ...v, amount: v.amount * factor };
	}
	return out;
}
