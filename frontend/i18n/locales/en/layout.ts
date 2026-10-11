// レイアウトと同期状態の表示
import type ja from '../ja/layout';

export default {
	skipToContent: 'Skip to content',
	appName: 'Mecalo',
	nav: {
		label: 'Main menu',
		home: 'Home',
		history: 'History',
		foods: 'Foods',
		settings: 'Settings',
	},
	syncStatus: {
		offline: 'You are offline. Changes are saved on this device and synced when you reconnect',
		syncing: 'Syncing ({count})',
		pending: '{count} not synced',
		failed: '{count} changes could not be synced',
		retry: 'Retry',
		discard: 'Discard',
		discardConfirm: 'Delete this unsynced record?',
		failedTitle: 'Changes that could not be synced',
	},
} satisfies typeof ja;
