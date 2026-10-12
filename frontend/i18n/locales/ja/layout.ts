// レイアウトと同期状態表示
export default {
	skipToContent: '本文へ移動',
	appName: 'Mecalo',
	nav: {
		label: 'メインメニュー',
		home: 'ホーム',
		history: '履歴',
		foods: '食品',
		settings: '設定',
		admin: '管理',
	},
	syncStatus: {
		offline: 'オフラインです。入力は端末に保存し、接続時に同期します',
		syncing: '同期中 ({count}件)',
		pending: '未同期 {count}件',
		failed: '同期できなかった操作 {count}件',
		retry: '再試行',
		discard: '破棄',
		discardConfirm: 'この未同期の記録を削除しますか?',
		failedTitle: '同期できなかった操作',
	},
};
