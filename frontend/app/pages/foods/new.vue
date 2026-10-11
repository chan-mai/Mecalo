<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const route = useRoute();
const onError = useErrorHandler();
const busy = ref(false);

function str(v: unknown): string | undefined {
	return typeof v === 'string' ? v : undefined;
}

async function create(body: Schemas['FoodInput']) {
	busy.value = true;
	try {
		const food = await unwrap(useApi().POST('/foods', { body }));
		toast(t('common.messages.saved'));
		await navigateTo(`/foods/${food.id}`, { replace: true });
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/foods" :title="t('common.newFood')" :eyebrow="t('common.eyebrow.create')"/>
	<FoodForm :busy="busy" :defaults="{ name: str(route.query.name), barcode: str(route.query.barcode) }" @submit="create"/>
</div>
</template>
