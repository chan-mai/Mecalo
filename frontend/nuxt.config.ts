import tailwindcss from '@tailwindcss/vite';

export default defineNuxtConfig({
	compatibilityDate: '2026-10-01',
	ssr: false,
	devtools: { enabled: false },
	telemetry: false,

	modules: ['@nuxtjs/i18n', '@vite-pwa/nuxt', '@vueuse/nuxt', 'vue-sonner/nuxt', '@nuxt/icon', '@nuxt/fonts'],
	css: ['~/assets/css/main.css', 'vue-sonner/style.css'],
	fonts: {
		families: [
			{ name: 'Unbounded', provider: 'google', weights: ['300 700'] },
			{ name: 'IBM Plex Mono', provider: 'google', weights: [400], subsets: ['latin'] },
			{ name: 'Zen Kaku Gothic New', provider: 'google', weights: [400, 700], subsets: ['latin', 'japanese'] },
			{ name: 'Shippori Mincho', provider: 'google', weights: [500], subsets: ['latin', 'japanese'] },
		],
		defaults: { styles: ['normal'] },
	},

	components: [{ path: '~/components', pathPrefix: false }],

	runtimeConfig: {
		mecaloBeApiUri: 'http://localhost:8080',
	},

	app: {
		pageTransition: { name: 'page', mode: 'out-in' },
		head: {
			htmlAttrs: { lang: 'ja' },
			title: 'Mecalo',
			meta: [
				{ name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
				{ name: 'theme-color', content: '#F6BED6' },
				{ name: 'description', content: 'Self-hosted nutrition tracker' },
			],
			link: [
				{ rel: 'icon', href: '/favicon.ico' },
				{ rel: 'apple-touch-icon', href: '/pwa-192x192.png' },
			],
		},
	},

	// アイコンはビルドに同梱し外部APIへ問い合わせない
	icon: {
		serverBundle: { collections: ['lucide'] },
		clientBundle: {
			scan: true,
			// TS参照分
			icons: [
				'lucide:chevrons-down',
				'lucide:chevron-down',
				'lucide:check',
				'lucide:chevron-up',
				'lucide:chevrons-up',
				'lucide:sunrise',
				'lucide:sun',
				'lucide:moon',
				'lucide:cookie',
			],
		},
		fallbackToApi: false,
	},

	i18n: {
		defaultLocale: 'ja',
		strategy: 'no_prefix',
		locales: [
			{ code: 'ja', language: 'ja-JP', file: 'ja/index.ts', name: '日本語' },
			{ code: 'en', language: 'en-US', file: 'en/index.ts', name: 'English' },
		],
		detectBrowserLanguage: { useCookie: true, cookieKey: 'mecalo_locale', fallbackLocale: 'ja' },
	},

	pwa: {
		registerType: 'autoUpdate',
		manifest: {
			name: 'Mecalo',
			short_name: 'Mecalo',
			description: 'Self-hosted nutrition tracker',
			lang: 'ja',
			start_url: '/',
			display: 'standalone',
			background_color: '#FFF9FC',
			theme_color: '#F6BED6',
			icons: [
				{ src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png' },
				{ src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png' },
				{ src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
			],
		},
		workbox: {
			navigateFallback: '/',
			navigateFallbackDenylist: [/^\/api\//, /^\/mcp$/, /^\/\.well-known\//],
			globPatterns: ['**/*.{js,css,html,png,svg,ico,glb,wasm,woff2}'],
			maximumFileSizeToCacheInBytes: 4 * 1024 * 1024,
			// オフライン表示用
			runtimeCaching: [{
				urlPattern: ({ url, sameOrigin }) => sameOrigin && url.pathname.startsWith('/api/') && !url.pathname.startsWith('/api/auth/') && !url.pathname.startsWith('/api/export/'),
				method: 'GET',
				handler: 'NetworkFirst',
				options: {
					cacheName: 'mecalo-api',
					networkTimeoutSeconds: 5,
					expiration: { maxEntries: 300, maxAgeSeconds: 14 * 24 * 60 * 60 },
					cacheableResponse: { statuses: [200] },
					plugins: [{
						fetchDidSucceed: async ({ response }) => {
							if (response.status >= 500) throw new Error('backend unavailable');
							return response;
						},
					}],
				},
			}],
		},
		client: { installPrompt: true },
		devOptions: { enabled: false },
	},

	nitro: {
		prerender: { routes: ['/'] },
	},

	vite: {
		plugins: [tailwindcss()],
	},

	typescript: {
		strict: true,
	},
});
