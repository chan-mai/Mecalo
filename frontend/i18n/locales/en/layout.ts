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
		admin: 'Admin',
	},
	syncStatus: {
		offline: 'Failed to connect to the server. Your changes have been saved to your device and will be synced when the connection is restored.',
		syncing: 'Syncing ({count})',
		pending: '{count} not synced',
		failed: '{count} changes could not be synced',
		retry: 'Retry',
		discard: 'Discard',
		discardConfirm: 'Delete this unsynced record?',
		failedTitle: 'Changes that could not be synced',
	},
} satisfies typeof ja;
