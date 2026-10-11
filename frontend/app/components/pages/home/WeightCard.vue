<script setup lang="ts">
import { toast } from 'vue-sonner';
import type { Schemas } from '~/composables/api/client';

const props = defineProps<{ date: string; latest?: Schemas['Weight'] }>();
const emit = defineEmits<{ saved: [] }>();
const { t } = useI18n();
const onError = useErrorHandler();
const outbox = useOutbox();
const uid = useId();

const kg = ref<number>();
const saving = ref(false);

watch(() => [props.date, props.latest], () => {
	kg.value = props.latest?.date === props.date ? props.latest.kg : undefined;
}, { immediate: true });

async function save() {
	if (!kg.value) return;
	saving.value = true;
	try {
		const res = await outbox.run({ kind: 'saveWeight', date: props.date, body: { kg: kg.value } });
		toast(res.queued ? t('common.messages.savedOffline') : t('common.messages.weightSaved'));
		emit('saved');
	} catch (err) {
		onError(err);
	} finally {
		saving.value = false;
	}
}
</script>

<template>
<section v-reveal class="panel p-6" :aria-labelledby="`${uid}-title`">
	<SectionHeading icon="lucide:scale" :eyebrow="t('common.eyebrow.weight')" :title="t('pages.home.weightCard.title')" :headingId="`${uid}-title`">
		<p v-if="latest" class="shrink-0 text-xs text-muted">{{ latest.date }}: <span class="numeric">{{ latest.kg }}</span> kg</p>
	</SectionHeading>
	<form class="mt-4 flex items-center gap-2" @submit.prevent="save">
		<input v-model.number="kg" type="number" inputmode="decimal" step="0.1" min="1" class="input flex-1 text-right" :aria-label="t('terms.weightKg')">
		<span class="eyebrow" aria-hidden="true">kg</span>
		<button class="btn btn-primary" :disabled="!kg || saving">{{ t('common.save') }}</button>
	</form>
</section>
</template>
