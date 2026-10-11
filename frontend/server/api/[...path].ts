// /api配下をバックエンドにプロキシ
export default defineEventHandler((event) => {
	const { mecaloBeApiUri } = useRuntimeConfig(event);
	const target = new URL(event.path, mecaloBeApiUri);
	return proxyRequest(event, target.toString(), {
		headers: { 'X-Forwarded-Host': getRequestHost(event), 'X-Forwarded-Proto': getRequestProtocol(event) },
	});
});
