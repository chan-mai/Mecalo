<script setup lang="ts">
const { t } = useI18n();
</script>

<template>
<div class="relative flex min-h-dvh flex-col overflow-hidden px-6 pt-[calc(2.5rem+env(safe-area-inset-top))] pb-10">
	<DecorScene class="absolute inset-0"/>
	<header class="relative">
		<h1 class="display text-[clamp(3rem,16vw,5.5rem)]">{{ t('layout.appName').toUpperCase() }}</h1>
	</header>
	<main class="panel relative mx-auto mt-auto w-full max-w-sm p-6">
		<slot></slot>
	</main>
</div>
</template>
