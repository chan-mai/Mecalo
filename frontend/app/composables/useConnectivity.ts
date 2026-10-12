import { createGlobalState, useOnline } from '@vueuse/core';
import { computed, ref } from 'vue';

// 端末の通信状態とサーバーへの到達可否
export const useConnectivity = createGlobalState(() => {
	const online = useOnline();
	// API応答の有無で更新
	const reachable = ref(true);
	const connected = computed(() => online.value && reachable.value);
	return { online, reachable, connected };
});
