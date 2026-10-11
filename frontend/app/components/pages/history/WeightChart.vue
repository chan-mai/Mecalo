<script setup lang="ts">
import { CategoryScale, Chart, LinearScale, LineController, LineElement, PointElement, Tooltip } from 'chart.js';
import { Line } from 'vue-chartjs';
import type { ChartData, ChartOptions } from 'chart.js';
import { revealLinesFromLeft } from '~/utils/chartAnimation';
import { addDays, daysBetween, formatDate } from '~/utils/date';

Chart.register(LineController, LineElement, PointElement, CategoryScale, LinearScale, Tooltip);

const props = defineProps<{
	// 期間の初日からの日数で横位置を決定
	from: string;
	points: { date: string; kg: number }[];
}>();
const { t, locale } = useI18n();
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

function label(day: number) {
	return formatDate(addDays(props.from, day), t('common.dateFormat.short'), locale.value);
}

const data = computed<ChartData<'line', { x: number; y: number }[]>>(() => ({
	datasets: [{
		label: t('terms.weightKg'),
		data: props.points.map((p) => ({ x: daysBetween(props.from, p.date), y: p.kg })),
		borderColor: theme.value.data,
		backgroundColor: theme.value.accent,
		borderWidth: 2,
		pointRadius: 5,
		pointBorderColor: theme.value.data,
		pointBorderWidth: 2,
		tension: 0.25,
		animation: false,
	}],
}));

const options = computed<ChartOptions<'line'>>(() => ({
	animation: animate ? undefined : false,
	responsive: true,
	maintainAspectRatio: false,
	plugins: {
		legend: { display: false },
		tooltip: {
			callbacks: {
				title: (items) => label(Number(items[0]?.parsed.x ?? 0)),
				label: (item) => `${item.parsed.y} kg`,
			},
		},
	},
	scales: {
		x: { type: 'linear', grid: { display: false }, ticks: { color: theme.value.text, maxTicksLimit: 5, callback: (v) => label(Number(v)) } },
		y: { grid: { color: theme.value.grid }, ticks: { color: theme.value.text, maxTicksLimit: 4 } },
	},
}));
</script>

<template>
<figure>
	<div ref="area" class="h-44" aria-hidden="true">
		<Line v-if="shown" :data="data" :options="options" :plugins="plugins"/>
	</div>
	<table class="sr-only">
		<caption>{{ t('pages.history.weightChart') }}</caption>
		<thead>
			<tr>
				<th scope="col">{{ t('terms.date') }}</th>
				<th scope="col">{{ t('terms.weightKg') }}</th>
			</tr>
		</thead>
		<tbody>
			<tr v-for="p in points" :key="p.date">
				<th scope="row">{{ p.date }}</th>
				<td>{{ p.kg }}</td>
			</tr>
		</tbody>
	</table>
</figure>
</template>
