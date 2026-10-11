import { describe, expect, it } from 'vitest';
import { defaultQuantity, gramsOf, scaleProfile } from './quantity';
import type { Schemas } from '~/composables/api/client';

const rev: Schemas['FoodRevision'] = {
	id: 'r1', number: 1, name: 'ごはん', brand: '', aliases: [], createdAt: '2026-10-12T00:00:00Z',
	source: { datasetId: 'user', reference: '' },
	portions: [{ unit: '杯', grams: 150 }],
	nutrients: { energy: { amount: 156, status: 'measured', coverage: 1 } },
};

describe('quantity', () => {
	it('converts portions to grams', () => {
		expect(gramsOf(rev, { amount: 2, unit: '杯' })).toBe(300);
		expect(gramsOf(rev, { amount: 80, unit: 'g' })).toBe(80);
		expect(gramsOf(rev, { amount: 1, unit: '個' })).toBeUndefined();
	});

	it('prefers the first portion as default', () => {
		expect(defaultQuantity(rev)).toEqual({ amount: 1, unit: '杯' });
		expect(defaultQuantity({ ...rev, portions: [] })).toEqual({ amount: 100, unit: 'g' });
	});

	it('scales per-100g values and keeps status', () => {
		expect(scaleProfile(rev.nutrients, 150).energy).toEqual({ amount: 234, status: 'measured', coverage: 1 });
	});
});
