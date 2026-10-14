<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';
import { tokenSchema } from '~/schemas/auth';
import { formatDate } from '~/utils/date';

const { t, locale } = useI18n();
const { me } = useAuth();
const api = useApi();
const onError = useErrorHandler();
const confirm = useConfirm();
const { copy } = useClipboard({ legacy: true });
const { errors, validate } = useValidation(tokenSchema);

const { data, refresh } = useAsyncData('tokens', () => unwrap(api.GET('/me/tokens')));
const { data: connections, refresh: refreshConnections } = useAsyncData('connections', () => unwrap(api.GET('/me/connections')));
const endpoint = computed(() => `${useRequestURL().origin}/mcp`);
const form = reactive<Schemas['TokenInput']>({ name: '', scope: 'read' });
const issued = ref<Schemas['IssuedToken']>();
const busy = ref(false);

async function copyText(text: string) {
	await copy(text);
	toast(t('pages.mcp.copied'));
}

async function issue() {
	const body = validate(form);
	if (!body) return;
	busy.value = true;
	try {
		issued.value = await unwrap(api.POST('/me/tokens', { body }));
		form.name = '';
		toast(t('pages.mcp.issued'));
		await refresh();
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}

async function revoke(token: Schemas['ApiToken']) {
	const ok = await confirm({
		title: t('pages.mcp.revokeConfirm', { name: token.name }),
		message: t('pages.mcp.revokeMessage'),
		confirmLabel: t('pages.mcp.revoke'),
		danger: true,
	});
	if (!ok) return;
	try {
		await unwrap(api.DELETE('/me/tokens/{id}', { params: { path: { id: token.id } } }));
		if (issued.value?.token.id === token.id) issued.value = undefined;
		toast(t('pages.mcp.revoked'));
		await refresh();
	} catch (err) {
		onError(err);
	}
}

async function disconnect(c: Schemas['Connection']) {
	const ok = await confirm({
		title: t('pages.mcp.disconnectConfirm', { name: c.clientName }),
		message: t('pages.mcp.disconnectMessage'),
		confirmLabel: t('pages.mcp.disconnect'),
		danger: true,
	});
	if (!ok) return;
	try {
		await unwrap(api.DELETE('/me/connections/{id}', { params: { path: { id: c.id } } }));
		toast(t('pages.mcp.disconnected'));
		await refreshConnections();
	} catch (err) {
		onError(err);
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader back="/settings" :title="t('pages.mcp.title')" :eyebrow="t('common.eyebrow.mcp')"/>

	<section v-if="data && !data.mcpEnabled" v-reveal class="panel space-y-2 p-6 text-sm">
		<p>{{ t('pages.mcp.disabled') }}</p>
		<template v-if="me?.role === 'admin'">
			<p class="text-muted">{{ t('pages.mcp.disabledAdmin') }}</p>
			<NuxtLink to="/admin/settings" class="underline">{{ t('pages.mcp.openSettings') }}</NuxtLink>
		</template>
	</section>

	<section v-if="data?.mcpEnabled" v-reveal class="panel space-y-3 p-6">
		<SectionHeading icon="lucide:plug" :eyebrow="t('common.eyebrow.endpoint')" :title="t('pages.mcp.endpoint')"/>
		<div class="flex items-center gap-2">
			<code class="min-w-0 flex-1 truncate rounded-xl border-2 border-[var(--input-border)] px-3 py-2 text-sm">{{ endpoint }}</code>
			<button type="button" class="btn btn-ghost size-11 shrink-0 !p-0" :aria-label="t('pages.mcp.copy')" @click="copyText(endpoint)">
				<Icon name="lucide:copy" size="18" aria-hidden="true"/>
			</button>
		</div>
		<p v-if="data.oauthEnabled" class="text-xs text-muted">{{ t('pages.mcp.endpointHintOAuth') }}</p>
		<template v-else>
			<p class="text-xs text-muted">{{ t('pages.mcp.endpointHint') }}</p>
			<p v-if="me?.role === 'admin'" class="text-xs text-muted">{{ t('pages.mcp.oauthUnavailable') }}</p>
		</template>
	</section>

	<section v-if="data?.oauthEnabled || connections?.length" v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:link" :eyebrow="t('common.eyebrow.connections')" :title="t('pages.mcp.connections')"/>
		<p v-if="!connections?.length" class="text-sm text-muted">{{ t('pages.mcp.connectionsEmpty') }}</p>
		<ul v-else class="divide-y divide-[var(--hairline)]">
			<li v-for="c in connections" :key="c.id" class="flex items-center gap-2 py-2 text-sm">
				<span class="min-w-0 flex-1">
					<span class="block truncate">{{ c.clientName }} <span class="text-xs text-muted">{{ t(`pages.mcp.scopes.${c.scope}.label`) }}</span></span>
					<span class="block truncate text-xs text-muted">
						<template v-if="c.clientHost">{{ c.clientHost }} / </template>{{ t('pages.mcp.connectedAt', { date: formatDate(c.createdAt, t('common.dateFormat.dateTime'), locale) }) }}
					</span>
				</span>
				<button type="button" class="btn btn-ghost shrink-0 text-danger" @click="disconnect(c)">{{ t('pages.mcp.disconnect') }}</button>
			</li>
		</ul>
	</section>

	<section v-if="issued" v-reveal class="panel space-y-3 p-6" role="status">
		<p class="text-sm font-semibold">{{ t('pages.mcp.issued') }}: {{ issued.token.name }}</p>
		<div class="flex items-center gap-2">
			<code class="min-w-0 flex-1 break-all rounded-xl border-2 border-[var(--input-border)] px-3 py-2 text-sm">{{ issued.secret }}</code>
			<button type="button" class="btn btn-ghost size-11 shrink-0 !p-0" :aria-label="t('pages.mcp.copy')" @click="copyText(issued.secret)">
				<Icon name="lucide:copy" size="18" aria-hidden="true"/>
			</button>
		</div>
		<p class="text-xs text-muted">{{ t('pages.mcp.secretNote') }}</p>
	</section>

	<section v-if="data" v-reveal class="panel p-6">
		<SectionHeading class="mb-3" icon="lucide:key-round" :eyebrow="t('common.eyebrow.tokens')" :title="t('pages.mcp.tokens')"/>
		<p v-if="data.oauthEnabled" class="mb-3 text-xs text-muted">{{ t('pages.mcp.tokensHint') }}</p>
		<p v-if="data.tokens.length === 0" class="text-sm text-muted">{{ t('pages.mcp.empty') }}</p>
		<ul v-else class="divide-y divide-[var(--hairline)]">
			<li v-for="tok in data.tokens" :key="tok.id" class="flex items-center gap-2 py-2 text-sm">
				<span class="min-w-0 flex-1">
					<span class="block truncate">{{ tok.name }} <span class="text-xs text-muted">{{ t(`pages.mcp.scopes.${tok.scope}.label`) }}</span></span>
					<span class="block text-xs text-muted">
						{{ t('pages.mcp.createdAt', { date: formatDate(tok.createdAt, t('common.dateFormat.dateTime'), locale) }) }}
					</span>
				</span>
				<button type="button" class="btn btn-ghost shrink-0 text-danger" @click="revoke(tok)">{{ t('pages.mcp.revoke') }}</button>
			</li>
		</ul>
	</section>

	<form v-if="data?.mcpEnabled" v-reveal class="panel space-y-3 p-6" novalidate @submit.prevent="issue">
		<SectionHeading class="mb-1" icon="lucide:plus" :eyebrow="t('common.eyebrow.create')" :title="t('pages.mcp.issue')"/>
		<FormField v-slot="{ field }" :label="t('pages.mcp.name')" :hint="t('pages.mcp.nameHint')" :error="errors.name">
			<input v-bind="field" v-model="form.name" class="input" maxlength="64"/>
		</FormField>
		<FormField v-slot="{ labelId }" :label="t('pages.mcp.scope')">
			<TokenScopeSelector v-model="form.scope" :labelledby="labelId"/>
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('pages.mcp.issue') }}</button>
	</form>
</div>
</template>
