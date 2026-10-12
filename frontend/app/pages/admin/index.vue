<script setup lang="ts">
definePageMeta({ middleware: 'admin' });
const { t } = useI18n();

const items = [
	{ to: '/admin/users', icon: 'lucide:users', key: 'users' },
	{ to: '/admin/settings', icon: 'lucide:sliders-horizontal', key: 'settings' },
	{ to: '/admin/sources', icon: 'lucide:database', key: 'sources' },
] as const;
</script>

<template>
<div class="space-y-4">
	<PageHeader :title="t('pages.adminIndex.title')" :eyebrow="t('common.eyebrow.admin')"/>
	<ul class="panel divide-y divide-[var(--hairline)]">
		<li v-for="item in items" :key="item.to">
			<NuxtLink :to="item.to" class="press flex items-center gap-3 px-4 py-3">
				<Icon :name="item.icon" size="20" class="shrink-0 text-muted" aria-hidden="true"/>
				<span class="min-w-0 flex-1">
					<span class="block font-medium">{{ t(`pages.adminIndex.items.${item.key}.title`) }}</span>
					<span class="block text-xs text-muted">{{ t(`pages.adminIndex.items.${item.key}.description`) }}</span>
				</span>
				<Icon name="lucide:chevron-right" size="16" class="shrink-0 text-muted" aria-hidden="true"/>
			</NuxtLink>
		</li>
	</ul>
</div>
</template>
