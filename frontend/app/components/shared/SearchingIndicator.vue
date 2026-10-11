<script setup lang="ts">
import gsap from 'gsap';

const { t } = useI18n();
const root = useTemplateRef<HTMLDivElement>('root');
const motion = usePreferredReducedMotion();
let ctx: gsap.Context | undefined;

onMounted(() => {
	if (motion.value === 'reduce' || !root.value) return;
	ctx = gsap.context(() => {
		gsap.timeline({ repeat: -1 })
			.to('.searching-orb', { y: -18, scaleX: 0.9, scaleY: 1.12, duration: 0.32, ease: 'power2.out', stagger: 0.1 })
			.to('.searching-orb', { y: 0, scaleX: 1.15, scaleY: 0.85, duration: 0.28, ease: 'power2.in', stagger: 0.1 }, 0.32)
			.to('.searching-orb', { scaleX: 1, scaleY: 1, duration: 0.2, ease: 'elastic.out(1, 0.4)', stagger: 0.1 }, 0.6);
		gsap.to('.searching-label span', { opacity: 0.25, duration: 0.5, stagger: { each: 0.06, repeat: -1, yoyo: true } });
		gsap.from(root.value, { autoAlpha: 0, y: 8, duration: 0.4, ease: 'expo.out' });
	}, root.value);
});

onBeforeUnmount(() => ctx?.revert());
</script>

<template>
<div ref="root" class="flex flex-col items-center py-10" role="status">
	<div class="flex h-8 items-end gap-2.5" aria-hidden="true">
		<span class="searching-orb bg-[var(--brand-primary)]"></span>
		<span class="searching-orb bg-[var(--brand-secondary)]"></span>
		<span class="searching-orb bg-[var(--brand-tertiary)]"></span>
	</div>
	<p class="searching-label eyebrow mt-4" aria-hidden="true">
		<span v-for="(c, i) in 'SEARCHING'" :key="i" class="inline-block">{{ c }}</span>
	</p>
	<span class="sr-only">{{ t('common.loading') }}</span>
</div>
</template>

<style scoped>
.searching-orb {
	display: block;
	width: 0.875rem;
	height: 0.875rem;
	border-radius: 50%;
	transform-origin: 50% 100%;
	box-shadow: 0 6px 12px -6px rgb(37 50 74 / 0.4);
}
</style>
