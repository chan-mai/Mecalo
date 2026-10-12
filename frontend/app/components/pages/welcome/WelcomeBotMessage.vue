<script setup lang="ts">
defineProps<{ typing?: boolean; label?: string }>();
</script>

<template>
<li class="flex items-end gap-2">
	<span class="avatar" aria-hidden="true">
		<svg viewBox="0 0 76 76" width="22" height="22">
			<circle cx="38" cy="26.6" r="20.27" fill="#B9DEFF" style="mix-blend-mode: multiply"/>
			<circle cx="26" cy="48.1" r="20.27" fill="#F6BED6" style="mix-blend-mode: multiply"/>
			<circle cx="50" cy="48.1" r="20.27" fill="#DDD6FF" style="mix-blend-mode: multiply"/>
		</svg>
	</span>
	<div class="bubble" :class="{ typing }" :aria-label="label">
		<template v-if="typing"><span></span><span></span><span></span></template>
		<slot v-else></slot>
	</div>
</li>
</template>

<style scoped>
.avatar {
	display: grid;
	flex-shrink: 0;
	place-items: center;
	inline-size: 2.25rem;
	block-size: 2.25rem;
	border: 1px solid rgb(255 255 255 / 0.7);
	border-radius: 9999px;
	background: var(--glass-strong);
	box-shadow: 0 6px 18px -10px rgb(37 50 74 / 0.4);
}

.bubble {
	inline-size: fit-content;
	max-inline-size: 85%;
	border: 1px solid rgb(255 255 255 / 0.7);
	border-radius: 1.25rem;
	border-end-start-radius: 0.375rem;
	background: var(--glass);
	padding: 0.7rem 1rem;
	font-size: 0.9375rem;
	line-height: 1.7;
	backdrop-filter: blur(18px) saturate(1.4);
	animation: bubble-in 0.45s var(--ease-out-expo);
}

.typing {
	display: flex;
	gap: 0.3rem;
	padding-block: 1rem;
}

.typing span {
	inline-size: 0.4rem;
	block-size: 0.4rem;
	border-radius: 9999px;
	background: var(--muted-foreground);
	animation: blink 1.2s infinite;
}

.typing span:nth-child(2) {
	animation-delay: 0.15s;
}

.typing span:nth-child(3) {
	animation-delay: 0.3s;
}

@keyframes bubble-in {
	from {
		opacity: 0;
		transform: translateY(8px);
	}
}

@keyframes blink {
	50% {
		opacity: 0.3;
	}
}

@media (prefers-reduced-motion: reduce) {
	.bubble,
	.typing span {
		animation: none;
	}
}
</style>
