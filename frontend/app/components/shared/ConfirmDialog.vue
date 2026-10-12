<script setup lang="ts">
const props = defineProps<{ shown: boolean; title: string; message?: string; confirmLabel: string; danger?: boolean }>();
const emit = defineEmits<{ confirm: []; cancel: [] }>();
const { t } = useI18n();

const dialog = useTemplateRef<HTMLDialogElement>('dialog');
const titleId = useId();
const messageId = useId();

function sync(shown: boolean) {
	const el = dialog.value;
	if (!el) return;
	if (shown && !el.open) el.showModal();
	if (!shown && el.open) el.close();
}

onMounted(() => sync(props.shown));
watch(() => props.shown, sync, { flush: 'post' });

// Escキー・背景のクリックは取消として親へ通知
function onCancel(event: Event) {
	event.preventDefault();
	emit('cancel');
}

function onClick(event: MouseEvent) {
	if (event.target === dialog.value) emit('cancel');
}
</script>

<template>
<Teleport to="body">
	<dialog
		ref="dialog"
		class="confirm"
		role="alertdialog"
		:aria-labelledby="titleId"
		:aria-describedby="message ? messageId : undefined"
		@cancel="onCancel"
		@click="onClick"
	>
		<h2 :id="titleId" class="font-semibold">{{ title }}</h2>
		<p v-if="message" :id="messageId" class="mt-2 text-sm text-muted">{{ message }}</p>
		<div class="mt-5 flex justify-end gap-2">
			<button type="button" class="btn btn-ghost" @click="emit('cancel')">{{ t('common.cancel') }}</button>
			<button type="button" class="btn btn-primary" :class="{ 'text-danger': danger }" @click="emit('confirm')">{{ confirmLabel }}</button>
		</div>
	</dialog>
</Teleport>
</template>

<style scoped>
.confirm {
	margin: auto;
	inline-size: calc(100% - 2rem);
	max-inline-size: 24rem;
	padding: 1.5rem;
	border: 0;
	border-radius: 1.75rem;
	background: var(--glass-strong);
	color: var(--foreground);
	box-shadow: 0 16px 48px -16px rgb(37 50 74 / 0.3);
	backdrop-filter: blur(24px) saturate(1.4);
}

.confirm[open] {
	animation: confirm-in 0.35s var(--ease-out-expo);
}

.confirm::backdrop {
	background: rgb(37 50 74 / 0.25);
	backdrop-filter: blur(4px);
	animation: fade-in 0.3s ease-out;
}

@keyframes confirm-in {
	from {
		opacity: 0;
		transform: scale(0.95);
	}
}

@keyframes fade-in {
	from {
		opacity: 0;
	}
}
</style>
