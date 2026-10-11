<script setup lang="ts">
import { unwrap, useApi } from '~/composables/api/client';
import { signUpSchema } from '~/schemas/auth';

const props = defineProps<{ title: string; hint?: string; path: '/auth/setup' | '/auth/signup' }>();
const { t } = useI18n();
const { refresh } = useAuth();
const onError = useErrorHandler();
const { errors, validate } = useValidation(signUpSchema);

const form = reactive({ loginName: '', displayName: '', password: '' });
const busy = ref(false);

async function submit() {
	const body = validate(form);
	if (!body) return;
	busy.value = true;
	try {
		await unwrap(useApi().POST(props.path, { body }));
		await refresh();
		await navigateTo('/settings');
	} catch (err) {
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
	<FormField v-slot="{ field }" :label="t('terms.loginName')" :error="errors.loginName">
		<input v-bind="field" v-model="form.loginName" class="input" autocomplete="username" autocapitalize="none" required>
	</FormField>
	<FormField v-slot="{ field }" :label="t('terms.displayName')" :error="errors.displayName" optional>
		<input v-bind="field" v-model="form.displayName" class="input" autocomplete="nickname">
	</FormField>
	<FormField v-slot="{ field }" :label="t('terms.password')" :error="errors.password">
		<input v-bind="field" v-model="form.password" type="password" class="input" autocomplete="new-password" required>
	</FormField>
	<button class="btn btn-primary w-full" :disabled="busy">{{ title }}</button>
</form>
</template>
