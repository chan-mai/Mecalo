import { describe, expect, it } from 'vitest';
import { defaultQuantity, gramsOf, nutrientsOf, scaleProfile, unitOptions } from './quantity';
import type { Schemas } from '~/composables/api/client';

const rev: Schemas['FoodRevision'] = {
	id: 'r1',
	number: 1,
	name: 'ごはん',
	brand: '',
	aliases: [],
	createdAt: '2026-10-12T00:00:00Z',
	source: { datasetId: 'user', reference: '', url: '' },
	portions: [{ unit: '杯', grams: 150 }],
	nutrients: { energy: { amount: 156, status: 'measured', coverage: 1 } },
};

const burger: Schemas['FoodRevision'] = {
	...rev,
	name: 'バーガー',
	basis: { unit: '1個' },
	portions: [],
	nutrients: { energy: { amount: 500, status: 'measured', coverage: 1 } },
};

describe('quantity', () => {
	it('converts portions to grams', () => {
		expect(gramsOf(rev, { amount: 2, unit: '杯' })).toBe(300);
		expect(gramsOf(rev, { amount: 80, unit: 'g' })).toBe(80);
		expect(gramsOf(rev, { amount: 1, unit: '個' })).toBeUndefined();
	});

	it('prefers the basis, then the first portion as default', () => {
		expect(defaultQuantity(rev)).toEqual({ amount: 1, unit: '杯' });
		expect(defaultQuantity({ ...rev, portions: [] })).toEqual({ amount: 100, unit: 'g' });
		expect(defaultQuantity(burger)).toEqual({ amount: 1, unit: '1個' });
	});

	it('scales values and keeps status', () => {
		expect(scaleProfile(rev.nutrients, 1.5).energy).toEqual({ amount: 234, status: 'measured', coverage: 1 });
		expect(nutrientsOf(rev, { amount: 1, unit: '杯' })?.energy?.amount).toBe(234);
	});

	it('handles per-unit values without weight', () => {
		expect(nutrientsOf(burger, { amount: 2, unit: '1個' })?.energy?.amount).toBe(1000);
		expect(gramsOf(burger, { amount: 2, unit: '1個' })).toBeUndefined();
		expect(nutrientsOf(burger, { amount: 100, unit: 'g' })).toBeUndefined();
		expect(unitOptions(burger).map((u) => u.unit)).toEqual(['1個']);
	});

	it('converts grams with a weighted basis', () => {
		const known = { ...burger, basis: { unit: '1個', grams: 200 } };
		expect(nutrientsOf(known, { amount: 50, unit: 'g' })?.energy?.amount).toBe(125);
		expect(unitOptions(known).map((u) => u.unit)).toEqual(['1個', 'g']);
	});
});
