<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';
import { signUpSchema } from '~/schemas/auth';

const { t } = useI18n();
const confirmDelete = useConfirmDelete();
const { me } = useAuth();
const onError = useErrorHandler();
const api = useApi();
const { errors, validate } = useValidation(signUpSchema);

const { data: users, refresh } = useAsyncData('admin-users', () => unwrap(api.GET('/admin/users')));
const form = reactive({ loginName: '', displayName: '', password: '' });
const roles = ['member', 'admin'] as const satisfies readonly Schemas['Role'][];
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

async function remove(id: string) {
	if (!await confirmDelete()) return;
	try {
		await unwrap(api.DELETE('/admin/users/{id}', { params: { path: { id } } }));
		await refresh();
	} catch (err) {
		onError(err);
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.admin.title')" :eyebrow="t('common.eyebrow.users')"/>
	<ul class="panel divide-y divide-[var(--hairline)]">
		<li v-for="u in users ?? []" :key="u.id" class="flex items-center justify-between px-4 py-2 text-sm">
			<span>{{ u.displayName }} <span class="text-xs text-muted">{{ t(`terms.role.${u.role}`) }}</span></span>
			<button v-if="u.id !== me?.id" type="button" class="btn btn-icon text-danger" :aria-label="t('common.delete')" @click="remove(u.id)"><Icon name="lucide:trash-2" size="16" aria-hidden="true"/></button>
		</li>
	</ul>
	<form v-reveal class="panel space-y-3 p-6" novalidate @submit.prevent="create">
		<SectionHeading class="mb-1" icon="lucide:user-plus" :eyebrow="t('common.eyebrow.create')" :title="t('pages.admin.create')"/>
		<FormField v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName"><input v-bind="field" v-model="form.loginName" class="input" autocapitalize="none"></FormField>
		<FormField v-slot="{ field }" :label="t('terms.displayName')" optional><input v-bind="field" v-model="form.displayName" class="input"></FormField>
		<FormField v-slot="{ field }" :label="t('terms.password')" :error="errors.password"><input v-bind="field" v-model="form.password" type="password" class="input" autocomplete="new-password"></FormField>
		<FormField v-slot="{ field, labelId }" :label="t('pages.admin.role')">
			<SelectField v-bind="field" v-model="role" :labelledby="labelId" :options="roles.map((r) => ({ value: r, label: t(`terms.role.${r}`) }))"/>
		</FormField>
		<button class="btn btn-primary w-full">{{ t('pages.admin.create') }}</button>
	</form>
</div>
</template>
