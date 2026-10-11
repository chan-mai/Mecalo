import { toast } from 'vue-sonner';
import { ApiError } from './api/client';

export function useErrorHandler() {
	const { t } = useI18n();
	return (err: unknown) => {
		if (err instanceof ApiError && err.status === 401) {
			void navigateTo('/login');
			return;
		}
		console.error(err);
		toast.error(err instanceof ApiError && err.problem?.detail ? err.problem.detail : t('errors.generic'));
	};
}
