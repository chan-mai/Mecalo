<script setup lang="ts">
const { t } = useI18n();
</script>

<template>
<div class="relative flex min-h-dvh flex-col overflow-hidden px-6 pt-[calc(2.5rem+env(safe-area-inset-top))] pb-10">
	<DecorScene class="absolute inset-0"/>
	<header class="relative">
		<h1><img src="/logo.svg" :alt="t('layout.appName')" width="427" height="76" class="h-[clamp(2.75rem,14vw,4.5rem)] w-auto"></h1>
	</header>
	<main class="panel relative mx-auto mt-auto w-full max-w-sm p-6">
		<slot></slot>
	</main>
</div>
</template>
