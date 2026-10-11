<script setup lang="ts">
import { BarController, BarElement, CategoryScale, Chart, Legend, LinearScale, LineController, LineElement, PointElement, Tooltip } from 'chart.js';
import { Bar } from 'vue-chartjs';
import type { ChartData, ChartOptions } from 'chart.js';
import { growFromBottom, revealLinesFromLeft } from '~/utils/chartAnimation';
import { formatInt } from '~/utils/nutrients';

Chart.register(BarController, BarElement, LineController, LineElement, PointElement, CategoryScale, LinearScale, Tooltip, Legend);

const props = defineProps<{
	days: { label: string; tick: string; kcal: number }[];
	target?: number;
}>();
const { t } = useI18n();
const theme = useChartTheme();
const reducedMotion = usePreferredReducedMotion();
const animate = reducedMotion.value !== 'reduce';
const plugins = animate ? [revealLinesFromLeft()] : [];

// カードの表示演出後に
const area = useTemplateRef<HTMLDivElement>('area');
const shown = ref(!animate);
const { stop } = useIntersectionObserver(area, ([entry]) => {
	if (!entry?.isIntersecting) return;
	stop();
	useTimeoutFn(() => (shown.value = true), 350);
}, { threshold: 0.4 });

const data = computed(() => ({
	labels: props.days.map((d) => d.tick),
	datasets: [
		{
			type: 'bar' as const,
			label: t('pages.home.energyCard.eaten'),
			data: props.days.map((d) => Math.round(d.kcal)),
			backgroundColor: theme.value.fill,
			hoverBackgroundColor: theme.value.fillHover,
			borderColor: theme.value.data,
			borderWidth: { top: 2 },
			borderSkipped: 'bottom',
			borderRadius: 4,
			...(animate ? { animations: growFromBottom() } : {}),
			maxBarThickness: 28,
		},
		...(props.target !== undefined ? [{
			type: 'line' as const,
			label: t('terms.target'),
			data: props.days.map(() => Math.round(props.target ?? 0)),
			borderColor: theme.value.ink,
			borderDash: [4, 3],
			animation: false,
			borderWidth: 1.5,
			pointRadius: 0,
		}] : []),
	],
}) as unknown as ChartData<'bar'>);

const options = computed<ChartOptions<'bar'>>(() => ({
	animation: animate ? undefined : false,
	responsive: true,
	maintainAspectRatio: false,
	interaction: { mode: 'index', intersect: false },
	plugins: {
		legend: { position: 'bottom', labels: { color: theme.value.text, boxWidth: 12 } },
		tooltip: {
			callbacks: {
				title: (items) => props.days[items[0]?.dataIndex ?? 0]?.label ?? '',
				label: (item) => `${item.dataset.label}: ${item.formattedValue} kcal`,
			},
		},
	},
	scales: {
		x: { grid: { display: false }, ticks: { color: theme.value.text } },
		y: { beginAtZero: true, grid: { color: theme.value.grid }, ticks: { color: theme.value.text, maxTicksLimit: 4 } },
	},
}));
</script>

<template>
<figure>
	<div ref="area" class="h-48" aria-hidden="true">
		<Bar v-if="shown" :data="data" :options="options" :plugins="plugins"/>
	</div>
	<table class="sr-only">
		<caption>{{ t('pages.history.energyChart') }}</caption>
		<thead>
			<tr>
				<th scope="col">{{ t('terms.date') }}</th>
				<th scope="col">{{ t('pages.home.energyCard.eaten') }} (kcal)</th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="d in days" :key="d.label">
				<th scope="row">{{ d.label }}</th>
				<td>{{ formatInt(d.kcal) }}</td>
			</tr>
		</tbody>
	</table>
	<figcaption v-if="target !== undefined" class="sr-only">{{ t('terms.target') }}: {{ formatInt(target) }} kcal</figcaption>
</figure>
</template>
