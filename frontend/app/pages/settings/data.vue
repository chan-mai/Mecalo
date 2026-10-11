<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const onError = useErrorHandler();
const busy = ref(false);

const exports = [
	{ href: '/api/export/json', label: 'pages.data.exportJson' },
	{ href: '/api/export/meals.csv', label: 'pages.data.exportMeals' },
	{ href: '/api/export/weights.csv', label: 'pages.data.exportWeights' },
] as const;

async function importFile(file: File) {
	busy.value = true;
	try {
		const body = JSON.parse(await file.text()) as Schemas['ExportBundle'];
		const res = await unwrap(useApi().POST('/import', { body }));
		toast(t('pages.data.importDone', { foods: res.foods, meals: res.meals, weights: res.weights }));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.data.title')" :eyebrow="t('common.eyebrow.data')"/>
	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-4" icon="lucide:download" :eyebrow="t('common.eyebrow.export')" :title="t('pages.data.export')"/>
		<ul class="space-y-2">
			<li v-for="item in exports" :key="item.href">
				<NuxtLink :to="item.href" external class="btn btn-ghost w-full !justify-between" download>
					{{ t(item.label) }}<Icon name="lucide:download" size="16" aria-hidden="true"/>
				</NuxtLink>
			</li>
		</ul>
	</section>
	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-4" icon="lucide:upload" :eyebrow="t('common.eyebrow.import')" :title="t('pages.data.import')"/>
		<FileDropzone accept="application/json,.json" :title="t('pages.data.dropTitle')" :hint="t('pages.data.importHint')" :busy="busy" @file="importFile"/>
	</section>
</div>
</template>
