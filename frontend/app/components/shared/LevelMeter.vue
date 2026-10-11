<script setup lang="ts">
// 目標に対する摂取量, 色は判定結果ごと
const props = withDefaults(defineProps<{
	value: number;
	max: number;
	color?: string;
	labelledby?: string;
}>(), { color: 'var(--primary-mid)', labelledby: undefined });

const safeMax = computed(() => (props.max > 0 ? props.max : 1));
</script>

<template>
<meter
	class="meter"
	:value="Math.min(value, safeMax)"
	min="0"
	:max="safeMax"
	:aria-labelledby="labelledby"
	:style="{ '--meter-color': color }"
></meter>
</template>

<style scoped>
.meter {
	animation: meter-grow 1.1s var(--ease-out-expo) both;
	display: block;
	inline-size: 100%;
	block-size: 0.25rem;
	border: 0;
	border-radius: 9999px;
	background: var(--hairline);
	appearance: none;
}

.meter::-webkit-meter-bar {
	border: 0;
	border-radius: 9999px;
	background: var(--hairline);
}

.meter::-webkit-meter-optimum-value,
.meter::-webkit-meter-suboptimum-value,
.meter::-webkit-meter-even-less-good-value {
	border-radius: 9999px;
	background: var(--meter-color);
}

.meter::-moz-meter-bar {
	border-radius: 9999px;
	background: var(--meter-color);
}

@keyframes meter-grow {
	from {
		clip-path: inset(0 100% 0 0 round 9999px);
	}

	to {
		clip-path: inset(0 0 0 0 round 9999px);
	}
}
</style>
