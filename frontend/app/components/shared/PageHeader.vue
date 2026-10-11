<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router';

// backHistory: 遷移元へ戻る
defineProps<{ title: string; eyebrow: string; back?: RouteLocationRaw; backHistory?: boolean }>();
const { t } = useI18n();
const router = useRouter();
</script>

<template>
<header class="mb-6 flex items-center gap-3">
	<NuxtLink v-if="back" :to="back" class="btn-round shrink-0" :aria-label="t('common.back')">
		<Icon name="lucide:arrow-left" size="18" aria-hidden="true"/>
	</NuxtLink>
	<button v-else-if="backHistory" type="button" class="btn-round shrink-0" :aria-label="t('common.back')" @click="router.back()">
		<Icon name="lucide:arrow-left" size="18" aria-hidden="true"/>
	</button>
	<div class="min-w-0 flex-1">
		<p class="eyebrow" aria-hidden="true">{{ eyebrow }}</p>
		<h1 class="mt-1 truncate text-2xl leading-tight font-bold">{{ title }}</h1>
	</div>
	<slot></slot>
</header>
</template>
