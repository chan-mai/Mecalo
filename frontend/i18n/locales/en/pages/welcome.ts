// 初回ログイン時の案内と初期設定
import type ja from '../../ja/pages/welcome';

export default {
	welcome: {
		skip: 'Skip',
		scroll: 'SCROLL',
		start: 'Get started',
		chapters: {
			0: { eyebrow: 'RECORD', line: 'I can\'t remember yesterday\'s lunch anymore', desc: 'If you record what you eat when you eat it, you won\'t have to remember it later. Try recording what you ate and how much in the breakfast, lunch, dinner and snack slots. Barcode scanning is also supported.' },
			1: { eyebrow: 'MEALS', line: 'Monday or Friday it\'s probably the same bread for breakfast', desc: 'When you open a record, the foods you ate before and your favorites are shown. You can also do things like copying yesterday\'s breakfast straight into today\'s.' },
			2: { eyebrow: 'UNKNOWN', line: 'The boxed lunch label doesn\'t show iron', desc: 'If the unlisted iron were added up as 0 mg, the data would make it look like you are short on iron. Mecalo excludes unknown values from the total and aggregates each nutrient by the weight of the foods whose values are known.' },
			3: { eyebrow: 'HISTORY', line: 'Even when the food tables are revised', desc: 'Each record keeps the values from the time it was logged. So even if the food tables are updated or you edit a food you registered, past values are not affected. (*You can also update them explicitly.)' },
			4: { eyebrow: 'TARGET', line: 'The same set meal is too much for some and not enough for others', desc: 'Based on sex, age, height, weight and usual activity level, daily targets for energy, iron, calcium, vitamins and more are calculated following the Dietary Reference Intakes for Japanese (Ministry of Health, Labour and Welfare).' },
		},
		setup: {
			eyebrow: 'SETUP',
			greeting: 'Nice to meet you, {name}. To calculate your targets, please tell me a few things.',
			questions: {
				sex: 'What is your sex?',
				menstruating: 'Do you menstruate? (Needed to calculate your iron target)',
				birthDate: 'When were you born?',
				age: 'So you are {age} years old. Is that right?',
				height: 'How tall are you in cm?',
				weight: 'What is your current weight? You can leave body fat empty.',
				activity: 'Choose the activity level closest to your daily life.',
				goal: 'What is your goal?',
			},
			options: {
				yes: 'Yes',
				no: 'No',
				correct: 'Yes',
				fix: 'Fix my birth date',
			},
			later: 'Log it later',
			next: 'Next',
			edit: 'Change answer',
			typing: 'Typing',
			saving: 'Saving',
			height: 'Height',
			weight: 'Weight',
			bodyFat: 'Body fat',
			done: 'Thank you. Your targets are ready.',
			summary: 'Your daily targets',
			perDay: 'day',
			macros: {
				protein: 'Protein',
				fat: 'Fat',
				carbohydrate: 'Carbohydrate',
			},
			note: 'You can change your targets anytime in Settings.',
			finish: 'Start recording',
		},
	},
} satisfies typeof ja;
