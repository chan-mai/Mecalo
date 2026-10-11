<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const route = useRoute();
const onError = useErrorHandler();
const api = useApi();
const id = computed(() => String(route.params.id));
const busy = ref(false);

const { data: food } = useAsyncData(() => `food-edit-${id.value}`, () => unwrap(api.GET('/foods/{id}', { params: { path: { id: id.value } } })));

async function update(body: Schemas['FoodInput']) {
	busy.value = true;
	try {
		await unwrap(api.PUT('/foods/{id}', { params: { path: { id: id.value } }, body }));
		clearNuxtData(`food-${id.value}`);
		toast(t('common.messages.saved'));
		await navigateTo(`/foods/${id.value}`, { replace: true });
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader :back="`/foods/${id}`" :title="t('common.edit')" :eyebrow="t('common.eyebrow.foods')"/>
	<FoodForm v-if="food" :initial="food" :busy="busy" :note="t('pages.foodEdit.updateNote')" @submit="update"/>
</div>
</template>
