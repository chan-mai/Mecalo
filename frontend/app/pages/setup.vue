<script setup lang="ts">
definePageMeta({ layout: 'auth' });
const { t } = useI18n();
</script>

<template>
<SignUpForm :title="t('pages.auth.setup')" :hint="t('pages.auth.setupHint')" path="/auth/setup"/>
</template>
