// MCPサーバへのプロキシ
export default defineEventHandler((event) => {
	const { mecaloBeApiUri } = useRuntimeConfig(event);
	return proxyRequest(event, new URL('/mcp', mecaloBeApiUri).toString());
});
