<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { settingsSchema } from '~/schemas/settings';
import { latestBirthDate, todayIso } from '~/utils/date';
import { formatInt, unitLabel } from '~/utils/nutrients';

const { t, locale, locales, setLocale } = useI18n();
const uid = useId();
const { logout } = useAuth();
const confirm = useConfirm();

async function confirmLogout() {
	if (await confirm({ title: t('pages.settings.logoutConfirm'), confirmLabel: t('pages.settings.logout'), danger: true })) await logout();
}

const { catalog } = useNutrientCatalog();
const onError = useErrorHandler();
const api = useApi();
const { errors, validate } = useValidation(settingsSchema);

const sexes = ['female', 'male'] as const;
const goals = ['lose', 'maintain', 'gain'] as const;
const activities = ['low', 'moderate', 'high'] as const;
const goalDefaults: Record<Schemas['Goal'], number> = { lose: -300, maintain: 0, gain: 300 };

const profile = reactive<Schemas['Profile']>({
	sex: 'female', birthDate: '', heightCm: 0, activity: 'moderate', goal: 'maintain', energyAdjustKcal: 0, menstruating: false,
});
const overrides = reactive<Record<string, { min?: number; max?: number }>>({});
const effectiveFrom = ref(todayIso());
const busy = ref(false);

watch(catalog, (list) => {
	for (const d of list) overrides[d.code] ??= {};
}, { immediate: true });

const { data: settings } = useAsyncData('settings', async () => {
	try {
		return await unwrap(api.GET('/settings'));
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) return null;
		throw err;
	}
});
const { data: targets, refresh: refreshTargets } = useAsyncData('targets-today', async () => {
	try {
		return await unwrap(api.GET('/targets/{date}', { params: { path: { date: todayIso() } } }));
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) return null;
		throw err;
	}
});
const { data: reference } = useAsyncData('dri', () => unwrap(api.GET('/reference/dri')));

watch(settings, (s) => {
	if (!s) return;
	Object.assign(profile, s.profile);
	for (const o of s.overrides) overrides[o.code] = { min: o.min, max: o.max };
}, { immediate: true });

const links = computed(() => [
	{ to: '/history', label: 'pages.settings.links.weight' },
	{ to: '/settings/data', label: 'pages.settings.links.data' },
	{ to: '/settings/account', label: 'pages.settings.links.account' },
	{ to: '/settings/mcp', label: 'pages.settings.links.mcp' },
	{ to: '/about', label: 'pages.settings.links.about' },
]);

function placeholder(code: string, side: 'min' | 'max'): string {
	const n = targets.value?.nutrients.find((x) => x.code === code);
	const v = side === 'min' ? n?.rda ?? n?.ai ?? n?.dgMin : n?.dgMax ?? n?.ul;
	return v === undefined ? '' : String(Math.round(v * 10) / 10);
}

function applyGoalDefault() {
	profile.energyAdjustKcal = goalDefaults[profile.goal];
}

// 空欄(NaN・空文字)除外
function cleanOverrides(): Schemas['TargetOverride'][] {
	const num = (x: unknown) => (typeof x === 'number' && Number.isFinite(x) ? x : undefined);
	return Object.entries(overrides)
		.map(([code, o]) => ({ code, min: num(o.min), max: num(o.max) }))
		.filter((o) => o.min !== undefined || o.max !== undefined);
}

async function save() {
	const body = validate({
		effectiveFrom: effectiveFrom.value,
		profile: { ...profile, menstruating: profile.sex === 'female' && profile.menstruating },
		overrides: cleanOverrides(),
	});
	if (!body) return;
	busy.value = true;
	try {
		settings.value = await unwrap(api.PUT('/settings', { body }));
		await refreshTargets();
		toast(t('common.messages.saved'));
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<div class="space-y-4">
	<PageHeader :title="t('pages.settings.title')" :eyebrow="t('common.eyebrow.settings')"/>

	<form v-reveal class="panel space-y-4 p-6" novalidate @submit.prevent="save">
		<SectionHeading icon="lucide:user-round" :eyebrow="t('common.eyebrow.profile')" :title="t('pages.settings.profile')"/>
		<div class="grid grid-cols-2 gap-3">
			<FormField v-slot="{ field, labelId }" :label="t('pages.settings.sex')">
				<SelectField v-bind="field" v-model="profile.sex" :labelledby="labelId" :options="sexes.map((s) => ({ value: s, label: t(`terms.sex.${s}`) }))"/>
			</FormField>
			<FormField v-slot="{ field }" :label="t('pages.settings.birthDate')" :error="errors['profile.birthDate']">
				<DateField v-bind="field" v-model="profile.birthDate" yearFirst :max="latestBirthDate()"/>
			</FormField>
			<FormField v-slot="{ field }" :label="t('pages.settings.height')" :error="errors['profile.heightCm']">
				<input v-bind="field" v-model.number="profile.heightCm" type="number" inputmode="decimal" step="0.1" class="input" required>
			</FormField>
			<FormField v-slot="{ field, labelId }" :label="t('pages.settings.goal')">
				<SelectField v-bind="field" v-model="profile.goal" :labelledby="labelId" :options="goals.map((g) => ({ value: g, label: t(`terms.goal.${g}`) }))" @update:modelValue="applyGoalDefault"/>
			</FormField>
		</div>
		<FormField v-slot="{ field, labelId }" :label="t('pages.settings.activity')">
			<SelectField v-bind="field" v-model="profile.activity" :labelledby="labelId" :options="activities.map((a) => ({ value: a, label: t(`terms.activity.${a}`) }))"/>
		</FormField>
		<FormField v-slot="{ field }" :label="t('pages.settings.energyAdjust')" :error="errors['profile.energyAdjustKcal']">
			<input v-bind="field" v-model.number="profile.energyAdjustKcal" type="number" inputmode="numeric" step="50" class="input">
		</FormField>
		<label v-if="profile.sex === 'female'" class="flex items-center gap-2 text-sm">
			<input v-model="profile.menstruating" type="checkbox" class="size-5">
			{{ t('pages.settings.menstruating') }}
		</label>

		<details class="disclosure rounded-2xl border-2 border-[var(--input-border)] p-4">
			<summary class="font-semibold">{{ t('pages.settings.targets') }}<Icon name="lucide:chevron-down" class="disclosure-icon text-muted" size="18" aria-hidden="true"/></summary>
			<p class="my-2 text-xs text-muted">{{ t('pages.settings.targetsHint') }}</p>
			<table class="w-full text-sm">
				<thead>
					<tr class="text-xs text-muted">
						<th class="text-left font-normal"><span class="sr-only">{{ t('pages.settings.nutrientColumn') }}</span></th>
						<th class="font-normal">{{ t('pages.settings.customMin') }}</th>
						<th class="font-normal">{{ t('pages.settings.customMax') }}</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="d in catalog.filter((c) => c.code !== 'energy')" :key="d.code">
						<th scope="row" class="py-1 text-left font-normal">{{ t(`terms.nutrient.${d.code}`) }} <span class="text-xs text-muted">{{ unitLabel[d.unit] }}</span></th>
						<td class="px-1 py-1"><input v-model.number="overrides[d.code]!.min" :aria-label="`${t(`terms.nutrient.${d.code}`)} ${t('pages.settings.customMin')}`" type="number" inputmode="decimal" step="any" class="input py-1 text-right" :placeholder="placeholder(d.code, 'min')"></td>
						<td class="px-1 py-1"><input v-model.number="overrides[d.code]!.max" :aria-label="`${t(`terms.nutrient.${d.code}`)} ${t('pages.settings.customMax')}`" type="number" inputmode="decimal" step="any" class="input py-1 text-right" :placeholder="placeholder(d.code, 'max')"></td>
					</tr>
				</tbody>
			</table>
			<p v-if="errors.overrides" class="mt-1 text-xs text-danger">{{ t(errors.overrides) }}</p>
		</details>

		<FormField v-slot="{ field }" :label="t('pages.settings.effectiveFrom')" :hint="t('pages.settings.effectiveFromHint')">
			<DateField v-bind="field" v-model="effectiveFrom"/>
		</FormField>
		<button class="btn btn-primary w-full" :disabled="busy">{{ t('common.save') }}</button>
	</form>

	<section v-if="targets" v-reveal class="panel p-6">
		<SectionHeading icon="lucide:target" :eyebrow="t('common.eyebrow.target')" :title="t('pages.settings.computed')"/>
		<p class="mt-4 flex items-baseline gap-2"><span class="numeric text-5xl">{{ formatInt(targets.energyKcal) }}</span><span class="eyebrow">kcal</span></p>
		<p class="mt-2 text-xs text-muted">{{ t('pages.settings.reference', { title: reference?.title ?? targets.edition }) }} ({{ targets.ageBand }})</p>
	</section>
	<p v-else-if="settings" v-reveal class="panel p-6 text-sm text-muted">{{ t('pages.settings.needWeight') }}</p>

	<nav v-reveal class="panel px-2 py-1" :aria-label="t('pages.settings.title')">
		<ul class="divide-y divide-[var(--hairline)]">
			<li v-for="link in links" :key="link.to">
				<NuxtLink :to="link.to" class="row flex items-center justify-between px-4 py-3">
					{{ t(link.label) }}<Icon name="lucide:arrow-up-right" class="text-muted" size="16" aria-hidden="true"/>
				</NuxtLink>
			</li>
		</ul>
	</nav>

	<section v-reveal class="panel divide-y divide-[var(--hairline)] px-2 py-1" :aria-label="t('pages.settings.account')">
		<div class="flex items-center justify-between gap-3 px-4 py-3">
			<span :id="`${uid}-language`">{{ t('pages.settings.language') }}</span>
			<SelectField
				compact
				:modelValue="locale"
				:labelledby="`${uid}-language`"
				:options="locales.map((l) => ({ value: l.code, label: l.name ?? l.code }))"
				@update:modelValue="(v) => setLocale(v as 'ja' | 'en')"
			/>
		</div>
		<button type="button" class="w-full px-4 py-3 text-left text-danger" @click="confirmLogout">{{ t('pages.settings.logout') }}</button>
	</section>
</div>
</template>
