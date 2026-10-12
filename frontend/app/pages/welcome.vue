<script setup lang="ts">
definePageMeta({ layout: false });

const { me } = useAuth();
const { complete } = useOnboarding();
const curtain = useEntryCurtain();
const phase = ref<'story' | 'setup'>('story');

function toSetup(from: Element) {
	void curtain.play('setup', () => {
		phase.value = 'setup';
		window.scrollTo(0, 0);
	}, from);
}

async function finish(from?: Element) {
	if (me.value) complete(me.value.id);
	await curtain.play('onboarded', () => navigateTo('/'), from);
}
</script>

<template>
<main id="main">
	<Transition name="welcome" mode="out-in">
		<WelcomeStory v-if="phase === 'story'" @done="toSetup"/>
		<WelcomeSetup v-else @finished="finish"/>
	</Transition>
</main>
</template>

<style scoped>
.welcome-enter-active,
.welcome-leave-active {
	transition: opacity 0.5s, transform 0.5s var(--ease-out-expo);
}

.welcome-enter-from {
	opacity: 0;
	transform: translateY(12px);
}

.welcome-leave-to {
	opacity: 0;
}
</style>
