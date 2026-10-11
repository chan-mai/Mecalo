// CSS変数色をChart.jsへ
export function useChartTheme() {
	const theme = ref({ fill: '#B9DEFF', fillHover: '#A8D6F7', data: '#4571B5', accent: '#F6BED6', text: '#5E6A80', grid: '#D9E3F2', ink: '#25324A', surface: '#FFFFFF' });

	onMounted(() => {
		const css = getComputedStyle(window.document.documentElement);
		const v = (name: string, fallback: string) => css.getPropertyValue(name).trim() || fallback;
		theme.value = {
			fill: v('--primary', theme.value.fill),
			fillHover: v('--primary-hover', theme.value.fillHover),
			data: v('--chart-protein', theme.value.data),
			accent: v('--brand-secondary', theme.value.accent),
			text: v('--muted-foreground', theme.value.text),
			grid: v('--border', theme.value.grid),
			ink: v('--foreground', theme.value.ink),
			surface: v('--surface', theme.value.surface),
		};
	});

	return theme;
}
