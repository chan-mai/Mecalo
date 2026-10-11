<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';
import { levelIcon, levelTone } from '~/utils/nutrients';

defineProps<{ level: Schemas['Level'] }>();
const { t } = useI18n();
</script>

<template>
<span class="inline-flex items-center gap-0.5 rounded-full border-2 border-current px-1.5 py-px text-xs font-bold" :class="levelTone[level]">
	<Icon v-if="levelIcon[level]" :name="levelIcon[level]!" mode="svg" size="12" class="level-icon" aria-hidden="true"/>
	{{ t(`terms.level.${level}`) }}
</span>
</template>

<style scoped>
/* 線を太く */
.level-icon :deep(*) {
	stroke-width: 3;
}
</style>
