import * as v from 'valibot';

// YYYY-MM-DD形式の暦日, 未入力も同じ扱い
export const isoDate = v.pipe(v.string('errors.validation.date'), v.isoDate('errors.validation.date'));
