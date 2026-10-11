<script setup lang="ts">
import { canUseWebGL } from '~/utils/webgl';

const root = useTemplateRef<HTMLDivElement>('root');
const motion = usePreferredReducedMotion();
const reduced = computed(() => motion.value === 'reduce');
const webgl = ref(false);
let dispose: (() => void) | undefined;

onMounted(async () => {
	webgl.value = canUseWebGL();
	if (!webgl.value || !root.value) return;
	const { mountDecor } = await import('~/utils/three/decor');
	dispose = await mountDecor(root.value, { animate: !reduced.value });
});

onBeforeUnmount(() => dispose?.());
</script>

<template>
<div ref="root" aria-hidden="true" class="pointer-events-none">
	<div v-if="!webgl" class="h-full w-full"></div>
</div>
</template>
