import dayjs from 'dayjs';
import customParseFormat from 'dayjs/plugin/customParseFormat';
import isoWeek from 'dayjs/plugin/isoWeek';
import 'dayjs/locale/ja';

dayjs.extend(customParseFormat);
dayjs.extend(isoWeek);

// APIとURLで扱う暦日の形式
const isoFormat = 'YYYY-MM-DD';

export function todayIso(): string {
	return dayjs().format(isoFormat);
}

export function addDays(date: string, n: number): string {
	return dayjs(date, isoFormat).add(n, 'day').format(isoFormat);
}

export function isIsoDate(value: unknown): value is string {
	return typeof value === 'string' && dayjs(value, isoFormat, true).isValid();
}

// 月曜始まり
export function startOfWeek(date: string): string {
	return dayjs(date, isoFormat).startOf('isoWeek').format(isoFormat);
}

export function daysBetween(from: string, to: string): number {
	return dayjs(to, isoFormat).diff(dayjs(from, isoFormat), 'day');
}

// 表示用の書式
export function formatDate(value: string, format: string, locale: string): string {
	return dayjs(value).locale(locale).format(format);
}
