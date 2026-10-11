<script setup lang="ts">
import { toast } from 'vue-sonner';
import { unwrap, useApi } from '~/composables/api/client';
import { passwordChangeSchema } from '~/schemas/auth';

const { t } = useI18n();
const onError = useErrorHandler();
const { errors, validate } = useValidation(passwordChangeSchema);
const form = reactive({ current: '', next: '' });
const busy = ref(false);

async function submit() {
	const body = validate(form);
	if (!body) return;
	busy.value = true;
	try {
		await unwrap(useApi().PUT('/me/password', { body }));
		toast(t('common.messages.saved'));
		await navigateTo('/settings');
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.password.title')" :eyebrow="t('common.eyebrow.password')"/>
	<form v-reveal class="panel space-y-4 p-6" novalidate @submit.prevent="submit">
		<FormField v-slot="{ field }" :label="t('pages.password.current')" :error="errors.current">
			<input v-bind="field" v-model="form.current" type="password" class="input" autocomplete="current-password">
		</FormField>
		<FormField v-slot="{ field }" :label="t('pages.password.new')" :error="errors.next">
			<input v-bind="field" v-model="form.next" type="password" class="input" autocomplete="new-password">
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
	</form>
</div>
</template>
