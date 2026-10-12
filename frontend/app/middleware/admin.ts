// 管理者以外はホームへ
export default defineNuxtRouteMiddleware(async () => {
	const { ensureLoaded } = useAuth();
	const me = await ensureLoaded();
	if (me?.role !== 'admin') return navigateTo('/');
});
