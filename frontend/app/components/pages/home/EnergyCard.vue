<script setup lang="ts">
import gsap from 'gsap';
import type { Schemas } from '~/composables/api/client';
import { amountOf, coveragePercent, formatAmount, formatInt } from '~/utils/nutrients';

const props = defineProps<{ report: Schemas['DailyReport'] }>();
const { t } = useI18n();
const motion = usePreferredReducedMotion();
const reduced = computed(() => motion.value === 'reduce');

const uid = useId();
const eaten = computed(() => amountOf(props.report.summary.total, 'energy'));
const target = computed(() => props.report.targets?.energyKcal);
const ratio = computed(() => (target.value ? eaten.value / target.value : 0));
const over = computed(() => ratio.value > 1.1);
const energyCoverage = computed(() => (props.report.summary.entryCount === 0 ? 100 : coveragePercent(props.report.summary.total.energy)));

const colors = { protein: 'var(--chart-protein-soft)', fat: 'var(--chart-fat-soft)', carbohydrate: 'var(--chart-carbs-soft)' } as const;
const kcalPerGram = { protein: 4, fat: 9, carbohydrate: 4 } as const;

const macros = computed(() => (['protein', 'fat', 'carbohydrate'] as const).map((code) => {
	const amount = amountOf(props.report.summary.total, code);
	const tg = props.report.targets?.nutrients.find((n) => n.code === code);
	const lower = tg?.customMin ?? tg?.dgMin ?? tg?.rda;
	const upper = tg?.customMax ?? tg?.dgMax;
	return {
		code,
		amount,
		color: colors[code],
		range: lower !== undefined && upper !== undefined ? `${formatInt(lower)}–${formatInt(upper)}g` : undefined,
		progress: upper ? Math.min(amount / upper, 1) : 0,
	};
}));

const pfcBars = computed(() => {
	const total = eaten.value;
	return macros.value.map((m) => {
		const tg = props.report.targets;
		const goal = tg?.nutrients.find((n) => n.code === m.code);
		const toPct = (g?: number) => (g !== undefined && tg ? g * kcalPerGram[m.code] / tg.energyKcal * 100 : undefined);
		return {
			value: total > 0 ? m.amount * kcalPerGram[m.code] / total * 100 : 0,
			min: toPct(goal?.dgMin),
			max: toPct(goal?.dgMax),
			color: getComputedColor(m.color),
		};
	});
});

function getComputedColor(cssVar: string): string {
	if (typeof window === 'undefined') return '#888888';
	const name = cssVar.slice(4, -1);
	const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
	return toHex(value) ?? '#888888';
}

// oklch等をThree.jsで扱えるhexへ変換
function toHex(color: string): string | undefined {
	if (!color) return undefined;
	const ctx = document.createElement('canvas').getContext('2d');
	if (!ctx) return undefined;
	ctx.fillStyle = color;
	ctx.fillRect(0, 0, 1, 1);
	const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data;
	return `#${[r, g, b].map((v) => (v ?? 0).toString(16).padStart(2, '0')).join('')}`;
}

const displayed = ref(Math.round(eaten.value));
watch(eaten, (to, from) => {
	if (reduced.value || window.document.hidden) {
		displayed.value = Math.round(to);
		return;
	}
	const state = { v: from };
	gsap.to(state, {
		v: to,
		duration: 0.6,
		ease: 'power2.out',
		onUpdate: () => {
			displayed.value = Math.round(state.v);
		},
	});
});
</script>

<template>
<section v-reveal class="panel p-6" :aria-labelledby="`${uid}-title`">
	<SectionHeading icon="lucide:flame" :eyebrow="t('common.eyebrow.energy')" :title="t('pages.home.energyCard.eaten')" :headingId="`${uid}-title`"/>
	<p class="mt-5 flex items-baseline gap-2">
		<span class="numeric text-[clamp(3.5rem,18vw,5rem)] leading-none" aria-hidden="true">{{ formatInt(displayed) }}</span>
		<span class="sr-only">{{ formatInt(eaten) }}</span>
		<span class="eyebrow">kcal</span>
	</p>
	<p :id="`${uid}-target`" class="mt-3 text-sm tabular-nums" :class="over ? 'text-danger' : 'text-muted'">
		<template v-if="target">{{ over ? t('pages.home.energyCard.over') : t('pages.home.energyCard.remaining') }} {{ formatInt(Math.abs(target - eaten)) }} / {{ t('terms.target') }} {{ formatInt(target) }} kcal</template>
		<template v-else>{{ t('pages.home.energyCard.noTarget') }}</template>
	</p>
	<LevelMeter v-if="target" class="mt-2" :value="ratio" :max="1" :color="over ? 'var(--brand-secondary)' : 'var(--primary-mid)'" :labelledby="`${uid}-target`"/>
	<p v-if="energyCoverage < 100" class="mt-2 text-xs text-warning">
		{{ t('terms.coverage', { percent: energyCoverage }) }}
		<span class="sr-only">{{ t('terms.coverageHint') }}</span>
	</p>

	<hr class="rule my-5">

	<div class="grid grid-cols-[1fr_9rem] items-center gap-4">
		<ul class="space-y-3">
			<li v-for="m in macros" :key="m.code">
				<div class="flex items-baseline justify-between text-sm">
					<span :id="`${uid}-${m.code}`" class="flex items-center gap-2"><span class="size-2 rounded-full" :style="{ background: m.color }" aria-hidden="true"></span>{{ t(`terms.nutrient.${m.code}`) }}</span>
					<span class="tabular-nums"><span class="numeric">{{ formatAmount(m.amount) }}</span>g<span v-if="m.range" class="text-xs text-muted"> / {{ m.range }}</span></span>
				</div>
				<LevelMeter class="mt-1.5" :value="m.progress" :max="1" :color="m.color" :labelledby="`${uid}-${m.code}`"/>
			</li>
		</ul>
		<PfcScene class="h-36" :bars="pfcBars"/>
	</div>
</section>
</template>

<style scoped>
.rule {
	border: 0;
	border-block-start: 1px solid var(--hairline);
}
</style>
