<script setup lang="ts">
import { addDays, formatDate, todayIso } from '~/utils/date';

const model = defineModel<string>({ required: true });
const { t, locale } = useI18n();
const today = todayIso();
const calendarOpen = ref(false);

const label = computed(() => {
	if (model.value === today) return t('common.today');
	if (model.value === addDays(today, -1)) return t('common.yesterday');
	return formatDate(model.value, t('common.dateFormat.long'), locale.value);
});
const short = computed(() => formatDate(model.value, 'MM.DD', 'en'));
const weekday = computed(() => `${formatDate(model.value, 'YYYY', 'en')} / ${formatDate(model.value, 'ddd', 'en').toUpperCase()}`);
</script>

<template>
<div>
	<button type="button" class="press group block text-left" :aria-label="`${label} ${t('common.calendar.open')}`" @click="calendarOpen = true">
		<span class="eyebrow flex items-center gap-1.5" aria-hidden="true">{{ weekday }}<Icon name="lucide:calendar-days" size="12"/></span>
		<time :datetime="model" class="numeric mt-2 block text-[clamp(3.25rem,17vw,4.75rem)] leading-[0.9]" aria-hidden="true">{{ short }}</time>
		<span class="mt-2 flex items-center gap-1 text-sm" aria-hidden="true">{{ label }}<Icon name="lucide:chevron-down" size="14" class="text-muted transition-transform duration-300 group-hover:translate-y-0.5"/></span>
	</button>
	<BottomSheet :shown="calendarOpen" :title="t('common.calendar.title')" @closed="calendarOpen = false">
		<CalendarPicker v-model="model" :max="today" @update:modelValue="calendarOpen = false"/>
	</BottomSheet>
</div>
</template>
