<script setup lang="ts">
import { unwrap, useApi } from '~/composables/api/client';

const { t } = useI18n();
const api = useApi();
const { data: datasets } = useAsyncData('datasets', () => unwrap(api.GET('/reference/datasets')));
const { data: dri } = useAsyncData('dri', () => unwrap(api.GET('/reference/dri')));
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.about.title')" :eyebrow="t('common.eyebrow.about')"/>
	<section v-reveal class="panel space-y-3 p-6 text-sm">
		<SectionHeading class="mb-1" icon="lucide:database" :eyebrow="t('common.eyebrow.data')" :title="t('pages.about.datasets')"/>
		<div v-for="d in datasets ?? []" :key="d.id">
			<p class="font-medium">{{ d.name }} <span v-if="d.version" class="text-muted">({{ d.version }})</span></p>
			<p v-if="d.attribution" class="text-muted">{{ d.attribution }}</p>
			<p v-if="d.license" class="text-muted">{{ d.license }}</p>
			<NuxtLink v-if="d.url" :to="d.url" class="link break-all" target="_blank">{{ d.url }}</NuxtLink>
		</div>
		<p v-if="datasets?.some((d) => d.kind === 'openfoodfacts')" class="text-xs text-muted">{{ t('pages.about.offAttribution') }}</p>
	</section>
	<section v-if="dri" v-reveal class="panel p-6 text-sm">
		<SectionHeading class="mb-3" icon="lucide:target" :eyebrow="t('common.eyebrow.reference')" :title="t('pages.about.dri')"/>
		<p>{{ dri.title }}</p>
		<NuxtLink :to="dri.url" class="link break-all" target="_blank">{{ dri.url }}</NuxtLink>
	</section>
	<section v-reveal class="panel p-6 text-sm">
		<SectionHeading class="mb-3" icon="lucide:code" :eyebrow="t('common.eyebrow.software')" :title="t('pages.about.software')"/>
		<p>{{ t('pages.about.license') }}</p>
		<NuxtLink to="https://github.com/chan-mai/mecalo" class="link" target="_blank">github.com/chan-mai/mecalo</NuxtLink>
	</section>
</div>
</template>
