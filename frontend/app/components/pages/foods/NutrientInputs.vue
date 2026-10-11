<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { unitLabel } from '~/utils/nutrients';

defineProps<{ codes: Schemas['NutrientDefinition'][] }>();
const model = defineModel<Record<string, number | undefined>>({ required: true });
const { t } = useI18n();

function update(code: string, value: number) {
	model.value = { ...model.value, [code]: Number.isNaN(value) ? undefined : value };
}
</script>

<template>
<div class="grid grid-cols-2 gap-x-3 gap-y-2">
	<label v-for="d in codes" :key="d.code" class="flex items-center gap-2 text-sm">
		<span class="flex-1">{{ t(`terms.nutrient.${d.code}`) }}</span>
		<input
			:value="model[d.code]"
			type="number"
			inputmode="decimal"
			step="any"
			min="0"
			class="input w-20 px-2 py-1 text-right"
			:placeholder="t('common.unknown')"
			@input="update(d.code, ($event.target as HTMLInputElement).valueAsNumber)"
		>
		<span class="w-8 text-xs text-muted">{{ unitLabel[d.unit] }}</span>
	</label>
</div>
</template>
