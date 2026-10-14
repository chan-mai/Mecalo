// OAuthのメタデータをバックエンドへ転送
const paths = ['/.well-known/oauth-protected-resource', '/.well-known/oauth-protected-resource/mcp', '/.well-known/oauth-authorization-server'];

export default defineEventHandler((event) => {
	const path = event.path.split('?')[0];
	if (!path || !paths.includes(path)) return;
	const { mecaloBeApiUri } = useRuntimeConfig(event);
	return proxyRequest(event, new URL(path, mecaloBeApiUri).toString());
});
