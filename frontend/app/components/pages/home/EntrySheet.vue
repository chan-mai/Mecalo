<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';
import { unwrap, useApi } from '~/composables/api/client';
import { pendingPrefix, slots } from '~/utils/meals';

const props = defineProps<{ entry?: Schemas['MealEntry'] }>();
const emit = defineEmits<{ closed: []; changed: [] }>();
const { t } = useI18n();
const confirmDelete = useConfirmDelete();
const uid = useId();
const onError = useErrorHandler();
const api = useApi();
const outbox = useOutbox();

const food = ref<Schemas['Food']>();
const busy = ref(false);
const date = ref('');
const slot = ref<Schemas['MealSlot']>('breakfast');
const isPending = computed(() => props.entry?.id.startsWith(pendingPrefix) ?? false);

// 単位換算と表示は記録時の値で行う
const snapshotFood = computed<Schemas['Food'] | undefined>(() => {
	const entry = props.entry;
	const current = food.value;
	if (!entry || !current) return undefined;
	const scaled = (factor: number) => Object.fromEntries(Object.entries(entry.nutrients).map(([k, v]) => [k, { ...v, amount: v.amount * factor }]));
	// 重量不明は記録した単位1つあたりの値
	if (entry.grams <= 0) {
		const basis = { unit: entry.quantity.unit };
		return { ...current, revision: { ...current.revision, name: entry.foodName, brand: entry.brand, nutrients: scaled(1 / entry.quantity.amount), basis, portions: [] } };
	}
	const per100 = scaled(100 / entry.grams);
	const portions = current.revision.id === entry.foodRevisionId ? [...current.revision.portions] : [];
	if (current.revision.basis && current.revision.id === entry.foodRevisionId && current.revision.basis.grams) {
		portions.unshift({ unit: current.revision.basis.unit, grams: current.revision.basis.grams });
	}
	if (entry.quantity.unit !== 'g' && !portions.some((p) => p.unit === entry.quantity.unit)) {
		portions.unshift({ unit: entry.quantity.unit, grams: entry.grams / entry.quantity.amount });
	}
	return { ...current, revision: { ...current.revision, name: entry.foodName, brand: entry.brand, nutrients: per100, basis: undefined, portions } };
});

watch(() => props.entry, async (entry) => {
	food.value = undefined;
	if (!entry) return;
	date.value = entry.date;
	slot.value = entry.slot;
	try {
		food.value = await unwrap(api.GET('/foods/{id}', { params: { path: { id: entry.foodId } } }));
	} catch {
		food.value = {
			id: entry.foodId, kind: 'custom', barcode: '', editable: false, favorite: false, deleted: true, discontinued: false,
			revision: { id: entry.foodRevisionId, number: 0, name: entry.foodName, brand: entry.brand, aliases: [], nutrients: {}, portions: [], source: entry.source, createdAt: entry.createdAt },
		};
	}
});

async function save(quantity: Schemas['Quantity']) {
	const body = { quantity, date: date.value, slot: slot.value };
	await run(async (id) => {
		if (id.startsWith(pendingPrefix)) {
			const before = props.entry?.quantity;
			outbox.updatePendingAdd(id.slice(pendingPrefix.length), (op) => {
				Object.assign(op.body, body);
				const kcal = op.preview.kcal;
				const ratio = before?.unit === quantity.unit ? quantity.amount / before.amount : undefined;
				op.preview.kcal = kcal !== undefined && ratio !== undefined ? kcal * ratio : undefined;
				op.preview.grams = op.preview.grams !== undefined && ratio !== undefined ? op.preview.grams * ratio : undefined;
			});
			return true;
		}
		const res = await outbox.run({ kind: 'updateMeal', id, body });
		return res.queued;
	}, t('common.messages.saved'));
}

async function recalculate() {
	if (!outbox.online.value) {
		toast.error(t('errors.offlineUnavailable'));
		return;
	}
	await run(async (id) => {
		await unwrap(api.POST('/meals/{id}/recalculate', { params: { path: { id } } }));
		return false;
	}, t('pages.home.entrySheet.recalculated'));
}

async function remove() {
	if (!await confirmDelete()) return;
	await run(async (id) => {
		if (id.startsWith(pendingPrefix)) {
			const ref = id.slice(pendingPrefix.length);
			const item = outbox.items.value.find((i) => i.op.kind === 'addMeal' && i.op.body.clientRef === ref);
			if (item) outbox.remove(item.key);
			return false;
		}
		const res = await outbox.run({ kind: 'deleteMeal', id });
		return res.queued;
	}, t('common.messages.deleted'));
}

// fnは端末保存のみで完了した場合にtrue
async function run(fn: (id: string) => Promise<boolean>, message: string) {
	const id = props.entry?.id;
	if (!id) return;
	busy.value = true;
	try {
		const queued = await fn(id);
		toast(queued ? t('common.messages.savedOffline') : message);
		emit('changed');
		emit('closed');
	} catch (err) {
		onError(err);
	} finally {
		busy.value = false;
	}
}
</script>

<template>
<BottomSheet :shown="entry !== undefined" :title="t('pages.home.entrySheet.title')" @closed="emit('closed')">
	<template v-if="entry && food && snapshotFood">
		<QuantityForm :key="entry.id" :food="snapshotFood" :initial="entry.quantity" :busy="busy" :submitLabel="t('common.save')" @submit="save">
			<template #extra>
				<div class="grid grid-cols-2 gap-2">
					<label>
						<span class="mb-1 block text-sm text-muted">{{ t('terms.date') }}</span>
						<input v-model="date" type="date" class="input" required>
					</label>
					<div>
						<span :id="`${uid}-slot`" class="mb-1 block text-sm text-muted">{{ t('terms.mealSlot') }}</span>
						<SelectField v-model="slot" :labelledby="`${uid}-slot`" :options="slots.map((s) => ({ value: s, label: t(`terms.slot.${s}`) }))"/>
					</div>
				</div>
			</template>
		</QuantityForm>

		<details class="disclosure mt-4 rounded-2xl border-2 border-[var(--input-border)] p-3">
			<summary class="text-sm">{{ t('pages.home.entrySheet.snapshot') }}<Icon name="lucide:chevron-down" class="disclosure-icon text-muted" size="16" aria-hidden="true"/></summary>
			<NutrientTable class="mt-2" :profile="entry.nutrients"/>
			<p class="mt-2 text-xs text-muted">{{ t('pages.home.entrySheet.source') }}: {{ entry.source.datasetId }} {{ entry.source.reference }}</p>
		</details>

		<div class="mt-4 flex flex-col gap-2">
			<button v-if="!isPending && !food.deleted && food.revision.id !== entry.foodRevisionId" type="button" class="btn btn-ghost" :disabled="busy" @click="recalculate">
				<Icon name="lucide:refresh-cw" size="18" aria-hidden="true"/>{{ t('pages.home.entrySheet.recalculate') }}
			</button>
			<p v-if="!isPending && !food.deleted && food.revision.id !== entry.foodRevisionId" class="text-xs text-muted">{{ t('pages.home.entrySheet.recalculateHint') }}</p>
			<button type="button" class="btn text-danger" :disabled="busy" @click="remove">
				<Icon name="lucide:trash-2" size="18" aria-hidden="true"/>{{ t('common.delete') }}
			</button>
		</div>
	</template>
</BottomSheet>
</template>
