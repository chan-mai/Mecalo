<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { formatDate } from '~/utils/date';

definePageMeta({ middleware: 'admin' });

const { t, locale } = useI18n();
const onError = useErrorHandler();
const api = useApi();

const { data: sources, refresh } = useAsyncData('admin-sources', () => unwrap(api.GET('/admin/sources')));

// 同期中は状態を定期取得
const { pause, resume } = useIntervalFn(() => void refresh(), 5000, { immediate: false });
watch(sources, (list) => (list?.some((s) => s.running) ? resume() : pause()), { immediate: true });

const catalogs = computed(() => (sources.value ?? []).filter((s) => s.kind === 'catalog'));
const anyRunning = computed(() => catalogs.value.some((s) => s.running));
// 開始要求の送信中
const starting = ref(false);

async function syncAll() {
	starting.value = true;
	try {
		await unwrap(api.POST('/admin/sources/sync'));
		toast(t('pages.adminSources.startedAll'));
		await refresh();
	} catch (err) {
		if (err instanceof ApiError && err.status === 409) {
			toast(t('pages.adminSources.running'));
			return;
		}
		onError(err);
	} finally {
		starting.value = false;
	}
}

async function sync(s: Schemas['Source']) {
	starting.value = true;
	try {
		await unwrap(api.POST('/admin/sources/{id}/sync', { params: { path: { id: s.dataset.id } } }));
		toast(t('pages.adminSources.started', { name: s.dataset.name }));
		await refresh();
	} catch (err) {
		if (err instanceof ApiError && err.status === 409) {
			toast(t('pages.adminSources.running'));
			return;
		}
		onError(err);
	} finally {
		starting.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/admin" :title="t('pages.adminSources.title')" :eyebrow="t('common.eyebrow.data')"/>
	<p class="text-sm text-muted">{{ t('pages.adminSources.note') }}</p>
	<button v-if="catalogs.length" type="button" class="btn btn-primary w-full" :disabled="anyRunning || starting" @click="syncAll">
		<Icon :name="anyRunning ? 'lucide:loader-circle' : 'lucide:refresh-cw'" size="16" :class="{ 'animate-spin': anyRunning }" aria-hidden="true"/>
		{{ t(anyRunning ? 'pages.adminSources.running' : 'pages.adminSources.syncAll') }}
	</button>
	<p v-if="sources && sources.length === 0" class="panel p-6 text-sm text-muted">{{ t('pages.adminSources.empty') }}</p>
	<ul v-else class="panel divide-y divide-[var(--hairline)]">
		<li v-for="s in sources ?? []" :key="s.dataset.id" class="flex items-center gap-3 px-4 py-3 text-sm">
			<div class="min-w-0 flex-1">
				<p class="font-medium">{{ s.dataset.name }} <span class="text-xs text-muted">{{ t(`pages.adminSources.kind.${s.kind}`) }}</span></p>
				<p v-if="s.running && s.runningSince" class="text-xs text-muted" role="status">
					{{ t('pages.adminSources.progress', { date: formatDate(s.runningSince, t('common.dateFormat.dateTime'), locale), items: s.runningItems ?? 0 }) }}
				</p>
				<p v-else-if="s.running" class="text-xs text-muted" role="status">{{ t('pages.adminSources.queued') }}</p>
				<p v-else-if="s.syncedAt" class="text-xs text-muted">
					{{ t('pages.adminSources.syncedAt', { date: formatDate(s.syncedAt, t('common.dateFormat.dateTime'), locale), items: s.items }) }}
				</p>
				<p v-if="s.error" class="mt-1 break-all text-xs text-danger">
					<Icon name="lucide:circle-alert" size="14" class="mr-1 align-[-2px]" aria-hidden="true"/>{{ s.error }}
				</p>
			</div>
			<button v-if="s.kind === 'catalog'" type="button" class="btn btn-ghost shrink-0" :disabled="anyRunning || starting" @click="sync(s)">
				<Icon :name="s.runningSince ? 'lucide:loader-circle' : s.running ? 'lucide:clock' : 'lucide:refresh-cw'" size="16" :class="{ 'animate-spin': s.runningSince }" aria-hidden="true"/>
				{{ t(s.runningSince ? 'pages.adminSources.running' : s.running ? 'pages.adminSources.queued' : 'pages.adminSources.sync') }}
			</button>
		</li>
	</ul>
</div>
</template>
