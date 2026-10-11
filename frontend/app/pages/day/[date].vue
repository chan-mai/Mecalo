<script setup lang="ts">
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const route = useRoute();
const date = computed(() => String(route.params.date));

const { data: report } = useAsyncData(
	() => `report-${date.value}`,
	() => unwrap(useApi().GET('/reports/daily/{date}', { params: { path: { date: date.value } } })),
	{ watch: [date] },
);
</script>

<template>
<div class="space-y-3">
	<PageHeader :back="{ path: '/', query: { date } }" :title="date" :eyebrow="t('common.eyebrow.day')"/>
	<section v-if="report" v-reveal class="panel p-6">
		<NutrientTable :profile="report.summary.total" :evaluations="report.evaluations"/>
		<p class="mt-3 text-xs text-muted">{{ t('terms.coverageHint') }}</p>
	</section>
</div>
</template>
