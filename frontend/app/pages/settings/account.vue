<script setup lang="ts">
import { toast } from 'vue-sonner';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { profileUpdateSchema } from '~/schemas/auth';

const { t } = useI18n();
const { me, refresh } = useAuth();
const onError = useErrorHandler();
const { errors, validate } = useValidation(profileUpdateSchema);

const form = reactive({ displayName: '', loginName: '' });
const busy = ref(false);

watch(
	me,
	(u) => {
		if (u) Object.assign(form, { displayName: u.displayName, loginName: u.loginName ?? '' });
	},
	{ immediate: true },
);

async function submit() {
	const body = validate({ displayName: form.displayName, loginName: me.value?.loginName ? form.loginName : undefined });
	if (!body) return;
	busy.value = true;
	try {
		await unwrap(useApi().PATCH('/me', { body }));
		await refresh();
		toast(t('common.messages.saved'));
	} catch (err) {
		if (err instanceof ApiError && err.status === 409) {
			errors.value = { loginName: 'errors.auth.loginNameTaken' };
			return;
		}
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.account.title')" :eyebrow="t('common.eyebrow.account')"/>
	<form v-reveal class="panel space-y-4 p-6" novalidate @submit.prevent="submit">
		<FormField v-slot="{ field }" :label="t('terms.displayName')" :error="errors.displayName">
			<input v-bind="field" v-model="form.displayName" class="input" autocomplete="nickname"/>
		</FormField>
		<FormField v-if="me?.loginName" v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName">
			<input v-bind="field" v-model="form.loginName" class="input" autocapitalize="none" autocomplete="username"/>
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
	</form>
	<NuxtLink v-if="me?.loginName" to="/settings/password" class="panel press flex items-center justify-between px-6 py-4 text-sm">
		{{ t('pages.settings.links.password') }}<Icon name="lucide:chevron-right" class="text-muted" size="16" aria-hidden="true"/>
	</NuxtLink>
</div>
</template>
