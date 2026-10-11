<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { formatInt } from '~/utils/nutrients';
import { formatQuantity, pendingPrefix } from '~/utils/meals';

const props = defineProps<{ mealSlot: Schemas['MealSlot']; entries: Schemas['MealEntry'][]; waitingIds?: Set<string> }>();
const emit = defineEmits<{
	add: [slot: Schemas['MealSlot']];
	copyPrevious: [slot: Schemas['MealSlot']];
	select: [entry: Schemas['MealEntry']];
}>();
const { t } = useI18n();
const uid = useId();

const slotIcon: Record<Schemas['MealSlot'], string> = {
	breakfast: 'lucide:sunrise',
	lunch: 'lucide:sun',
	dinner: 'lucide:moon',
	snack: 'lucide:cookie',
};

const totalKcal = computed(() => props.entries.reduce((sum, entry) => sum + (entry.nutrients.energy?.amount ?? 0), 0));
</script>

<template>
<section v-reveal class="panel p-6" :aria-labelledby="`${uid}-title`">
	<SectionHeading :icon="slotIcon[mealSlot]" :eyebrow="t(`common.eyebrow.${mealSlot}`)" :title="t(`terms.slot.${mealSlot}`)" :headingId="`${uid}-title`">
		<p class="shrink-0 text-sm tabular-nums"><span class="numeric text-lg">{{ formatInt(totalKcal) }}</span><span class="eyebrow ml-1">kcal</span></p>
	</SectionHeading>
	<TransitionGroup v-if="entries.length" tag="ul" name="list" class="relative mt-3">
		<li v-for="entry in entries" :key="entry.id" class="border-b border-[var(--hairline)] last:border-0">
			<button type="button" class="row flex w-full items-center gap-3 py-3 text-left" @click="emit('select', entry)">
				<span class="min-w-0 flex-1">
					<span class="block truncate">
						{{ entry.foodName }}
						<span v-if="entry.id.startsWith(pendingPrefix) || waitingIds?.has(entry.id)" class="ml-1 rounded-full bg-warning-soft px-1.5 text-[10px] text-warning">
							{{ entry.id.startsWith(pendingPrefix) ? t('pages.home.mealSlotCard.pending') : t('pages.home.mealSlotCard.waiting') }}
						</span>
					</span>
					<span class="mt-0.5 block text-xs text-muted">{{ formatQuantity(entry.quantity) }}<template v-if="entry.quantity.unit !== 'g'"> ({{ formatInt(entry.grams) }}g)</template></span>
				</span>
				<span class="numeric text-sm">{{ entry.nutrients.energy ? formatInt(entry.nutrients.energy.amount) : '–' }}</span>
			</button>
		</li>
	</TransitionGroup>
	<div class="mt-4 flex gap-2">
		<button type="button" class="btn btn-primary flex-1" @click="emit('add', mealSlot)">
			<Icon name="lucide:plus" size="18" aria-hidden="true"/>{{ t('common.addFood') }}
		</button>
		<button v-if="entries.length === 0" type="button" class="btn btn-ghost" @click="emit('copyPrevious', mealSlot)">
			<Icon name="lucide:copy" size="16" aria-hidden="true"/><span class="text-sm">{{ t('pages.home.mealSlotCard.copyPrevious') }}</span>
		</button>
	</div>
</section>
</template>
