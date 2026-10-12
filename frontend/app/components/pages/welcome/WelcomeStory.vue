<script setup lang="ts">
import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import type { WelcomeFrame, WelcomeScene } from '~/utils/three/welcome';
import { canUseWebGL } from '~/utils/webgl';

const emit = defineEmits<{ done: [from: Element] }>();
const { t } = useI18n();
const motion = usePreferredReducedMotion();
const reduced = computed(() => motion.value === 'reduce');

const chapters = [0, 1, 2, 3, 4] as const;
const last = chapters.length - 1;
const root = useTemplateRef<HTMLDivElement>('root');
const stage = useTemplateRef<HTMLDivElement>('stage');
const pin = useTemplateRef<HTMLDivElement>('pin');
// 0〜4, 小数は章の間
const progress = ref(0);
const current = computed(() => Math.round(progress.value));
const frame = shallowRef<WelcomeFrame>();
let scene: WelcomeScene | undefined;
let ctx: gsap.Context | undefined;

// 章に留まっている間の表示の強さ
function weight(c: number): number {
	return Math.min(Math.max(1 - Math.abs(progress.value - c) * 2.4, 0), 1);
}

// 左から書き出すように表示
function reveal(c: number): Record<string, string> {
	const w = weight(c);
	return { opacity: String(Math.min(w * 1.5, 1)), clipPath: `inset(0 ${(1 - w) * 100}% 0 0)` };
}

function at(p: { x: number; y: number } | undefined, dx = 0, dy = 0): Record<string, string> {
	return p ? { transform: `translate(${p.x + dx}px, ${p.y + dy}px)` } : { display: 'none' };
}

// 栄養素の一覧, 狭い画面は左上に固定
const { width: viewWidth } = useWindowSize();
const readoutAt = computed(() => (viewWidth.value < 640 ? { transform: 'translate(20px, 76px)' } : at(frame.value?.pebbles[2], 120, -70)));

// 単語の間でのみ改行, 助詞と句読点は前の語へ付ける
const segmenter = new Intl.Segmenter(undefined, { granularity: 'word' });

function phrases(key: string): string[][] {
	const out: string[] = [];
	for (const { segment } of segmenter.segment(t(key))) {
		const prev = out.length - 1;
		const joined = prev >= 0 && (/^[ぁ-ゖ、。,.!?\s]{1,2}$/.test(segment) || (/[ァ-ヿ]$/.test(out[prev] ?? '') && /^[ァ-ヿ]/.test(segment)));
		if (joined) out[prev] += segment;
		else out.push(segment);
	}
	return out.map((p) => [...p]);
}

// 0章: 進む時計
const now = useNow();
const clock = computed(() => {
	const d = now.value;
	const p = (n: number) => String(n).padStart(2, '0');
	const days = ['SUN', 'MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT'];
	return `${d.getFullYear()}.${p(d.getMonth() + 1)}.${p(d.getDate())} ${days[d.getDay()]} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
});

// 朝・昼・おやつ・夜
const meals = ['07:30 BREAKFAST', '12:15 LUNCH', '15:30 SNACK', '19:00 DINNER'];
// 未測定値空欄
const readout = [
	{ name: 'ENERGY', value: '182 kcal' },
	{ name: 'PROTEIN', value: '6.1 g' },
	{ name: 'FAT', value: '4.3 g' },
	{ name: 'CARBOHYDRATE', value: '27.0 g' },
	{ name: 'IRON', value: '' },
	{ name: 'VITAMIN D', value: '' },
	{ name: 'FOLATE', value: '' },
];
// 直近6日
const days = computed(() =>
	Array.from({ length: 6 }, (_, i) => {
		const d = new Date(now.value);
		d.setDate(d.getDate() - (6 - i));
		return `${String(d.getMonth() + 1).padStart(2, '0')}.${String(d.getDate()).padStart(2, '0')}`;
	}),
);
// 食事摂取基準の目標量
const pfc = ['P 13–20%', 'F 20–30%', 'C 50–65%'];

const scanY = computed(() => {
	const f = frame.value;
	const top = f?.discs[5]?.y ?? 0;
	const bottom = f?.discs[0]?.y ?? 0;
	return top - 40 + (f?.scan ?? 0) * (bottom - top + 80);
});

const lifetime = new AbortController();

onMounted(async () => {
	gsap.registerPlugin(ScrollTrigger);
	window.scrollTo(0, 0);
	if (canUseWebGL() && stage.value) {
		const { mountWelcome } = await import('~/utils/three/welcome');
		try {
			scene = await mountWelcome(stage.value, { animate: !reduced.value, onFrame: (f) => (frame.value = f), signal: lifetime.signal });
		} catch (err) {
			if (lifetime.signal.aborted) return;
			throw err;
		}
	}
	if (reduced.value) {
		ctx = gsap.context(() => {
			ScrollTrigger.create({
				trigger: root.value,
				start: 'top top',
				end: 'bottom bottom',
				onUpdate: (self) => {
					progress.value = Math.round(self.progress * last);
					scene?.setProgress(progress.value);
				},
			});
		});
		return;
	}
	ctx = gsap.context(() => {
		const tl = gsap.timeline({
			defaults: { ease: 'none' },
			scrollTrigger: {
				trigger: pin.value,
				start: 'top top',
				end: `+=${last * 120}%`,
				pin: true,
				// 慣性のある追従
				scrub: 1.4,
				// 速く動かしても1章ずつ
				snap: { snapTo: 1 / last, inertia: false, duration: { min: 0.5, max: 1.1 }, delay: 0.08, ease: 'power3.inOut' },
			},
			onUpdate: () => {
				progress.value = tl.progress() * last;
				scene?.setProgress(progress.value);
			},
		});
		const layers = gsap.utils.toArray<HTMLElement>('.layer');
		layers.forEach((layer, i) => {
			const inner = layer.querySelectorAll('.layer-char');
			const extras = layer.querySelectorAll('.layer-extra');
			const number = layer.querySelectorAll('.layer-number');
			if (i > 0) {
				tl.set(layer, { visibility: 'visible' }, i - 0.55)
					.fromTo(number, { yPercent: 40, autoAlpha: 0 }, { yPercent: 0, autoAlpha: 1, duration: 0.5, ease: 'power3.out' }, i - 0.55)
					.fromTo(
						inner,
						{ yPercent: 80, rotate: 8, autoAlpha: 0, filter: 'blur(12px)' },
						{ yPercent: 0, rotate: 0, autoAlpha: 1, filter: 'blur(0px)', duration: 0.45, stagger: 0.014, ease: 'power3.out' },
						i - 0.5,
					)
					.fromTo(extras, { autoAlpha: 0, y: 20 }, { autoAlpha: 1, y: 0, duration: 0.35, ease: 'power3.out' }, i - 0.3);
			}
			if (i < last) {
				tl.to(inner, { yPercent: -70, rotate: -6, autoAlpha: 0, filter: 'blur(10px)', duration: 0.4, stagger: 0.008, ease: 'power2.in' }, i + 0.08)
					.to(extras, { autoAlpha: 0, y: -20, duration: 0.3, ease: 'power2.in' }, i + 0.05)
					.to(number, { yPercent: -40, autoAlpha: 0, duration: 0.45, ease: 'power2.in' }, i + 0.05)
					.set(layer, { visibility: 'hidden' }, i + 0.55);
			}
		});
		tl.set({}, {}, last);
	}, root.value ?? undefined);
});

onBeforeUnmount(() => {
	lifetime.abort();
	ctx?.revert();
});
</script>

<template>
<div ref="root" class="relative">
	<div ref="stage" class="aberrate pointer-events-none fixed inset-0" aria-hidden="true"></div>
	<WelcomeEffects :progress="progress"/>

	<!-- 3Dに沿う情報, 効果の上 -->
	<div
		v-if="frame && !reduced"
		class="pointer-events-none fixed inset-0 z-30 overflow-hidden font-mono text-[10px] tracking-[0.12em] text-foreground"
		aria-hidden="true"
	>
		<p class="label absolute top-[calc(1rem+env(safe-area-inset-top)+4rem)] left-5" :style="reveal(0)">{{ clock }}<br/>TODAY · 0 ENTRIES</p>

		<p v-for="(m, i) in meals" :key="m" class="label absolute top-0 left-0" :style="{ ...at(frame.pebbles[i], 18, -26), ...reveal(1) }">{{ m }}</p>
		<span
			class="ripple absolute top-0 left-0"
			:style="{ ...at(frame.pebbles[1]), '--phase': String(frame.tap), opacity: String(frame.tapWeight * (1 - frame.tap)) }"
		></span>
		<p class="label absolute top-0 left-0" :style="{ ...at(frame.pebbles[1], 40, -78), opacity: String(frame.tapWeight * Math.min(frame.tap * 4, 1)) }">
			+1 TAP<br/>SAME AS LAST TIME
		</p>

		<div class="label absolute top-0 left-0 w-44" :style="{ ...readoutAt, ...reveal(2) }">
			<p class="mb-2 border-b border-current pb-1">NUTRIENTS / 1 SERVING</p>
			<p v-for="r in readout" :key="r.name" class="flex justify-between">
				<span>{{ r.name }}</span><span :class="r.value ? '' : 'blink'">{{ r.value || '—' }}</span>
			</p>
			<p class="mt-2">NOT MEASURED ≠ 0</p>
		</div>

		<p v-for="(d, i) in days" :key="d" class="label absolute top-0 left-0" :style="{ ...at(frame.discs[i], 14, -6), ...reveal(3) }">{{ d }}</p>
		<div class="absolute inset-x-0 top-0" :style="{ transform: `translateY(${scanY}px)`, opacity: String(frame.scanWeight) }">
			<div class="h-px w-full bg-[var(--lavender-ink)] opacity-60"></div>
			<p class="label mt-1 ml-5">FOOD DATA rev.2 → rev.3 · RECORDS UNCHANGED</p>
		</div>

		<p v-for="(l, i) in pfc" :key="l" class="label absolute top-0 left-0" :style="{ ...at(frame.rings[i], 10, -8), ...reveal(4) }">{{ l }}</p>
		<p class="label absolute top-[calc(1rem+env(safe-area-inset-top)+4rem)] left-5" :style="reveal(4)">EER = BMR × WEIGHT × PAL<br/>DRI 2025 · JAPAN</p>
	</div>

	<header class="fixed inset-x-0 top-0 z-40 flex items-center justify-between px-5 pt-[calc(1rem+env(safe-area-inset-top))]">
		<Transition name="fade">
			<img v-if="current > 0" src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="h-5 w-auto"/>
		</Transition>
		<button type="button" class="btn btn-ghost press ml-auto" @click="emit('done', $event.currentTarget as Element)">
			{{ t('pages.welcome.skip') }}<Icon name="lucide:arrow-right" size="16" aria-hidden="true"/>
		</button>
	</header>

	<!-- 章の位置 -->
	<nav class="fixed top-1/2 right-5 z-40 -translate-y-1/2" aria-hidden="true">
		<ol class="relative flex flex-col gap-5">
			<span class="absolute top-0 left-[3px] h-full w-px bg-[var(--hairline)]"></span>
			<span class="absolute top-0 left-[3px] w-px origin-top bg-foreground" :style="{ height: `${(progress / last) * 100}%` }"></span>
			<li v-for="c in chapters" :key="c" class="relative flex items-center gap-3">
				<span
					class="size-[7px] rounded-full border border-foreground transition-colors duration-500"
					:class="c <= current ? 'bg-foreground' : 'bg-[var(--background)]'"
				></span>
			</li>
		</ol>
	</nav>

	<div v-if="!reduced" ref="pin" class="relative z-30 h-dvh overflow-hidden">
		<section
			v-for="c in chapters"
			:key="c"
			class="layer absolute inset-0 flex flex-col justify-end px-6 pb-[calc(6rem+env(safe-area-inset-bottom))] landscape:justify-center landscape:pb-0 landscape:pl-[7vw]"
			:class="c > 0 ? 'invisible' : ''"
			:aria-labelledby="`chapter-${c}`"
		>
			<p class="layer-number number" aria-hidden="true">{{ String(c + 1).padStart(2, '0') }}</p>
			<div class="relative mx-auto w-full max-w-xl landscape:mx-0 landscape:max-w-[min(34rem,42vw)]">
				<p class="layer-extra eyebrow mb-5">{{ t(`pages.welcome.chapters.${c}.eyebrow`) }}</p>
				<h1 v-if="c === 0" class="layer-extra mb-6">
					<img src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="h-[clamp(2.5rem,11vw,3.75rem)] w-auto"/>
				</h1>
				<h2 :id="`chapter-${c}`" class="line" :aria-label="t(`pages.welcome.chapters.${c}.line`)">
					<span v-for="(ph, i) in phrases(`pages.welcome.chapters.${c}.line`)" :key="i" class="inline-block whitespace-nowrap" aria-hidden="true"><span v-for="(ch, j) in ph" :key="j" class="layer-char inline-block whitespace-pre">{{ ch }}</span></span>
				</h2>
				<p class="layer-extra desc mt-4">{{ t(`pages.welcome.chapters.${c}.desc`) }}</p>
				<p v-if="c === 0" class="layer-extra eyebrow mt-10 flex items-center gap-3" aria-hidden="true">
					<span class="scroll-cue"></span>{{ t('pages.welcome.scroll') }}
				</p>
				<button v-if="c === last" type="button" class="layer-extra btn btn-primary press mt-10 w-full" @click="emit('done', $event.currentTarget as Element)">
					{{ t('pages.welcome.start') }}
				</button>
			</div>
		</section>
	</div>

	<template v-else>
		<section
			v-for="c in chapters"
			:key="c"
			class="relative z-30 flex min-h-dvh flex-col justify-end px-6 pb-[calc(6rem+env(safe-area-inset-bottom))]"
			:aria-labelledby="`chapter-${c}`"
		>
			<div class="mx-auto w-full max-w-xl">
				<p class="eyebrow mb-5">{{ t(`pages.welcome.chapters.${c}.eyebrow`) }}</p>
				<h1 v-if="c === 0" class="mb-6">
					<img src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="h-[clamp(2.5rem,11vw,3.75rem)] w-auto"/>
				</h1>
				<h2 :id="`chapter-${c}`" class="line">{{ t(`pages.welcome.chapters.${c}.line`) }}</h2>
				<p class="desc mt-4">{{ t(`pages.welcome.chapters.${c}.desc`) }}</p>
				<button v-if="c === last" type="button" class="btn btn-primary press mt-10 w-full" @click="emit('done', $event.currentTarget as Element)">
					{{ t('pages.welcome.start') }}
				</button>
			</div>
		</section>
	</template>
</div>
</template>

<style scoped>
/* 色収差 */
.aberrate {
	filter: url('#welcome-aberration');
}

.line {
	font-family: var(--font-sans);
	font-size: clamp(1.6rem, 6.2vw, 2.5rem);
	font-weight: 700;
	line-height: 1.65;
	letter-spacing: 0.04em;
	color: var(--foreground);
}

.desc {
	max-inline-size: 30rem;
	font-size: 0.9375rem;
	line-height: 1.9;
	color: var(--muted-foreground);
}

/* 輪郭文字 */
.number {
	position: absolute;
	inset-block-start: 9%;
	inset-inline-end: 1.5rem;
	font-family: var(--font-display);
	font-size: clamp(7rem, 34vw, 16rem);
	line-height: 1;
	color: transparent;
	-webkit-text-stroke: 1px var(--foreground);
	opacity: 0.35;
}

@media (orientation: portrait) {
	.number {
		inset-block-start: 44%;
		font-size: clamp(5rem, 26vw, 9rem);
	}
}

@media (orientation: landscape) {
	.number {
		inset-inline: 6vw auto;
	}
}

.label {
	white-space: nowrap;
	line-height: 1.7;
	will-change: transform;
}

.ripple {
	inline-size: 2px;
	block-size: 2px;
	border-radius: 9999px;
	box-shadow: 0 0 0 1px var(--foreground);
	scale: calc(1 + var(--phase) * 60);
}

.blink {
	animation: blink 1s steps(2) infinite;
}

.scroll-cue {
	position: relative;
	display: block;
	inline-size: 2.5rem;
	block-size: 1px;
	overflow: hidden;
	background: var(--hairline);
}

.scroll-cue::after {
	content: '';
	position: absolute;
	inset: 0;
	background: var(--foreground);
	animation: cue 1.8s var(--ease-out-expo) infinite;
}

.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.6s;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}

@keyframes blink {
	50% {
		opacity: 0;
	}
}

@keyframes cue {
	from {
		transform: translateX(-100%);
	}

	to {
		transform: translateX(100%);
	}
}
</style>
