<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';
import { formatInt } from '~/utils/nutrients';

const props = defineProps<{ query: string }>();
const emit = defineEmits<{ select: [food: Schemas['Food']] }>();
const { t } = useI18n();
const onError = useErrorHandler();
const api = useApi();

const result = ref<Schemas['ExternalSearchResult']>();
const searching = ref(false);
const saving = ref<string>();
// 外部検索が無効なインスタンス
const disabled = useState('external-search-disabled', () => false);

watch(
	() => props.query,
	() => {
		result.value = undefined;
	},
);

async function search() {
	searching.value = true;
	try {
		result.value = await unwrap(api.GET('/foods/external', { params: { query: { q: props.query } } }));
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) {
			disabled.value = true;
			return;
		}
		onError(err);
	} finally {
		searching.value = false;
	}
}

async function choose(f: Schemas['ExternalFood']) {
	const key = `${f.datasetId}/${f.reference}`;
	saving.value = key;
	try {
		const food = f.savedId
			? await unwrap(api.GET('/foods/{id}', { params: { path: { id: f.savedId } } }))
			: await unwrap(api.POST('/foods/external', { body: { datasetId: f.datasetId, reference: f.reference } }));
		emit('select', food);
	} catch (err) {
		onError(err);
	} finally {
		saving.value = undefined;
	}
}

function energyLabel(f: Schemas['ExternalFood']): string {
	const per = f.basis ? t('terms.perBasis', { unit: f.basis.unit }) : t('terms.per100g');
	const energy = f.nutrients.energy;
	return `${per} ${energy ? `${formatInt(energy.amount)} kcal` : t('common.unknown')}`;
}
</script>

<template>
<div v-if="!disabled" class="mt-3">
	<button v-if="!result" type="button" class="btn btn-ghost w-full" :disabled="searching" @click="search">
		<Icon name="lucide:globe" size="18" aria-hidden="true"/>
		{{ t('components.externalResults.search') }}
	</button>
	<SearchingIndicator v-if="searching"/>
	<section v-if="result" :aria-label="t('components.externalResults.title')">
		<p class="eyebrow mb-1">{{ t('components.externalResults.title') }}</p>
		<p v-if="result.failed.length" class="mb-2 rounded-lg bg-warning-soft p-2 text-sm text-warning" role="status">
			{{ t('components.externalResults.failed') }}
		</p>
		<p v-if="result.foods.length === 0" class="py-4 text-center text-sm text-muted">{{ t('components.foodPicker.noResults') }}</p>
		<ul class="divide-y divide-[var(--hairline)]">
			<li v-for="f in result.foods" :key="`${f.datasetId}/${f.reference}`" class="py-2">
				<button type="button" class="row w-full text-left" :disabled="saving !== undefined" @click="choose(f)">
					<span class="block truncate">{{ f.name }}</span>
					<span class="block truncate text-xs text-muted">
						<template v-if="f.brand">{{ f.brand }} · </template>{{ f.datasetName }} · {{ energyLabel(f) }}
					</span>
				</button>
			</li>
		</ul>
		<p v-for="a in result.attributions" :key="a" class="mt-2 text-xs text-muted">{{ a }}</p>
	</section>
</div>
</template>
