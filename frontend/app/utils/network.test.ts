import { describe, expect, it } from 'vitest';
import { ApiError } from '~/composables/api/client';
import { isNetworkError } from '~/composables/useOutbox';

describe('isNetworkError', () => {
	it('treats fetch failures and gateway errors as offline', () => {
		expect(isNetworkError(new TypeError('Failed to fetch'))).toBe(true);
		expect(isNetworkError(new ApiError(502, undefined))).toBe(true);
		expect(isNetworkError(new ApiError(504, undefined))).toBe(true);
	});

	it('does not queue client errors', () => {
		expect(isNetworkError(new ApiError(400, undefined))).toBe(false);
		expect(isNetworkError(new ApiError(404, undefined))).toBe(false);
	});
});
