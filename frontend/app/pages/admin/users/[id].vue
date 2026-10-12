<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { passwordResetSchema, profileUpdateSchema } from '~/schemas/auth';

definePageMeta({ middleware: 'admin' });

const { t } = useI18n();
const route = useRoute();
const { me, refresh: refreshMe } = useAuth();
const onError = useErrorHandler();
const confirm = useConfirm();
const confirmDelete = useConfirmDelete();
const api = useApi();
const profile = useValidation(profileUpdateSchema);
const reset = useValidation(passwordResetSchema);

const id = computed(() => String(route.params.id));
const { data: target } = useAsyncData(() => `admin-user-${id.value}`, () => unwrap(api.GET('/admin/users/{id}', { params: { path: { id: id.value } } })));

const form = reactive<{ displayName: string; loginName: string; role: Schemas['Role'] }>({ displayName: '', loginName: '', role: 'member' });
const password = ref('');
const busy = ref(false);
const self = computed(() => target.value?.id === me.value?.id);

watch(
	target,
	(u) => {
		if (!u) return;
		Object.assign(form, { displayName: u.displayName, loginName: u.loginName ?? '', role: u.role });
	},
	{ immediate: true },
);

async function save() {
	const body = profile.validate({ displayName: form.displayName, loginName: target.value?.loginName ? form.loginName : undefined });
	if (!body) return;
	busy.value = true;
	try {
		target.value = await unwrap(api.PATCH('/admin/users/{id}', { params: { path: { id: id.value } }, body: { ...body, role: form.role } }));
		if (self.value) await refreshMe();
		toast(t('common.messages.saved'));
	} catch (err) {
		if (err instanceof ApiError && err.status === 409) {
			profile.errors.value = { loginName: 'errors.auth.loginNameTaken' };
			return;
		}
		onError(err);
	} finally {
		busy.value = false;
	}
}

async function remove() {
	if (!await confirmDelete()) return;
	busy.value = true;
	try {
		await unwrap(api.DELETE('/admin/users/{id}', { params: { path: { id: id.value } } }));
		toast(t('common.messages.deleted'));
		await navigateTo('/admin/users');
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}

async function resetPassword() {
	const body = reset.validate({ password: password.value });
	if (!body) return;
	if (
		!(await confirm({
			title: t('pages.adminUser.resetConfirm'),
			message: self.value ? undefined : t('pages.adminUser.resetMessage'),
			confirmLabel: t('pages.adminUser.reset'),
			danger: true,
		}))
	) return;
	busy.value = true;
	try {
		await unwrap(api.PATCH('/admin/users/{id}', { params: { path: { id: id.value } }, body }));
		password.value = '';
		toast(t('pages.adminUser.resetDone'));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/admin/users" :title="target?.displayName ?? ''" :eyebrow="t('common.eyebrow.users')"/>
	<form v-if="target" class="panel space-y-4 p-6" novalidate @submit.prevent="save">
		<FormField v-slot="{ field }" :label="t('terms.displayName')" :error="profile.errors.value.displayName">
			<input v-bind="field" v-model="form.displayName" class="input" autocomplete="off"/>
		</FormField>
		<FormField v-if="target.loginName" v-slot="{ field }" :label="t('terms.loginName')" :error="profile.errors.value.loginName">
			<input v-bind="field" v-model="form.loginName" class="input" autocapitalize="none" autocomplete="off"/>
		</FormField>
		<p v-else class="text-sm text-muted">{{ t('pages.admin.externalOnly') }}</p>
		<FormField v-slot="{ labelId }" :label="t('pages.admin.role')">
			<RoleSelector v-model="form.role" :labelledby="labelId" :disabled="self"/>
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
	</form>
	<form v-if="target?.loginName" class="panel space-y-4 p-6" novalidate @submit.prevent="resetPassword">
		<h2 class="font-medium">{{ t('pages.adminUser.reset') }}</h2>
		<FormField v-slot="{ field }" :label="t('pages.password.new')" :error="reset.errors.value.password">
			<input v-bind="field" v-model="password" type="password" class="input" autocomplete="new-password"/>
		</FormField>
		<button class="btn btn-ghost w-full text-danger" :disabled="busy">{{ t('pages.adminUser.reset') }}</button>
	</form>
	<button v-if="target && !self" type="button" class="btn btn-ghost w-full text-danger" :disabled="busy" @click="remove">
		<Icon name="lucide:trash-2" size="16" aria-hidden="true"/>
		{{ t('pages.adminUser.delete') }}
	</button>
</div>
</template>
