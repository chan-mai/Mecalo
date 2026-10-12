<script setup lang="ts">
// 画面全体のコンポジット
const props = defineProps<{ progress: number }>();

// 擬似グレインっぽいの
const bokeh = Array.from({ length: 14 }, (_, i) => ({
	x: (i * 37) % 100,
	y: (i * 61) % 100,
	size: 18 + ((i * 29) % 70),
	depth: 0.6 + ((i * 13) % 10) / 10,
	tone: ['--brand-primary', '--brand-secondary', '--brand-tertiary', '--background'][i % 4],
}));
const grain = `url("data:image/svg+xml,${encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="220" height="220"><filter id="n"><feTurbulence type="fractalNoise" baseFrequency="0.85" numOctaves="3" stitchTiles="stitch"/><feColorMatrix values="0 0 0 0 0.5 0 0 0 0 0.45 0 0 0 0 0.6 0 0 0 1.4 -0.2"/></filter><rect width="100%" height="100%" filter="url(#n)"/></svg>')}")`;
</script>

<template>
<div class="fx" aria-hidden="true">
	<!-- ディフュージョン -->
	<div class="fx-lens"></div>
	<!-- 光源補正 -->
	<div class="fx-leak fx-leak-a"></div>
	<div class="fx-leak fx-leak-b"></div>
	<div class="fx-vignette"></div>
	<!-- フレア -->
	<span
		v-for="(b, i) in bokeh"
		:key="i"
		class="fx-bokeh"
		:style="{
			left: `${b.x}%`,
			top: `${b.y}%`,
			width: `${b.size}px`,
			height: `${b.size}px`,
			background: `var(${b.tone})`,
			transform: `translate3d(0, ${-props.progress * 18 * b.depth}vh, 0)`,
			filter: `blur(${4 + b.depth * 6}px)`,
		}"
	></span>
	<div class="fx-grain" :style="{ backgroundImage: grain }"></div>
	<!-- 色収差 -->
	<svg class="absolute h-0 w-0">
		<filter id="welcome-aberration" color-interpolation-filters="sRGB">
			<feColorMatrix in="SourceGraphic" type="matrix" values="1 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1 0" result="r"/>
			<feOffset in="r" dx="-0.8" dy="0" result="ro"/>
			<feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 1 0 0 0  0 0 0 0 0  0 0 0 1 0" result="g"/>
			<feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 1 0 0  0 0 0 1 0" result="b"/>
			<feOffset in="b" dx="0.8" dy="0" result="bo"/>
			<feBlend in="ro" in2="g" mode="screen" result="rg"/>
			<feBlend in="rg" in2="bo" mode="screen"/>
		</filter>
	</svg>
</div>
</template>

<style scoped>
.fx {
	position: fixed;
	inset: 0;
	z-index: 20;
	overflow: hidden;
	pointer-events: none;
}

.fx > div {
	position: absolute;
}

.fx-lens {
	inset: 0;
	backdrop-filter: blur(3px);
	mask-image: radial-gradient(ellipse 75% 70% at 50% 50%, transparent 55%, black 100%);
}

.fx-leak {
	inline-size: 70vmax;
	block-size: 70vmax;
	border-radius: 9999px;
	filter: blur(70px);
	mix-blend-mode: screen;
	opacity: 0.55;
}

.fx-leak-a {
	inset-block-start: -35vmax;
	inset-inline-start: -25vmax;
	background: radial-gradient(circle, var(--brand-secondary), transparent 65%);
	animation: leak-a 18s ease-in-out infinite alternate;
}

.fx-leak-b {
	inset-block-end: -40vmax;
	inset-inline-end: -30vmax;
	background: radial-gradient(circle, var(--brand-primary), transparent 65%);
	animation: leak-b 22s ease-in-out infinite alternate;
}

.fx-bokeh {
	position: absolute;
	border-radius: 9999px;
	opacity: 0.55;
	mix-blend-mode: screen;
	transition: transform 0.2s linear;
}

.fx-vignette {
	inset: 0;
	background: radial-gradient(ellipse 80% 75% at 50% 50%, transparent 55%, rgb(109 78 172 / 0.22) 100%);
	mix-blend-mode: multiply;
}

.fx-grain {
	inset: -50%;
	opacity: 0.32;
	mix-blend-mode: soft-light;
	animation: grain 0.6s steps(6) infinite;
}

@keyframes leak-a {
	to {
		transform: translate(18vmax, 12vmax) scale(1.15);
	}
}

@keyframes leak-b {
	to {
		transform: translate(-16vmax, -10vmax) scale(1.2);
	}
}

@keyframes grain {
	0% {
		transform: translate(0, 0);
	}
	20% {
		transform: translate(-3%, 2%);
	}
	40% {
		transform: translate(2%, -4%);
	}
	60% {
		transform: translate(-4%, -1%);
	}
	80% {
		transform: translate(3%, 3%);
	}
	100% {
		transform: translate(-1%, 4%);
	}
}

@media (prefers-reduced-motion: reduce) {
	.fx-leak,
	.fx-grain {
		animation: none;
	}
}
</style>
