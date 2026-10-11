<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { formatInt } from '~/utils/nutrients';
import { unwrap, useApi } from '~/composables/api/client';
import { formatQuantity } from '~/utils/meals';

const { t } = useI18n();
const route = useRoute();
const onError = useErrorHandler();
const api = useApi();
const id = computed(() => String(route.params.id));
const editing = ref(false);
const busy = ref(false);

const { data: recipe, refresh } = useAsyncData(() => `recipe-${id.value}`, () => unwrap(api.GET('/recipes/{id}', { params: { path: { id: id.value } } })));

async function update(body: Schemas['RecipeInput']) {
	busy.value = true;
	try {
		await unwrap(api.PUT('/recipes/{id}', { params: { path: { id: id.value } }, body }));
		await refresh();
		editing.value = false;
		toast(t('common.messages.saved'));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div v-if="recipe" class="space-y-4">
	<PageHeader back="/foods" :title="recipe.food.revision.name" :eyebrow="t('common.eyebrow.recipes')">
		<button v-if="recipe.food.editable && !editing" type="button" class="btn btn-ghost shrink-0" @click="editing = true">{{ t('common.edit') }}</button>
	</PageHeader>

	<RecipeForm v-if="editing" :initial="recipe" :busy="busy" :note="t('pages.recipe.updateNote')" @submit="update"/>
	<template v-else>
		<section v-reveal class="panel p-6 text-sm">
			<SectionHeading class="mb-3" icon="lucide:list" :eyebrow="t('common.eyebrow.ingredients')" :title="t('terms.ingredients')"/>
			<ul class="space-y-1">
				<li v-for="(l, i) in recipe.lines" :key="i" class="flex justify-between">
					<span>{{ l.name }}</span><span class="tabular-nums text-muted">{{ formatQuantity(l.quantity) }}</span>
				</li>
			</ul>
			<p class="mt-3 text-muted">{{ t('terms.rawGrams', { grams: formatInt(recipe.rawGrams) }) }} → {{ formatInt(recipe.cookedGrams) }}g</p>
			<p v-if="recipe.servings" class="text-muted">{{ t('pages.recipe.perServing', { grams: formatInt(recipe.cookedGrams / recipe.servings) }) }}</p>
		</section>
		<section v-reveal class="panel p-6">
			<SectionHeading class="mb-3" icon="lucide:leaf" :eyebrow="t('common.eyebrow.nutrients')" :title="t('terms.per100g')"/>
			<NutrientTable :profile="recipe.food.revision.nutrients"/>
		</section>
	</template>
</div>
</template>
