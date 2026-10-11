<script setup lang="ts" generic="T extends string">
import { Listbox, ListboxButton, ListboxOption, ListboxOptions } from '@headlessui/vue';

defineOptions({ inheritAttrs: false });

const props = withDefaults(defineProps<{
	options: readonly { value: T; label: string }[];
	// 表示ラベルがない場合の名前
	label?: string;
	labelledby?: string;
	compact?: boolean;
}>(), { label: undefined, labelledby: undefined, compact: false });

const model = defineModel<T>({ required: true });
const attrs = useAttrs();
const buttonId = computed(() => (attrs.id as string | undefined) ?? fallbackId);
const fallbackId = useId();
const current = computed(() => props.options.find((o) => o.value === model.value));

// ホバー・キー操作中の項目へ背景を移動
const activeEl = shallowRef<HTMLElement>();
const highlight = computed(() => (activeEl.value ? { transform: `translateY(${activeEl.value.offsetTop}px)`, height: `${activeEl.value.offsetHeight}px` } : { opacity: 0 }));

function trackActive(el: unknown, active: boolean) {
	if (active && el instanceof HTMLElement) activeEl.value = el;
}
</script>

<template>
<Listbox v-model="model" as="div" class="relative">
	<ListboxButton
		v-bind="attrs"
		:id="buttonId"
		v-slot="{ open }"
		class="input flex w-full items-center justify-between gap-2 text-left"
		:class="compact ? '!rounded-full !px-3 !py-1.5 text-sm' : ''"
		:aria-label="label"
		:aria-labelledby="labelledby ? `${labelledby} ${buttonId}` : undefined"
	>
		<span class="truncate">{{ current?.label }}</span>
		<Icon name="lucide:chevron-down" size="16" class="shrink-0 text-muted transition-transform duration-300" :class="{ 'rotate-180': open }" aria-hidden="true"/>
	</ListboxButton>
	<Transition
		enterActiveClass="transition duration-300 ease-[var(--ease-out-expo)]"
		enterFromClass="opacity-0 -translate-y-1 scale-95"
		leaveActiveClass="transition duration-150 ease-in"
		leaveToClass="opacity-0 scale-95"
	>
		<ListboxOptions class="panel absolute z-50 mt-2 max-h-64 w-full min-w-max origin-top overflow-auto !rounded-2xl !bg-[var(--glass-strong)] p-1.5" @vue:unmounted="activeEl = undefined">
			<span class="select-highlight pointer-events-none absolute inset-x-1.5 top-0 rounded-xl bg-surface-pink" :style="highlight" aria-hidden="true"></span>
			<ListboxOption v-for="o in options" :key="o.value" v-slot="{ active, selected }" :value="o.value" as="template">
				<li
					:ref="(el) => trackActive(el, active)"
					class="relative flex cursor-pointer items-center justify-between gap-3 rounded-xl px-3 py-2 text-sm transition-colors duration-200"
					:class="selected ? 'font-bold text-primary-ink' : ''"
				>
					{{ o.label }}
					<Icon v-if="selected" name="lucide:check" size="16" aria-hidden="true"/>
				</li>
			</ListboxOption>
		</ListboxOptions>
	</Transition>
</Listbox>
</template>

<style scoped>
.select-highlight {
	transition: transform 0.4s var(--ease-out-expo), height 0.4s var(--ease-out-expo), opacity 0.2s;
}
</style>
