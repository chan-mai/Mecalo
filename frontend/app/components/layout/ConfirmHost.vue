<script setup lang="ts">
const request = useConfirmRequest();

function answer(ok: boolean) {
	const r = request.value;
	request.value = undefined;
	r?.resolve(ok);
}
</script>

<template>
<ConfirmDialog
	:shown="request !== undefined"
	:title="request?.title ?? ''"
	:message="request?.message"
	:confirmLabel="request?.confirmLabel ?? ''"
	:danger="request?.danger"
	@confirm="answer(true)"
	@cancel="answer(false)"
/>
</template>
