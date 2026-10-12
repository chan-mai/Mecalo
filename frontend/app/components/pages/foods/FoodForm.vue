<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { foodInputSchema } from '~/schemas/food';

const props = defineProps<{ initial?: Schemas['Food']; busy?: boolean; note?: string; defaults?: { name?: string; barcode?: string } }>();
const emit = defineEmits<{ submit: [input: Schemas['FoodInput']] }>();
const { t } = useI18n();
const { catalog } = useNutrientCatalog();
const { errors, validate } = useValidation(foodInputSchema);

const primary = ['energy', 'protein', 'fat', 'carbohydrate', 'fiber', 'salt_equivalent'];
const primaryCodes = computed(() => catalog.value.filter((d) => primary.includes(d.code)));
const secondaryCodes = computed(() => catalog.value.filter((d) => !primary.includes(d.code)));

function initialForm() {
	const rev = props.initial?.revision;
	return {
		name: rev?.name ?? props.defaults?.name ?? '',
		brand: rev?.brand ?? '',
		aliases: rev?.aliases.join(', ') ?? '',
		barcode: props.initial?.barcode ?? props.defaults?.barcode ?? '',
		portions: rev?.portions.map((p) => ({ ...p })) ?? [] as Schemas['Portion'][],
		perUnit: rev?.basis !== undefined,
		basisUnit: rev?.basis?.unit ?? t('pages.foods.form.defaultBasisUnit'),
		basisGrams: rev?.basis?.grams as number | undefined,
		nutrients: Object.fromEntries(Object.entries(rev?.nutrients ?? {}).map(([k, v]) => [k, v.amount])) as Record<string, number | undefined>,
	};
}

const form = reactive(initialForm());

function submit() {
	const nutrients: Schemas['FoodInput']['nutrients'] = {};
	for (const [code, amount] of Object.entries(form.nutrients)) {
		if (typeof amount === 'number' && Number.isFinite(amount)) nutrients[code] = { amount };
	}
	const aliases = form.aliases.split(/[,、]/).map((a) => a.trim()).filter((a) => a !== '');
	const { perUnit, basisUnit, basisGrams, ...rest } = form;
	const grams = typeof basisGrams === 'number' && Number.isFinite(basisGrams) ? basisGrams : undefined;
	const basis = perUnit ? { unit: basisUnit, grams } : undefined;
	const body = validate({ ...rest, aliases, nutrients, basis });
	if (body) emit('submit', body);
}
</script>

<template>
<form class="space-y-4" novalidate @submit.prevent="submit">
	<section v-reveal class="panel space-y-3 p-6">
		<FormField v-slot="{ field }" :label="t('terms.foodName')" :error="errors.name">
			<input v-bind="field" v-model="form.name" class="input" required>
		</FormField>
		<FormField v-slot="{ field }" :label="t('terms.aliases')" :hint="t('pages.foods.form.aliasesHint')" :error="errors.aliases" optional>
			<input v-bind="field" v-model="form.aliases" class="input">
		</FormField>
		<div class="grid grid-cols-2 gap-3">
			<FormField v-slot="{ field }" :label="t('pages.foods.form.brand')" optional>
				<input v-bind="field" v-model="form.brand" class="input">
			</FormField>
			<FormField v-slot="{ field }" :label="t('pages.foods.form.barcode')" optional>
				<input v-bind="field" v-model="form.barcode" class="input" inputmode="numeric">
			</FormField>
		</div>
	</section>

	<section v-reveal class="panel space-y-2 p-6">
		<SectionHeading class="mb-3" icon="lucide:utensils" :eyebrow="t('common.eyebrow.portions')" :title="t('pages.foods.form.portions')"/>
		<div v-for="(p, i) in form.portions" :key="i" class="flex items-end gap-2">
			<FormField v-slot="{ field }" class="flex-1" :label="t('pages.foods.form.portionUnit')" :error="errors[`portions.${i}.unit`]">
				<input v-bind="field" v-model="p.unit" class="input">
			</FormField>
			<FormField v-slot="{ field }" class="w-28" :label="t('pages.foods.form.portionGrams')" :error="errors[`portions.${i}.grams`]">
				<input v-bind="field" v-model.number="p.grams" type="number" inputmode="decimal" step="any" class="input text-right">
			</FormField>
			<button type="button" class="btn btn-icon text-muted" :aria-label="t('common.delete')" @click="form.portions.splice(i, 1)"><Icon name="lucide:trash-2" size="18" aria-hidden="true"/></button>
		</div>
		<button type="button" class="btn btn-ghost w-full" @click="form.portions.push({ unit: '', grams: 0 })">
			<Icon name="lucide:plus" size="18" aria-hidden="true"/>{{ t('pages.foods.form.addPortion') }}
		</button>
	</section>

	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:leaf" :eyebrow="t('common.eyebrow.nutrients')" :title="form.perUnit ? t('terms.perBasis', { unit: form.basisUnit }) : t('pages.foods.form.nutrientsPer100g')"/>
		<fieldset class="mb-3 flex gap-1">
			<legend class="mb-1 text-sm text-muted">{{ t('pages.foods.form.basis') }}</legend>
			<label class="choice">
				<input v-model="form.perUnit" type="radio" name="food-basis" :value="false">
				<span>{{ t('terms.per100g') }}</span>
			</label>
			<label class="choice">
				<input v-model="form.perUnit" type="radio" name="food-basis" :value="true">
				<span>{{ t('pages.foods.form.perUnit') }}</span>
			</label>
		</fieldset>
		<div v-if="form.perUnit" class="mb-3 grid grid-cols-2 gap-3">
			<FormField v-slot="{ field }" :label="t('pages.foods.form.basisUnit')" :error="errors['basis.unit']">
				<input v-bind="field" v-model="form.basisUnit" class="input">
			</FormField>
			<FormField v-slot="{ field }" :label="t('pages.foods.form.basisGrams')" :hint="t('pages.foods.form.basisGramsHint')" :error="errors['basis.grams']" optional>
				<input v-bind="field" v-model.number="form.basisGrams" type="number" inputmode="decimal" step="any" class="input text-right">
			</FormField>
		</div>
		<p class="mb-3 text-xs text-muted">{{ t('pages.foods.form.blankIsUnknown') }}</p>
		<NutrientInputs v-model="form.nutrients" :codes="primaryCodes"/>
		<details class="mt-3">
			<summary class="cursor-pointer text-sm font-semibold">{{ t('pages.foods.form.moreNutrients') }}</summary>
			<NutrientInputs v-model="form.nutrients" class="mt-2" :codes="secondaryCodes"/>
		</details>
	</section>

	<p v-if="note" class="text-xs text-muted">{{ note }}</p>
	<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
</form>
</template>
