<script setup lang="ts">
import { toast } from 'vue-sonner';
import { unwrap, useApi } from '~/composables/api/client';
import { weightSchema } from '~/schemas/settings';
import { addDays, formatDate, startOfWeek, todayIso } from '~/utils/date';

const { t, locale } = useI18n();
const confirmDelete = useConfirmDelete();
const onError = useErrorHandler();
const api = useApi();
const { errors, validate } = useValidation(weightSchema);
const outbox = useOutbox();

const modes = ['week', 'month'] as const;
const mode = ref<(typeof modes)[number]>('week');
const today = todayIso();
const anchor = ref(today);

const from = computed(() => (mode.value === 'week' ? startOfWeek(anchor.value) : addDays(anchor.value, -29)));
const to = computed(() => (mode.value === 'week' ? addDays(from.value, 6) : anchor.value));

function shift(dir: number) {
	anchor.value = addDays(anchor.value, dir * (mode.value === 'week' ? 7 : 30));
	if (anchor.value > today) anchor.value = today;
}

const { data: report } = useAsyncData(
	() => `range-${from.value}-${to.value}`,
	() => unwrap(api.GET('/reports/range', { params: { query: { from: from.value, to: to.value } } })),
	{ watch: [from, to] },
);
const weightFrom = computed(() => addDays(to.value, -89));
const { data: weights, refresh: refreshWeights } = useAsyncData(
	() => `weights-${weightFrom.value}-${to.value}`,
	() => unwrap(api.GET('/weights', { params: { query: { from: weightFrom.value, to: to.value } } })),
	{ watch: [weightFrom, to] },
);

function shortLabel(date: string) {
	return formatDate(date, t('common.dateFormat.short'), locale.value);
}

const energyDays = computed(() => (report.value?.days ?? []).map((d) => ({
	label: shortLabel(d.summary.date),
	tick: formatDate(d.summary.date, mode.value === 'week' ? 'dd' : 'D', locale.value),
	kcal: d.summary.total.energy?.amount ?? 0,
})));
const energyTarget = computed(() => report.value?.days.at(-1)?.targets?.energyKcal);

const weightForm = reactive<{ date: string; kg?: number }>({ date: today });

async function saveWeight() {
	const body = validate({ kg: weightForm.kg });
	if (!body) return;
	try {
		const res = await outbox.run({ kind: 'saveWeight', date: weightForm.date, body });
		toast(res.queued ? t('common.messages.savedOffline') : t('common.messages.weightSaved'));
		await refreshWeights();
	} catch (err) {
		onError(err);
	}
}

async function deleteWeight(date: string) {
	if (!await confirmDelete()) return;
	try {
		await unwrap(api.DELETE('/weights/{date}', { params: { path: { date } } }));
		await refreshWeights();
	} catch (err) {
		onError(err);
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader :title="t('pages.history.title')" :eyebrow="t('common.eyebrow.history')">
		<fieldset class="flex gap-1">
			<legend class="sr-only">{{ t('pages.history.period') }}</legend>
			<label v-for="m in modes" :key="m" class="choice">
				<input v-model="mode" type="radio" name="history-period" :value="m">
				<span>{{ t(`terms.period.${m}`) }}</span>
			</label>
		</fieldset>
	</PageHeader>

	<div class="flex items-center justify-between gap-3">
		<button type="button" class="btn-round" :aria-label="t('common.back')" @click="shift(-1)"><Icon name="lucide:arrow-left" size="18" aria-hidden="true"/></button>
		<span class="numeric text-sm">{{ from.replaceAll('-', '.') }} – {{ to.replaceAll('-', '.') }}</span>
		<button type="button" class="btn-round" :disabled="to >= today" :aria-label="t('common.nextDay')" @click="shift(1)"><Icon name="lucide:arrow-right" size="18" aria-hidden="true"/></button>
	</div>

	<template v-if="report">
		<section v-reveal class="panel p-6">
			<SectionHeading class="mb-4" icon="lucide:flame" :eyebrow="t('common.eyebrow.energy')" :title="t('pages.history.energyChart')">
				<span class="shrink-0 text-xs text-muted">{{ t('pages.history.recordedDays', { days: report.recordedDays }) }}</span>
			</SectionHeading>
			<EnergyChart :days="energyDays" :target="energyTarget"/>
		</section>

		<section v-reveal class="panel p-6">
			<SectionHeading icon="lucide:leaf" :eyebrow="t('common.eyebrow.nutrients')" :title="t('pages.history.nutrients')"/>
			<p class="mt-3 mb-4 text-xs text-muted">{{ t('pages.history.average') }}</p>
			<NutrientTable :key="`${from}-${to}`" :profile="report.average" :evaluations="report.averageEvaluations"/>
		</section>
	</template>

	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-4" icon="lucide:scale" :eyebrow="t('common.eyebrow.weight')" :title="t('pages.history.weightChart')"/>
		<WeightChart v-if="weights?.length" :from="weightFrom" :points="weights"/>
		<p v-else class="text-sm text-muted">{{ t('pages.history.noWeights') }}</p>

		<form class="mt-3 grid grid-cols-[1fr_1fr_auto] items-end gap-2" novalidate @submit.prevent="saveWeight">
			<FormField v-slot="{ field }" :label="t('terms.date')"><DateField v-bind="field" v-model="weightForm.date" :max="today"/></FormField>
			<FormField v-slot="{ field }" :label="t('terms.weightKg')" :error="errors.kg"><input v-bind="field" v-model.number="weightForm.kg" type="number" inputmode="decimal" step="0.1" class="input"></FormField>
			<button class="btn btn-primary">{{ t('common.save') }}</button>
		</form>
		<TransitionGroup tag="ul" name="list" class="relative mt-3 divide-y divide-[var(--hairline)] text-sm">
			<li v-for="w in [...(weights ?? [])].reverse()" :key="w.date" class="flex items-center justify-between py-2">
				<span class="numeric text-xs">{{ w.date.replaceAll('-', '.') }}</span>
				<span><span class="numeric">{{ w.kg }}</span> kg</span>
				<button type="button" class="btn btn-icon text-muted" :aria-label="t('common.delete')" @click="deleteWeight(w.date)"><Icon name="lucide:trash-2" size="16" aria-hidden="true"/></button>
			</li>
		</TransitionGroup>
	</section>
</div>
</template>
