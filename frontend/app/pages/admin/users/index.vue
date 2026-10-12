<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';
import { signUpSchema } from '~/schemas/auth';

definePageMeta({ middleware: 'admin' });

const { t } = useI18n();
const onError = useErrorHandler();
const api = useApi();
const { errors, validate } = useValidation(signUpSchema);

const { data: users, refresh } = useAsyncData('admin-users', () => unwrap(api.GET('/admin/users')));
const form = reactive({ loginName: '', displayName: '', password: '' });
const role = ref<Schemas['Role']>('member');

async function create() {
	const body = validate(form);
	if (!body) return;
	try {
		await unwrap(api.POST('/admin/users', { body: { ...body, role: role.value } }));
		toast(t('common.messages.saved'));
		Object.assign(form, { loginName: '', displayName: '', password: '' });
		await refresh();
	} catch (err) {
		onError(err);
	}
}

</script>

<template>
<div class="space-y-4">
	<PageHeader back="/admin" :title="t('pages.admin.title')" :eyebrow="t('common.eyebrow.users')"/>
	<ul class="panel divide-y divide-[var(--hairline)]">
		<li v-for="u in users ?? []" :key="u.id" class="flex items-center gap-2 px-4 py-2 text-sm">
			<NuxtLink :to="`/admin/users/${u.id}`" class="press min-w-0 flex-1">
				<span class="block truncate">{{ u.displayName }} <span class="text-xs text-muted">{{ t(`terms.role.${u.role}`) }}</span></span>
				<span class="block truncate text-xs text-muted">{{ u.loginName ?? t('pages.admin.externalOnly') }}</span>
			</NuxtLink>
			<Icon name="lucide:chevron-right" size="16" class="shrink-0 text-muted" aria-hidden="true"/>
		</li>
	</ul>
	<form v-reveal class="panel space-y-3 p-6" novalidate @submit.prevent="create">
		<SectionHeading class="mb-1" icon="lucide:user-plus" :eyebrow="t('common.eyebrow.create')" :title="t('pages.admin.create')"/>
		<FormField v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName"><input v-bind="field" v-model="form.loginName" class="input" autocapitalize="none"></FormField>
		<FormField v-slot="{ field }" :label="t('terms.displayName')" optional><input v-bind="field" v-model="form.displayName" class="input"></FormField>
		<FormField v-slot="{ field }" :label="t('terms.password')" :error="errors.password"><input v-bind="field" v-model="form.password" type="password" class="input" autocomplete="new-password"></FormField>
		<FormField v-slot="{ labelId }" :label="t('pages.admin.role')">
			<RoleSelector v-model="role" :labelledby="labelId"/>
		</FormField>
		<button class="btn btn-primary w-full">{{ t('pages.admin.create') }}</button>
	</form>
</div>
</template>
