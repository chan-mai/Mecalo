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

async function sync(s: Schemas['Source']) {
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
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/admin" :title="t('pages.adminSources.title')" :eyebrow="t('common.eyebrow.data')"/>
	<p class="text-sm text-muted">{{ t('pages.adminSources.note') }}</p>
	<p v-if="sources && sources.length === 0" class="panel p-6 text-sm text-muted">{{ t('pages.adminSources.empty') }}</p>
	<ul v-else class="panel divide-y divide-[var(--hairline)]">
		<li v-for="s in sources ?? []" :key="s.dataset.id" class="flex items-center gap-3 px-4 py-3 text-sm">
			<div class="min-w-0 flex-1">
				<p class="font-medium">{{ s.dataset.name }} <span class="text-xs text-muted">{{ t(`pages.adminSources.kind.${s.kind}`) }}</span></p>
				<p v-if="s.running" class="text-xs text-muted" role="status">{{ t('pages.adminSources.running') }}</p>
				<p v-else-if="s.syncedAt" class="text-xs text-muted">
					{{ t('pages.adminSources.syncedAt', { date: formatDate(s.syncedAt, t('common.dateFormat.full'), locale), items: s.items }) }}
				</p>
				<p v-if="s.error" class="mt-1 break-all text-xs text-danger">
					<Icon name="lucide:circle-alert" size="14" class="mr-1 align-[-2px]" aria-hidden="true"/>{{ s.error }}
				</p>
			</div>
			<button v-if="s.kind === 'catalog'" type="button" class="btn btn-ghost shrink-0" :disabled="s.running" @click="sync(s)">
				<Icon name="lucide:refresh-cw" size="16" aria-hidden="true"/>
				{{ t('pages.adminSources.sync') }}
			</button>
		</li>
	</ul>
</div>
</template>
