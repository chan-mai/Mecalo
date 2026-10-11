<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { formatInt } from '~/utils/nutrients';
import { unwrap, useApi } from '~/composables/api/client';
import { recipeInputSchema } from '~/schemas/food';
import { defaultQuantity, gramsOf } from '~/utils/quantity';

const props = defineProps<{ initial?: Schemas['Recipe']; busy?: boolean; note?: string }>();
const emit = defineEmits<{ submit: [input: Schemas['RecipeInput']] }>();
const { t } = useI18n();
const { errors, validate } = useValidation(recipeInputSchema);

interface Line {
	food: Schemas['Food'];
	quantity: Schemas['Quantity'];
}

const name = ref('');
const ingredients = ref<Line[]>([]);
const cookedGrams = ref<number | undefined>();
const servings = ref<number | undefined>();
const picking = ref(false);

onMounted(async () => {
	const initial = props.initial;
	if (!initial) return;
	name.value = initial.food.revision.name;
	cookedGrams.value = initial.cookedGrams;
	servings.value = initial.servings > 0 ? initial.servings : undefined;
	const api = useApi();
	ingredients.value = await Promise.all(initial.lines.map(async (l) => ({
		food: await unwrap(api.GET('/foods/{id}', { params: { path: { id: l.foodId } } })),
		quantity: { ...l.quantity },
	})));
});

const rawGrams = computed(() => ingredients.value.reduce((sum, l) => sum + (gramsOf(l.food.revision, l.quantity) ?? 0), 0));

function add(food: Schemas['Food'], quantity?: Schemas['Quantity']) {
	ingredients.value.push({ food, quantity: quantity ?? defaultQuantity(food.revision) });
	picking.value = false;
}

function finite(n: unknown): number | undefined {
	return typeof n === 'number' && Number.isFinite(n) ? n : undefined;
}

function submit() {
	const body = validate({
		name: name.value,
		ingredients: ingredients.value.map((l) => ({ foodId: l.food.id, quantity: l.quantity })),
		cookedGrams: finite(cookedGrams.value),
		servings: finite(servings.value),
	});
	if (body) emit('submit', body);
}
</script>

<template>
<div>
	<form class="space-y-4" novalidate @submit.prevent="submit">
		<section v-reveal class="panel p-6">
			<FormField v-slot="{ field }" :label="t('terms.foodName')" :error="errors.name">
				<input v-bind="field" v-model="name" class="input" required>
			</FormField>
		</section>

		<section v-reveal class="panel space-y-2 p-6">
			<SectionHeading class="mb-3" icon="lucide:list" :eyebrow="t('common.eyebrow.ingredients')" :title="t('terms.ingredients')"/>
			<ul class="divide-y divide-[var(--hairline)]">
				<li v-for="(ing, i) in ingredients" :key="i" class="flex items-center gap-2 py-2">
					<span class="min-w-0 flex-1 truncate text-sm">{{ ing.food.revision.name }}</span>
					<input v-model.number="ing.quantity.amount" type="number" inputmode="decimal" step="any" class="input w-20 px-2 py-1 text-right" :aria-label="t('terms.quantity')">
					<SelectField v-model="ing.quantity.unit" class="w-28" compact :label="t('terms.unit')" :options="[...ing.food.revision.portions.map((p) => ({ value: p.unit, label: p.unit })), { value: 'g', label: 'g' }]"/>
					<button type="button" class="btn btn-icon text-muted" :aria-label="t('common.delete')" @click="ingredients.splice(i, 1)"><Icon name="lucide:trash-2" size="18" aria-hidden="true"/></button>
				</li>
			</ul>
			<p v-if="errors.ingredients" class="text-xs text-danger">{{ t(errors.ingredients) }}</p>
			<button type="button" class="btn btn-ghost w-full" @click="picking = true"><Icon name="lucide:plus" size="18" aria-hidden="true"/>{{ t('pages.recipe.form.addIngredient') }}</button>
			<p class="text-xs text-muted">{{ t('terms.rawGrams', { grams: formatInt(rawGrams) }) }}</p>
		</section>

		<section v-reveal class="panel grid grid-cols-2 gap-3 p-6">
			<FormField v-slot="{ field }" :label="t('pages.recipe.form.cookedGrams')" :hint="t('pages.recipe.form.cookedGramsHint')" :error="errors.cookedGrams">
				<input v-bind="field" v-model.number="cookedGrams" type="number" inputmode="decimal" step="any" class="input text-right">
			</FormField>
			<FormField v-slot="{ field }" :label="t('pages.recipe.form.servings')" :error="errors.servings" optional>
				<input v-bind="field" v-model.number="servings" type="number" inputmode="decimal" step="any" class="input text-right">
			</FormField>
		</section>

		<p v-if="note" class="text-xs text-muted">{{ note }}</p>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
	</form>

	<BottomSheet :shown="picking" :title="t('pages.recipe.form.addIngredient')" @closed="picking = false">
		<FoodPicker :quickAdd="false" @select="add"/>
	</BottomSheet>
</div>
</template>
