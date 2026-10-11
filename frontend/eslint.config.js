import pluginMisskey from '@misskey-dev/eslint-plugin';
import tsParser from '@typescript-eslint/parser';
import pluginVue from 'eslint-plugin-vue';
import globals from 'globals';
import vueParser from 'vue-eslint-parser';

// 概ね@misskey-dev/eslint-plugin準拠
export default [
	{
		ignores: [
			'.nuxt/**',
			'.output/**',
			'dist/**',
			'node_modules/**',
			'app/types/api.gen.ts',
		],
	},
	...pluginMisskey.configs.recommended,
	{
		files: ['**/*.vue'],
		...pluginMisskey.configs.typescript,
	},
	...pluginVue.configs['flat/recommended'],
	{
		files: ['**/*.{js,ts,vue}'],
		languageOptions: {
			globals: {
				...globals.browser,
				...globals.node,
			},
			parser: vueParser,
			parserOptions: {
				extraFileExtensions: ['.vue'],
				parser: tsParser,
				projectService: {
					allowDefaultProject: ['eslint.config.js', 'vitest.config.js'],
				},
				sourceType: 'module',
				tsconfigRootDir: import.meta.dirname,
			},
		},
		rules: {
			// 未定義参照は型検査で検出, Nuxtのauto-import衝突
			'no-undef': 'off',
			'@stylistic/quotes': ['error', 'single', { avoidEscape: true }],
			'@typescript-eslint/no-unused-expressions': 'off',
			'import/consistent-type-specifier-style': ['error', 'prefer-top-level'],
			'id-denylist': [
				'warn',
				'window',
				'e',
				'close',
				'open',
				'fetch',
				'location',
				'document',
				'history',
				'scroll',
				'setTimeout',
				'setInterval',
				'clearTimeout',
				'clearInterval',
			],
			'no-shadow': ['warn'],
			'vue/block-order': ['error', { order: [['script:not([setup])', 'script[setup]'], 'template', 'style'] }],
			'vue/attributes-order': ['error', { alphabetical: false }],
			'vue/no-use-v-if-with-v-for': [
				'error',
				{ allowUsingIterationVar: false },
			],
			'vue/no-ref-as-operand': 'error',
			'vue/no-multi-spaces': ['error', { ignoreProperties: false }],
			'vue/no-v-html': 'warn',
			'vue/order-in-components': 'error',
			'vue/html-indent': [
				'warn',
				'tab',
				{
					attribute: 1,
					baseIndent: 0,
					closeBracket: 0,
					alignAttributesVertically: true,
					ignores: [],
				},
			],
			'vue/html-closing-bracket-spacing': [
				'warn',
				{ startTag: 'never', endTag: 'never', selfClosingTag: 'never' },
			],
			'vue/multi-word-component-names': 'warn',
			'vue/require-v-for-key': 'warn',
			'vue/no-unused-components': 'warn',
			'vue/no-unused-vars': 'warn',
			'vue/no-dupe-keys': 'warn',
			'vue/valid-v-for': 'warn',
			'vue/return-in-computed-property': 'warn',
			'vue/no-setup-props-reactivity-loss': 'warn',
			'vue/max-attributes-per-line': 'off',
			'vue/html-self-closing': [
				'error',
				{
					html: { void: 'any', normal: 'never', component: 'any' },
					svg: 'any',
					math: 'any',
				},
			],
			'vue/singleline-html-element-content-newline': 'off',
			'vue/v-on-event-hyphenation': ['error', 'never', { autofix: true }],
			'vue/attribute-hyphenation': ['error', 'never'],
			'vue/no-mutating-props': ['error', { shallowOnly: true }],
		},
	},
	{
		// Nuxt制約
		files: [
			'eslint.config.js',
			'vitest.config.js',
			'nuxt.config.ts',
			'server/**/*.ts',
			'app/middleware/**/*.ts',
			'app/plugins/**/*.ts',
			'i18n/**/*.ts',
		],
		rules: {
			'import/no-default-export': 'off',
		},
	},
	{
		// 翻訳キーはプロパティ名
		files: ['i18n/**/*.ts'],
		rules: {
			'id-denylist': 'off',
		},
	},
	{
		// Nuxt制約
		files: ['app/pages/**/*.vue', 'app/layouts/**/*.vue', 'app/app.vue'],
		rules: {
			'vue/multi-word-component-names': 'off',
		},
	},
];
