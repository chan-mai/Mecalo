<script setup lang="ts">
withDefaults(defineProps<{
	title: string;
	eyebrow: string;
	icon?: string;
	level?: 'h1' | 'h2';
	headingId?: string;
}>(), { icon: undefined, level: 'h2', headingId: undefined });
</script>

<template>
<div class="flex items-center justify-between gap-3">
	<div class="flex min-w-0 items-center gap-3">
		<span v-if="icon" class="grid size-10 shrink-0 place-items-center rounded-full bg-tertiary text-tertiary-ink" aria-hidden="true">
			<Icon :name="icon" size="18"/>
		</span>
		<div class="min-w-0">
			<p class="eyebrow" aria-hidden="true">{{ eyebrow }}</p>
			<component :is="level" :id="headingId" class="mt-1 truncate leading-tight font-bold" :class="level === 'h1' ? 'text-2xl' : 'text-base'">{{ title }}</component>
		</div>
	</div>
	<slot></slot>
</div>
</template>
