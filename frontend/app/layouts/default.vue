<script setup lang="ts">
const { t } = useI18n();

const items = [
	{ to: '/', icon: 'lucide:house', label: 'layout.nav.home' },
	{ to: '/history', icon: 'lucide:chart-column', label: 'layout.nav.history' },
	{ to: '/foods', icon: 'lucide:salad', label: 'layout.nav.foods' },
	{ to: '/settings', icon: 'lucide:settings', label: 'layout.nav.settings' },
] as const;
</script>

<template>
<div class="mx-auto flex min-h-dvh max-w-xl flex-col">
	<NuxtLink to="#main" external class="skip-link">{{ t('layout.skipToContent') }}</NuxtLink>
	<header class="flex items-center justify-between px-5 pt-[calc(1rem+env(safe-area-inset-top))]">
		<NuxtLink to="/" class="display text-sm tracking-[0.3em]">{{ t('layout.appName').toUpperCase() }}</NuxtLink>
	</header>
	<main id="main" tabindex="-1" class="flex-1 px-5 pt-4 pb-[calc(7rem+env(safe-area-inset-bottom))]">
		<SyncStatus/>
		<slot></slot>
	</main>
	<nav
		class="nav-bar panel fixed z-30 px-2 py-1.5"
		:aria-label="t('layout.nav.label')"
	>
		<ul class="grid h-full grid-cols-4 items-center">
			<li v-for="item in items" :key="item.to">
				<NuxtLink
					:to="item.to"
					class="press flex flex-col items-center gap-0.5 py-1 text-[10px] tracking-[0.08em] text-muted"
					activeClass="nav-active !text-primary-ink"
				>
					<span class="nav-icon"><Icon :name="item.icon" size="20" aria-hidden="true"/></span>
					{{ t(item.label) }}
				</NuxtLink>
			</li>
		</ul>
	</nav>
</div>
</template>

<style scoped>
.skip-link {
	position: absolute;
	inset-block-start: -100%;
	inset-inline-start: 1rem;
	z-index: 60;
	border-radius: 9999px;
	background: var(--primary);
	padding: 0.5rem 1rem;
	color: var(--primary-ink);
}

.skip-link:focus {
	inset-block-start: 1rem;
}

.nav-active {
	font-weight: 700;
}

.nav-icon {
	display: grid;
	place-items: center;
	border-radius: 9999px;
	padding: 0.375rem 0.875rem;
	transition: background-color 0.3s;
}

.nav-active .nav-icon {
	background: var(--primary);
}

/* 下部ナビ */
.nav-bar {
	inset-inline: 1rem;
	inset-block-end: calc(0.75rem + env(safe-area-inset-bottom));
	margin-inline: auto;
	max-inline-size: 28rem;
	block-size: var(--nav-height);
	border-radius: 9999px;
}

.nav-active .nav-icon {
	animation: pop 0.5s var(--ease-out-expo);
}

@keyframes pop {
	40% {
		transform: scale(1.25);
	}
}
</style>
