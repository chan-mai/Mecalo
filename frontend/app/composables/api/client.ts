import createClient from 'openapi-fetch';
import type { components, paths } from '~/types/api.gen';

export type Schemas = components['schemas'];

export class ApiError extends Error {
	constructor(
		public readonly status: number,
		public readonly problem: Schemas['Problem'] | undefined,
	) {
		super(problem?.detail ?? problem?.title ?? `HTTP ${status}`);
	}
}

let client: ReturnType<typeof createClient<paths>> | undefined;

export function useApi() {
	client ??= createClient<paths>({
		baseUrl: '/api',
		credentials: 'same-origin',
		headers: { 'X-Mecalo-Request': '1' },
	});
	return client;
}

// エラー時ApiError送出
export async function unwrap<T>(promise: Promise<{ data?: T; error?: unknown; response: Response }>): Promise<T> {
	const { data, error, response } = await promise;
	if (!response.ok) {
		throw new ApiError(response.status, error as Schemas['Problem'] | undefined);
	}
	return data as T;
}
