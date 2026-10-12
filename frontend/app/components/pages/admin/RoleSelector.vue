<script setup lang="ts">
import type { Schemas } from '~/composables/api/client';

defineProps<{ labelledby?: string; disabled?: boolean }>();
const model = defineModel<Schemas['Role']>({ required: true });
const { t } = useI18n();
const name = useId();

const roles = [
	{ value: 'member', icon: 'lucide:user' },
	{ value: 'admin', icon: 'lucide:shield' },
] as const satisfies readonly { value: Schemas['Role']; icon: string }[];
</script>

<template>
<div role="radiogroup" :aria-labelledby="labelledby" class="grid grid-cols-2 gap-2" :class="{ 'opacity-60': disabled }">
	<label v-for="r in roles" :key="r.value" class="role" :class="[`role-${r.value}`, { 'cursor-not-allowed': disabled }]">
		<input v-model="model" type="radio" :name="name" :value="r.value" :disabled="disabled">
		<span>
			<Icon :name="r.icon" size="18" class="role-icon" aria-hidden="true"/>
			{{ t(`terms.role.${r.value}`) }}
		</span>
	</label>
</div>
</template>

<style scoped>
.role {
	display: block;
	cursor: pointer;
}

.role input {
	position: absolute;
	opacity: 0;
	pointer-events: none;
}

.role > span {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 0.5rem;
	border: 2px solid var(--input-border);
	border-radius: 1rem;
	background: var(--glass);
	padding: 0.6rem 1rem;
	font-size: 0.875rem;
	transition: background-color 0.3s, color 0.3s, border-color 0.3s;
}

/* 権限ごとの色 */
.role-member {
	--role-bg: var(--primary);
	--role-ink: var(--primary-ink);
}

.role-admin {
	--role-bg: var(--accent);
	--role-ink: var(--lavender-ink);
}

.role-icon {
	color: var(--muted-foreground);
	transition: color 0.3s;
}

.role > input:checked + span {
	border-color: var(--role-bg);
	background: var(--role-bg);
	color: var(--role-ink);
	font-weight: 700;
}

.role > input:checked + span .role-icon {
	color: var(--role-ink);
}

.role > input:focus-visible + span {
	box-shadow: var(--focus-shadow);
}
</style>
