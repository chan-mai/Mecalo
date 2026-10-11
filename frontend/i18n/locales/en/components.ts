// 複数画面で使う部品
import type ja from '../ja/components';

export default {
	foodPicker: {
		clear: 'Clear search',
		searchPlaceholder: 'Search foods',
		recent: 'Recent',
		frequent: 'Frequent',
		history: 'Searches',
		noResults: 'No results',
		createFood: 'Create a food',
		scan: 'Barcode',
		barcodeNotFound: 'No matching food',
		empty: 'Nothing recorded yet. Search to add your first food',
	},
	barcodeScanner: {
		title: 'Scan a barcode',
		unsupported: 'Barcode scanning is not supported on this device. Enter the number instead',
		placeholder: 'Barcode number',
	},
	dropzone: {
		release: 'Release to import',
		or: 'or click to choose',
	},
} satisfies typeof ja;
