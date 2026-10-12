<script setup lang="ts">
const { locale } = useI18n();
useHead({ htmlAttrs: { lang: locale } });
</script>

<template>
<AtmosphereBackground/>
<IntroLoader/>
<EntryCurtain/>
<NuxtLayout>
	<NuxtPage/>
</NuxtLayout>
<Toaster position="bottom-center" :offset="{ bottom: 'calc(6.5rem + env(safe-area-inset-bottom))' }"/>
<ConfirmHost/>
<NuxtPwaManifest/>
</template>
