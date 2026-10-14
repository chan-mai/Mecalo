<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';

// max: 選択できる上限
const props = defineProps<{ labelledby?: string; max?: Schemas['TokenScope'] }>();
const model = defineModel<Schemas['TokenScope']>({ required: true });
const { t } = useI18n();
const name = useId();

const scopes = [
	{ value: 'read', icon: 'lucide:eye' },
	{ value: 'write', icon: 'lucide:pencil' },
] as const satisfies readonly { value: Schemas['TokenScope']; icon: string }[];
</script>

<template>
<div role="radiogroup" :aria-labelledby="labelledby" class="grid grid-cols-2 gap-2">
	<label v-for="s in scopes" :key="s.value" class="scope" :class="{ 'cursor-not-allowed opacity-60': props.max === 'read' && s.value === 'write' }">
		<input v-model="model" type="radio" :name="name" :value="s.value" :disabled="props.max === 'read' && s.value === 'write'" :aria-describedby="`${name}-${s.value}`">
		<span>
			<span class="flex items-center gap-2 font-semibold">
				<Icon :name="s.icon" size="18" class="scope-icon" aria-hidden="true"/>
				{{ t(`pages.mcp.scopes.${s.value}.label`) }}
			</span>
			<span :id="`${name}-${s.value}`" class="text-xs">{{ t(`pages.mcp.scopes.${s.value}.description`) }}</span>
		</span>
	</label>
</div>
</template>

<style scoped>
.scope {
	display: block;
	cursor: pointer;
}

.scope input {
	position: absolute;
	opacity: 0;
	pointer-events: none;
}

.scope > span {
	display: flex;
	height: 100%;
	flex-direction: column;
	gap: 0.25rem;
	border: 2px solid var(--input-border);
	border-radius: 1rem;
	background: var(--glass);
	padding: 0.6rem 1rem;
	font-size: 0.875rem;
	transition: background-color 0.3s, color 0.3s, border-color 0.3s;
}

.scope-icon {
	color: var(--muted-foreground);
	transition: color 0.3s;
}

.scope > input:checked + span {
	border-color: var(--primary);
	background: var(--primary);
	color: var(--primary-ink);
}

.scope > input:checked + span .scope-icon {
	color: var(--primary-ink);
}

.scope > input:focus-visible + span {
	box-shadow: var(--focus-shadow);
}
</style>
