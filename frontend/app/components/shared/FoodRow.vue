<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { formatInt } from '~/utils/nutrients';
import { formatQuantity } from '~/utils/meals';
import { nutrientsOf } from '~/utils/quantity';

const props = defineProps<{ food: Schemas['Food']; quantity?: Schemas['Quantity'] }>();
const emit = defineEmits<{ select: []; quickAdd: [] }>();
const { t } = useI18n();

const basisEnergy = computed(() => {
	const energy = props.food.revision.nutrients.energy;
	return energy ? `${formatInt(energy.amount)} kcal` : t('common.unknown');
});

const basisLabel = computed(() => {
	const { basis } = props.food.revision;
	return basis ? t('terms.perBasis', { unit: basis.unit }) : t('terms.per100g');
});

function kcalFor(q: Schemas['Quantity']): string {
	const energy = nutrientsOf(props.food.revision, q)?.energy;
	return energy ? `${formatInt(energy.amount)} kcal` : t('common.unknown');
}
</script>

<template>
<li class="flex items-center gap-2 py-2">
	<button type="button" class="row min-w-0 flex-1 text-left" @click="emit('select')">
		<span class="block truncate">{{ food.revision.name }}</span>
		<span class="block truncate text-xs text-muted">
			<template v-if="food.revision.brand">{{ food.revision.brand }} · </template>
			<template v-if="quantity">{{ formatQuantity(quantity) }} · {{ kcalFor(quantity) }}</template>
			<template v-else>{{ basisLabel }} {{ basisEnergy }}</template>
		</span>
	</button>
	<button
		v-if="quantity"
		type="button"
		class="press grid size-10 shrink-0 place-items-center rounded-full bg-tertiary text-tertiary-ink hover:bg-[#cfc5ff]"
		:aria-label="`${t('common.add')} ${food.revision.name}`"
		@click="emit('quickAdd')"
	>
		<Icon name="lucide:plus" size="20" aria-hidden="true"/>
	</button>
</li>
</template>
