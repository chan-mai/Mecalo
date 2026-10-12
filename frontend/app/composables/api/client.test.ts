import { describe, expect, it } from 'vitest';
import { useConnectivity } from '../useConnectivity';
import { connectivityMiddleware } from './client';
import type { MiddlewareCallbackParams } from 'openapi-fetch';

const params = {} as MiddlewareCallbackParams;

describe('connectivityMiddleware', () => {
	it('marks the server unreachable on fetch failures and gateway errors', async () => {
		const { reachable, connected } = useConnectivity();
		await connectivityMiddleware.onError?.({ ...params, error: new TypeError('Failed to fetch') });
		expect(reachable.value).toBe(false);
		expect(connected.value).toBe(false);

		await connectivityMiddleware.onResponse?.({ ...params, response: new Response(null, { status: 200 }), options: {} as never });
		expect(reachable.value).toBe(true);

		await connectivityMiddleware.onResponse?.({ ...params, response: new Response(null, { status: 502 }), options: {} as never });
		expect(reachable.value).toBe(false);
	});

	it('treats client errors as reachable', async () => {
		const { reachable } = useConnectivity();
		reachable.value = false;
		await connectivityMiddleware.onResponse?.({ ...params, response: new Response(null, { status: 404 }), options: {} as never });
		expect(reachable.value).toBe(true);
	});
});
