import { defineConfig } from '@playwright/test'

const port = 3100

export default defineConfig({
  testDir: 'tests/e2e',
  timeout: 30_000,
  retries: 0,
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    serviceWorkers: 'block',
  },
  webServer: {
    command: 'node .output/server/index.mjs',
    port,
    reuseExistingServer: false,
    timeout: 60_000,
    env: {
      PORT: String(port),
      HOST: '127.0.0.1',
      NUXT_PUBLIC_API_BASE: 'http://127.0.0.1:3199',
      NUXT_PUBLIC_GOOGLE_CLIENT_ID: 'test-client-id',
    },
  },
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
