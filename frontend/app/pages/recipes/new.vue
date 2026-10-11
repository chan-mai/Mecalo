<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const onError = useErrorHandler();
const busy = ref(false);

async function create(body: Schemas['RecipeInput']) {
	busy.value = true;
	try {
		const recipe = await unwrap(useApi().POST('/recipes', { body }));
		toast(t('common.messages.saved'));
		await navigateTo(`/recipes/${recipe.food.id}`, { replace: true });
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/foods" :title="t('common.newRecipe')" :eyebrow="t('common.eyebrow.create')"/>
	<RecipeForm :busy="busy" @submit="create"/>
</div>
</template>
