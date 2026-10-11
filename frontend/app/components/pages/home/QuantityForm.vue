<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { formatAmount } from '~/utils/nutrients';
import { gramsOf, scaleProfile } from '~/utils/quantity';

const props = defineProps<{
	food: Schemas['Food'];
	initial: Schemas['Quantity'];
	busy?: boolean;
	submitLabel?: string;
}>();
const emit = defineEmits<{ submit: [quantity: Schemas['Quantity']] }>();
const { t } = useI18n();
const uid = useId();

const amount = ref<number>();
const unit = ref('g');

watch(() => props.initial, (q) => {
	amount.value = q.amount;
	unit.value = q.unit;
}, { immediate: true });
const amountInput = useTemplateRef<HTMLInputElement>('amountInput');
defineExpose({ amountInput });

const summaryCodes = ['energy', 'protein', 'fat', 'carbohydrate'] as const;

const units = computed(() => [
	...props.food.revision.portions.map((p) => ({ unit: p.unit, label: `${p.unit} (${Math.round(p.grams)}g)` })),
	{ unit: 'g', label: 'g' },
]);

const presets = computed(() => (unit.value === 'g' ? [50, 100, 150, 200, 300] : [0.5, 1, 1.5, 2, 3]));

const preview = computed(() => {
	const grams = gramsOf(props.food.revision, { amount: amount.value ?? 0, unit: unit.value });
	return scaleProfile(props.food.revision.nutrients, grams ?? 0);
});

// 単位変更時は同じ重量になるよう数量を換算
function changeUnit(to: string) {
	const from = unit.value;
	unit.value = to;
	if (amount.value === undefined) return;
	const grams = gramsOf(props.food.revision, { amount: amount.value, unit: from });
	const per = to === 'g' ? 1 : props.food.revision.portions.find((p) => p.unit === to)?.grams;
	if (grams !== undefined && per) amount.value = Math.round(grams / per * 100) / 100;
}
</script>

<template>
<form class="space-y-4" @submit.prevent="emit('submit', { amount: amount!, unit })">
	<div class="rounded-xl bg-surface-soft p-3">
		<p class="font-semibold">{{ food.revision.name }}</p>
		<p v-if="food.revision.brand" class="text-sm text-muted">{{ food.revision.brand }}</p>
		<dl class="mt-2 grid grid-cols-4 gap-1 text-center text-xs">
			<div v-for="code in summaryCodes" :key="code">
				<dt class="text-muted">{{ t(`terms.nutrient.${code}`) }}</dt>
				<dd class="text-sm font-semibold tabular-nums">{{ preview[code] ? formatAmount(preview[code].amount) : '–' }}</dd>
			</div>
		</dl>
	</div>

	<div class="flex gap-2">
		<label class="flex-1">
			<span class="mb-1 block text-sm text-muted">{{ t('terms.quantity') }}</span>
			<input ref="amountInput" v-model.number="amount" type="number" inputmode="decimal" min="0" step="any" class="input text-right text-lg" required>
		</label>
		<div class="w-36">
			<span :id="`${uid}-unit`" class="mb-1 block text-sm text-muted">{{ t('terms.unit') }}</span>
			<SelectField :modelValue="unit" :labelledby="`${uid}-unit`" :options="units.map((u) => ({ value: u.unit, label: u.label }))" @update:modelValue="changeUnit"/>
		</div>
	</div>
	<div class="flex flex-wrap gap-1.5">
		<button v-for="preset in presets" :key="preset" type="button" class="press rounded-full border-2 border-[var(--input-border)] bg-[var(--glass)] px-3 py-1 text-sm tabular-nums" @click="amount = preset">
			{{ preset }}
		</button>
	</div>

	<slot name="extra"></slot>

	<button class="btn btn-primary w-full" :disabled="!amount || amount <= 0 || busy">{{ submitLabel ?? t('pages.home.quantityForm.submit') }}</button>
</form>
</template>
