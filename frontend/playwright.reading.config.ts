import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: 'tests/e2e',
  testMatch: 'reading.spec.ts',
  timeout: 180_000,
  workers: 1,
  retries: 0,
  outputDir: '/workspace/cloud-setup/reading-flow/task-5-logs/playwright',
  use: {
    baseURL: 'http://127.0.0.1:3100',
    viewport: { width: 390, height: 844 },
    serviceWorkers: 'allow',
    launchOptions: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH
      ? { executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH } : {},
  },
  webServer: [
    {
      command: 'node tests/e2e/helpers/reading-server.mjs',
      url: 'http://127.0.0.1:3299/ready',
      reuseExistingServer: false,
      timeout: 90_000,
      gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
    },
    {
      command: 'node .output/server/index.mjs',
      port: 3100,
      reuseExistingServer: false,
      env: {
        PORT: '3100', HOST: '127.0.0.1',
        NUXT_PUBLIC_API_BASE: 'http://127.0.0.1:3199',
        NUXT_PUBLIC_GOOGLE_CLIENT_ID: 'test-client-id',
      },
    },
  ],
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
