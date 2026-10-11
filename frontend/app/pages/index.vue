<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';
import { addDays, isIsoDate, todayIso } from '~/utils/date';
import { defaultSlotForNow, slots } from '~/utils/meals';

const { t } = useI18n();
const route = useRoute();
const router = useRouter();
const onError = useErrorHandler();
const api = useApi();

const date = computed({
	get: () => (isIsoDate(route.query.date) ? route.query.date : todayIso()),
	set: (value: string) => {
		void router.replace({ query: value === todayIso() ? {} : { date: value } });
	},
});

const { data: day, pending, error, refresh } = useAsyncData(
	() => `day-${date.value}`,
	() => unwrap(api.GET('/days/{date}', { params: { path: { date: date.value } } })),
	{ watch: [date] },
);

const { added, waitingIds, merge } = usePendingEntries(date);
const entries = computed(() => merge(day.value?.entries ?? []));

const recordOpen = ref(false);
const recordSlot = ref<Schemas['MealSlot']>('breakfast');
const editing = ref<Schemas['MealEntry']>();

function openRecord(slot: Schemas['MealSlot']) {
	recordSlot.value = slot;
	recordOpen.value = true;
}

async function copyPrevious(slot: Schemas['MealSlot']) {
	try {
		const copied = await unwrap(api.POST('/meals/copy', {
			body: { fromDate: addDays(date.value, -1), fromSlot: slot, toDate: date.value },
		}));
		toast(copied.length ? t('pages.home.copied', { count: copied.length }) : t('pages.home.nothingToCopy'));
		if (copied.length) await refresh();
	} catch (err) {
		onError(err);
	}
}

const root = useTemplateRef<HTMLDivElement>('root');
useSwipe(root, {
	threshold: 60,
	onSwipeEnd(_, direction) {
		if (recordOpen.value || editing.value || (direction !== 'left' && direction !== 'right')) return;
		const next = addDays(date.value, direction === 'left' ? 1 : -1);
		if (next <= todayIso()) date.value = next;
	},
});
</script>

<template>
<div ref="root" class="space-y-4">
	<h1 class="sr-only">{{ t('pages.home.title') }}</h1>
	<div class="relative pt-2 pb-4">
		<DateSwitcher v-model="date"/>
	</div>

	<p v-if="error" class="panel p-5 text-sm" role="alert">
		{{ t('errors.generic') }}
		<button type="button" class="link ml-2" @click="refresh()">{{ t('common.retry') }}</button>
	</p>

	<template v-if="day">
		<EnergyCard :report="day.report"/>

		<NuxtLink v-if="!day.report.targets" v-reveal to="/settings" class="panel block p-6 text-sm">
			{{ t('pages.home.setupProfile') }}
			<span class="link mt-2 flex items-center gap-1 font-bold">{{ t('pages.home.setupProfileAction') }}<Icon name="lucide:arrow-right" size="16" aria-hidden="true"/></span>
		</NuxtLink>
		<AttentionCard v-else :evaluations="day.report.evaluations" :recorded="day.report.summary.entryCount > 0" :date="date"/>

		<MealSlotCard
			v-for="slot in slots"
			:key="slot"
			:mealSlot="slot"
			:entries="entries.filter((entry) => entry.slot === slot)"
			:waitingIds="waitingIds"
			@add="openRecord"
			@copyPrevious="copyPrevious"
			@select="(entry) => (editing = entry)"
		/>

		<p v-if="added.length" class="text-center text-xs text-muted">{{ t('pages.home.pendingNote') }}</p>

		<WeightCard :date="date" :latest="day.latestWeight" @saved="refresh()"/>
	</template>
	<div v-else-if="pending" class="space-y-4" aria-busy="true">
		<p class="sr-only">{{ t('common.loading') }}</p>
		<div v-for="i in 3" :key="i" class="panel h-40 animate-pulse"></div>
	</div>

	<button
		type="button"
		class="fab press fixed z-30 grid size-14 place-items-center rounded-full bg-secondary text-secondary-ink shadow-[0_12px_32px_-12px_rgb(177_76_120/0.6)]"
		:aria-label="t('common.addFood')"
		@click="openRecord(defaultSlotForNow())"
	>
		<Icon name="lucide:plus" size="26" aria-hidden="true"/>
	</button>

	<RecordSheet :shown="recordOpen" :date="date" :initialSlot="recordSlot" @closed="recordOpen = false" @recorded="refresh()"/>
	<EntrySheet :entry="editing" @closed="editing = undefined" @changed="refresh()"/>
</div>
</template>

<style scoped>
.fab {
	inset-inline-end: 1.25rem;
	inset-block-end: calc(6.25rem + env(safe-area-inset-bottom));
}

@media (min-width: 1024px) {
	.fab {
		inset-inline-end: 1.5rem;
		inset-block-end: calc(0.75rem + env(safe-area-inset-bottom) + (var(--nav-height) - 3.5rem) / 2);
	}
}
</style>
