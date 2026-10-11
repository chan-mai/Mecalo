<script setup lang="ts">
const props = defineProps<{ shown: boolean; title: string }>();
const emit = defineEmits<{ closed: [] }>();
const { t } = useI18n();

const dialog = useTemplateRef<HTMLDialogElement>('dialog');
const titleId = useId();
const scrollLock = useScrollLock(import.meta.client ? window.document.body : null);
const rendered = ref(false);

function sync(shown: boolean) {
	const el = dialog.value;
	if (!el) return;
	if (shown) {
		el.classList.remove('is-closing');
		rendered.value = true;
		if (!el.open) el.showModal();
		scrollLock.value = true;
		return;
	}
	if (!el.open) return;
	el.classList.add('is-closing');
}

function onAnimationEnd(event: AnimationEvent) {
	const el = dialog.value;
	if (!el || event.target !== el || !event.animationName.startsWith('sheet-out') || props.shown) return;
	el.classList.remove('is-closing');
	el.close();
	rendered.value = false;
	scrollLock.value = false;
}

onMounted(() => sync(props.shown));
watch(() => props.shown, sync, { flush: 'post' });

// Escキーは閉じる要求として親へ通知し, 表示状態は親が管理
function onCancel(event: Event) {
	event.preventDefault();
	emit('closed');
}

// 背景(::backdrop)のクリックはdialog要素自身へ届く
function onClick(event: MouseEvent) {
	if (event.target === dialog.value) emit('closed');
}
</script>

<template>
<Teleport to="body">
	<dialog ref="dialog" class="sheet" :aria-labelledby="titleId" @cancel="onCancel" @click="onClick" @animationend="onAnimationEnd">
		<div class="flex items-center gap-2 px-5 pt-3 pb-2">
			<h2 :id="titleId" class="flex-1 truncate font-semibold">{{ title }}</h2>
			<button type="button" class="btn btn-icon" :aria-label="t('common.close')" @click="emit('closed')">
				<Icon name="lucide:x" size="22" aria-hidden="true"/>
			</button>
		</div>
		<div class="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 pb-4">
			<slot v-if="rendered"></slot>
		</div>
	</dialog>
</Teleport>
</template>

<style scoped>
.sheet {
	margin: auto auto 0;
	inline-size: 100%;
	max-inline-size: 36rem;
	max-block-size: 90dvh;
	padding: 0 0 env(safe-area-inset-bottom);
	border: 0;
	border-start-start-radius: 2rem;
	border-start-end-radius: 2rem;
	background: var(--glass-strong);
	color: var(--foreground);
	box-shadow: 0 -16px 48px -16px rgb(37 50 74 / 0.3);
	backdrop-filter: blur(24px) saturate(1.4);
}

.sheet[open] {
	display: flex;
	flex-direction: column;
	animation: sheet-in 0.45s var(--ease-out-expo);
}

.sheet::before {
	content: "";
	align-self: center;
	margin-block-start: 0.5rem;
	inline-size: 2.5rem;
	block-size: 0.25rem;
	border-radius: 9999px;
	background: var(--hairline);
}

.sheet.is-closing {
	animation: sheet-out 0.35s cubic-bezier(0.4, 0, 1, 1) forwards;
}

.sheet.is-closing::backdrop {
	animation: fade-out 0.35s ease-in forwards;
}

.sheet::backdrop {
	background: rgb(37 50 74 / 0.25);
	backdrop-filter: blur(4px);
	animation: fade-in 0.3s ease-out;
}

@keyframes sheet-in {
	from {
		transform: translateY(100%);
	}
}

@keyframes sheet-out {
	to {
		transform: translateY(100%);
	}
}

@keyframes fade-out {
	to {
		opacity: 0;
	}
}

@keyframes fade-in {
	from {
		opacity: 0;
	}
}
</style>
