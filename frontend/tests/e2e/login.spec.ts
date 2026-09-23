import { expect, test } from '@playwright/test'

const SIGN_IN = {
  access_token: 'jwt-test',
  token_type: 'Bearer',
  expires_in: 86400,
  user: { id: 'u1', email: 'user@example.com', full_name: 'Nguyen Hendrix', cefr_current: 'B1' },
}
const DAILY = {
  date: '2026-09-23',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 1200,
  is_target_met: false,
  tasks: [
    { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng Email Công việc', duration_minutes: 10, is_completed: true, content_json: {} },
    { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu Mẫu Thư Thương mại', duration_minutes: 10, is_completed: false, content_json: {} },
    { id: 'ex-3', task_type: 'practice', title: 'Viết Phản hồi Khách hàng', duration_minutes: 10, is_completed: false, content_json: {} },
  ],
}
const PET = { plant_name: 'My Green Buddy', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-22T13:00:00Z' }

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/**', async (route) => {
    const path = new URL(route.request().url()).pathname
    const hit = path.endsWith('/auth/google') ? SIGN_IN : path.endsWith('/quests/daily') ? DAILY : path.endsWith('/pet/status') ? PET : null
    await route.fulfill({
      status: hit ? 200 : 404,
      contentType: 'application/json',
      body: JSON.stringify(hit ?? { error: 'not_stubbed' }),
    })
  })
  await page.route('https://accounts.google.com/**', route =>
    route.fulfill({ status: 200, contentType: 'text/html', body: '<title>google-stub</title>' }))
})

test('/login renders and sends the user to Google with the backend scope list', async ({ page }) => {
  await page.goto('/login')
  await expect(page.getByRole('heading', { name: /Chào mừng bạn/ })).toBeVisible()
  await page.getByRole('button', { name: 'Đăng nhập bằng Google' }).click()
  await page.waitForURL(/accounts\.google\.com/)
  const u = new URL(page.url())
  expect(u.searchParams.get('access_type')).toBe('offline')
  expect(u.searchParams.get('prompt')).toBe('consent')
  expect(u.searchParams.get('scope')).toContain('https://www.googleapis.com/auth/calendar.events')
  expect(u.searchParams.get('scope')).toContain('https://www.googleapis.com/auth/tasks')
  expect(u.searchParams.get('redirect_uri')).toBe('http://127.0.0.1:3100/login')
})

test('a guarded route without a token redirects to /login', async ({ page }) => {
  await page.goto('/')
  await page.waitForURL(/\/login$/)
  await expect(page.getByRole('button', { name: 'Đăng nhập bằng Google' })).toBeVisible()
})

test('the Google callback exchanges the code (§6.1 shape) and lands on the dashboard', async ({ page }) => {
  await page.addInitScript(() => sessionStorage.setItem('aelp.oauth_state', 'state-1'))
  await page.goto('/login?code=code-1&state=state-1')
  await page.waitForURL(/127\.0\.0\.1:3100\/$/)
  await expect(page.getByText('20 / 30 phút')).toBeVisible()
  await expect(page.getByText(/Streak: 5 ngày/)).toBeVisible()
  await expect(page.getByText('Đọc hiểu Mẫu Thư Thương mại')).toBeVisible()
  await expect(page.getByRole('link', { name: /Học: Đọc hiểu/ })).toBeVisible()
})
