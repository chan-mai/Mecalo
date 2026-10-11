import type { Schemas } from '~/composables/api/client';

export type NutrientCode =
	| 'energy' | 'protein' | 'fat' | 'carbohydrate' | 'fiber' | 'salt_equivalent' | 'cholesterol'
	| 'sodium' | 'potassium' | 'calcium' | 'magnesium' | 'phosphorus' | 'iron' | 'zinc' | 'copper'
	| 'manganese' | 'iodine' | 'selenium' | 'chromium' | 'molybdenum'
	| 'vitamin_a' | 'vitamin_d' | 'vitamin_e' | 'vitamin_k' | 'vitamin_b1' | 'vitamin_b2'
	| 'niacin_equivalent' | 'vitamin_b6' | 'vitamin_b12' | 'folate' | 'pantothenic_acid' | 'biotin' | 'vitamin_c';

export const macroCodes: NutrientCode[] = ['protein', 'fat', 'carbohydrate'];

export const unitLabel: Record<Schemas['NutrientUnit'], string> = {
	kcal: 'kcal',
	g: 'g',
	mg: 'mg',
	ug: 'µg',
};

export function formatInt(value: number): string {
	return Math.round(value).toLocaleString('en-US');
}

export function formatAmount(amount: number): string {
	const abs = Math.abs(amount);
	if (abs >= 100) return formatInt(amount);
	if (abs >= 10) return (Math.round(amount * 10) / 10).toLocaleString();
	return (Math.round(amount * 100) / 100).toLocaleString();
}

export function coveragePercent(v: Schemas['NutrientValue'] | undefined): number {
	return Math.round((v?.coverage ?? 0) * 100);
}

export function amountOf(profile: Schemas['NutrientProfile'] | undefined, code: string): number {
	return profile?.[code]?.amount ?? 0;
}

export const levelTone: Record<Schemas['Level'], string> = {
	deficient: 'bg-danger-soft text-danger',
	low: 'bg-warning-soft text-warning',
	adequate: 'bg-success-soft text-success',
	high: 'bg-warning-soft text-warning',
	excessive: 'bg-danger-soft text-danger',
	unknown: 'border-transparent text-muted',
	none: 'border-transparent text-muted',
};

// 判定結果ごとの向きを示すアイコン
export const levelIcon: Partial<Record<Schemas['Level'], string>> = {
	deficient: 'lucide:chevrons-down',
	low: 'lucide:chevron-down',
	adequate: 'lucide:check',
	high: 'lucide:chevron-up',
	excessive: 'lucide:chevrons-up',
};

// 表の到達度バー, 右端は目標上限(なければ下限), 適正は全幅
export function barProgress(evaluation: Schemas['Evaluation']): number {
	if (evaluation.level === 'adequate') return 1;
	const ref = evaluation.upper ?? evaluation.lower;
	if (!ref) return 0;
	return Math.min(evaluation.value.amount / ref, 1);
}

// 判定結果ごとの文字色
export const levelText: Record<Schemas['Level'], string> = {
	deficient: 'text-danger',
	low: 'text-warning',
	adequate: 'text-success',
	high: 'text-warning',
	excessive: 'text-danger',
	unknown: 'text-muted',
	none: 'text-muted',
};

// 判定結果ごとのメーターの色, 値は文字でも表示
export const levelColor: Record<Schemas['Level'], string> = {
	deficient: 'var(--brand-secondary)',
	low: 'var(--chart-carbs-soft)',
	adequate: 'var(--chart-fiber-soft)',
	high: 'var(--chart-carbs-soft)',
	excessive: 'var(--brand-secondary)',
	unknown: 'var(--control-border)',
	none: 'var(--control-border)',
};

// 目標の下限・上限に対する進捗率(0〜1.5)
export function progressOf(evaluation: Schemas['Evaluation']): number {
	const ref = evaluation.lower ?? evaluation.upper;
	if (!ref) return 0;
	return Math.min(evaluation.value.amount / ref, 1.5);
}

// 目標範囲からの過不足, 範囲内はundefined
export function diffOf(evaluation: Schemas['Evaluation']): number | undefined {
	const { amount } = evaluation.value;
	if (evaluation.lower !== undefined && amount < evaluation.lower) return amount - evaluation.lower;
	if (evaluation.upper !== undefined && amount > evaluation.upper) return amount - evaluation.upper;
	return undefined;
}

export function formatSigned(value: number): string {
	return `${value > 0 ? '+' : '−'}${formatAmount(Math.abs(value))}`;
}

// 注意喚起の優先度順に並べた評価
export function attentionOrder(evaluations: Schemas['Evaluation'][]): Schemas['Evaluation'][] {
	const rank: Partial<Record<Schemas['Level'], number>> = { excessive: 0, high: 1, deficient: 2, low: 3 };
	return evaluations
		.filter((evaluation) => evaluation.code !== 'energy' && rank[evaluation.level] !== undefined)
		.sort((a, b) => (rank[a.level] ?? 9) - (rank[b.level] ?? 9));
}
