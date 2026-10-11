<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { attentionOrder, coveragePercent, formatAmount, levelColor, progressOf } from '~/utils/nutrients';

const props = defineProps<{ evaluations: Schemas['Evaluation'][]; recorded: boolean; date: string }>();
const { t } = useI18n();
const { unitOf } = useNutrientCatalog();

const uid = useId();
const items = computed(() => attentionOrder(props.evaluations).slice(0, 5));
</script>

<template>
<section v-reveal class="panel p-6" :aria-labelledby="`${uid}-title`">
	<SectionHeading icon="lucide:leaf" :eyebrow="t('common.eyebrow.nutrients')" :title="t('pages.home.attentionCard.title')" :headingId="`${uid}-title`">
		<NuxtLink :to="`/day/${date}`" class="link shrink-0 text-xs">{{ t('pages.home.attentionCard.allNutrients') }}</NuxtLink>
	</SectionHeading>
	<p v-if="!recorded" class="mt-4 text-sm text-muted">{{ t('pages.home.attentionCard.noRecords') }}</p>
	<p v-else-if="items.length === 0" class="mt-4 text-sm text-muted">{{ t('pages.home.attentionCard.allGood') }}</p>
	<ul v-else class="mt-4 space-y-4">
		<li v-for="evaluation in items" :key="evaluation.code">
			<div class="flex items-baseline justify-between gap-2 text-sm">
				<span :id="`${uid}-${evaluation.code}`">{{ t(`terms.nutrient.${evaluation.code}`) }}</span>
				<span class="tabular-nums">
					<LevelBadge :level="evaluation.level"/>
					<span class="ml-2 text-muted">{{ formatAmount(evaluation.value.amount) }}{{ unitOf(evaluation.code) }}<template v-if="evaluation.lower ?? evaluation.upper"> / {{ formatAmount((evaluation.lower ?? evaluation.upper)!) }}</template></span>
				</span>
			</div>
			<LevelMeter class="mt-1.5" :value="progressOf(evaluation)" :max="1" :color="levelColor[evaluation.level]" :labelledby="`${uid}-${evaluation.code}`"/>
			<p v-if="evaluation.value.coverage < 1" class="mt-1 text-xs text-muted">{{ t('terms.coverage', { percent: coveragePercent(evaluation.value) }) }}</p>
		</li>
	</ul>
</section>
</template>
