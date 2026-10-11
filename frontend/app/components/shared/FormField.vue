<script setup lang="ts">
const props = defineProps<{ label: string; error?: string; hint?: string; optional?: boolean }>();
const { t } = useI18n();

const id = useId();
const messageId = `${id}-message`;
const labelId = `${id}-label`;

// 入力欄へv-bindする関連付け属性
const field = computed(() => ({
	'id': id,
	'aria-describedby': props.error || props.hint ? messageId : undefined,
	'aria-invalid': props.error ? true : undefined,
}));
</script>

<template>
<div>
	<label :id="labelId" :for="id" class="mb-1 block text-sm text-muted">
		{{ label }}<span v-if="optional" class="ml-1 text-xs">({{ t('common.optional') }})</span>
	</label>
	<slot :field="field" :labelId="labelId"></slot>
	<p v-if="error" :id="messageId" class="mt-1 text-xs text-danger" role="alert">{{ t(error) }}</p>
	<p v-else-if="hint" :id="messageId" class="mt-1 text-xs text-muted">{{ hint }}</p>
</div>
</template>
