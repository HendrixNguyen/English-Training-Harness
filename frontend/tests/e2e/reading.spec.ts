import { expect, test } from '@playwright/test'
import type { APIRequestContext, Page } from '@playwright/test'
import { writeFile } from 'node:fs/promises'

const fixtureBase = 'http://127.0.0.1:3299'
interface Fixture {
  exercise: string
  passage: string
  questions: { id: string }[]
  auth: { accessToken: string, expiresAt: number, user: { id: string } }
}
async function seed(page: Page, request: APIRequestContext, failures = 0): Promise<Fixture> {
  expect((await request.post(`${fixtureBase}/control?failures=${failures}`)).ok()).toBe(true)
  const response = await request.post(`${fixtureBase}/fixture`)
  expect(response.ok()).toBe(true)
  const fixture = await response.json() as Fixture
  // Browser storage uses the same persisted shape as signIn; backend still
  // verifies the issued HS256 token against the real Redis session.
  await page.addInitScript(auth => localStorage.setItem('aelp.auth', JSON.stringify(auth)), fixture.auth)
  return fixture
}
async function evidence(request: APIRequestContext, exercise: string) {
  const response = await request.get(`${fixtureBase}/evidence?exercise=${exercise}`)
  expect(response.ok()).toBe(true)
  return response.json()
}
const passageRegion = (page: Page) => page.getByRole('region', { name: 'Đoạn văn', exact: true })

test('real API: passage, Vietnamese gloss, answers, durable completion and cached reuse', async ({ page, request }) => {
  const fixture = await seed(page, request)
  expect((await request.get(`http://127.0.0.1:3199/api/v1/quests/exercises/${fixture.exercise}`)).status()).toBe(401)
  await page.goto(`/learn/${fixture.exercise}`)
  await expect(page.getByLabel('Đang tải')).toBeVisible()
  await expect(page.getByRole('radiogroup')).toHaveCount(0)
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  const persisted = await evidence(request, fixture.exercise)
  expect(persisted.schema).toBe(1)
  expect(persisted.content.passage).toBe(fixture.passage)
  expect(persisted.calls).toBe(1)

  const term = page.getByRole('button', { name: 'Giải nghĩa Negotiated' })
  await term.focus()
  await page.keyboard.press('Enter')
  const card = page.getByRole('region', { name: 'Giải nghĩa', exact: true })
  await expect(card).toContainText('đàm phán, thương lượng')
  await expect(card).toContainText('To discuss something to reach an agreement.')
  await expect(card).toHaveAttribute('aria-live', 'polite')
  await expect(term).toHaveAttribute('aria-expanded', 'true')
  expect(await term.getAttribute('aria-controls')).toBe(await card.getAttribute('id'))
  const countdown = page.getByText('Thời gian:').locator('..')
  const before = await countdown.innerText()
  await expect.poll(() => countdown.innerText(), { timeout: 4000 }).not.toBe(before)
  await page.screenshot({ path: '/tmp/reading-flow-task4-gloss-390x844.png' })
  await page.keyboard.press('Escape')
  await expect(card).toHaveCount(0)
  await expect(term).toBeFocused()
  await term.click()
  await page.getByRole('button', { name: 'Đóng giải nghĩa' }).click()
  await expect(term).toBeFocused()

  // Seed a resumed one-minute anchor, then keep using the normal wall clock.
  // This gives a nonzero durable minute assertion without a minute-long sleep.
  await page.evaluate(exercise => {
    const timers = JSON.parse(localStorage.getItem('aelp.timers')!)
    timers[exercise].startedAt = Date.now() - 65_000
    localStorage.setItem('aelp.timers', JSON.stringify(timers))
  }, fixture.exercise)
  await page.reload()
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  expect((await evidence(request, fixture.exercise)).calls).toBe(1)
  // Call the authenticated endpoint too: persisted cache returns the same DTO
  // even if the browser's daily/content caches have already avoided that GET.
  const cached = await request.get(`http://127.0.0.1:3199/api/v1/quests/exercises/${fixture.exercise}`, { headers: { Authorization: `Bearer ${fixture.auth.accessToken}` } })
  expect(cached.ok()).toBe(true)
  expect((await cached.json()).content_json.content.passage).toBe(fixture.passage)
  expect((await evidence(request, fixture.exercise)).calls).toBe(1)

  await page.getByRole('button', { name: 'Đọc xong →' }).click()
  await expect(page.getByText('Câu 1 / 3', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Xem lại đoạn' }).click()
  await expect(page.getByRole('button', { name: 'Ẩn đoạn văn' })).toHaveAttribute('aria-expanded', 'true')
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  await term.click()
  await page.getByRole('radio').first().click()
  await page.getByRole('button', { name: 'Gửi đáp án' }).click()
  await expect(page.getByText('Câu 2 / 3', { exact: true })).toBeVisible()
  await expect(card).toHaveCount(0)
  await expect(passageRegion(page)).toBeVisible()
  await term.click()
  await page.getByRole('button', { name: 'Ẩn đoạn văn' }).click()
  await expect(card).toHaveCount(0)
  for (let question = 2; question <= 3; question++) {
    await expect(page.getByText(`Câu ${question} / 3`, { exact: true })).toBeVisible()
    await page.getByRole('radio').first().click()
    await page.getByRole('button', { name: 'Gửi đáp án' }).click()
  }
  const posted = page.waitForRequest(req => req.url().endsWith('/quests/progress') && req.method() === 'POST')
  await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click()
  const body = (await posted).postDataJSON()
  expect(body.exercise_id).toBe(fixture.exercise)
  expect(body.user_answers).toEqual({ q1: 'A', q2: 'A', q3: 'A' })
  expect(body.duration_seconds).toBeGreaterThanOrEqual(65)
  expect(body.duration_seconds).toBeLessThan(110)
  await page.waitForURL('/')
  const saved = await evidence(request, fixture.exercise)
  expect(saved.completed).toBe(true)
  expect(saved.seconds).toBe(body.duration_seconds)
  expect(saved.minutes).toBe(Math.floor(body.duration_seconds / 60))
  expect(saved.calls).toBe(1)
  const proof = JSON.stringify({ posted: body, saved }, null, 2)
  await writeFile('/tmp/reading-flow-task4-db-evidence.json', proof)
  await test.info().attach('database-and-redis-evidence', { body: proof, contentType: 'application/json' })
  await page.goto(`/learn/${fixture.exercise}`)
  await page.reload()
  await expect(page.getByText('✓ Đã hoàn thành', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Đã hoàn thành', exact: true })).toBeDisabled()
  expect((await evidence(request, fixture.exercise)).calls).toBe(1)
})

test('real API generation failure leaves outline untouched and retry persists content', async ({ page, request }) => {
  const fixture = await seed(page, request, 1)
  const failure = page.waitForResponse(res => res.url().endsWith(`/exercises/${fixture.exercise}`))
  await page.goto(`/learn/${fixture.exercise}`)
  expect((await failure).status()).toBe(502)
  await expect(page.getByText('Chưa tải được nội dung học. Thử lại.', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toBeDisabled()
  const failed = await evidence(request, fixture.exercise)
  expect(failed.schema).toBeNull()
  expect(failed.completed).toBe(false)
  expect(failed.calls).toBe(1)
  expect(await page.evaluate(exercise => JSON.parse(localStorage.getItem('aelp.timers') ?? '{}')[exercise], fixture.exercise)).toBeUndefined()
  await page.getByRole('button', { name: 'Thử lại' }).click()
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  const retried = await evidence(request, fixture.exercise)
  expect(retried.schema).toBe(1)
  expect(retried.calls).toBe(2)
})

test('real service worker daily cache and typed content support offline app re-entry', async ({ page, context, request }) => {
  const fixture = await seed(page, request)
  await page.goto(`/learn/${fixture.exercise}`)
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  await page.evaluate(async () => { await navigator.serviceWorker.ready })
  // A controlled reload populates the real NetworkFirst daily cache.
  await page.reload()
  await expect.poll(() => page.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true)
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  await expect.poll(() => page.evaluate(async () => {
    const cache = await caches.open('api-state')
    return (await cache.keys()).some(req => req.url.endsWith('/quests/daily'))
  })).toBe(true)
  await page.getByRole('link', { name: '‹ Quay lại' }).click()
  await page.waitForURL('/')
  await context.setOffline(true)
  const offlineDaily = await page.evaluate(async token => {
    const response = await fetch('http://127.0.0.1:3199/api/v1/quests/daily', { headers: { Authorization: `Bearer ${token}` } })
    return { status: response.status, daily: await response.json() }
  }, fixture.auth.accessToken)
  expect(offlineDaily.status).toBe(200)
  expect(offlineDaily.daily.tasks.find((task: { id: string }) => task.id === fixture.exercise)).toBeTruthy()
  await page.getByRole('link', { name: /Học: Community library/ }).click()
  await expect(passageRegion(page)).toHaveText(fixture.passage)
  await page.getByRole('button', { name: 'Giải nghĩa Negotiated' }).click()
  await expect(page.getByRole('region', { name: 'Giải nghĩa', exact: true })).toContainText('đàm phán, thương lượng')
  await expect(page.getByText('Cần kết nối để ghi tiến độ', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toBeDisabled()
  await context.setOffline(false)
  expect((await evidence(request, fixture.exercise)).calls).toBe(1)
})
