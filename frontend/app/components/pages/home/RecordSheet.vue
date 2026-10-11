<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { newClientRef, slots } from '~/utils/meals';
import { defaultQuantity, gramsOf } from '~/utils/quantity';

const props = defineProps<{ shown: boolean; date: string; initialSlot: Schemas['MealSlot'] }>();
const emit = defineEmits<{ closed: []; recorded: [] }>();
const { t } = useI18n();
const onError = useErrorHandler();
const outbox = useOutbox();

const slot = ref<Schemas['MealSlot']>('breakfast');
const selected = ref<{ food: Schemas['Food']; quantity: Schemas['Quantity'] }>();
const busy = ref(false);

watch(() => props.shown, (shown) => {
	if (shown) {
		slot.value = props.initialSlot;
		selected.value = undefined;
	}
}, { immediate: true });

async function record(food: Schemas['Food'], quantity: Schemas['Quantity']) {
	busy.value = true;
	try {
		const grams = gramsOf(food.revision, quantity);
		const energy = food.revision.nutrients.energy;
		const res = await outbox.run({
			kind: 'addMeal',
			body: { date: props.date, slot: slot.value, foodId: food.id, quantity, clientRef: newClientRef() },
			preview: {
				foodName: food.revision.name,
				grams,
				kcal: energy && grams !== undefined ? energy.amount * grams / 100 : undefined,
			},
		});
		toast(res.queued ? t('common.messages.savedOffline') : t('pages.home.recordSheet.added', { name: food.revision.name }));
		selected.value = undefined;
		emit('recorded');
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<BottomSheet :shown="shown" :title="t('pages.home.recordSheet.title', { slot: t(`terms.slot.${slot}`) })" @closed="emit('closed')">
	<fieldset class="mb-3 flex gap-1 overflow-x-auto">
		<legend class="sr-only">{{ t('terms.mealSlot') }}</legend>
		<label v-for="s in slots" :key="s" class="choice shrink-0">
			<input v-model="slot" type="radio" name="record-slot" :value="s">
			<span>{{ t(`terms.slot.${s}`) }}</span>
		</label>
	</fieldset>

	<div v-if="selected">
		<button type="button" class="mb-2 flex items-center text-sm link" @click="selected = undefined">
			<Icon name="lucide:chevron-left" size="16" aria-hidden="true"/>{{ t('common.back') }}
		</button>
		<QuantityForm :food="selected.food" :initial="selected.quantity" :busy="busy" @submit="(q) => record(selected!.food, q)"/>
	</div>
	<FoodPicker
		v-else
		@select="(food, quantity) => (selected = { food, quantity: quantity ?? defaultQuantity(food.revision) })"
		@quickAdd="record"
	/>
</BottomSheet>
</template>
