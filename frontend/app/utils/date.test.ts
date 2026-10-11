import { describe, expect, it } from 'vitest';
import { addDays, daysBetween, formatDate, isIsoDate, startOfWeek } from './date';

describe('date', () => {
	it('adds days across month boundaries', () => {
		expect(addDays('2026-10-31', 1)).toBe('2026-11-01');
		expect(addDays('2026-03-01', -1)).toBe('2026-02-28');
	});

	it('starts weeks on Monday', () => {
		expect(startOfWeek('2026-10-18')).toBe('2026-10-12');
		expect(startOfWeek('2026-10-12')).toBe('2026-10-12');
	});

	it('validates iso dates strictly', () => {
		expect(isIsoDate('2026-10-12')).toBe(true);
		expect(isIsoDate('2026-02-30')).toBe(false);
		expect(isIsoDate('2026/10/12')).toBe(false);
		expect(isIsoDate(undefined)).toBe(false);
	});

	it('counts days and formats by locale', () => {
		expect(daysBetween('2026-10-01', '2026-10-12')).toBe(11);
		expect(formatDate('2026-10-12', 'M/D(dd)', 'ja')).toBe('10/12(月)');
		expect(formatDate('2026-10-12', 'MMM D', 'en')).toBe('Oct 12');
	});
});
