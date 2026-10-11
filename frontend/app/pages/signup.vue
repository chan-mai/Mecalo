<script setup lang="ts">
definePageMeta({ layout: 'auth' });
const { t } = useI18n();
</script>

<template>
<SignUpForm :title="t('pages.auth.signup')" path="/auth/signup"/>
</template>
