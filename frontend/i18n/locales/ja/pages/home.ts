// ホーム画面と記録・編集シート
export default {
	home: {
		title: '今日の記録',
		setupProfile: 'プロフィールと体重を登録すると、目標量と不足・過剰を表示します',
		setupProfileAction: 'プロフィールを設定',
		copied: '{count}件を記録しました',
		nothingToCopy: '前日の記録がありません',
		pendingNote: '未同期の記録は集計に含まれていません',
		energyCard: {
			eaten: '摂取',
			remaining: '残り',
			over: '超過',
			noTarget: '目標未設定',
		},
		attentionCard: {
			title: '気になる栄養素',
			noRecords: '食事を記録すると不足や過剰の栄養素が算出されます',
			allGood: '大きな偏りはありません',
			allNutrients: 'すべての栄養素を見る',
		},
		mealSlotCard: {
			copyPrevious: '前日と同じ',
			kcal: '{value} kcal',
			pending: '未同期',
			waiting: '同期待ち',
		},
		weightCard: {
			title: '体重',
		},
		recordSheet: {
			title: '{slot}に追加',
			added: '{name}を記録しました',
		},
		entrySheet: {
			title: '記録の編集',
			recalculate: '最新データで再計算',
			recalculateHint: '食品データの更新を反映します。通常は記録時の値を保持します',
			recalculated: '再計算しました',
			source: '出典',
			snapshot: '記録時の栄養値',
		},
		quantityForm: {
			submit: '記録する',
		},
	},
};
