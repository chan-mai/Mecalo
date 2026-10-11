import type { Chart, ChartOptions, ChartType, Plugin, ScriptableContext } from 'chart.js';

type Animations = NonNullable<ChartOptions<ChartType>['animations']>;
type Ctx = ScriptableContext<ChartType> & { grown?: boolean };

export function growFromBottom(): Animations {
	return {
		y: {
			duration: 1000,
			easing: 'easeOutExpo',
			from: (c) => {
				const ctx = c as Ctx;
				if (ctx.type !== 'data' || ctx.grown) return undefined;
				ctx.grown = true;
				return ctx.chart.chartArea.bottom;
			},
			delay: (c) => (c.type === 'data' ? c.dataIndex * 70 : 0),
		},
	};
}

export function revealLinesFromLeft(duration = 1000): Plugin<ChartType> {
	let start: number | undefined;
	const isLine = (chart: Chart, index: number) => chart.getDatasetMeta(index).type === 'line';
	return {
		id: 'revealLinesFromLeft',
		beforeDatasetDraw(chart, { index }) {
			if (!isLine(chart, index)) return;
			start ??= performance.now();
			const progress = Math.min((performance.now() - start) / duration, 1);
			const eased = 1 - (1 - progress) ** 3;
			const { left, right, top, bottom } = chart.chartArea;
			chart.ctx.save();
			chart.ctx.beginPath();
			chart.ctx.rect(left - 8, top - 8, (right - left + 16) * eased, bottom - top + 16);
			chart.ctx.clip();
			if (progress < 1) requestAnimationFrame(() => chart.draw());
		},
		afterDatasetDraw(chart, { index }) {
			if (isLine(chart, index)) chart.ctx.restore();
		},
	};
}
