<script setup lang="ts">
const props = defineProps<{ accept: string; title: string; hint?: string; busy?: boolean }>();
const emit = defineEmits<{ file: [file: File] }>();
const { t } = useI18n();

const zone = useTemplateRef<HTMLLabelElement>('zone');
const input = useTemplateRef<HTMLInputElement>('input');
const fileName = ref<string>();

const { isOverDropZone } = useDropZone(zone, {
	multiple: false,
	onDrop: (files) => files?.[0] && pick(files[0]),
});

function pick(file: File) {
	if (props.busy) return;
	fileName.value = file.name;
	emit('file', file);
}

function onChange() {
	const file = input.value?.files?.[0];
	if (file) pick(file);
	if (input.value) input.value.value = '';
}
</script>

<template>
<label
	ref="zone"
	class="dropzone"
	:class="{ 'is-over': isOverDropZone, 'is-busy': busy }"
>
	<input ref="input" type="file" class="sr-only" :accept="accept" :disabled="busy" @change="onChange">
	<span class="dropzone-icon" aria-hidden="true">
		<Icon :name="busy ? 'lucide:loader-circle' : 'lucide:file-up'" size="26" :class="{ 'animate-spin': busy }"/>
	</span>
	<span class="mt-3 block font-bold">{{ isOverDropZone ? t('components.dropzone.release') : title }}</span>
	<span class="mt-1 block text-xs text-muted">{{ t('components.dropzone.or') }}</span>
	<span v-if="hint" class="mt-3 block text-xs text-muted">{{ hint }}</span>
	<span v-if="fileName" class="eyebrow mt-3 block truncate" aria-live="polite">{{ fileName }}</span>
</label>
</template>

<style scoped>
.dropzone {
	display: flex;
	flex-direction: column;
	align-items: center;
	border: 2px dashed var(--input-border);
	border-radius: 1.5rem;
	background: var(--glass);
	padding: 2rem 1.25rem;
	text-align: center;
	cursor: pointer;
	transition: background-color 0.3s, border-color 0.3s, transform 0.4s var(--ease-out-expo);
}

.dropzone:hover,
.dropzone.is-over {
	border-color: var(--primary-ink);
	background: var(--surface-pink);
}

.dropzone.is-over {
	transform: scale(1.02);
}

.dropzone:focus-within {
	box-shadow: var(--focus-shadow);
}

.dropzone.is-busy {
	cursor: progress;
}

.dropzone-icon {
	display: grid;
	place-items: center;
	width: 3.5rem;
	height: 3.5rem;
	border-radius: 50%;
	background: var(--primary);
	color: var(--primary-ink);
	transition: transform 0.4s var(--ease-out-expo);
}

.dropzone:hover .dropzone-icon,
.dropzone.is-over .dropzone-icon {
	transform: translateY(-4px);
}
</style>
