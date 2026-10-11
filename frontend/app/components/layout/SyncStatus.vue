<script setup lang="ts">
const { t } = useI18n();
const { online, pending, failed, remove, retryFailed } = useOutbox();
const showFailed = ref(false);

async function retry() {
	await retryFailed();
	await refreshNuxtData();
	if (failed.value.length === 0) showFailed.value = false;
}
</script>

<template>
<div v-if="!online || pending.length || failed.length" class="sticky top-0 z-30 -mx-4 mb-2 px-4 pt-[env(safe-area-inset-top)]">
	<div class="flex items-center gap-2 rounded-xl px-3 py-2 text-xs" :class="!online ? 'bg-tertiary text-tertiary-ink' : 'bg-warning-soft text-warning'" role="status">
		<span class="flex-1">
			<template v-if="!online">{{ t('layout.syncStatus.offline') }}<template v-if="pending.length"> ({{ t('layout.syncStatus.pending', { count: pending.length }) }})</template></template>
			<template v-else-if="pending.length">{{ t('layout.syncStatus.syncing', { count: pending.length }) }}</template>
			<template v-if="failed.length"><span v-if="!online || pending.length"> / </span>{{ t('layout.syncStatus.failed', { count: failed.length }) }}</template>
		</span>
		<button v-if="failed.length" type="button" class="font-semibold underline" @click="showFailed = true">{{ t('common.edit') }}</button>
	</div>

	<BottomSheet :shown="showFailed" :title="t('layout.syncStatus.failedTitle')" @closed="showFailed = false">
		<ul class="divide-y divide-[var(--hairline)]">
			<li v-for="item in failed" :key="item.key" class="py-2">
				<p class="text-sm">{{ t(`terms.syncKind.${item.op.kind}`) }}<template v-if="item.op.kind === 'addMeal'">: {{ item.op.preview.foodName }}</template></p>
				<p class="text-xs text-danger">{{ item.error }}</p>
				<button type="button" class="mt-1 text-xs text-muted underline" @click="remove(item.key)">{{ t('layout.syncStatus.discard') }}</button>
			</li>
		</ul>
		<button type="button" class="btn btn-primary mt-3 w-full" :disabled="!online" @click="retry">{{ t('layout.syncStatus.retry') }}</button>
	</BottomSheet>
</div>
</template>
