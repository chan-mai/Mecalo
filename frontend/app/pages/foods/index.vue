<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const api = useApi();

const { data: favorites } = useAsyncData('favorites', () => unwrap(api.GET('/favorites')));
const { data: mine } = useAsyncData('my-foods', () => unwrap(api.GET('/foods/mine')));
const visibleMine = computed(() => (mine.value ?? []).filter((f) => !f.deleted));

function linkOf(f: Schemas['Food']) {
	return f.kind === 'recipe' ? `/recipes/${f.id}` : `/foods/${f.id}`;
}

function openFood(f: Schemas['Food']) {
	void navigateTo(linkOf(f));
}
</script>

<template>
<div class="space-y-4">
	<PageHeader :title="t('pages.foods.title')" :eyebrow="t('common.eyebrow.foods')"/>
	<div class="grid grid-cols-2 gap-2">
		<NuxtLink to="/foods/new" class="btn btn-primary"><Icon name="lucide:plus" size="18" aria-hidden="true"/>{{ t('common.newFood') }}</NuxtLink>
		<NuxtLink to="/recipes/new" class="btn btn-ghost"><Icon name="lucide:plus" size="18" aria-hidden="true"/>{{ t('common.newRecipe') }}</NuxtLink>
	</div>

	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:star" :eyebrow="t('common.eyebrow.favorites')" :title="t('terms.favorites')"/>
		<ul class="divide-y divide-[var(--hairline)]">
			<FoodRow v-for="f in favorites ?? []" :key="f.id" :food="f" @select="openFood(f)"/>
		</ul>
		<p v-if="favorites?.length === 0" class="text-sm text-muted">{{ t('pages.foods.noFavorites') }}</p>
	</section>

	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:book-open" :eyebrow="t('common.eyebrow.mine')" :title="t('pages.foods.mine')"/>
		<ul class="divide-y divide-[var(--hairline)]">
			<li v-for="f in visibleMine" :key="f.id">
				<NuxtLink :to="linkOf(f)" class="row flex items-center justify-between py-3">
					<span class="min-w-0 truncate">{{ f.revision.name }}</span>
					<span class="eyebrow shrink-0">{{ t(`terms.foodKind.${f.kind}`) }}</span>
				</NuxtLink>
			</li>
		</ul>
		<p v-if="visibleMine.length === 0" class="text-sm text-muted">{{ t('pages.foods.empty') }}</p>
	</section>

	<section v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:search" :eyebrow="t('common.eyebrow.search')" :title="t('common.search')"/>
		<FoodPicker :quickAdd="false" @select="openFood"/>
	</section>
</div>
</template>
