// 食品・レシピの画面
export default {
	foods: {
		mine: '登録した食品',
		empty: '登録した食品はありません',
		title: '食品',
		noFavorites: 'お気に入りに登録した食品はありません',
		form: {
			brand: 'ブランド・メーカー',
			barcode: 'バーコード',
			portions: '単位',
			portionUnit: '単位名(例: 個, 枚, 人前)',
			portionGrams: 'グラム数',
			addPortion: '単位を追加',
			nutrientsPer100g: '100gあたりの栄養値',
			blankIsUnknown: '空欄は未計測として扱います',
			moreNutrients: 'ビタミン・ミネラル',
			aliasesHint: '検索対象に加える呼び名, 複数はカンマ区切り',
		},
	},
	food: {
		favorite: 'お気に入り',
		revisions: '変更履歴',
		revision: '第{number}版',
		source: '出典',
		deletedNote: 'この食品は削除されています',
	},
	foodEdit: {
		updateNote: '保存すると新しい版として記録されます。過去の記録の栄養値は変わりません',
	},
	recipe: {
		perServing: '1人前 {grams}g',
		updateNote: '保存時に材料の最新データで再計算します',
		form: {
			title: 'レシピ',
			addIngredient: '材料を追加',
			cookedGrams: '出来上がりの総重量(g)',
			cookedGramsHint: '空欄の場合は材料の合計重量を使います',
			servings: '何人前',
		},
	},
};
