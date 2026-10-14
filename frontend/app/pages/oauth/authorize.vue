<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';

definePageMeta({ layout: 'auth' });

const { t } = useI18n();
const route = useRoute();
const { me } = useAuth();
const api = useApi();
const onError = useErrorHandler();

const keys = ['response_type', 'client_id', 'redirect_uri', 'code_challenge', 'code_challenge_method', 'scope', 'state', 'resource'] as const;
const request = computed(() => {
	const out: Schemas['OAuthAuthorizationRequest'] = {};
	for (const k of keys) {
		const v = route.query[k];
		if (typeof v === 'string') out[k] = v;
	}
	return out;
});

const consent = ref<Schemas['OAuthConsent']>();
const scope = ref<Schemas['TokenScope']>('write');
const invalid = ref('');
const leaving = ref(false);
const busy = ref(false);

function leave(to: string) {
	leaving.value = true;
	window.location.replace(to);
}

onMounted(async () => {
	try {
		const res = await unwrap(api.POST('/oauth/authorization/check', { body: request.value }));
		if (res.redirectTo) return leave(res.redirectTo);
		consent.value = res.consent;
		if (res.consent) scope.value = res.consent.scope;
	} catch (err) {
		if (err instanceof ApiError && (err.status === 400 || err.status === 404)) invalid.value = err.message;
		else onError(err);
	}
});

async function decide(approve: boolean) {
	busy.value = true;
	try {
		const res = await unwrap(api.POST('/oauth/authorization/decide', { body: { request: request.value, approve, scope: scope.value } }));
		leave(res.redirectTo);
	} catch (err) {
		onError(err);
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<h2 class="text-lg font-bold">{{ t('pages.oauth.title') }}</h2>

	<p v-if="leaving" role="status" class="text-sm">{{ t('pages.oauth.redirecting') }}</p>

	<div v-else-if="invalid" role="alert" class="space-y-2 text-sm">
		<p class="font-semibold text-danger">{{ t('pages.oauth.invalid') }}</p>
		<p class="text-muted">{{ t('pages.oauth.invalidHint') }}</p>
		<p class="text-xs break-all text-muted">{{ invalid }}</p>
	</div>

	<template v-else-if="consent">
		<p>{{ t('pages.oauth.heading', { name: consent.clientName || consent.clientId }) }}</p>
		<dl class="space-y-1 text-sm">
			<div v-if="consent.clientHost">
				<dt class="sr-only">{{ t('pages.oauth.clientHostLabel') }}</dt>
				<dd>{{ t('pages.oauth.clientHost', { host: consent.clientHost }) }}</dd>
			</div>
			<div>
				<dt class="sr-only">{{ t('pages.oauth.redirectLabel') }}</dt>
				<dd>{{ t('pages.oauth.redirect', { host: consent.redirectHost }) }}</dd>
			</div>
		</dl>
		<p v-if="consent.loopback" class="flex gap-2 rounded-lg bg-danger-soft p-2 text-sm text-danger">
			<Icon name="lucide:triangle-alert" size="18" class="mt-0.5 shrink-0" aria-hidden="true"/>
			{{ t('pages.oauth.loopback') }}
		</p>
		<FormField v-slot="{ labelId }" :label="t('pages.mcp.scope')">
			<TokenScopeSelector v-model="scope" :labelledby="labelId" :max="consent.scope"/>
		</FormField>
		<p v-if="me" class="text-xs text-muted">{{ t('pages.oauth.account', { name: me.displayName }) }}</p>
		<div class="grid grid-cols-2 gap-2">
			<button type="button" class="btn btn-ghost" :disabled="busy" @click="decide(false)">{{ t('pages.oauth.deny') }}</button>
			<button type="button" class="btn btn-primary" :disabled="busy" @click="decide(true)">{{ t('pages.oauth.approve') }}</button>
		</div>
	</template>

	<p v-else role="status" class="text-sm text-muted">{{ t('common.loading') }}</p>
</div>
</template>
