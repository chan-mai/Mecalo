<script setup lang="ts">
import { getDayOfWeek, parseDate } from '@internationalized/date';
import { CalendarCell, CalendarCellTrigger, CalendarGrid, CalendarGridBody, CalendarGridHead, CalendarGridRow, CalendarHeadCell, CalendarNext, CalendarPrev, CalendarRoot } from 'reka-ui';
import type { DateValue } from '@internationalized/date';
import { todayIso } from '~/utils/date';

const props = withDefaults(defineProps<{ max?: string }>(), { max: undefined });
const model = defineModel<string>({ required: true });
const { t, locale } = useI18n();
const today = todayIso();

const value = computed({
	get: () => parseDate(model.value),
	set: (v: DateValue | undefined) => {
		if (v) model.value = v.toString();
	},
});
const placeholder = shallowRef<DateValue>(parseDate(model.value));
const maxValue = computed(() => (props.max ? parseDate(props.max) : undefined));
const monthLabel = computed(() => `${placeholder.value.year}.${String(placeholder.value.month).padStart(2, '0')}`);

// 月送りの向き, 切替演出に利用
const direction = ref<'next' | 'prev'>('next');
watch(placeholder, (to, from) => {
	direction.value = to.compare(from) >= 0 ? 'next' : 'prev';
});

// 祝日データは開いた時に読み込み
const holidays = shallowRef<Record<string, { name: string; name_en: string }>>({});
onMounted(async () => {
	const { default: holidayJp } = await import('@holiday-jp/holiday_jp');
	holidays.value = holidayJp.holidays;
});

function holidayName(iso: string) {
	const h = holidays.value[iso];
	if (!h) return undefined;
	return locale.value === 'ja' ? h.name : h.name_en;
}

function tone(date: DateValue) {
	const weekday = getDayOfWeek(date, 'en-US');
	if (holidayName(date.toString()) || weekday === 0) return 'text-[var(--sakura-ink)]';
	if (weekday === 6) return 'text-[var(--soda-ink)]';
	return '';
}

const monthHolidays = computed(() => {
	const prefix = monthLabel.value.replace('.', '-');
	return Object.keys(holidays.value)
		.filter((iso) => iso.startsWith(prefix))
		.map((iso) => ({ iso, day: Number(iso.slice(8)), name: holidayName(iso) }));
});
</script>

<template>
<CalendarRoot
	v-slot="{ weekDays, grid }"
	v-model="value"
	v-model:placeholder="placeholder"
	:locale="locale"
	:weekStartsOn="1"
	:maxValue="maxValue"
	fixedWeeks
	weekdayFormat="short"
	class="select-none"
>
	<div class="mb-4 flex items-center justify-between">
		<CalendarPrev class="btn-round" :aria-label="t('common.calendar.prevMonth')">
			<Icon name="lucide:arrow-left" size="18" aria-hidden="true"/>
		</CalendarPrev>
		<p class="numeric text-xl" aria-hidden="true">{{ monthLabel }}</p>
		<CalendarNext class="btn-round" :aria-label="t('common.calendar.nextMonth')">
			<Icon name="lucide:arrow-right" size="18" aria-hidden="true"/>
		</CalendarNext>
	</div>
	<div class="relative overflow-hidden">
		<Transition :name="`month-${direction}`" mode="out-in">
			<div :key="monthLabel">
				<CalendarGrid v-for="month in grid" :key="month.value.toString()" class="w-full border-collapse">
					<CalendarGridHead>
						<CalendarGridRow class="grid grid-cols-7">
							<CalendarHeadCell v-for="(day, i) in weekDays" :key="day" class="eyebrow py-1 text-center" :class="{ 'text-[var(--soda-ink)]': i === 5, 'text-[var(--sakura-ink)]': i === 6 }">
								{{ day }}
							</CalendarHeadCell>
						</CalendarGridRow>
					</CalendarGridHead>
					<CalendarGridBody>
						<CalendarGridRow v-for="(week, w) in month.rows" :key="w" class="grid grid-cols-7 gap-y-1">
							<CalendarCell v-for="date in week" :key="date.toString()" :date="date" class="flex justify-center">
								<CalendarCellTrigger
									:day="date"
									:month="month.value"
									class="calendar-day press relative grid size-11 place-items-center rounded-full text-sm tabular-nums"
									:class="[tone(date), { 'is-today': date.toString() === today }]"
									:title="holidayName(date.toString())"
								>
									{{ date.day }}
									<span v-if="holidayName(date.toString())" class="absolute bottom-1.5 size-1 rounded-full bg-current" aria-hidden="true"></span>
									<span v-if="holidayName(date.toString())" class="sr-only">{{ holidayName(date.toString()) }}</span>
								</CalendarCellTrigger>
							</CalendarCell>
						</CalendarGridRow>
					</CalendarGridBody>
				</CalendarGrid>
			</div>
		</Transition>
	</div>
	<ul v-if="monthHolidays.length" class="mt-3 space-y-1 text-xs">
		<li v-for="h in monthHolidays" :key="h.iso" class="flex gap-2 text-[var(--sakura-ink)]">
			<span class="numeric w-6 text-right">{{ h.day }}</span>{{ h.name }}
		</li>
	</ul>
	<button v-if="model !== today" type="button" class="btn btn-ghost mt-4 w-full" @click="model = today">
		{{ t('common.calendar.today') }}
	</button>
</CalendarRoot>
</template>

<style scoped>
.calendar-day {
	transition: background-color 0.3s, color 0.3s, transform 0.35s var(--ease-out-expo);
}

.calendar-day:hover:not([data-disabled]) {
	background: var(--surface-pink);
}

.calendar-day.is-today {
	border: 2px solid var(--input-border);
}

.calendar-day[data-selected] {
	background: var(--primary);
	color: var(--primary-ink);
	font-weight: 700;
}

.calendar-day[data-outside-view] {
	opacity: 0.35;
}

.calendar-day[data-disabled] {
	cursor: not-allowed;
	opacity: 0.3;
}

.month-next-enter-active,
.month-next-leave-active,
.month-prev-enter-active,
.month-prev-leave-active {
	transition: transform 0.35s var(--ease-out-expo), opacity 0.25s;
}

.month-next-enter-from,
.month-prev-leave-to {
	opacity: 0;
	transform: translateX(2rem);
}

.month-next-leave-to,
.month-prev-enter-from {
	opacity: 0;
	transform: translateX(-2rem);
}
</style>
