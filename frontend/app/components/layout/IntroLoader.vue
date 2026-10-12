<script setup lang="ts">
import gsap from 'gsap';

const motion = usePreferredReducedMotion();
const root = useTemplateRef<HTMLDivElement>('root');

// 起動時1回のみ表示
const seen = useSessionStorage('mecalo-intro-seen', false);
const shown = ref(!seen.value);

// ロゴ3色円
const dots = [
	{ cx: 38, cy: 26.6, fill: 'var(--brand-primary)' },
	{ cx: 26, cy: 48.1, fill: 'var(--brand-secondary)' },
	{ cx: 50, cy: 48.1, fill: 'var(--brand-tertiary)' },
];
const sparkles = [
	{ x: -110, y: -70, size: 16 },
	{ x: 112, y: -52, size: 11 },
	{ x: -86, y: 78, size: 10 },
	{ x: 100, y: 84, size: 14 },
];
// 中央で弾む間のロゴの円の倍率
const big = 3.2;

onMounted(() => {
	if (!shown.value) return;
	seen.value = true;
	if (motion.value === 'reduce' || !root.value) {
		shown.value = false;
		return;
	}
	const q = gsap.utils.selector(root.value);
	const mark = q('.intro-mark')[0];
	if (!mark) return;
	// ロゴ円を画面中央へ
	const r = mark.getBoundingClientRect();
	const dx = window.innerWidth / 2 - (r.left + r.width / 2);
	const dy = window.innerHeight / 2 - (r.top + r.height / 2);
	gsap.set(mark, { x: dx, y: dy, scale: big, transformOrigin: '50% 50%' });
	gsap.set(q('.intro-text'), { clipPath: 'inset(0 100% 0 0)' });

	const tl = gsap.timeline({ onComplete: () => (shown.value = false) });
	// 落下, 着地でつぶれて弾む
	q('.intro-dot').forEach((el, i) => {
		const at = 0.15 + i * 0.16;
		tl.fromTo(el, { y: -44, autoAlpha: 0 }, { y: 0, autoAlpha: 1, duration: 0.42, ease: 'power2.in' }, at)
			.to(el, { scaleX: 1.25, scaleY: 0.72, duration: 0.09, ease: 'power1.out', transformOrigin: '50% 100%' }, at + 0.42)
			.to(el, { scaleX: 1, scaleY: 1, duration: 0.6, ease: 'elastic.out(1.2, 0.35)', transformOrigin: '50% 100%' }, at + 0.51);
	});
	tl.fromTo(
		q('.intro-sparkle'),
		{ scale: 0, rotation: -45, autoAlpha: 1 },
		{ scale: 1, rotation: 0, duration: 0.4, ease: 'back.out(3)', stagger: 0.07, yoyo: true, repeat: 1, repeatDelay: 0.2 },
		0.95,
	)
		// 3つそろった後に小さく跳ねてロゴの位置
		.to(mark, { y: dy - 14, duration: 0.18, ease: 'power2.out' }, 1.15)
		.to(mark, { x: 0, y: 0, scale: 1, duration: 0.7, ease: 'expo.inOut' }, 1.33)
		.to(q('.intro-text'), { clipPath: 'inset(0 0% 0 0)', duration: 0.6, ease: 'expo.out' }, 1.75)
		// 画面中央へ縮む円で本体を表示
		.fromTo(root.value, { clipPath: 'circle(150% at 50% 50%)' }, { clipPath: 'circle(0% at 50% 50%)', duration: 0.75, ease: 'expo.inOut' }, '+=0.3');
	// 描画間引き時も操作を妨げない上限
	useTimeoutFn(() => {
		tl.kill();
		shown.value = false;
	}, 4500);
});
</script>

<template>
<div v-if="shown" ref="root" class="intro" aria-hidden="true">
	<span class="intro-glow intro-glow-a"></span>
	<span class="intro-glow intro-glow-b"></span>
	<svg
		v-for="(s, i) in sparkles"
		:key="i"
		class="intro-sparkle"
		viewBox="0 0 24 24"
		:width="s.size"
		:height="s.size"
		:style="{ left: `calc(50% + ${s.x}px)`, top: `calc(50% + ${s.y}px)` }"
	>
		<path d="M12 0c1 6 5 11 12 12-7 1-11 6-12 12-1-6-5-11-12-12 7-1 11-6 12-12z" :style="{ fill: dots[i % 3]?.fill }"/>
	</svg>
	<!-- ロゴの円と文字を分けて表示, 文字はロゴ画像の右側を切り出し -->
	<div class="intro-logo">
		<svg class="intro-mark" viewBox="0 0 76 76">
			<circle v-for="d in dots" :key="d.cx" class="intro-dot" :cx="d.cx" :cy="d.cy" r="20.27" :style="{ fill: d.fill }"/>
		</svg>
		<span class="intro-text"><img src="/logo.svg" alt="" width="427" height="76"></span>
	</div>
</div>
</template>

<style scoped>
.intro {
	--logo-h: 2.25rem;

	position: fixed;
	inset: 0;
	z-index: 100;
	display: grid;
	place-items: center;
	overflow: hidden;
	background: var(--brand-cream);
}

.intro-glow {
	position: absolute;
	inline-size: 70vmax;
	block-size: 70vmax;
	border-radius: 9999px;
	filter: blur(60px);
	opacity: 0.55;
}

.intro-glow-a {
	inset-block-start: -30vmax;
	inset-inline-start: -25vmax;
	background: radial-gradient(circle, var(--brand-primary), transparent 65%);
}

.intro-glow-b {
	inset-block-end: -32vmax;
	inset-inline-end: -25vmax;
	background: radial-gradient(circle, var(--brand-secondary), transparent 65%);
}

.intro-logo {
	position: relative;
	display: flex;
	block-size: var(--logo-h);
}

.intro-mark {
	inline-size: var(--logo-h);
	block-size: var(--logo-h);
	overflow: visible;
}

.intro-dot {
	visibility: hidden;
	mix-blend-mode: multiply;
}

/* ロゴ画像のx=76以降が文字 */
.intro-text {
	display: block;
	inline-size: calc(var(--logo-h) * 351 / 76);
	overflow: hidden;
}

.intro-text img {
	block-size: var(--logo-h);
	inline-size: auto;
	max-inline-size: none;
	margin-inline-start: calc(var(--logo-h) * -1);
}

.intro-sparkle {
	position: absolute;
	translate: -50% -50%;
	visibility: hidden;
}
</style>
