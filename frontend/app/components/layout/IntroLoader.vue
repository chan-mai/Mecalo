<script setup lang="ts">
import gsap from 'gsap';

const { t } = useI18n();
const motion = usePreferredReducedMotion();
const root = useTemplateRef<HTMLDivElement>('root');

// 起動時1回のみ表示
const seen = useSessionStorage('mecalo-intro-seen', false);
const shown = ref(!seen.value);

onMounted(() => {
	if (!shown.value) return;
	seen.value = true;
	if (motion.value === 'reduce' || !root.value) {
		shown.value = false;
		return;
	}
	const tl = gsap.timeline({ onComplete: () => (shown.value = false) });
	tl.from('.intro-dot', { scale: 0, duration: 0.6, stagger: 0.12, ease: 'back.out(2)' }, 0)
		.from('.intro-word span', { yPercent: 110, duration: 0.7, stagger: 0.04, ease: 'expo.out' }, 0.3)
		.to('.intro-fill', { scaleX: 1, duration: 0.9, ease: 'power2.inOut' }, 0.2)
		.to(root.value, { autoAlpha: 0, duration: 0.6, ease: 'power2.out' }, '+=0.15');
	// 描画間引き時も操作を妨げない上限
	useTimeoutFn(() => {
		tl.kill();
		shown.value = false;
	}, 3000);
});
</script>

<template>
<div v-if="shown" ref="root" class="intro" aria-hidden="true">
	<div class="flex items-center gap-3">
		<span class="intro-dot bg-[var(--brand-primary)]"></span>
		<span class="intro-dot bg-[var(--brand-secondary)]"></span>
		<span class="intro-dot bg-[var(--brand-tertiary)]"></span>
	</div>
	<p class="intro-word display mt-6 overflow-hidden text-2xl">
		<span v-for="(c, i) in t('layout.appName').toUpperCase()" :key="i" class="inline-block">{{ c }}</span>
	</p>
	<div class="mt-6 h-px w-40 overflow-hidden bg-[var(--hairline)]">
		<div class="intro-fill h-full origin-left scale-x-0 bg-foreground"></div>
	</div>
</div>
</template>

<style scoped>
.intro {
	position: fixed;
	inset: 0;
	z-index: 100;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	background: var(--brand-cream);
}

.intro-dot {
	width: 1.25rem;
	height: 1.25rem;
	border-radius: 50%;
}
</style>
