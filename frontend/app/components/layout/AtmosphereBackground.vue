<script setup lang="ts">
import gsap from 'gsap';

const root = useTemplateRef<HTMLDivElement>('root');
const motion = usePreferredReducedMotion();

// 位置・大きさはviewport比, 色はブランド色
const orbs = [
	{ x: -12, y: -8, size: 62, from: 'var(--brand-primary)', to: 'var(--brand-tertiary)' },
	{ x: 58, y: 6, size: 48, from: 'var(--brand-secondary)', to: 'var(--brand-cream)' },
	{ x: 64, y: 58, size: 70, from: 'var(--brand-tertiary)', to: 'var(--brand-primary)' },
	{ x: -18, y: 66, size: 44, from: 'var(--brand-secondary)', to: 'var(--brand-tertiary)' },
];

let ctx: gsap.Context | undefined;

onMounted(() => {
	if (motion.value === 'reduce' || !root.value) return;
	ctx = gsap.context(() => {
		gsap.utils.toArray<HTMLElement>('.orb').forEach((el, i) => {
			gsap.to(el, {
				xPercent: gsap.utils.random(-10, 10),
				yPercent: gsap.utils.random(-12, 12),
				scale: gsap.utils.random(0.92, 1.08),
				duration: gsap.utils.random(14, 22),
				delay: i * -3,
				ease: 'sine.inOut',
				repeat: -1,
				yoyo: true,
			});
		});
	}, root.value);
});

onBeforeUnmount(() => ctx?.revert());
</script>

<template>
<div ref="root" class="atmosphere" aria-hidden="true">
	<div
		v-for="(orb, i) in orbs"
		:key="i"
		class="orb"
		:style="{ left: `${orb.x}vw`, top: `${orb.y}vh`, width: `${orb.size}vmax`, height: `${orb.size}vmax`, '--from': orb.from, '--to': orb.to }"
	></div>
	<svg class="threads" viewBox="0 0 100 100" preserveAspectRatio="none">
		<path d="M-5 30 C 25 18, 55 52, 105 38"/>
		<path d="M-5 78 C 30 62, 62 96, 105 70"/>
	</svg>
	<div class="grain"></div>
</div>
</template>

<style scoped>
.atmosphere {
	position: fixed;
	inset: 0;
	z-index: -1;
	overflow: hidden;
	background: linear-gradient(170deg, #eaf5ff 0%, var(--brand-cream) 48%, #f4f0ff 100%);
	pointer-events: none;
}

.orb {
	position: absolute;
	border-radius: 50%;
	background: radial-gradient(circle at 35% 30%, var(--from), var(--to) 70%, transparent 72%);
	filter: blur(48px);
	opacity: 0.75;
}

.threads {
	position: absolute;
	inset: 0;
	width: 100%;
	height: 100%;
	fill: none;
	stroke: rgb(255 255 255 / 0.9);
}

.threads path {
	stroke-width: 1px;
	vector-effect: non-scaling-stroke;
}

.grain {
	position: absolute;
	inset: 0;
	background-image: url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='160' height='160'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/></filter><rect width='100%' height='100%' filter='url(%23n)' opacity='0.5'/></svg>");
	mix-blend-mode: soft-light;
	opacity: 0.35;
}
</style>
