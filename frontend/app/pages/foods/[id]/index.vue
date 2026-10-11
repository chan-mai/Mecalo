<script setup lang="ts">
import { toast } from 'vue-sonner';
import { unwrap, useApi } from '~/composables/api/client';
import { formatDate } from '~/utils/date';

const { t, locale } = useI18n();
const route = useRoute();
const onError = useErrorHandler();
const api = useApi();
const id = computed(() => String(route.params.id));

const { data: food, refresh } = useAsyncData(() => `food-${id.value}`, () => unwrap(api.GET('/foods/{id}', { params: { path: { id: id.value } } })));
const { data: revisions } = useAsyncData(() => `food-revisions-${id.value}`, () => unwrap(api.GET('/foods/{id}/revisions', { params: { path: { id: id.value } } })));
const { data: datasets } = useAsyncData('datasets', () => unwrap(api.GET('/reference/datasets')));

const datasetLabel = computed(() => datasets.value?.find((d) => d.id === food.value?.revision.source.datasetId)?.name ?? food.value?.revision.source.datasetId);

async function toggleFavorite() {
	if (!food.value) return;
	try {
		const path = { params: { path: { id: id.value } } };
		await unwrap(food.value.favorite ? api.DELETE('/foods/{id}/favorite', path) : api.PUT('/foods/{id}/favorite', path));
		await refresh();
	} catch (err) {
		onError(err);
	}
}

async function remove() {
	if (!window.confirm(t('common.confirmDelete'))) return;
	try {
		await unwrap(api.DELETE('/foods/{id}', { params: { path: { id: id.value } } }));
		toast(t('common.messages.deleted'));
		await navigateTo('/foods', { replace: true });
	} catch (err) {
		onError(err);
	}
}
</script>

<template>
<div v-if="food" class="space-y-4">
	<PageHeader backHistory :title="food.revision.name" :eyebrow="t('common.eyebrow.foods')">
		<button type="button" class="btn-round favorite shrink-0" :class="food.favorite ? 'text-warning' : 'text-muted'" :aria-pressed="food.favorite" :aria-label="t('pages.food.favorite')" @click="toggleFavorite">
			<Icon name="lucide:star" size="18" aria-hidden="true"/>
		</button>
	</PageHeader>
	<p v-if="food.deleted" class="rounded-lg bg-warning-soft p-2 text-sm text-warning">{{ t('pages.food.deletedNote') }}</p>

	<section v-reveal class="panel p-6">
		<p class="text-sm text-muted">
			{{ t(`terms.foodKind.${food.kind}`) }}<template v-if="food.revision.brand"> · {{ food.revision.brand }}</template>
			<template v-if="food.barcode"> · {{ food.barcode }}</template>
		</p>
		<p v-if="food.revision.aliases.length" class="mt-1 text-sm text-muted">{{ t('terms.aliases') }}: {{ food.revision.aliases.join(', ') }}</p>
		<p v-if="food.revision.portions.length" class="mt-1 text-sm">
			<span v-for="p in food.revision.portions" :key="p.unit" class="mr-3">1{{ p.unit }} = {{ p.grams }}g</span>
		</p>
		<SectionHeading class="mt-5 mb-3" icon="lucide:leaf" :eyebrow="t('common.eyebrow.nutrients')" :title="t('terms.per100g')"/>
		<NutrientTable :profile="food.revision.nutrients"/>
		<p class="mt-3 text-xs text-muted">{{ t('pages.food.source') }}: {{ datasetLabel }} {{ food.revision.source.reference }}</p>
	</section>

	<div v-if="food.editable && !food.deleted" class="grid grid-cols-2 gap-2">
		<NuxtLink :to="`/foods/${food.id}/edit`" class="btn btn-ghost">{{ t('common.edit') }}</NuxtLink>
		<button type="button" class="btn btn-ghost text-danger" @click="remove">{{ t('common.delete') }}</button>
	</div>

	<details v-if="revisions && revisions.length > 1" class="panel p-6">
		<summary class="cursor-pointer font-semibold">{{ t('pages.food.revisions') }}</summary>
		<ul class="mt-2 space-y-1 text-sm">
			<li v-for="r in [...revisions].reverse()" :key="r.id" class="flex justify-between">
				<span>{{ t('pages.food.revision', { number: r.number }) }} {{ r.name }}</span>
				<span class="text-muted">{{ formatDate(r.createdAt, t('common.dateFormat.full'), locale) }}</span>
			</li>
		</ul>
	</details>
</div>
</template>

<style scoped>
.favorite[aria-pressed="true"] svg,
.favorite[aria-pressed="true"] .iconify {
	animation: pop 0.5s var(--ease-out-expo);
}

@keyframes pop {
	40% {
		transform: scale(1.25);
	}
}
</style>
