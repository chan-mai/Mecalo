<script setup lang="ts">
const emit = defineEmits<{ found: [code: string] }>();
const { t } = useI18n();

const video = useTemplateRef<HTMLVideoElement>('video');
const manual = ref('');
const cameraAvailable = ref(true);
let stream: MediaStream | undefined;
let timer: ReturnType<typeof setInterval> | undefined;

onMounted(async () => {
	try {
		stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: 'environment' }, audio: false });
	} catch {
		cameraAvailable.value = false;
		return;
	}
	if (!video.value) return;
	video.value.srcObject = stream;
	await video.value.play();

	const detector = await createDetector();
	timer = setInterval(async () => {
		if (!video.value || video.value.readyState < 2) return;
		const codes = await detector.detect(video.value).catch(() => []);
		const hit = codes.find((c) => /^\d{8,14}$/.test(c.rawValue));
		if (hit) {
			stop();
			emit('found', hit.rawValue);
		}
	}, 300);
});

// 非対応ブラウザは同梱のwasm実装を使用
async function createDetector() {
	const formats = ['ean_13', 'ean_8', 'upc_a', 'upc_e'];
	if ('BarcodeDetector' in window) {
		const Native = (window as unknown as { BarcodeDetector: typeof import('barcode-detector/ponyfill').BarcodeDetector }).BarcodeDetector;
		const supported = await Native.getSupportedFormats();
		if (formats.some((f) => supported.includes(f as never))) return new Native({ formats: formats as never });
	}
	const [{ BarcodeDetector, prepareZXingModule }, { default: wasmUrl }] = await Promise.all([
		import('barcode-detector/ponyfill'),
		import('zxing-wasm/reader/zxing_reader.wasm?url'),
	]);
	prepareZXingModule({
		overrides: { locateFile: (path: string, prefix: string) => (path.endsWith('.wasm') ? wasmUrl : prefix + path) },
	});
	return new BarcodeDetector({ formats: formats as never });
}

function submit() {
	stop();
	emit('found', manual.value);
}

function stop() {
	clearInterval(timer);
	stream?.getTracks().forEach((track) => track.stop());
	stream = undefined;
}

onBeforeUnmount(stop);
</script>

<template>
<div>
	<div v-if="cameraAvailable" class="relative mb-3 aspect-video overflow-hidden rounded-2xl bg-black">
		<video ref="video" class="h-full w-full object-cover" muted playsinline></video>
		<div class="pointer-events-none absolute inset-x-8 top-1/2 h-0.5 -translate-y-1/2 bg-[var(--brand-secondary)]"></div>
	</div>
	<p v-else class="mb-2 text-sm text-muted">{{ t('components.barcodeScanner.unsupported') }}</p>
	<form class="flex gap-2" @submit.prevent="submit">
		<input v-model="manual" class="input" :aria-label="t('components.barcodeScanner.placeholder')" inputmode="numeric" pattern="\d{8,14}" :placeholder="t('components.barcodeScanner.placeholder')">
		<button class="btn btn-primary btn-icon !size-12" :disabled="!/^\d{8,14}$/.test(manual)" :aria-label="t('common.search')">
			<Icon name="lucide:search" size="20" aria-hidden="true"/>
		</button>
	</form>
</div>
</template>
