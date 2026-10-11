<script setup lang="ts">
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { credentialsSchema } from '~/schemas/auth';

definePageMeta({ layout: 'auth' });

const { t } = useI18n();
const route = useRoute();
const { refresh } = useAuth();
const onError = useErrorHandler();
const api = useApi();
const { errors, validate } = useValidation(credentialsSchema);

const form = reactive({ loginName: '', password: '' });
const busy = ref(false);
const failed = ref(false);

const { data: config } = useAsyncData('auth-config', () => unwrap(api.GET('/auth/config')));

watch(config, (c) => {
	if (c?.setupRequired) void navigateTo('/setup');
}, { immediate: true });

const returnTo = computed(() => {
	const r = route.query.returnTo;
	return typeof r === 'string' && r.startsWith('/') && !r.startsWith('//') ? r : '/';
});
const oidcHref = computed(() => `/api/auth/oidc/start?returnTo=${encodeURIComponent(returnTo.value)}`);

const errorMessage = computed(() => {
	if (failed.value) return t('errors.auth.failed');
	if (route.query.error === 'oidc') return t('errors.auth.oidc');
	if (route.query.error === 'not_registered') return t('errors.auth.notRegistered');
	return '';
});

async function submit() {
	const body = validate(form);
	if (!body) return;
	busy.value = true;
	failed.value = false;
	try {
		await unwrap(api.POST('/auth/login', { body }));
		await refresh();
		await navigateTo(returnTo.value);
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) failed.value = true;
		else onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div>
	<p v-if="errorMessage" role="alert" class="mb-4 rounded-lg bg-danger-soft p-2 text-sm text-danger">{{ errorMessage }}</p>

	<form v-if="config?.localEnabled" class="space-y-4" novalidate @submit.prevent="submit">
		<FormField v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName">
			<input v-bind="field" v-model="form.loginName" class="input" autocomplete="username" autocapitalize="none" required>
		</FormField>
		<FormField v-slot="{ field }" :label="t('terms.password')" :error="errors.password">
			<input v-bind="field" v-model="form.password" type="password" class="input" autocomplete="current-password" required>
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('pages.auth.login') }}</button>
	</form>

	<NuxtLink v-if="config?.oidcEnabled" :to="oidcHref" external class="btn btn-ghost mt-3 w-full">
		{{ t('pages.auth.loginWith', { name: config.oidcName ?? 'OIDC' }) }}
	</NuxtLink>

	<p v-if="config?.registrationOpen" class="mt-4 text-center text-sm text-muted">
		{{ t('pages.auth.noAccount') }}
		<NuxtLink to="/signup" class="link">{{ t('pages.auth.signup') }}</NuxtLink>
	</p>
</div>
</template>
