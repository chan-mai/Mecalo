<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';

definePageMeta({ middleware: 'admin' });

type Setting = Schemas['InstanceSetting'];

const { t, te } = useI18n();
const confirm = useConfirm();
const onError = useErrorHandler();
const api = useApi();

const { data: settings } = useAsyncData('admin-settings', () => unwrap(api.GET('/admin/settings')));

// 入力中の値, 秘密の値は入力時のみ送信
const form = reactive<Record<string, string>>({});
const busy = ref(false);

// 保存済みの値へ戻す
function discard() {
	for (const s of settings.value ?? []) form[s.key] = s.kind === 'secret' ? '' : s.value;
}

watch(settings, discard, { immediate: true });

const groups = computed(() => {
	const out: { name: string; items: Setting[] }[] = [];
	for (const s of settings.value ?? []) {
		const g = out.find((x) => x.name === s.group);
		if (g) g.items.push(s);
		else out.push({ name: s.group, items: [s] });
	}
	return out;
});

const changes = computed(() => {
	const out: Record<string, string> = {};
	for (const s of settings.value ?? []) {
		const v = form[s.key] ?? '';
		if (s.kind === 'secret' ? v !== '' : v !== s.value) out[s.key] = v;
	}
	return out;
});

const dirty = computed(() => Object.keys(changes.value).length > 0);

// 未保存の変更がある場合は遷移前に確認
onBeforeRouteLeave(async () => {
	if (!dirty.value) return true;
	return confirm({ title: t('pages.adminSettings.leaveTitle'), message: t('pages.adminSettings.leaveMessage'), confirmLabel: t('pages.adminSettings.leave'), danger: true });
});

useEventListener(window, 'beforeunload', (event: BeforeUnloadEvent) => {
	if (dirty.value) event.preventDefault();
});

function label(s: Setting): string {
	return t(`pages.adminSettings.items.${s.key}`);
}

function optionLabel(s: Setting, o: string): string {
	const key = `pages.adminSettings.options.${s.key}.${o}`;
	return te(key) ? t(key) : o;
}

// WebUIから設定した値の戻り先
function fallbackHint(s: Setting): string | undefined {
	if (s.origin !== 'ui' || s.kind === 'secret') return undefined;
	if (s.hasEnv) return t('pages.adminSettings.envValue', { value: s.envValue || t('pages.adminSettings.empty') });
	return t('pages.adminSettings.defaultValue', { value: s.defaultValue || t('pages.adminSettings.empty') });
}

function hint(s: Setting): string | undefined {
	const parts = [fallbackHint(s)];
	if (s.kind === 'duration' || s.kind === 'list') parts.push(t(`pages.adminSettings.format.${s.kind}`));
	if (s.kind === 'secret') parts.push(t(s.configured ? 'pages.adminSettings.secretSet' : 'pages.adminSettings.secretUnset'));
	return parts.filter(Boolean).join(' / ') || undefined;
}

function selected(s: Setting): string[] {
	return (form[s.key] ?? '').split(',').filter(Boolean);
}

// 「all」は他の選択肢と排他
function allSelected(s: Setting): boolean {
	return s.options.includes('all') && selected(s).includes('all');
}

function checked(s: Setting, o: string): boolean {
	return allSelected(s) || selected(s).includes(o);
}

function toggle(s: Setting, o: string, on: boolean) {
	if (o === 'all') {
		form[s.key] = on ? 'all' : '';
		return;
	}
	const cur = selected(s).filter((x) => x !== o);
	if (on) cur.push(o);
	form[s.key] = s.options.filter((x) => cur.includes(x)).join(',');
}

async function save() {
	if (!dirty.value) return;
	busy.value = true;
	try {
		settings.value = await unwrap(api.PUT('/admin/settings', { body: { values: changes.value } }));
		toast(t('common.messages.saved'));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}

async function reset(s: Setting) {
	busy.value = true;
	try {
		settings.value = await unwrap(api.DELETE('/admin/settings/{key}', { params: { path: { key: s.key } } }));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}

async function resetAll() {
	if (!await confirm({ title: t('pages.adminSettings.confirmResetAll'), confirmLabel: t('pages.adminSettings.resetAll'), danger: true })) return;
	busy.value = true;
	try {
		settings.value = await unwrap(api.DELETE('/admin/settings'));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4" :class="{ 'pb-20': dirty }">
	<PageHeader back="/admin" :title="t('pages.adminSettings.title')" :eyebrow="t('common.eyebrow.settings')"/>
	<p class="text-sm text-muted">{{ t('pages.adminSettings.note') }}</p>
	<form class="space-y-4" novalidate @submit.prevent="save">
		<section v-for="g in groups" :key="g.name" class="panel relative space-y-4 p-6 focus-within:z-10">
			<h2 class="font-medium">{{ t(`pages.adminSettings.groups.${g.name}`) }}</h2>
			<div v-for="s in g.items" :key="s.key">
				<FormField v-slot="{ field, labelId }" :label="label(s)" :hint="hint(s)">
					<div class="mb-1 flex flex-wrap items-center gap-2 text-xs text-muted">
						<code>{{ s.key }}</code>
						<span class="rounded-full px-2 py-0.5" :class="s.origin === 'ui' ? 'bg-info-soft text-info' : 'bg-surface-soft'">{{
							t(`pages.adminSettings.origin.${s.origin}`)
						}}</span>
						<button v-if="s.origin === 'ui'" type="button" class="underline" :disabled="busy" @click="reset(s)">{{ t('pages.adminSettings.reset') }}</button>
					</div>
					<label v-if="s.kind === 'bool'" class="flex items-center gap-2 text-sm">
						<input
							v-bind="field"
							type="checkbox"
							class="size-5"
							:checked="form[s.key] === 'true'"
							@change="form[s.key] = String(($event.target as HTMLInputElement).checked)"
						/>
						{{ t('pages.adminSettings.enabled') }}
					</label>
					<SelectField
						v-else-if="s.kind === 'choice'"
						v-bind="field"
						v-model="form[s.key]!"
						:labelledby="labelId"
						:options="s.options.map((o) => ({ value: o, label: optionLabel(s, o) }))"
					/>
					<div v-else-if="s.kind === 'choices'" role="group" :aria-labelledby="labelId" class="grid grid-cols-2 gap-2 sm:grid-cols-3">
						<label v-for="o in s.options" :key="o" class="flex items-center gap-2 text-sm" :class="{ 'opacity-50': o !== 'all' && allSelected(s) }">
							<input type="checkbox" class="size-5" :checked="checked(s, o)" :disabled="o !== 'all' && allSelected(s)" @change="toggle(s, o, ($event.target as HTMLInputElement).checked)"/>
							{{ optionLabel(s, o) }}
						</label>
					</div>
					<input
						v-else-if="s.kind === 'secret'"
						v-bind="field"
						v-model="form[s.key]"
						type="password"
						class="input"
						autocomplete="off"
						:placeholder="s.configured ? '********' : ''"
					/>
					<input v-else v-bind="field" v-model="form[s.key]" class="input" autocapitalize="none" :placeholder="s.defaultValue"/>
				</FormField>
			</div>
		</section>
		<Transition
			enterActiveClass="transition duration-300 ease-[var(--ease-out-expo)]"
			enterFromClass="opacity-0 translate-y-4"
			leaveActiveClass="transition duration-200 ease-in"
			leaveToClass="opacity-0 translate-y-4"
		>
			<div v-if="dirty" class="save-bar panel fixed z-30 flex items-center gap-2 py-2 ps-5 pe-2" role="region" :aria-label="t('pages.adminSettings.unsaved', { n: Object.keys(changes).length })">
				<p class="min-w-0 flex-1 text-sm">{{ t('pages.adminSettings.unsaved', { n: Object.keys(changes).length }) }}</p>
				<button type="button" class="btn btn-ghost shrink-0" :disabled="busy" @click="discard">{{ t('pages.adminSettings.discard') }}</button>
				<button class="btn btn-primary shrink-0" :disabled="busy">{{ t('common.save') }}</button>
			</div>
		</Transition>
	</form>
	<button type="button" class="btn btn-ghost w-full text-danger" :disabled="busy" @click="resetAll">
		<Icon name="lucide:rotate-ccw" size="16" aria-hidden="true"/>
		{{ t('pages.adminSettings.resetAll') }}
	</button>
</div>
</template>

<style scoped>
/* 下部ナビの上に表示 */
.save-bar {
	inset-inline: 1rem;
	inset-block-end: calc(var(--nav-height) + 1.5rem + env(safe-area-inset-bottom));
	margin-inline: auto;
	max-inline-size: 28rem;
	border-radius: 9999px;
}
</style>
