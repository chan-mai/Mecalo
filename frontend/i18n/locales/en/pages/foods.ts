// 食品・レシピの画面
import type ja from '../../ja/pages/foods';

export default {
	foods: {
		mine: 'My foods',
		empty: 'No foods yet',
		title: 'Foods',
		noFavorites: 'No favorite foods yet',
		form: {
			brand: 'Brand',
			barcode: 'Barcode',
			portions: 'Units',
			portionUnit: 'Unit (e.g. piece, slice, serving)',
			portionGrams: 'Grams',
			addPortion: 'Add unit',
			nutrientsPer100g: 'Nutrients per 100 g',
			blankIsUnknown: 'Blank fields are treated as not measured',
			moreNutrients: 'Vitamins and minerals',
			aliasesHint: 'Other names used for search, separated by commas',
		},
	},
	food: {
		favorite: 'Favorite',
		revisions: 'Revisions',
		revision: 'Revision {number}',
		source: 'Source',
		deletedNote: 'This food has been deleted',
	},
	foodEdit: {
		updateNote: 'Saving creates a new revision. Past records keep their values',
	},
	recipe: {
		perServing: '{grams} g per serving',
		updateNote: 'Nutrients are recalculated from the latest ingredient data on save',
		form: {
			title: 'Recipe',
			addIngredient: 'Add ingredient',
			cookedGrams: 'Cooked weight (g)',
			cookedGramsHint: 'Leave blank to use the total ingredient weight',
			servings: 'Servings',
		},
	},
} satisfies typeof ja;
