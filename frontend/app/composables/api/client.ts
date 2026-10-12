import createClient from 'openapi-fetch';
import { useConnectivity } from '../useConnectivity';
import type { Middleware } from 'openapi-fetch';
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

// 中継先のBEへ届かなかった応答
export function isGatewayStatus(status: number): boolean {
	return status === 502 || status === 503 || status === 504;
}

// 応答の有無でサーバーへの到達可否を更新
export const connectivityMiddleware: Middleware = {
	onResponse({ response }) {
		useConnectivity().reachable.value = !isGatewayStatus(response.status);
	},
	onError() {
		useConnectivity().reachable.value = false;
	},
};

let client: ReturnType<typeof createClient<paths>> | undefined;

export function useApi() {
	if (!client) {
		client = createClient<paths>({
			baseUrl: '/api',
			credentials: 'same-origin',
			headers: { 'X-Mecalo-Request': '1' },
		});
		client.use(connectivityMiddleware);
	}
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
