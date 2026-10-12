<script setup lang="ts">
import { toast } from 'vue-sonner';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { setupSchema, signUpSchema } from '~/schemas/auth';

const props = defineProps<{ title: string; hint?: string; path: '/auth/setup' | '/auth/signup' }>();
const { t } = useI18n();
const { refresh } = useAuth();
const curtain = useEntryCurtain();
const submitButton = useTemplateRef<HTMLButtonElement>('submitButton');
const onError = useErrorHandler();
const signUp = useValidation(signUpSchema);
const setup = useValidation(setupSchema);
const isSetup = computed(() => props.path === '/auth/setup');
const errors = computed(() => (isSetup.value ? setup.errors.value : signUp.errors.value));

const form = reactive({ loginName: '', displayName: '', password: '', setupToken: '' });
const busy = ref(false);

// 入力不備はfalse
async function post(): Promise<boolean> {
	if (isSetup.value) {
		const body = setup.validate(form);
		if (body) await unwrap(useApi().POST('/auth/setup', { body }));
		return body !== undefined;
	}
	const body = signUp.validate(form);
	if (body) await unwrap(useApi().POST('/auth/signup', { body }));
	return body !== undefined;
}

async function submit() {
	busy.value = true;
	try {
		if (!await post()) return;
		await refresh();
		await curtain.enter(() => navigateTo('/settings'), submitButton.value);
	} catch (err) {
		if (isSetup.value && err instanceof ApiError && err.status === 403) {
			setup.errors.value = { setupToken: 'errors.auth.setupToken' };
			return;
		}
		if (isSetup.value && err instanceof ApiError && err.status === 409) {
			toast.error(t('errors.auth.setupDone'));
			await navigateTo('/login');
			return;
		}
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<form class="space-y-4" novalidate @submit.prevent="submit">
	<div>
		<SectionHeading icon="lucide:user-plus" :eyebrow="t('common.eyebrow.account')" :title="title"/>
		<p v-if="hint" class="mt-2 text-sm text-muted">{{ hint }}</p>
	</div>
	<FormField v-if="isSetup" v-slot="{ field }" :label="t('pages.auth.setupToken')" :hint="t('pages.auth.setupTokenHint')" :error="errors.setupToken">
		<input v-bind="field" v-model="form.setupToken" type="password" class="input" autocomplete="off" required>
	</FormField>
	<FormField v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName">
		<input v-bind="field" v-model="form.loginName" class="input" autocomplete="username" autocapitalize="none" required>
	</FormField>
	<FormField v-slot="{ field }" :label="t('terms.displayName')" :error="errors.displayName" optional>
		<input v-bind="field" v-model="form.displayName" class="input" autocomplete="nickname">
	</FormField>
	<FormField v-slot="{ field }" :label="t('terms.password')" :error="errors.password">
		<input v-bind="field" v-model="form.password" type="password" class="input" autocomplete="new-password" required>
	</FormField>
	<button ref="submitButton" class="btn btn-primary w-full" :disabled="busy">{{ title }}</button>
</form>
</template>
