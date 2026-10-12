<script setup lang="ts">
import { toast } from 'vue-sonner';
import { Tab, TabGroup, TabList, TabPanel, TabPanels } from '@headlessui/vue';
import type { Schemas } from '~/composables/api/client';
import { ApiError, unwrap, useApi } from '~/composables/api/client';

withDefaults(defineProps<{ quickAdd?: boolean }>(), { quickAdd: true });
const emit = defineEmits<{
	select: [food: Schemas['Food'], quantity?: Schemas['Quantity']];
	quickAdd: [food: Schemas['Food'], quantity: Schemas['Quantity']];
}>();
const { t } = useI18n();
const onError = useErrorHandler();
const api = useApi();

const tabs = [
	{ key: 'recent', label: 'components.foodPicker.recent' },
	{ key: 'frequent', label: 'components.foodPicker.frequent' },
	{ key: 'favorites', label: 'terms.favorites' },
] as const;

const query = ref('');
const tabIndex = ref(0);

function clearQuery() {
	query.value = '';
	searchInput.value?.focus();
}

const results = ref<Schemas['Food'][]>([]);
const searching = ref(false);
const scanning = ref(false);
const quickPicks = ref<Schemas['QuickPicks']>();
const searchInput = useTemplateRef<HTMLInputElement>('searchInput');

defineExpose({ searchInput });

onMounted(async () => {
	try {
		quickPicks.value = await unwrap(api.GET('/meals/quick-picks'));
	} catch (err) {
		onError(err);
	}
});

let timer: ReturnType<typeof setTimeout> | undefined;
let seq = 0;
watch(query, (q) => {
	clearTimeout(timer);
	if (!q.trim()) {
		results.value = [];
		return;
	}
	searching.value = true;
	timer = setTimeout(async () => {
		const current = ++seq;
		try {
			const list = await unwrap(api.GET('/foods', { params: { query: { q, limit: 40 } } }));
			if (current === seq) results.value = list;
		} catch (err) {
			onError(err);
		} finally {
			if (current === seq) searching.value = false;
		}
	}, 250);
});

// 次回の検索順位に反映, 失敗しても選択は継続
function selectResult(f: Schemas['Food']) {
	const q = query.value.trim();
	if (q) {
		void unwrap(api.POST('/foods/{id}/selections', { params: { path: { id: f.id } }, body: { query: q } })).catch(() => undefined);
	}
	emit('select', f);
}

async function onBarcode(code: string) {
	scanning.value = false;
	try {
		const food = await unwrap(api.GET('/foods/barcode', { params: { query: { code } } }));
		emit('select', food);
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) {
			toast.error(t('components.foodPicker.barcodeNotFound'));
			await navigateTo({ path: '/foods/new', query: { barcode: code } });
			return;
		}
		onError(err);
	}
}

</script>

<template>
<div>
	<div class="sticky top-0 z-10 flex gap-2 pb-2">
		<label class="relative flex-1">
			<span class="sr-only">{{ t('common.search') }}</span>
			<Icon name="lucide:search" class="absolute top-1/2 left-3 -translate-y-1/2 text-muted" size="18" aria-hidden="true"/>
			<input
				ref="searchInput"
				v-model="query"
				type="search"
				enterkeyhint="search"
				class="input pr-10 pl-9"
				:placeholder="t('components.foodPicker.searchPlaceholder')"
				autocomplete="off"
			>
			<button
				v-if="query"
				type="button"
				class="absolute top-1/2 right-2 grid size-7 -translate-y-1/2 place-items-center rounded-full text-muted"
				:aria-label="t('components.foodPicker.clear')"
				@click="clearQuery"
			>
				<Icon name="lucide:x" size="16" aria-hidden="true"/>
			</button>
		</label>
		<button type="button" class="btn-round !size-12 shrink-0" :aria-label="t('components.foodPicker.scan')" @click="scanning = true">
			<Icon name="lucide:scan-barcode" size="20" aria-hidden="true"/>
		</button>
	</div>

	<BottomSheet :shown="scanning" :title="t('components.barcodeScanner.title')" @closed="scanning = false">
		<BarcodeScanner @found="onBarcode"/>
	</BottomSheet>

	<template v-if="query.trim()">
		<SearchingIndicator v-if="searching"/>
		<div v-else-if="results.length === 0" class="py-6 text-center text-sm text-muted">
			<p>{{ t('components.foodPicker.noResults') }}</p>
			<NuxtLink :to="{ path: '/foods/new', query: { name: query } }" class="mt-2 inline-block link">{{ t('components.foodPicker.createFood') }}</NuxtLink>
		</div>
		<TransitionGroup v-else tag="ul" name="list" class="relative divide-y divide-[var(--hairline)]" appear>
			<FoodRow v-for="f in results" :key="f.id" :food="f" @select="selectResult(f)"/>
		</TransitionGroup>
		<ExternalResults v-if="!searching" :query="query.trim()" @select="selectResult"/>
	</template>

	<template v-else-if="quickPicks">
		<div v-if="quickPicks.searches.length" class="mb-2 flex flex-wrap gap-1.5">
			<button v-for="s in quickPicks.searches.slice(0, 8)" :key="s" type="button" class="press rounded-full border-2 border-[var(--input-border)] bg-[var(--glass)] px-3 py-1 text-sm" @click="query = s">
				{{ s }}
			</button>
		</div>
		<TabGroup :selectedIndex="tabIndex" as="div" @change="(i: number) => (tabIndex = i)">
			<TabList class="relative mb-1 grid grid-cols-3 rounded-full border-2 border-[var(--input-border)] p-1">
				<span
					class="tab-indicator absolute inset-y-1 left-1 rounded-full bg-primary"
					:style="{ width: `calc((100% - 0.5rem) / ${tabs.length})`, transform: `translateX(${tabIndex * 100}%)` }"
					aria-hidden="true"
				></span>
				<Tab v-for="tab in tabs" :key="tab.key" v-slot="{ selected }" as="template">
					<button type="button" class="tab press relative rounded-full py-1.5 text-center text-sm transition-colors duration-300" :class="selected ? 'font-bold text-primary-ink' : 'text-muted'">
						{{ t(tab.label) }}
					</button>
				</Tab>
			</TabList>
			<TabPanels>
				<TabPanel v-for="tab in tabs" :key="tab.key">
					<p v-if="quickPicks[tab.key].length === 0" class="py-6 text-center text-sm text-muted">{{ t('components.foodPicker.empty') }}</p>
					<ul class="divide-y divide-[var(--hairline)]">
						<FoodRow
							v-for="p in quickPicks[tab.key]"
							:key="p.food.id"
							:food="p.food"
							:quantity="quickAdd ? p.quantity : undefined"
							@select="emit('select', p.food, p.quantity)"
							@quickAdd="emit('quickAdd', p.food, p.quantity)"
						/>
					</ul>
				</TabPanel>
			</TabPanels>
		</TabGroup>
	</template>
</div>
</template>

<style scoped>
/* 選択中タブへフォーカス移動, 位置は背景で表示 */
.tab:focus-visible {
	box-shadow: none;
}

.tab-indicator {
	transition: transform 0.55s var(--ease-out-expo);
}
</style>
