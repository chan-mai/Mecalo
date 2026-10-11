<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { barProgress, coveragePercent, diffOf, formatAmount, formatSigned, levelColor, levelText, unitLabel } from '~/utils/nutrients';

const props = withDefaults(defineProps<{
	profile: Schemas['NutrientProfile'];
	evaluations?: Schemas['Evaluation'][];
	showGroups?: boolean;
}>(), { evaluations: undefined, showGroups: true });
const { t } = useI18n();
const { groups } = useNutrientCatalog();
const uid = useId();

const order: Schemas['NutrientGroup'][] = ['energy', 'macro', 'mineral', 'vitamin'];
const visibleGroups = computed(() => order.filter((g) => groups.value[g].length > 0));

function evaluationOf(code: string) {
	return props.evaluations?.find((evaluation) => evaluation.code === code);
}

// 行背景の到達度バー
function barStyle(code: string, row: number) {
	const evaluation = evaluationOf(code);
	if (!evaluation || (evaluation.lower === undefined && evaluation.upper === undefined)) return undefined;
	return {
		'--bar': `${barProgress(evaluation) * 100}%`,
		'--row': row,
		'--bar-color': `color-mix(in srgb, ${levelColor[evaluation.level]} 40%, transparent)`,
	};
}
</script>

<template>
<div class="space-y-3">
	<section v-for="(group, g) in visibleGroups" :key="group" :aria-labelledby="`${uid}-${group}`">
		<h3 :id="`${uid}-${group}`" class="mb-1 text-xs font-semibold text-muted" :class="{ 'sr-only': !showGroups }">{{ t(`terms.nutrientGroup.${group}`) }}</h3>
		<table class="w-full text-sm" :aria-labelledby="`${uid}-${group}`">
			<tbody>
				<tr v-for="(d, i) in groups[group]" :key="d.code" class="border-b border-[var(--hairline)] last:border-0" :class="{ 'data-bar': barStyle(d.code, g * 6 + i) }" :style="barStyle(d.code, g * 6 + i)">
					<th scope="row" class="py-1.5 pl-2 text-left font-normal">{{ t(`terms.nutrient.${d.code}`) }}</th>
					<td class="py-1.5 text-right tabular-nums">
						<template v-if="profile[d.code]">
							{{ formatAmount(profile[d.code]!.amount) }}<span class="ml-0.5 text-xs text-muted">{{ unitLabel[d.unit] }}</span>
							<span v-if="profile[d.code]!.status === 'estimated' || profile[d.code]!.status === 'trace'" class="ml-1 text-xs text-muted">({{ t(`terms.status.${profile[d.code]!.status}`) }})</span>
							<span v-else-if="profile[d.code]!.coverage < 1" class="ml-1 text-xs text-warning">{{ coveragePercent(profile[d.code]) }}%</span>
						</template>
						<span v-else class="text-xs text-muted">{{ t('common.unknown') }}</span>
					</td>
					<td v-if="evaluations" class="w-20 py-1.5 text-right text-xs tabular-nums" :class="evaluationOf(d.code) ? levelText[evaluationOf(d.code)!.level] : ''">
						<template v-if="evaluationOf(d.code) && diffOf(evaluationOf(d.code)!) !== undefined">{{ formatSigned(diffOf(evaluationOf(d.code)!)!) }}{{ unitLabel[d.unit] }}</template>
					</td>
					<td v-if="evaluations" class="w-24 py-1.5 pr-1 text-right text-xs">
						<LevelBadge v-if="evaluationOf(d.code)" :level="evaluationOf(d.code)!.level"/>
					</td>
				</tr>
			</tbody>
		</table>
	</section>
</div>
</template>

<style scoped>
/* 行背景の到達度バー */
.data-bar {
	background-image: linear-gradient(var(--bar-color), var(--bar-color));
	background-repeat: no-repeat;
	background-size: var(--bar) 100%;
	animation: bar-grow 1.2s var(--ease-out-expo) both;
	animation-delay: calc(0.3s + var(--row, 0) * 40ms);
}

/* カードの表示演出が始まるまで待機 */
.reveal:not(.is-revealed) .data-bar {
	animation-play-state: paused;
}

@keyframes bar-grow {
	from {
		background-size: 0 100%;
	}
}
</style>
