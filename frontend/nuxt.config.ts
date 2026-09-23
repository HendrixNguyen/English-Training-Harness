export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  // Client-rendered PWA: the JWT lives in localStorage and every screen is
  // per-user, so there is nothing to render on a server (design §5).
  ssr: false,
  devtools: { enabled: false },
  modules: ['@pinia/nuxt', '@nuxtjs/tailwindcss', '@nuxt/eslint', '@vite-pwa/nuxt'],
  components: [{ path: '~/components', pathPrefix: false }],
  css: [
    '@fontsource-variable/fraunces/index.css',
    '@fontsource/source-sans-3/400.css',
    '@fontsource/source-sans-3/600.css',
    '~/assets/css/main.css',
  ],
  app: {
    head: {
      title: 'Học 30 phút',
      htmlAttrs: { lang: 'vi' },
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, viewport-fit=cover' },
        { name: 'theme-color', content: '#10B981' },
      ],
    },
  },
  // NUXT_PUBLIC_API_BASE, NUXT_PUBLIC_GOOGLE_CLIENT_ID, NUXT_PUBLIC_VAPID_PUBLIC_KEY,
  // NUXT_PUBLIC_STUB_ONBOARDING override these at runtime (Nuxt convention;
  // the idea's bare API_BASE/GOOGLE_CLIENT_ID/VAPID_PUBLIC_KEY names are the
  // same values under the NUXT_PUBLIC_ prefix).
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      googleClientId: '',
      vapidPublicKey: '',
      stubOnboarding: 'true',
    },
  },
  pwa: {
    strategies: 'injectManifest',
    srcDir: 'service-worker',
    filename: 'sw.ts',
    registerType: 'autoUpdate',
    injectRegister: 'auto',
    manifest: {
      name: 'Học 30 phút',
      short_name: 'Học30',
      description: 'Học tiếng Anh 30 phút mỗi ngày và nuôi một cái cây.',
      lang: 'vi',
      display: 'standalone',
      orientation: 'portrait',
      start_url: '/',
      scope: '/',
      theme_color: '#10B981',
      background_color: '#F8FAFC',
      icons: [
        { src: 'pwa-64x64.png', sizes: '64x64', type: 'image/png' },
        { src: 'pwa-192x192.png', sizes: '192x192', type: 'image/png' },
        { src: 'pwa-512x512.png', sizes: '512x512', type: 'image/png' },
        { src: 'maskable-icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
      ],
    },
    injectManifest: { globPatterns: ['**/*.{js,css,html,svg,png,woff2}'] },
    client: { installPrompt: true },
    devOptions: { enabled: false },
  },
  typescript: { strict: true },
})
