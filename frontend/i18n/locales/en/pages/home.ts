// ホーム画面と記録・編集シート
import type ja from '../../ja/pages/home';

export default {
	home: {
		title: "Today's record",
		setupProfile: 'Add your profile and weight to see targets and deficiencies',
		setupProfileAction: 'Set up profile',
		copied: 'Recorded {count} items',
		nothingToCopy: 'Nothing recorded on the previous day',
		pendingNote: 'Unsynced records are not included in the totals',
		energyCard: {
			eaten: 'Eaten',
			remaining: 'Remaining',
			over: 'Over',
			noTarget: 'No target',
		},
		attentionCard: {
			title: 'Needs attention',
			noRecords: 'Recording your meals calculates your nutrient deficiencies and excesses.',
			allGood: 'No major imbalance',
			allNutrients: 'See all nutrients',
		},
		mealSlotCard: {
			copyPrevious: 'Same as yesterday',
			kcal: '{value} kcal',
			pending: 'Not synced',
			waiting: 'Waiting to sync',
		},
		weightCard: {
			title: 'Weight',
		},
		recordSheet: {
			title: 'Add to {slot}',
			added: 'Recorded {name}',
		},
		entrySheet: {
			title: 'Edit record',
			recalculate: 'Recalculate with latest data',
			recalculateHint: 'Applies updates to the food data. Records normally keep the values from when they were added',
			recalculated: 'Recalculated',
			source: 'Source',
			snapshot: 'Nutrients at recording',
		},
		quantityForm: {
			submit: 'Record',
		},
	},
} satisfies typeof ja;
