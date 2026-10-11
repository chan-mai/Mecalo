import { describe, expect, it } from 'vitest';
import { attentionOrder, diffOf, formatAmount, formatSigned, progressOf } from './nutrients';
import type { Schemas } from '~/composables/api/client';

function evaluation(code: string, level: Schemas['Level'], amount: number, lower?: number): Schemas['Evaluation'] {
	return { code, level, lower, value: { amount, status: 'calculated', coverage: 1 } };
}

describe('nutrients', () => {
	it('formats by magnitude', () => {
		expect(formatAmount(1234.5)).toBe((1235).toLocaleString());
		expect(formatAmount(12.34)).toBe((12.3).toLocaleString());
		expect(formatAmount(0.123)).toBe((0.12).toLocaleString());
	});

	it('orders excess before deficiency and skips energy', () => {
		const list = [
			evaluation('iron', 'low', 5, 6),
			evaluation('energy', 'excessive', 3000, 2000),
			evaluation('salt_equivalent', 'high', 9),
			evaluation('vitamin_c', 'adequate', 100, 100),
		];
		expect(attentionOrder(list).map((x) => x.code)).toEqual(['salt_equivalent', 'iron']);
	});

	it('measures distance from target range', () => {
		expect(diffOf(evaluation('iron', 'low', 4, 6))).toBe(-2);
		expect(diffOf({ ...evaluation('salt_equivalent', 'high', 9), upper: 7.5 })).toBe(1.5);
		expect(diffOf(evaluation('vitamin_c', 'adequate', 100, 100))).toBeUndefined();
		expect(formatSigned(-2)).toBe(`−${(2).toLocaleString()}`);
	});

	it('caps progress', () => {
		expect(progressOf(evaluation('iron', 'excessive', 100, 6))).toBe(1.5);
		expect(progressOf(evaluation('biotin', 'none', 10))).toBe(0);
	});
});
