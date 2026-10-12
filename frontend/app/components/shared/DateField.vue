<script setup lang="ts">
import { parseDate } from '@internationalized/date';
import {
	DatePickerCalendar,
	DatePickerCell,
	DatePickerCellTrigger,
	DatePickerContent,
	DatePickerField,
	DatePickerGrid,
	DatePickerGridBody,
	DatePickerGridHead,
	DatePickerGridRow,
	DatePickerHeadCell,
	DatePickerInput,
	DatePickerNext,
	DatePickerPrev,
	DatePickerRoot,
	DatePickerTrigger,
} from 'reka-ui';
import type { DateValue } from '@internationalized/date';
import { todayIso } from '~/utils/date';

defineOptions({ inheritAttrs: false });

// yearFirstは年, 月, 日の順に選ぶ 生年月日向け
const props = withDefaults(defineProps<{ max?: string; min?: string; yearFirst?: boolean }>(), { max: undefined, min: undefined, yearFirst: false });
// ISO形式, 未入力は空文字
const model = defineModel<string>({ required: true });
const { t, locale } = useI18n();
const attrs = useAttrs();

function parse(iso: string | undefined): DateValue | undefined {
	if (!iso) return undefined;
	try {
		return parseDate(iso);
	} catch {
		return undefined;
	}
}

const value = computed({
	get: () => parse(model.value),
	set: (v: DateValue | undefined) => {
		model.value = v ? v.toString() : '';
	},
});
const maxValue = computed(() => parse(props.max));
const minValue = computed(() => parse(props.min));

// 表示中の月, 未入力時は上限の日付から
const placeholder = shallowRef<DateValue>(parse(model.value) ?? maxValue.value ?? parseDate(todayIso()));
const isOpen = ref(false);
const view = ref<'year' | 'month' | 'day'>('day');
watch(isOpen, (o) => {
	if (!o) return;
	placeholder.value = value.value ?? placeholder.value;
	view.value = props.yearFirst ? 'year' : 'day';
});

// 12年ずつ表示
const yearPage = ref(0);
watch(view, (v) => {
	if (v === 'year') yearPage.value = Math.floor(placeholder.value.year / 12);
});
const years = computed(() => Array.from({ length: 12 }, (_, i) => yearPage.value * 12 + i));
const months = Array.from({ length: 12 }, (_, i) => i + 1);

function yearDisabled(y: number): boolean {
	return (maxValue.value !== undefined && y > maxValue.value.year) || (minValue.value !== undefined && y < minValue.value.year);
}

function monthDisabled(m: number): boolean {
	const y = placeholder.value.year;
	const max = maxValue.value;
	const min = minValue.value;
	return (max !== undefined && (y > max.year || (y === max.year && m > max.month))) || (min !== undefined && (y < min.year || (y === min.year && m < min.month)));
}

// ロケールに合わせた月の表記
function monthName(m: number): string {
	return new Date(2000, m - 1, 1).toLocaleString(locale.value, { month: 'short' });
}

// 入力欄でのEnterは囲むフォームの送信
function submitForm(event: KeyboardEvent) {
	const form = (event.currentTarget as HTMLElement).closest('form');
	// カレンダーボタン上は開閉を優先
	if (!form || isOpen.value || (event.target as HTMLElement).closest('button')) return;
	event.preventDefault();
	event.stopPropagation();
	form.requestSubmit();
}

function pickYear(y: number) {
	placeholder.value = placeholder.value.set({ year: y, day: 1 });
	view.value = 'month';
}

function pickMonth(m: number) {
	placeholder.value = placeholder.value.set({ month: m, day: 1 });
	view.value = 'day';
}
</script>

<template>
<DatePickerRoot v-model="value" v-model:placeholder="placeholder" v-model:open="isOpen" :locale="locale" :weekStartsOn="1" :maxValue="maxValue" :minValue="minValue" fixedWeeks weekdayFormat="short">
	<DatePickerField v-slot="{ segments }" v-bind="attrs" class="date-field input flex items-center gap-0.5 tabular-nums" @keydown.enter.capture="submitForm">
		<template v-for="item in segments" :key="item.part">
			<DatePickerInput v-if="item.part === 'literal'" :part="item.part" class="text-muted">{{ item.value }}</DatePickerInput>
			<DatePickerInput v-else :part="item.part" class="segment rounded-md px-0.5">{{ item.value }}</DatePickerInput>
		</template>
		<DatePickerTrigger class="press ml-auto grid size-8 place-items-center rounded-full text-muted" :aria-label="t('common.calendar.open')">
			<Icon name="lucide:calendar" size="18" aria-hidden="true"/>
		</DatePickerTrigger>
	</DatePickerField>
	<Teleport to="body">
		<DatePickerContent :sideOffset="8" align="end" class="panel z-50 w-[20.5rem] max-w-[calc(100vw-2rem)] !rounded-3xl !bg-[var(--glass-strong)] p-4">
			<!-- 年 -->
			<div v-if="view === 'year'" class="select-none">
				<div class="mb-3 flex items-center justify-between">
					<button type="button" class="btn-round" :aria-label="t('common.calendar.prevYears')" @click="yearPage--">
						<Icon name="lucide:arrow-left" size="18" aria-hidden="true"/>
					</button>
					<p class="numeric text-lg" aria-hidden="true">{{ years[0] }} - {{ years[11] }}</p>
					<button type="button" class="btn-round" :aria-label="t('common.calendar.nextYears')" @click="yearPage++">
						<Icon name="lucide:arrow-right" size="18" aria-hidden="true"/>
					</button>
				</div>
				<div class="grid grid-cols-3 gap-2">
					<button
						v-for="y in years"
						:key="y"
						type="button"
						class="calendar-day press grid h-11 place-items-center rounded-full text-center text-sm tabular-nums"
						:data-selected="y === value?.year ? '' : undefined"
						:data-disabled="yearDisabled(y) ? '' : undefined"
						:disabled="yearDisabled(y)"
						@click="pickYear(y)"
					>
						{{ y }}
					</button>
				</div>
			</div>
			<!-- 月 -->
			<div v-else-if="view === 'month'" class="select-none">
				<div class="mb-3 flex items-center justify-center">
					<button type="button" class="numeric press rounded-full px-3 text-lg" @click="view = 'year'">{{ placeholder.year }}</button>
				</div>
				<div class="grid grid-cols-3 gap-2">
					<button
						v-for="m in months"
						:key="m"
						type="button"
						class="calendar-day press grid h-11 place-items-center rounded-full text-center text-sm tabular-nums"
						:data-selected="value && m === value.month && placeholder.year === value.year ? '' : undefined"
						:data-disabled="monthDisabled(m) ? '' : undefined"
						:disabled="monthDisabled(m)"
						@click="pickMonth(m)"
					>
						{{ monthName(m) }}
					</button>
				</div>
			</div>
			<DatePickerCalendar v-else v-slot="{ weekDays, grid }" class="select-none">
				<div class="mb-3 flex items-center justify-between">
					<DatePickerPrev class="btn-round" :aria-label="t('common.calendar.prevMonth')">
						<Icon name="lucide:arrow-left" size="18" aria-hidden="true"/>
					</DatePickerPrev>
					<button v-for="month in grid" :key="month.value.toString()" type="button" class="numeric press rounded-full px-3 text-lg" :aria-label="t('common.calendar.pickYear')" @click="view = 'year'">
						{{ month.value.year }}.{{ String(month.value.month).padStart(2, '0') }}
					</button>
					<DatePickerNext class="btn-round" :aria-label="t('common.calendar.nextMonth')">
						<Icon name="lucide:arrow-right" size="18" aria-hidden="true"/>
					</DatePickerNext>
				</div>
				<DatePickerGrid v-for="month in grid" :key="month.value.toString()" class="w-full border-collapse">
					<DatePickerGridHead>
						<DatePickerGridRow class="grid grid-cols-7">
							<DatePickerHeadCell
								v-for="(day, i) in weekDays"
								:key="day"
								class="eyebrow py-1 text-center"
								:class="{ 'text-[var(--soda-ink)]': i === 5, 'text-[var(--sakura-ink)]': i === 6 }"
							>
								{{ day }}
							</DatePickerHeadCell>
						</DatePickerGridRow>
					</DatePickerGridHead>
					<DatePickerGridBody>
						<DatePickerGridRow v-for="(week, w) in month.rows" :key="w" class="grid grid-cols-7 gap-y-1">
							<DatePickerCell v-for="date in week" :key="date.toString()" :date="date" class="flex justify-center">
								<DatePickerCellTrigger
									:day="date"
									:month="month.value"
									class="calendar-day press grid size-10 place-items-center rounded-full text-sm tabular-nums"
								>
									{{ date.day }}
								</DatePickerCellTrigger>
							</DatePickerCell>
						</DatePickerGridRow>
					</DatePickerGridBody>
				</DatePickerGrid>
			</DatePickerCalendar>
		</DatePickerContent>
	</Teleport>
</DatePickerRoot>
</template>

<style scoped>
.segment:focus,
.segment:focus-visible {
	outline: none;
	box-shadow: none;
}

.date-field:focus-within {
	border-color: var(--focus-ring);
	outline: 2px solid transparent;
	box-shadow: 0 0 0 5px rgb(246 190 214 / 0.45);
}

.segment[data-placeholder] {
	color: var(--muted-foreground);
}

.calendar-day {
	transition:
		background-color 0.3s,
		color 0.3s,
		transform 0.35s var(--ease-out-expo);
}

.calendar-day:hover:not([data-disabled]) {
	background: var(--surface-pink);
}

.calendar-day[data-today] {
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
</style>
