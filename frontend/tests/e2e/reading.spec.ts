import { expect, test } from '@playwright/test';
import type { APIRequestContext, Page } from '@playwright/test';
import { writeFile } from 'node:fs/promises';
const infra = 'http://127.0.0.1:3299';
const api = 'http://127.0.0.1:3199/api/v1/quests';
const logs = '/workspace/cloud-setup/reading-flow/task-5-logs';
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
    await page.getByRole('link', { name: '‹ Quay lại' }).click();
    await page.waitForURL('/');
    await context.setOffline(true);
    await page.getByRole('link', { name: /Học: Community library/ }).click();
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
