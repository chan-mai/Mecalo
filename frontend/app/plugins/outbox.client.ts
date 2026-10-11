// 未送信操作の自動送信(再接続・画面復帰・30秒ごと)
export default defineNuxtPlugin(() => {
	const { me } = useAuth();
	const outbox = useOutbox();
	const visibility = useDocumentVisibility();

	async function sync() {
		if (!outbox.online.value) return;
		const sent = await outbox.flush();
		if (sent > 0) await refreshNuxtData();
	}

	watch(() => me.value?.id, () => void sync(), { immediate: true });
	watch(outbox.online, (online) => {
		if (online) void sync();
	});
	watch(visibility, (state) => {
		if (state === 'visible') void sync();
	});
	useIntervalFn(() => {
		if (outbox.pending.value.length > 0) void sync();
	}, 30_000);
});
