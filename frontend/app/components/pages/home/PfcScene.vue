<script setup lang="ts">
import type { PfcBar, PfcHandle } from '~/utils/three/pfc';
import { canUseWebGL } from '~/utils/webgl';

const props = defineProps<{ bars: PfcBar[] }>();

const root = useTemplateRef<HTMLDivElement>('root');
const motion = usePreferredReducedMotion();
const reduced = computed(() => motion.value === 'reduce');
const webgl = ref(true);
let handle: PfcHandle | undefined;

onMounted(async () => {
	webgl.value = canUseWebGL();
	if (!webgl.value || !root.value) return;
	const { mountPfc } = await import('~/utils/three/pfc');
	handle = await mountPfc(root.value, props.bars, { animate: !reduced.value });
});

watch(() => props.bars, (bars) => handle?.update(bars), { deep: true });

onBeforeUnmount(() => handle?.dispose());
</script>

<template>
<div ref="root" class="relative" aria-hidden="true">
	<div v-if="!webgl" class="flex h-full items-end justify-center gap-3 pb-2">
		<div v-for="(bar, i) in bars" :key="i" class="relative h-24 w-8 overflow-hidden rounded-t-xl rounded-b-2xl border-2 border-[var(--input-border)] bg-[var(--glass)]">
			<div class="absolute inset-x-0 bottom-0" :style="{ height: `${Math.min(bar.value / (bar.max ?? 70), 1) * 100}%`, background: bar.color }"></div>
		</div>
	</div>
</div>
</template>
