const publicPages = ['/login', '/setup', '/signup'];

export default defineNuxtRouteMiddleware(async (to) => {
	const { ensureLoaded } = useAuth();
	const me = await ensureLoaded();
	if (!me && !publicPages.includes(to.path)) {
		return navigateTo({ path: '/login', query: to.fullPath === '/' ? {} : { returnTo: to.fullPath } });
	}
	if (me && publicPages.includes(to.path)) {
		return navigateTo('/');
	}
});
