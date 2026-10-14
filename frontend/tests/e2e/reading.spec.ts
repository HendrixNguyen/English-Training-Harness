import { expect, test } from '@playwright/test';
import type { APIRequestContext, Page } from '@playwright/test';
import { writeFile } from 'node:fs/promises';
const infra = 'http://127.0.0.1:3299';
const api = 'http://127.0.0.1:3199/api/v1/quests';
const logs = process.env.LEARNING_PROOF_LOGS || '/workspace/cloud-setup/reading-flow/task-5-logs';
const live = process.env.LEARNING_GENUINE === '1';
interface Q {
    id: string;
    prompt: string;
    options: Record<string, string>;
    answer: string;
    explanation: string;
}
interface Content {
    questions: Q[];
    passage?: string;
    words?: {
        term: string;
        definition: string;
        example?: string;
    }[];
}
interface Fixture {
    exercise: string;
    auth: {
        accessToken: string;
        user: {
            id: string;
        };
    };
}
const auth = (f: Fixture) => ({ Authorization: `Bearer ${f.auth.accessToken}` });
const passage = (p: Page) => p.getByRole('region', { name: 'Đoạn văn', exact: true });
test('retro learning room stays readable at 375 and 200% layout zoom', async ({ page, request }) => {
    test.skip(live, 'Deterministic visual fixture; genuine content is covered by the three lifecycle cases');
    await page.setViewportSize({ width: 375, height: 812 });
    await page.emulateMedia({ reducedMotion: 'reduce' });
    const f = await seed(page, request);
    await page.goto(`/learn/${f.exercise}`);
    await expect(passage(page)).toBeVisible();
    await page.screenshot({ path: `${logs}/learning-room-375.png`, fullPage: true });
    await page.evaluate(() => { document.documentElement.style.zoom = '200%' });
    await expect(passage(page)).toBeVisible();
    const metrics = await page.evaluate(() => ({ dpr: devicePixelRatio, zoom: getComputedStyle(document.documentElement).zoom, viewport: innerWidth, html: document.documentElement.scrollWidth, body: document.body.scrollWidth }));
    expect(metrics.zoom).toBe('2');
    expect(metrics.dpr).toBe(1);
    expect(metrics.html).toBeLessThanOrEqual(metrics.viewport + 1);
    expect(metrics.body).toBeLessThanOrEqual(metrics.viewport + 1);
    await expect(page.getByRole('button', { name: 'Đọc xong →' })).toBeVisible();
    await page.screenshot({ path: `${logs}/learning-room-375-zoom200.png`, fullPage: true });
    await writeFile(`${logs}/learning-room-zoom.json`, JSON.stringify(metrics, null, 2));
});
async function seed(p: Page, r: APIRequestContext, params = ''): Promise<Fixture> {
    await r.post(`${infra}/control?reset=1`);
    const f = await (await r.post(`${infra}/fixture?${params}`)).json() as Fixture;
    await p.addInitScript(a => localStorage.setItem('aelp.auth', JSON.stringify(a)), f.auth);
    return f;
}
async function evidence(r: APIRequestContext, f: Fixture) { return (await r.get(`${infra}/evidence?exercise=${f.exercise}`)).json(); }
async function dto(r: APIRequestContext, f: Fixture) {
    const res = await r.get(`${api}/exercises/${f.exercise}`, { headers: auth(f) });
    expect(res.ok()).toBe(true);
    return res.json();
}
async function anchor(p: Page, f: Fixture, seconds = 65) {
    await p.evaluate(({ id, seconds }) => { const t = JSON.parse(localStorage.getItem('aelp.timers')!); t[id].startedAt = Date.now() - seconds * 1000; localStorage.setItem('aelp.timers', JSON.stringify(t)); }, { id: f.exercise, seconds });
    await p.reload();
}
async function study(p: Page, c: Content) {
    if (c.words)
        for (const [i, w] of c.words.entries()) {
            await expect(p.getByText(`Từ ${i + 1} / ${c.words.length}`, { exact: true })).toBeVisible();
            await expect(p.getByText(w.term, { exact: true })).toBeVisible();
            await expect(p.getByText(w.definition, { exact: true })).toBeVisible();
            if (w.example)
                await expect(p.getByText(w.example, { exact: true })).toBeVisible();
            await expect(p.getByRole('radio')).toHaveCount(0);
            if (i === 0) {
                await p.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-word-390x844.png` });
                await p.setViewportSize({ width: 1280, height: 900 });
                await p.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-word-desktop.png` });
                await p.setViewportSize({ width: 390, height: 844 });
            }
            if (i === 1) {
                await p.reload();
                await expect(p.getByText(`Từ 2 / ${c.words.length}`, { exact: true })).toBeVisible();
            }
            await p.getByRole('button', { name: i === c.words.length - 1 ? 'Làm câu hỏi' : 'Tiếp tục', exact: true }).click();
        }
    else if (c.passage) {
        await expect(passage(p)).toHaveText(c.passage);
        await p.getByRole('button', { name: 'Đọc xong →' }).click();
    }
}
async function quiz(p: Page, c: Content, shots = false) {
    const answers: Record<string, string> = {};
    for (const [i, q] of c.questions.entries()) {
        await expect(p.getByText(`Câu ${i + 1} / ${c.questions.length}`, { exact: true })).toBeVisible();
        await expect(p.getByText(`Đã trả lời ${i} / ${c.questions.length}`, { exact: true })).toBeVisible();
        await expect(p.getByRole('button', { name: 'Trả lời', exact: true })).toBeDisabled();
        await expect(p.getByText(q.explanation, { exact: true })).toHaveCount(0);
        await expect(p.getByText('✔ Đáp án đúng', { exact: true })).toHaveCount(0);
        await expect(p.getByRole('button', { name: 'Hoàn thành', exact: true })).toHaveCount(0);
        const choice = i === 1 ? Object.keys(q.options).find(k => k !== q.answer)! : q.answer;
        answers[q.id] = choice;
        const option = p.getByRole('radio').filter({ hasText: `(${choice})` });
        await option.focus();
        await p.keyboard.press('Space');
        await expect(option).toBeFocused();
        await expect(option).toHaveAttribute('aria-checked', 'true');
        expect((await option.boundingBox())!.height).toBeGreaterThanOrEqual(56);
        expect(await option.evaluate(el => getComputedStyle(el).outlineStyle)).not.toBe('none');
        if (i === 0) {
            await p.reload();
            await expect(p.getByRole('radio').filter({ hasText: `(${choice})` })).toHaveAttribute('aria-checked', 'true');
        }
        await p.getByRole('button', { name: 'Trả lời', exact: true }).click();
        await expect(p.getByText(q.explanation, { exact: true })).toBeVisible();
        await expect(p.getByRole('heading', { name: i === 1 ? 'Trượt…' : 'Trúng!', exact: true })).toBeFocused();
        await expect(p.locator('[role="status"][aria-live="polite"]')).toHaveCount(1);
        await expect(p.getByRole('radio').first()).toBeDisabled();
        if (i === 0) {
            await p.reload();
            await expect(p.getByText(q.explanation, { exact: true })).toBeVisible();
            await expect(p.getByText(`Đã trả lời 1 / ${c.questions.length}`, { exact: true })).toBeVisible();
        }
        if (shots && i === 1) {
            await p.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-feedback-390x844.png`, fullPage: true });
            await p.setViewportSize({ width: 1280, height: 900 });
            await p.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-feedback-desktop.png`, fullPage: true });
            await p.setViewportSize({ width: 390, height: 844 });
        }
        await p.getByRole('button', { name: 'Tiếp tục', exact: true }).click();
    }
    await expect(p.getByText('Chưa lưu kết quả', { exact: true })).toBeVisible();
    await expect(p.getByRole('heading', { name: 'Community library', exact: true })).toBeFocused();
    return answers;
}
for (const type of ['reading', 'vocabulary', 'practice'])
    test(`${live ? 'genuine 9Router' : 'deterministic transport'} ${type}: study, feedback, grade, durable time and cache`, async ({ page, request }) => {
        await page.emulateMedia({ reducedMotion: 'reduce' });
        const f = await seed(page, request, `type=${type}`);
        expect((await request.get(`${api}/exercises/${f.exercise}`)).status()).toBe(401);
        const started = Date.now();
        const generation = page.waitForResponse(res => res.url().includes(`/exercises/${f.exercise}`) && !res.url().includes('repair'));
        await page.goto(`/learn/${f.exercise}`);
        const response = await generation;
        if (!response.ok()) await writeFile(`${logs}/${live ? 'genuine' : 'deterministic'}-${type}-unavailable.json`, JSON.stringify({ generation_ms: Date.now()-started, status: response.status(), error: await response.json(), persisted: await evidence(request,f) }, null, 2));
        expect(response.ok(), `actual ${type} generation HTTP ${response.status()}`).toBe(true);
        await expect(page.getByRole('button', { name: /Đọc xong|Tiếp tục|Trả lời/ }).first()).toBeVisible();
        const task = await dto(request, f);
        const generation_ms = Date.now() - started;
        expect(task.content_origin).toBe('generated');
        expect(task.content_revision).toMatch(/^[0-9a-f]{64}$/);
        if (!live)
            expect((await evidence(request, f)).calls).toBe(1);
        await anchor(page, f);
        if (type === 'reading') {
            const term = page.getByRole('button', { name: /^Giải nghĩa / }).first();
            await term.focus();
            await page.keyboard.press('Enter');
            await expect(page.getByRole('region', { name: 'Giải nghĩa', exact: true })).toBeVisible();
            await page.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-gloss-390x844.png` });
            await page.keyboard.press('Escape');
            await expect(term).toBeFocused();
        }
        await study(page, task.content_json.content);
        if (type === 'reading') {
            await page.getByRole('button', { name: 'Xem lại đoạn' }).click();
            await expect(passage(page)).toHaveText(task.content_json.content.passage);
        }
        const answers = await quiz(page, task.content_json.content, type === 'reading');
        const posted = page.waitForRequest(r => r.url().endsWith('/quests/progress') && r.method() === 'POST');
        await page.getByRole('button', { name: 'Hoàn thành', exact: true }).dblclick();
        const body = (await posted).postDataJSON();
        expect(body.content_revision).toBe(task.content_revision);
        expect(body.user_answers).toEqual(answers);
        await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
        const saved = await evidence(request, f);
        const n = task.content_json.content.questions.length;
        expect(saved.completed).toBe(true);
        expect(saved.result.status).toBe('graded');
        expect(saved.result.correct_count).toBe(n - 1);
        expect(saved.result.score_percent).toBe(Math.round(100 * (n - 1) / n));
        expect(saved.result.answers.map((a: {
            question_id: string;
            selected_option: string;
        }) => [a.question_id, a.selected_option])).toEqual(Object.entries(answers));
        expect(saved.result.answers).toEqual((task.content_json.content.questions as Q[]).map(q => ({ question_id: q.id, selected_option: answers[q.id], correct_option: q.answer, is_correct: answers[q.id] === q.answer, explanation: q.explanation })));
        await expect(page.getByText(`Đúng ${n-1} / ${n} · ${saved.result.score_percent}%`, { exact: true })).toBeVisible();
        expect(saved.result.duration_seconds).toBe(body.duration_seconds);
        expect(saved.seconds).toBe(body.duration_seconds);
        expect(saved.minutes).toBe(Math.floor(body.duration_seconds / 60));
        expect(saved.progressPosts).toBe(1);
        const time = `Thời gian học: ${Math.floor(body.duration_seconds / 60)} phút ${body.duration_seconds % 60} giây`;
        await expect(page.getByText(time, { exact: true })).toBeVisible();
        await page.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-${type}-summary-390x844.png`, fullPage: true });
        await page.setViewportSize({ width: 1280, height: 900 });
        await page.screenshot({ path: `${logs}/${live ? 'genuine' : 'deterministic'}-${type}-summary-desktop.png`, fullPage: true });
        await page.reload();
        await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
        await expect(page.getByText(time, { exact: true })).toBeVisible();
        expect((await dto(request, f)).quiz_result).toEqual(saved.result);
        const final = await evidence(request, f);
        expect(final.progressPosts).toBe(1);
        expect(final.calls).toBe(saved.calls);
        expect(final.generations).toBe(saved.generations);
        expect(await page.evaluate(() => Object.keys(localStorage).filter(k => k.startsWith('aelp.learning-draft')))).toEqual([]);
        await writeFile(`${logs}/${live ? 'genuine' : 'deterministic'}-${type}-evidence.json`, JSON.stringify({ generation_ms, body, task, saved, final }, null, 2));
    });
test('strict API rejects partial/tampered/stale before writes; expiry and offline never bypass unanswered quiz', async ({ page, context, request }) => {
    test.skip(live, 'Deterministic negative transport');
    const f = await seed(page, request);
    await page.goto(`/learn/${f.exercise}`);
    await expect(passage(page)).toBeVisible();
    const task = await dto(request, f);
    const base = { exercise_id: f.exercise, duration_seconds: 65, content_revision: task.content_revision };
    for (const body of [{ ...base, user_answers: { q1: 'A' } }, { ...base, score: 100 }, { ...base, user_answers: { q1: 'Z', q2: 'A', q3: 'A' } }, { ...base, user_answers: { foreign: 'A' } }])
        expect((await request.post(`${api}/progress`, { headers: auth(f), data: body })).status()).toBe(400);
    expect((await request.post(`${api}/progress`, { headers: auth(f), data: { ...base, content_revision: 'stale', user_answers: { q1: 'A', q2: 'A', q3: 'A' } } })).status()).toBe(409);
    const before = await evidence(request, f);
    expect(before.completed).toBe(false);
    expect(before.result).toBeNull();
    expect(before.seconds).toBe(0);
    expect(before.minutes).toBeNull();
    await anchor(page, f, 700);
    await page.getByRole('button', { name: 'Đọc xong →' }).click();
    await expect(page.getByRole('button', { name: 'Trả lời', exact: true })).toBeDisabled();
    await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toHaveCount(0);
    await page.getByRole('radio').first().click();
    await page.getByRole('link', { name: '‹ Trại' }).click();
    await page.waitForURL('/');
    await context.setOffline(true);
    await page.getByRole('button', { name: /Community library, Vào/ }).click();
    await expect(page.getByRole('radio').first()).toHaveAttribute('aria-checked', 'true');
    await expect(page.getByText('Cần kết nối để ghi tiến độ', { exact: true })).toBeVisible();
    for (let i = 0; i < 3; i++) {
        await page.getByRole('radio').first().click();
        await page.getByRole('button', { name: 'Trả lời', exact: true }).click();
        await page.getByRole('button', { name: 'Tiếp tục', exact: true }).click();
    }
    await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toBeDisabled();
    await context.setOffline(false);
    expect((await evidence(request, f)).calls).toBe(1);
});
test('explicit invalid repair keeps original bytes on bounded failure then caches success', async ({ page, request }) => {
    test.skip(live, 'Deterministic malformed transport');
    const f = await seed(page, request, 'invalid=1');
    const original = await evidence(request, f);
    await page.goto(`/learn/${f.exercise}`);
    await expect(page.getByText('Nội dung này cần được tạo lại.', { exact: true })).toBeVisible();
    expect((await evidence(request, f)).calls).toBe(0);
    await request.post(`${infra}/control?malformed=2`);
    await page.getByRole('button', { name: 'Tạo lại nội dung', exact: true }).click();
    await expect(page.getByText('Chưa tạo được nội dung phù hợp. Quay lại sau.', { exact: true })).toBeVisible();
    const failed = await evidence(request, f);
    expect(failed.envelope).toEqual(original.envelope);
    expect(failed.calls).toBe(2);
    expect(failed.completed).toBe(false);
    await page.getByRole('button', { name: /Tạo lại nội dung|Thử lại/ }).click();
    await expect(passage(page)).toBeVisible();
    expect((await evidence(request, f)).calls).toBe(3);
    await page.reload();
    await expect(passage(page)).toBeVisible();
    expect((await evidence(request, f)).calls).toBe(3);
    await writeFile(`${logs}/repair-evidence.json`, JSON.stringify({ original, failed, repaired: await evidence(request, f) }, null, 2));
});
for (const fault of ['before', 'after'])
    test(`connection lost ${fault} actual upstream save retains/reconciles`, async ({ page, request }) => {
        test.skip(live, 'Deterministic network fault');
        const f = await seed(page, request);
        await page.goto(`/learn/${f.exercise}`);
        await expect(passage(page)).toBeVisible();
        const task = await dto(request, f);
        await anchor(page, f);
        await study(page, task.content_json.content);
        const answers = await quiz(page, task.content_json.content);
        await request.post(`${infra}/control?fault=${fault}`);
        await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
        if (fault === 'before') {
            await expect(page.getByText('Chưa ghi được tiến độ. Thử lại.', { exact: true })).toBeVisible();
            const incomplete = await evidence(request, f);
            expect(incomplete.completed).toBe(false);
            expect(incomplete.seconds).toBe(0);
            const draft = await page.evaluate(() => JSON.parse(localStorage.getItem(Object.keys(localStorage).find(k => k.startsWith('aelp.learning-draft'))!)!).session);
            expect(draft.answers).toEqual(answers);
            expect(draft.duration).toBeGreaterThanOrEqual(65);
            await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
        }
        await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
        const saved = await evidence(request, f);
        expect(saved.progressPosts).toBe(fault === 'before' ? 2 : 1);
        expect(saved.result.correct_count).toBe(2);
        expect(saved.seconds).toBe(saved.result.duration_seconds);
        await page.reload();
        await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
        expect((await evidence(request, f)).progressPosts).toBe(saved.progressPosts);
        await writeFile(`${logs}/save-${fault}-evidence.json`, JSON.stringify(saved, null, 2));
    });
test('legacy study-only persists honest ungraded null score and durable time', async ({ page, request }) => {
    test.skip(live, 'Legacy is not generated quality evidence');
    const f = await seed(page, request, 'legacy=1');
    await page.goto(`/learn/${f.exercise}`);
    await expect(passage(page)).toBeVisible();
    await page.getByRole('button', { name: 'Đọc xong →' }).click();
    await expect(page.getByText('Không chấm điểm', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
    await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    const saved = await evidence(request, f);
    expect(saved.result.status).toBe('ungraded');
    expect(saved.result.score_percent).toBeNull();
    expect(saved.result.correct_count).toBeNull();
    expect(saved.result.answers).toEqual([]);
    await page.reload();
    await expect(page.getByText('Không chấm điểm', { exact: true })).toBeVisible();
    expect((await evidence(request, f)).progressPosts).toBe(1);
});
test('unknown lost-response outcome offers read-only checking then adopts the real saved grade', async ({ page, request }) => {
    test.skip(live, 'Deterministic connection faults');
    const f = await seed(page, request);
    await page.goto(`/learn/${f.exercise}`);
    await expect(passage(page)).toBeVisible();
    const task = await dto(request, f);
    await study(page, task.content_json.content);
    await quiz(page, task.content_json.content);
    await request.post(`${infra}/control?fault=after-unknown`);
    await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
    await expect(page.getByText('Chưa xác nhận được kết quả. Kết nối rồi kiểm tra lại.', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toHaveCount(0);
    const committed = await evidence(request, f);
    expect(committed.completed).toBe(true);
    expect(committed.progressPosts).toBe(1);
    await request.post(`${infra}/control`);
    await page.getByRole('button', { name: 'Kiểm tra lại', exact: true }).click();
    await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    expect((await evidence(request, f)).progressPosts).toBe(1);
    await writeFile(`${logs}/save-unknown-evidence.json`, JSON.stringify(await evidence(request, f), null, 2));
});
test('keyless legacy quiz records a choice without inventing correctness', async ({ page, request }) => {
    test.skip(live, 'Legacy has no provider quality claim');
    const f = await seed(page, request, 'type=practice&keyless=1');
    await page.goto(`/learn/${f.exercise}`);
    await page.getByRole('radio').first().click();
    await page.getByRole('button', { name: 'Trả lời', exact: true }).click();
    await expect(page.getByText('Câu này chưa có đáp án để chấm.', { exact: true })).toBeVisible();
    await expect(page.getByText('✔ Đáp án đúng', { exact: true })).toHaveCount(0);
    await page.getByRole('button', { name: 'Tiếp tục', exact: true }).click();
    await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
    await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    const result = (await evidence(request, f)).result;
    expect(result.status).toBe('ungraded');
    expect(result.correct_count).toBeNull();
    expect(result.score_percent).toBeNull();
    expect(result.answers).toEqual([{ question_id: 'q1', selected_option: 'A', correct_option: null, is_correct: null, explanation: null }]);
});
test('historical completed null result has unknown time and never POSTs', async ({ page, request }) => {
    test.skip(live, 'Historical fixture');
    const f = await seed(page, request, 'legacy=1&historical=1');
    await page.goto(`/learn/${f.exercise}`);
    await expect(page.getByText('Không có kết quả câu hỏi đã lưu', { exact: true })).toBeVisible();
    await expect(page.getByText('Thời gian học: chưa có dữ liệu', { exact: true })).toBeVisible();
    await page.reload();
    await expect(page.getByText('Thời gian học: chưa có dữ liệu', { exact: true })).toBeVisible();
    expect((await evidence(request, f)).progressPosts).toBe(0);
});
test('denied draft storage keeps study and server saving usable', async ({ page, request }) => {
    test.skip(live, 'Storage negative');
    const f = await seed(page, request);
    // Keep auth/timer/cache available, deny only the draft namespace.
    await page.addInitScript(() => { const set = Storage.prototype.setItem; Storage.prototype.setItem = function (key, value) { if (key.startsWith('aelp.learning-draft'))
        throw new DOMException('Denied', 'SecurityError'); return set.call(this, key, value); }; });
    await page.goto(`/learn/${f.exercise}`);
    await expect(passage(page)).toBeVisible();
    await expect(page.getByText('Tiến độ chưa lưu chỉ giữ trong lần mở này.', { exact: true })).toBeVisible();
    const task = await dto(request, f);
    await study(page, task.content_json.content);
    // Do not reload: memory-only continuity is intentionally limited to this mount.
    for (const q of task.content_json.content.questions) {
        await page.getByRole('radio').filter({ hasText: `(${q.answer})` }).click();
        await page.getByRole('button', { name: 'Trả lời', exact: true }).click();
        await page.getByRole('button', { name: 'Tiếp tục', exact: true }).click();
    }
    await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click();
    await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    expect((await evidence(request, f)).result.score_percent).toBe(100);
});

// These regressions use the production worker and real auth/store actions. The
// gateway only delays actual upstream bytes; it does not fabricate API DTOs.
async function worker(p: Page) {
    await p.evaluate(async () => { await navigator.serviceWorker.ready; });
    if (!await p.evaluate(() => !!navigator.serviceWorker.controller)) await p.reload();
    expect(await p.evaluate(() => !!navigator.serviceWorker.controller)).toBe(true);
}
async function replaceSession(p: Page, f: Fixture) {
    await p.evaluate(async a => {
        const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { signOut(): Promise<void>; signIn(r: unknown): Promise<void> }> } } } } }).__vue_app__.config.globalProperties.$pinia;
        const auth = pinia._s.get('auth')!;
        await auth.signOut(); await auth.signIn({ access_token: a.accessToken, expires_in: 86400, user: a.user });
    }, f.auth);
}
async function loadStore(p: Page, wait = true) {
    return p.evaluate(wait => {
        const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { load(): Promise<boolean> }> } } } } }).__vue_app__.config.globalProperties.$pinia;
        const promise = pinia._s.get('quest')!.load();
        if (wait) return promise; void promise; return true;
    }, wait);
}
test('saved result survives initial outline worker fallback, offline hub/reopen and a re-created store', async ({ page, context, request }) => {
    test.setTimeout(45_000);
    const f = await seed(page, request); await page.goto('/'); await expect(page.getByRole('button', { name: /Community library, Vào/ })).toBeVisible(); await worker(page); await loadStore(page);
    await page.getByRole('button', { name: /Community library, Vào/ }).click(); await expect(passage(page)).toBeVisible();
    // Prime the real initial outline response before hydration in a fresh store.
    // The first navigation daily already contains that outline; wait for cachePut.
    await expect.poll(() => page.evaluate(async id => { const cache = await caches.open('api-state'); for (const key of await cache.keys()) { if (key.url.includes('/quests/daily')) { const body = await (await cache.match(key))!.json(); if (body.tasks.some((t: { id: string; content_origin: string }) => t.id === id && t.content_origin === 'outline')) return true; } } return false; }, f.exercise)).toBe(true);
    const task = await dto(request, f); await study(page, task.content_json.content);
    for (const q of task.content_json.content.questions as Q[]) { await page.getByRole('radio').filter({ hasText: `(${q.answer})` }).click(); await page.getByRole('button', { name: 'Trả lời', exact: true }).click(); await page.getByRole('button', { name: 'Tiếp tục', exact: true }).click(); }
    await page.getByRole('button', { name: 'Hoàn thành', exact: true }).click(); await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    const saved = await evidence(request, f); expect(saved.result.score_percent).toBe(100); expect(saved.progressPosts).toBe(1);
    await context.setOffline(true); await page.getByRole('button', { name: 'Về trại', exact: true }).click(); await page.waitForURL('/'); expect(await loadStore(page)).toBe(true);
    await expect(page.getByRole('button', { name: /Community library, Đã xong/ })).toBeVisible();
    await page.evaluate(id => { const app = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $router: { push(path: string): Promise<void> } } } } }).__vue_app__; return app.config.globalProperties.$router.push(`/learn/${id}`); }, f.exercise); await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible();
    // Reload document online while blocking only daily transport. Re-created
    // Pinia must read the durable checkpoint + real worker fallback.
    await context.setOffline(false); await request.post(`${infra}/control?fault=unknown`); await page.reload();
    await expect(page.getByText('Đã lưu kết quả', { exact: true })).toBeVisible(); await expect(page.getByText('Đúng 3 / 3 · 100%', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Hoàn thành', exact: true })).toHaveCount(0);
    await request.post(`${infra}/control`); const final = await evidence(request, f); expect(final.result).toEqual(saved.result); expect(final.seconds).toBe(saved.seconds); expect(final.progressPosts).toBe(1);
    await writeFile(`${logs}/saved-worker-fallback.json`, JSON.stringify({ saved, final }, null, 2));
});
for (const path of ['/api/v1/quests/daily', '/api/v1/pet/status']) test(`late prior-owner worker cache write cannot serve B fallback: ${path}`, async ({ page, context, request }) => {
    test.setTimeout(45_000);
    const a = await seed(page, request, 'type=practice&keyless=1'); const privateTask = await dto(request, a); expect((await request.post(`${api}/progress`, { headers: auth(a), data: { exercise_id: a.exercise, duration_seconds: 65, content_revision: privateTask.content_revision, user_answers: { q1: 'B' } } })).ok()).toBe(true); const b = await (await request.post(`${infra}/fixture?legacy=1`)).json() as Fixture;
    await page.goto('/'); await expect(page.getByRole('button', { name: /Community library, Đã xong/ })).toBeVisible(); await worker(page); expect(await loadStore(page)).toBe(true);
    await request.post(`${infra}/control?defer=${a.auth.user.id}&path=${encodeURIComponent(path)}`);
    if (path.endsWith('/daily')) await loadStore(page, false);
    else await page.evaluate(() => { const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { load(): Promise<void> }> } } } } }).__vue_app__.config.globalProperties.$pinia; void pinia._s.get('pet')!.load(); });
    await expect.poll(async () => (await (await request.get(`${infra}/deferred`)).json()).count).toBe(1);
    await replaceSession(page, b); await request.post(`${infra}/release`);
    await expect.poll(() => page.evaluate(async path => { const c = await caches.open('api-state'); return (await c.keys()).some(k => new URL(k.url).pathname === path); }, path)).toBe(true);
    if (path.endsWith('/daily')) {
        const leakedCandidate = await page.evaluate(async ({ owner, exercise }) => { const cache = await caches.open('api-state'); for (const key of await cache.keys()) if (new URL(key.url).searchParams.get('aelp_owner') === owner && new URL(key.url).pathname.endsWith('/daily')) { const dto = await (await cache.match(key))!.json(); return dto.tasks.find((t: { id: string }) => t.id === exercise)?.quiz_result; } return null; }, { owner: a.auth.user.id, exercise: a.exercise });
        expect(leakedCandidate.answers[0].selected_option).toBe('B');
    }
    await context.setOffline(true);
    if (path.endsWith('/daily')) { expect(await loadStore(page)).toBe(false); await expect(page.getByRole('button', { name: /Community library, Vào/ })).toHaveCount(0); }
    else {
        const response = await page.evaluate(async () => { const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { $reset(): void; load(): Promise<void>; status: unknown; error: string | null }> } } } } }).__vue_app__.config.globalProperties.$pinia; const pet = pinia._s.get('pet')!; pet.$reset(); await pet.load(); return { status: pet.status, error: pet.error }; }); expect(response.status).toBeNull(); expect(response.error).toBe('network_error');
    }
    await context.setOffline(false); expect(await loadStore(page)).toBe(true);
    await expect.poll(() => page.evaluate(async owner => { const c = await caches.open('api-state'); return (await c.keys()).some(k => new URL(k.url).searchParams.get('aelp_owner') === owner && new URL(k.url).pathname.endsWith('/daily')); }, b.auth.user.id)).toBe(true);
    const ids = await page.evaluate(async owner => { const c = await caches.open('api-state'); const entries = []; for (const k of await c.keys()) entries.push({ owner: new URL(k.url).searchParams.get('aelp_owner'), path: new URL(k.url).pathname }); return { owner, entries }; }, b.auth.user.id);
    expect(ids.entries.some(e => e.owner === b.auth.user.id && e.path.endsWith('/daily'))).toBe(true);
    const renewed = await (await request.post(`${infra}/renew?exercise=${b.exercise}`)).json() as { access_token: string };
    expect(renewed.access_token).not.toBe(b.auth.accessToken);
    await page.evaluate(async ({ token, user }) => { const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { signIn(r: unknown): Promise<void> }> } } } } }).__vue_app__.config.globalProperties.$pinia; await pinia._s.get('auth')!.signIn({ access_token: token, expires_in: 86400, user }); }, { token: renewed.access_token, user: b.auth.user });
    await context.setOffline(true); expect(await loadStore(page)).toBe(true); await context.setOffline(false);
    expect((await evidence(request, a)).progressPosts).toBe(1); expect((await evidence(request, b)).progressPosts).toBe(1);
    await writeFile(`${logs}/late-owner-${path.endsWith('/daily') ? 'daily' : 'pet'}.json`, JSON.stringify({ a: a.auth.user.id, b: b.auth.user.id, cache: ids, renewed_token_offline_reuse: true }, null, 2));
});
for (const route of ['/', '/settings']) test(`central signout from ${route} clears draft before same-owner relogin`, async ({ page, request }) => {
    const f = await seed(page, request); await page.goto(`/learn/${f.exercise}`); await expect(passage(page)).toBeVisible();
    await page.getByRole('button', { name: 'Đọc xong →' }).click(); await page.getByRole('radio').first().click();
    expect(await page.evaluate(() => Object.keys(localStorage).filter(k => k.startsWith('aelp.learning-draft')).length)).toBe(1);
    await page.getByRole('link', { name: '‹ Trại' }).click(); await page.waitForURL('/');
    if (route === '/settings') await page.goto('/settings');
    else await page.getByRole('link', { name: /Cài đặt tài khoản/ }).click();
    await page.waitForURL('/settings');
    await page.getByRole('button', { name: 'Rời quán · Đăng xuất' }).click();
    await page.getByRole('dialog').getByRole('button', { name: 'Đăng xuất' }).click(); await page.waitForURL('/login');
    expect(await page.evaluate(() => Object.keys(localStorage).filter(k => k.startsWith('aelp.learning-draft')))).toEqual([]);
    await replaceSession(page, f); await page.evaluate(id => { const app = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $router: { push(path: string): Promise<void> } } } } }).__vue_app__; return app.config.globalProperties.$router.push(`/learn/${id}`); }, f.exercise);
    await expect(passage(page)).toBeVisible(); await page.getByRole('button', { name: 'Đọc xong →' }).click(); await expect(page.getByRole('radio').first()).toHaveAttribute('aria-checked', 'false'); expect((await evidence(request, f)).progressPosts).toBe(0);
});

test('API session lifetime protects same-owner replacement and renewal from deferred real responses', async ({ page, request }) => {
    const f = await seed(page, request); await page.goto(`/learn/${f.exercise}`); await expect(passage(page)).toBeVisible();
    await page.getByRole('button', { name: 'Đọc xong →' }).click(); await page.getByRole('radio').first().click();
    await page.getByRole('link', { name: '‹ Trại' }).click(); await page.waitForURL('/'); expect(await loadStore(page)).toBe(true);
    await expect.poll(() => page.evaluate(() => { const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { loading: boolean }> } } } } }).__vue_app__.config.globalProperties.$pinia; return pinia._s.get('pet')!.loading; })).toBe(false);
    const outcomes = [];
    for (const scenario of ['relogin200', 'relogin401', 'renewal401', 'renewal200']) {
        const rejected = scenario.endsWith('401'); const replacement = scenario.startsWith('relogin');
        let token = '';
        if (rejected) token = ((await (await request.post(`${infra}/renew?exercise=${f.exercise}`)).json()) as { access_token: string }).access_token;
        await request.post(`${infra}/control?defer=${f.auth.user.id}&path=${encodeURIComponent('/api/v1/pet/status')}`);
        const epoch = await page.evaluate(() => {
            const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { $reset(): void; load(): Promise<void>; sessionEpoch: number }> } } } } }).__vue_app__.config.globalProperties.$pinia;
            const pet = pinia._s.get('pet')!; pet.$reset();
            (window as Window & { __petReadSettled?: boolean }).__petReadSettled = false;
            void pet.load().finally(() => { (window as Window & { __petReadSettled?: boolean }).__petReadSettled = true });
            return pinia._s.get('auth')!.sessionEpoch;
        });
        await expect.poll(async () => (await (await request.get(`${infra}/deferred`)).json()).count).toBe(1);
        if (!rejected) token = ((await (await request.post(`${infra}/renew?exercise=${f.exercise}`)).json()) as { access_token: string }).access_token;
        const nextEpoch = await page.evaluate(async ({ token, user, replacement }) => {
            const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { signOut(): Promise<void>; signIn(r: unknown): Promise<void>; sessionEpoch: number }> } } } } }).__vue_app__.config.globalProperties.$pinia;
            const auth = pinia._s.get('auth')!; if (replacement) await auth.signOut(); await auth.signIn({ access_token: token, expires_in: 86400, user }); return auth.sessionEpoch;
        }, { token, user: f.auth.user, replacement });
        expect(nextEpoch).toBe(epoch + (replacement ? 2 : 0));
        // Create/retain a real controlled learning draft in the current session
        // before the old response is released; unauthorized cleanup must not erase it.
        await page.evaluate(id => { const app = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $router: { push(path: string): Promise<void> } } } } }).__vue_app__; return app.config.globalProperties.$router.push(`/learn/${id}`); }, f.exercise);
        const studied = page.getByRole('button', { name: 'Đọc xong →' });
        await expect(page.getByRole('button', { name: /Đọc xong|Trả lời/ }).first()).toBeVisible();
        if (await studied.count()) await studied.click(); await page.getByRole('radio').first().click();
        expect(await page.evaluate(() => Object.keys(localStorage).filter(k => k.startsWith('aelp.learning-draft')).length)).toBe(1);
        await request.post(`${infra}/release`);
        await expect.poll(() => page.evaluate(() => (window as Window & { __petReadSettled?: boolean }).__petReadSettled)).toBe(true);
        const outcome = await page.evaluate(() => {
            const pinia = (document.querySelector('#__nuxt') as unknown as { __vue_app__: { config: { globalProperties: { $pinia: { _s: Map<string, { status: unknown; error: string | null; isAuthenticated: boolean; sessionEpoch: number }> } } } } }).__vue_app__.config.globalProperties.$pinia;
            const pet = pinia._s.get('pet')!, auth = pinia._s.get('auth')!; return { authenticated: auth.isAuthenticated, epoch: auth.sessionEpoch, pet: pet.status, error: pet.error, drafts: Object.keys(localStorage).filter(k => k.startsWith('aelp.learning-draft')).length };
        });
        expect(outcome.authenticated).toBe(true); expect(outcome.epoch).toBe(nextEpoch); expect(outcome.drafts).toBe(1); await expect(page.getByRole('radio').first()).toHaveAttribute('aria-checked', 'true');
        if (scenario === 'renewal200') { expect(outcome.pet).not.toBeNull(); expect(outcome.error).toBeNull(); }
        else {
            expect(outcome.pet).toBeNull();
            // Account replacement retires the old store request before it can
            // write an error into the current account. Renewal keeps that store
            // lifetime and reports the rejected prior-token response.
            expect(outcome.error).toBe(replacement ? null : 'session_changed');
        }
        outcomes.push({ scenario, ...outcome });
    }
    expect((await evidence(request, f)).progressPosts).toBe(0);
    await writeFile(`${logs}/api-session-lifetime.json`, JSON.stringify(outcomes, null, 2));
});
