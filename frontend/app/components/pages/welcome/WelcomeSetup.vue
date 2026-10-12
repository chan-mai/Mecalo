<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { ageOf, formatDate, latestBirthDate, todayIso } from '~/utils/date';
import { formatInt } from '~/utils/nutrients';

const emit = defineEmits<{ finished: [from: Element] }>();
const { t, locale } = useI18n();
const { me, refresh } = useAuth();
const onError = useErrorHandler();
const api = useApi();
const motion = usePreferredReducedMotion();

type Step = 'sex' | 'menstruating' | 'birthDate' | 'age' | 'height' | 'weight' | 'activity' | 'goal';

interface Answers {
	sex?: Schemas['Sex'];
	menstruating?: boolean;
	birthDate?: string;
	// 生年月日から求めた年齢の確認済み
	age?: boolean;
	heightCm?: number;
	// nullはあとで記録
	weight?: { kg: number; bodyFatPct?: number } | null;
	activity?: Schemas['ActivityLevel'];
	goal?: Schemas['Goal'];
}

const answers = reactive<Answers>({});
const steps = computed<Step[]>(() => [
	'sex',
	...(answers.sex === 'female' ? ['menstruating' as const] : []),
	'birthDate',
	'age',
	'height',
	'weight',
	'activity',
	'goal',
]);
const index = ref(0);
const step = computed(() => steps.value[index.value]);
// 質問の表示前の入力中表示
const typing = ref(false);
const phase = ref<'asking' | 'saving' | 'summary'>('asking');
const targets = ref<Schemas['Targets'] | null>(null);
const log = useTemplateRef<HTMLElement>('log');

const goalAdjust: Record<Schemas['Goal'], number> = { lose: -300, maintain: 0, gain: 300 };
const options = {
	sex: ['female', 'male'],
	menstruating: ['yes', 'no'],
	age: ['correct', 'fix'],
	activity: ['low', 'moderate', 'high'],
	goal: ['lose', 'maintain', 'gain'],
} as const;

// 入力欄の値
const draft = reactive({ birthDate: '', height: '', kg: '', bodyFat: '' });

function answered(s: Step): boolean {
	switch (s) {
		case 'height':
			return answers.heightCm !== undefined;
		case 'weight':
			return answers.weight !== undefined;
		default:
			return answers[s] !== undefined;
	}
}

function answerText(s: Step): string {
	switch (s) {
		case 'sex':
			return t(`terms.sex.${answers.sex}`);
		case 'menstruating':
			return t(`pages.welcome.setup.options.${answers.menstruating ? 'yes' : 'no'}`);
		case 'age':
			return t('pages.welcome.setup.options.correct');
		case 'birthDate':
			return formatDate(answers.birthDate ?? '', t('common.dateFormat.full'), locale.value);
		case 'height':
			return `${answers.heightCm} cm`;
		case 'weight': {
			const w = answers.weight;
			if (!w) return t('pages.welcome.setup.later');
			return w.bodyFatPct === undefined ? `${w.kg} kg` : `${w.kg} kg / ${w.bodyFatPct} %`;
		}
		case 'activity':
			return t(`terms.activity.${answers.activity}`);
		case 'goal':
			return t(`terms.goal.${answers.goal}`);
	}
}

function question(s: Step): string {
	return t(`pages.welcome.setup.questions.${s}`, { age: answers.birthDate ? ageOf(answers.birthDate) : '' });
}

// 現在の質問までの会話
const asked = computed(() => steps.value.slice(0, index.value).filter(answered));

// 最後の吹き出しを下の回答欄に隠れない位置まで表示
async function reveal() {
	await nextTick();
	log.value?.lastElementChild?.scrollIntoView({ behavior: motion.value === 'reduce' ? 'auto' : 'smooth', block: 'end' });
}

async function ask() {
	typing.value = motion.value !== 'reduce';
	if (typing.value) await new Promise((r) => setTimeout(r, 650));
	typing.value = false;
	await reveal();
}

// 入力中の表示, 回答, 保存中, 目標の表示で吹き出しが増えた時
watch([typing, () => asked.value.length, phase], () => void reveal());

// 未回答の次の質問, なければ保存
async function advance() {
	const next = steps.value.findIndex((s, i) => i > index.value && !answered(s));
	const rest = steps.value.findIndex((s) => !answered(s));
	if (next >= 0) index.value = next;
	else if (rest >= 0) index.value = rest;
	else {
		index.value = steps.value.length;
		await finish();
		return;
	}
	await ask();
}

function choose(s: Step, value: string) {
	if (s === 'sex') {
		const prev = answers.sex;
		answers.sex = value as Schemas['Sex'];
		// 男性から女性へ変えた場合は改めて質問
		if (value === 'male') answers.menstruating = false;
		else if (prev === 'male') answers.menstruating = undefined;
	}
	if (s === 'menstruating') answers.menstruating = value === 'yes';
	if (s === 'age') {
		if (value === 'fix') {
			edit('birthDate');
			return;
		}
		answers.age = true;
	}
	if (s === 'activity') answers.activity = value as Schemas['ActivityLevel'];
	if (s === 'goal') answers.goal = value as Schemas['Goal'];
	void advance();
}

const birthValid = computed(() => /^\d{4}-\d{2}-\d{2}$/.test(draft.birthDate) && draft.birthDate <= latestBirthDate());
const birthTooYoung = computed(() => /^\d{4}-\d{2}-\d{2}$/.test(draft.birthDate) && draft.birthDate > latestBirthDate());
const heightValid = computed(() => Number(draft.height) > 50 && Number(draft.height) < 260);
const weightValid = computed(
	() => Number(draft.kg) > 10 && Number(draft.kg) < 400 && (draft.bodyFat === '' || (Number(draft.bodyFat) >= 0 && Number(draft.bodyFat) <= 100)),
);

function submitBirth() {
	if (!birthValid.value) return;
	// 変更時は年齢を改めて確認
	if (answers.birthDate !== draft.birthDate) answers.age = undefined;
	answers.birthDate = draft.birthDate;
	void advance();
}

function submitHeight() {
	if (!heightValid.value) return;
	answers.heightCm = Number(draft.height);
	void advance();
}

function submitWeight(later = false) {
	if (later) answers.weight = null;
	else if (weightValid.value) answers.weight = { kg: Number(draft.kg), bodyFatPct: draft.bodyFat === '' ? undefined : Number(draft.bodyFat) };
	else return;
	void advance();
}

// 答えた質問へ戻る
function edit(s: Step) {
	if (phase.value === 'saving') return;
	phase.value = 'asking';
	index.value = steps.value.indexOf(s);
	void ask();
}

async function finish() {
	const { sex, birthDate, heightCm, activity, goal } = answers;
	if (!sex || !birthDate || heightCm === undefined || !activity || !goal) return;
	phase.value = 'saving';
	try {
		await unwrap(
			api.PUT('/settings', {
				body: {
					profile: {
						sex,
						birthDate,
						heightCm,
						activity,
						goal,
						energyAdjustKcal: goalAdjust[goal],
						menstruating: sex === 'female' && answers.menstruating === true,
					},
				},
			}),
		);
		if (answers.weight) {
			await unwrap(api.PUT('/weights/{date}', { params: { path: { date: todayIso() } }, body: answers.weight }));
		}
		await unwrap(api.POST('/me/onboarding'));
		await refresh();
		try {
			targets.value = await unwrap(api.GET('/targets/{date}', { params: { path: { date: todayIso() } } }));
		} catch (err) {
			if (!(err instanceof ApiError && err.status === 404)) throw err;
		}
		phase.value = 'summary';
		await ask();
	} catch (err) {
		phase.value = 'asking';
		index.value = steps.value.length - 1;
		onError(err);
	}
}

// 目標量の範囲
function range(code: string): string | undefined {
	const n = targets.value?.nutrients.find((x) => x.code === code);
	const lo = n?.dgMin ?? n?.rda ?? n?.ai;
	const hi = n?.dgMax;
	if (lo === undefined) return undefined;
	return hi === undefined ? `${formatInt(lo)} g` : `${formatInt(lo)}–${formatInt(hi)} g`;
}

// 答え直す質問の入力欄へ前回の値
watch(step, (s) => {
	if (s === 'birthDate') draft.birthDate = answers.birthDate ?? draft.birthDate;
	if (s === 'height' && answers.heightCm !== undefined) draft.height = String(answers.heightCm);
	if (s === 'weight' && answers.weight) {
		draft.kg = String(answers.weight.kg);
		draft.bodyFat = answers.weight.bodyFatPct === undefined ? '' : String(answers.weight.bodyFatPct);
	}
});

onMounted(() => void ask());
</script>

<template>
<div class="mx-auto flex min-h-dvh max-w-xl flex-col px-5 pt-[calc(1.5rem+env(safe-area-inset-top))]">
	<header class="mb-6 flex items-center gap-3">
		<img src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="h-5 w-auto">
		<p class="eyebrow">{{ t('pages.welcome.setup.eyebrow') }}</p>
		<p class="eyebrow ml-auto tabular-nums" aria-hidden="true">{{ Math.min(index + 1, steps.length) }} / {{ steps.length }}</p>
	</header>

	<ol ref="log" class="log flex-1 space-y-3 pb-6" aria-live="polite">
		<WelcomeBotMessage>{{ t('pages.welcome.setup.greeting', { name: me?.displayName ?? '' }) }}</WelcomeBotMessage>
		<template v-for="s in asked" :key="s">
			<WelcomeBotMessage>{{ question(s) }}</WelcomeBotMessage>
			<li class="flex justify-end">
				<button type="button" class="bubble bubble-user press" :aria-label="`${answerText(s)} (${t('pages.welcome.setup.edit')})`" @click="edit(s)">
					{{ answerText(s) }}<Icon name="lucide:pencil" size="12" class="ml-2 opacity-60" aria-hidden="true"/>
				</button>
			</li>
		</template>
		<WelcomeBotMessage v-if="typing" typing :label="t('pages.welcome.setup.typing')"/>
		<WelcomeBotMessage v-else-if="phase === 'asking' && step">{{ question(step) }}</WelcomeBotMessage>
		<WelcomeBotMessage v-else-if="phase === 'saving'" typing :label="t('pages.welcome.setup.saving')"/>
		<template v-else-if="phase === 'summary'">
			<WelcomeBotMessage>{{ t('pages.welcome.setup.done') }}</WelcomeBotMessage>
			<li class="panel p-6">
				<p class="eyebrow">{{ t('pages.welcome.setup.summary') }}</p>
				<template v-if="targets">
					<p class="mt-3">
						<span class="display text-4xl tabular-nums">{{ formatInt(targets.energyKcal) }}</span><span class="ml-2 text-sm text-muted">kcal / {{ t('pages.welcome.setup.perDay') }}</span>
					</p>
					<dl class="mt-4 grid grid-cols-3 gap-2 text-center text-sm">
						<div v-for="code in ['protein', 'fat', 'carbohydrate']" :key="code" class="rounded-2xl bg-surface-soft px-2 py-3">
							<dt class="text-xs text-muted">{{ t(`pages.welcome.setup.macros.${code}`) }}</dt>
							<dd class="mt-1 tabular-nums">{{ range(code) ?? '-' }}</dd>
						</div>
					</dl>
				</template>
				<p v-else class="mt-3 text-sm text-muted">{{ t('pages.settings.needWeight') }}</p>
				<p class="mt-4 text-xs text-muted">{{ t('pages.welcome.setup.note') }}</p>
			</li>
		</template>
	</ol>

	<div class="answer panel sticky bottom-[calc(1rem+env(safe-area-inset-bottom))] mb-4 p-4">
		<template v-if="phase === 'summary'">
			<button type="button" class="btn btn-primary press w-full" @click="emit('finished', $event.currentTarget as Element)">{{ t('pages.welcome.setup.finish') }}</button>
		</template>
		<template v-else-if="phase === 'asking' && step && !typing">
			<div
				v-if="step === 'sex' || step === 'menstruating' || step === 'age' || step === 'activity' || step === 'goal'"
				class="grid gap-2"
				:class="step === 'activity' ? 'grid-cols-1' : step === 'goal' ? 'grid-cols-3' : 'grid-cols-2'"
			>
				<button v-for="o in options[step]" :key="o" type="button" class="choice-btn press" @click="choose(step, o)">
					{{ step === 'sex' || step === 'activity' || step === 'goal' ? t(`terms.${step}.${o}`) : t(`pages.welcome.setup.options.${o}`) }}
				</button>
			</div>
			<form v-else-if="step === 'birthDate'" class="flex gap-2" novalidate @submit.prevent="submitBirth">
				<div class="flex-1">
					<DateField v-model="draft.birthDate" yearFirst :max="latestBirthDate()" :aria-label="t('pages.settings.birthDate')" :aria-invalid="birthTooYoung || undefined"/>
					<p v-if="birthTooYoung" class="mt-1 text-xs text-danger" role="alert">{{ t('errors.validation.birthDate') }}</p>
				</div>
				<button class="btn btn-primary press" :disabled="!birthValid">{{ t('pages.welcome.setup.next') }}</button>
			</form>
			<form v-else-if="step === 'height'" class="flex gap-2" novalidate @submit.prevent="submitHeight">
				<label class="relative flex-1">
					<input v-model="draft.height" type="number" inputmode="decimal" class="input w-full pr-12" :aria-label="t('pages.welcome.setup.height')"/>
					<span class="absolute top-1/2 right-4 -translate-y-1/2 text-sm text-muted">cm</span>
				</label>
				<button class="btn btn-primary press" :disabled="!heightValid">{{ t('pages.welcome.setup.next') }}</button>
			</form>
			<form v-else-if="step === 'weight'" class="space-y-2" novalidate @submit.prevent="submitWeight()">
				<div class="grid grid-cols-2 gap-2">
					<label class="relative">
						<input v-model="draft.kg" type="number" inputmode="decimal" class="input w-full pr-12" :aria-label="t('pages.welcome.setup.weight')"/>
						<span class="absolute top-1/2 right-4 -translate-y-1/2 text-sm text-muted">kg</span>
					</label>
					<label class="relative">
						<input
							v-model="draft.bodyFat"
							type="number"
							inputmode="decimal"
							class="input w-full pr-12"
							:placeholder="t('common.optional')"
							:aria-label="t('pages.welcome.setup.bodyFat')"
						/>
						<span class="absolute top-1/2 right-4 -translate-y-1/2 text-sm text-muted">%</span>
					</label>
				</div>
				<div class="flex gap-2">
					<button type="button" class="btn btn-ghost press flex-1" @click="submitWeight(true)">{{ t('pages.welcome.setup.later') }}</button>
					<button class="btn btn-primary press flex-1" :disabled="!weightValid">{{ t('pages.welcome.setup.next') }}</button>
				</div>
			</form>
		</template>
		<p v-else class="text-center text-xs text-muted" aria-hidden="true">…</p>
	</div>
</div>
</template>

<style scoped>
/* 下に固定した回答欄の高さ分を空けて表示 */
.log > :deep(*) {
	scroll-margin-block-end: 9rem;
}

.bubble {
	inline-size: fit-content;
	max-inline-size: 85%;
	border-radius: 1.25rem;
	padding: 0.7rem 1rem;
	font-size: 0.9375rem;
	line-height: 1.7;
	animation: bubble-in 0.45s var(--ease-out-expo);
}

.bubble-user {
	display: inline-flex;
	align-items: center;
	border-end-end-radius: 0.375rem;
	background: var(--primary);
	color: var(--primary-ink);
	font-weight: 700;
	text-align: start;
}

.choice-btn {
	border: 2px solid var(--input-border);
	border-radius: 1rem;
	background: var(--glass);
	padding: 0.7rem 1rem;
	font-size: 0.875rem;
	text-align: center;
	transition:
		background-color 0.3s,
		border-color 0.3s;
}

.choice-btn:hover {
	border-color: var(--primary);
	background: var(--primary);
	color: var(--primary-ink);
}

@keyframes bubble-in {
	from {
		opacity: 0;
		transform: translateY(8px);
	}
}

@media (prefers-reduced-motion: reduce) {
	.bubble {
		animation: none;
	}
}
</style>
