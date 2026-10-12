export interface ConfirmOptions {
	title: string;
	message?: string;
	confirmLabel: string;
	danger?: boolean;
}

interface ConfirmRequest extends ConfirmOptions {
	resolve: (ok: boolean) => void;
}

const current = shallowRef<ConfirmRequest>();

export function useConfirm() {
	return (options: ConfirmOptions) => new Promise<boolean>((resolve) => {
		current.value?.resolve(false);
		current.value = { ...options, resolve };
	});
}

// 削除の確認
export function useConfirmDelete() {
	const confirm = useConfirm();
	const { t } = useI18n();
	return () => confirm({ title: t('common.confirmDelete'), confirmLabel: t('common.delete'), danger: true });
}

export function useConfirmRequest() {
	return current;
}
