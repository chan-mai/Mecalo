// v-reveal: 時間差表示
export default defineNuxtPlugin((nuxtApp) => {
	const step = 90;
	let order = 0;
	let last = 0;

	const observer = new IntersectionObserver((entries) => {
		for (const entry of entries) {
			if (!entry.isIntersecting) continue;
			const now = performance.now();
			order = now - last < 500 ? order + 1 : 0;
			last = now;
			const el = entry.target as HTMLElement;
			el.style.transitionDelay = `${Math.min(order, 8) * step}ms`;
			el.classList.add('is-revealed');
			observer.unobserve(el);
		}
	}, { rootMargin: '0px 0px -8% 0px' });

	nuxtApp.vueApp.directive<HTMLElement>('reveal', {
		mounted(el) {
			el.classList.add('reveal');
			observer.observe(el);
		},
		unmounted(el) {
			observer.unobserve(el);
		},
	});
});
