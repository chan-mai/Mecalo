<script setup lang="ts">
import gsap from 'gsap';
import type { CurtainKind } from '~/composables/useEntryCurtain';
import { registerCurtain } from '~/composables/useEntryCurtain';

const { t } = useI18n();
const { me } = useAuth();
const motion = usePreferredReducedMotion();
const { takePending, loginKind } = useEntryCurtain();
const root = useTemplateRef<HTMLDivElement>('root');

const restored = takePending();
const shown = ref(restored);
const kind = ref<CurtainKind | undefined>();

// ロゴ3色
const blobs = ['#B9DEFF', '#F6BED6', '#DDD6FF'];
const dots = [
	{ cx: 38, cy: 26.6, fill: '#B9DEFF' },
	{ cx: 26, cy: 48.1, fill: '#F6BED6' },
	{ cx: 50, cy: 48.1, fill: '#DDD6FF' },
];

function q(sel: string): Element[] {
	return root.value ? gsap.utils.toArray<Element>(sel, root.value) : [];
}

function place(sel: string, o: { x: number; y: number }, scale: number) {
	const w = window.innerWidth;
	const h = window.innerHeight;
	const d = Math.hypot(Math.max(o.x, w - o.x), Math.max(o.y, h - o.y)) * 2 + 4;
	gsap.set(q(sel), { left: o.x, top: o.y, width: d, height: d, xPercent: -50, yPercent: -50, scale });
}

function center() {
	return { x: window.innerWidth / 2, y: window.innerHeight / 2 };
}

// ロゴ円登場
function markIn(tl: gsap.core.Timeline, at: number) {
	const els = q('.curtain-dot');
	if (kind.value === 'onboarded') {
		// 離れた3つの円が寄ってロゴの形へ
		tl.fromTo(
			els,
			{ scale: 1, x: gsap.utils.wrap([0, -36, 36]), y: gsap.utils.wrap([-30, 22, 22]), autoAlpha: 0 },
			{ x: 0, y: 0, autoAlpha: 1, duration: 0.8, ease: 'expo.out', stagger: 0.05 },
			at,
		);
	} else if (kind.value === 'setup') {
		// 上から順に弾んで着地, 対話の入力中表示へ
		tl.fromTo(els, { y: -48, autoAlpha: 0 }, { y: 0, autoAlpha: 1, duration: 0.7, ease: 'bounce.out', stagger: 0.1 }, at);
	} else if (kind.value === 'first') {
		tl.fromTo(els, { scale: 0, autoAlpha: 1 }, { scale: 1, duration: 0.6, ease: 'back.out(2)', stagger: 0.12 }, at);
	} else {
		tl.fromTo(
			els,
			{ scale: 0, rotation: -90, autoAlpha: 1, transformOrigin: '50% 50%' },
			{ scale: 1, rotation: 0, duration: 0.5, ease: 'expo.out', stagger: 0.06 },
			at,
		);
	}
	tl.fromTo(q('.curtain-label'), { autoAlpha: 0, y: 10 }, { autoAlpha: 1, y: 0, duration: 0.5, ease: 'expo.out' }, at + 0.25);
}

let covering: gsap.core.Timeline | undefined;

async function cover(k: CurtainKind, origin?: { x: number; y: number }) {
	if (motion.value === 'reduce') return;
	kind.value = k;
	shown.value = true;
	await nextTick();
	const o = origin ?? center();
	place('.curtain-blob', o, 0);
	place('.curtain-sheet', o, 0);
	gsap.set(q('.curtain-mark'), { autoAlpha: 1, scale: 1 });
	gsap.set(q('.curtain-dot, .curtain-label'), { autoAlpha: 0 });
	const tl = gsap.timeline();
	covering = tl;
	tl.to(q('.curtain-blob'), { scale: 1, duration: 0.85, ease: 'expo.inOut', stagger: 0.09 }, 0).to(
		q('.curtain-sheet'),
		{ scale: 1, duration: 0.75, ease: 'expo.inOut' },
		0.3,
	);
	markIn(tl, 0.75);
	// 初回は読み取れる間を確保
	tl.to({}, { duration: k === 'return' || k === 'setup' ? 0.15 : 0.45 });
	await tl;
}

async function reveal() {
	if (!shown.value) return;
	covering?.progress(1).kill();
	covering = undefined;
	// 画面中央へ縮む円で遷移先を表示
	place('.curtain-sheet', center(), 1);
	gsap.set(q('.curtain-blob'), { autoAlpha: 0 });
	root.value?.classList.remove('is-covered');
	const tl = gsap.timeline();
	tl.to(q('.curtain-mark'), { autoAlpha: 0, scale: 0.85, duration: 0.35, ease: 'power2.in' }, 0).to(
		q('.curtain-sheet'),
		{ scale: 0, duration: 0.75, ease: 'expo.inOut' },
		0.15,
	);
	await tl;
	shown.value = false;
	kind.value = undefined;
}

onMounted(async () => {
	registerCurtain({ cover, reveal });
	if (!restored) return;
	if (motion.value === 'reduce' || !me.value) {
		shown.value = false;
		return;
	}
	place('.curtain-sheet', center(), 1);
	kind.value = await loginKind();
	await nextTick();
	const tl = gsap.timeline();
	markIn(tl, 0);
	tl.to({}, { duration: 0.4 });
	await tl;
	await reveal();
});

onBeforeUnmount(() => registerCurtain(undefined));
</script>

<template>
<div v-if="shown" ref="root" class="curtain" :class="{ 'is-covered': restored }" aria-hidden="true">
	<span v-for="c in blobs" :key="c" class="curtain-disc curtain-blob" :style="{ background: c }"></span>
	<span class="curtain-disc curtain-sheet"></span>
	<div class="curtain-mark">
		<svg viewBox="0 0 76 76" width="76" height="76">
			<circle v-for="d in dots" :key="d.fill" class="curtain-dot" :cx="d.cx" :cy="d.cy" r="20.27" :fill="d.fill" style="mix-blend-mode: multiply"/>
		</svg>
		<p v-if="kind === 'return' && me" class="curtain-label display mt-5 text-xl">{{ me.displayName }}</p>
		<p v-else-if="kind === 'setup'" class="curtain-label curtain-typing mt-6"><span></span><span></span><span></span></p>
		<img v-else-if="kind" src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="curtain-label mt-6 h-7 w-auto"/>
	</div>
</div>
</template>

<style scoped>
.curtain {
	position: fixed;
	inset: 0;
	z-index: 110;
	overflow: hidden;
}

.curtain.is-covered {
	background: var(--background);
}

.curtain-disc {
	position: absolute;
	border-radius: 9999px;
}

.curtain-blob {
	mix-blend-mode: multiply;
}

.curtain-sheet {
	background: var(--background);
}

.curtain-mark {
	position: absolute;
	inset: 0;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
}

.curtain-typing {
	display: flex;
	gap: 0.375rem;
	border: 1px solid rgb(255 255 255 / 0.7);
	border-radius: 1.25rem;
	border-end-start-radius: 0.375rem;
	background: var(--glass-strong);
	padding: 0.85rem 1.1rem;
}

.curtain-typing span {
	inline-size: 0.4rem;
	block-size: 0.4rem;
	border-radius: 9999px;
	background: var(--muted-foreground);
	animation: curtain-typing 1s ease-in-out infinite;
}

.curtain-typing span:nth-child(2) {
	animation-delay: 0.15s;
}

.curtain-typing span:nth-child(3) {
	animation-delay: 0.3s;
}

@keyframes curtain-typing {
	50% {
		opacity: 0.35;
		transform: translateY(-3px);
	}
}
</style>
