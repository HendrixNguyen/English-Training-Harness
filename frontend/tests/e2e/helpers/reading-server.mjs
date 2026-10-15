// Test-only infrastructure. No application route, OAuth stub or auth bypass.
import { createServer, request as httpRequest } from 'node:http';
import { createHmac, randomBytes, randomUUID } from 'node:crypto';
import { execFileSync, spawn } from 'node:child_process';
import { createWriteStream } from 'node:fs';
import { fileURLToPath } from 'node:url';
const backend = fileURLToPath(new URL('../../../../backend/', import.meta.url));
const compose = ['compose', '-p', process.env.LEARNING_COMPOSE_PROJECT || 'reading-flow', '-f', `${backend}docker-compose.yml`];
function docker(args) {
    return execFileSync('docker', [...compose, ...args], { encoding: 'utf8' }).trim();
}
// Refuse to seed or query anything except the nominated disposable services.
if (!docker(['port', 'postgres', '5432']).endsWith(':55432')
    || !docker(['port', 'redis', '6379']).endsWith(':56379')) {
    throw new Error('reading-flow requires disposable PostgreSQL :55432 and Redis :56379');
}
const sql = query => docker(['exec', '-T', 'postgres', 'psql', '-U', 'english', '-d', 'english', '-At', '-v', 'ON_ERROR_STOP=1', '-c', query]);
const redis = (...args) => docker(['exec', '-T', 'redis', 'redis-cli', '--raw', ...args]);
const secret = randomBytes(32).toString('hex');
const logs = process.env.LEARNING_PROOF_LOGS || '/workspace/cloud-setup/reading-flow/task-5-logs';
const live = process.env.LEARNING_GENUINE === '1';
if (live && !process.env.NINE_ROUTER_API_KEY)
    throw new Error('NINE_ROUTER_API_KEY binding missing');
const apiLog = createWriteStream(`${logs}/${live ? 'genuine' : 'deterministic'}-api.log`);
const providerLog = createWriteStream(`${logs}/deterministic-provider.log`);
const api = spawn(`${logs}/api`, [], {
    cwd: backend,
    env: {
        ...process.env,
        DATABASE_URL: 'postgres://english:english@localhost:55432/english?sslmode=disable',
        REDIS_URL: 'redis://localhost:56379/0',
        PORT: '3198', FRONTEND_ORIGIN: 'http://127.0.0.1:3100',
        GOOGLE_CLIENT_ID: 'fixture-only', GOOGLE_CLIENT_SECRET: 'fixture-only',
        JWT_SECRET: secret, ENCRYPTION_SECRET_KEY: randomBytes(32).toString('hex'),
        GEMINI_API_KEY: '', DEEPSEEK_API_KEY: '',
        OPENAI_API_KEY: live ? process.env.NINE_ROUTER_API_KEY : 'local-fixture-only',
        OPENAI_BASE_URL: live ? 'https://9router.huyndg.io.vn/v1' : 'http://127.0.0.1:3299/v1',
        OPENAI_MODEL: live ? 'English-bot-free' : 'deterministic-reading', VAPID_PUBLIC_KEY: '', VAPID_PRIVATE_KEY: '',
    },
    stdio: ['ignore', 'pipe', 'pipe'],
});
let generations = 0;
function log(chunk) {
    let value = chunk.toString();
    if (value.includes('openai-compat(') && value.includes('tokens'))
        generations++;
    for (const privateValue of [process.env.NINE_ROUTER_API_KEY, secret])
        if (privateValue)
            value = value.split(privateValue).join('[REDACTED]');
    apiLog.write(value.replace(/Bearer\s+\S+/g, 'Bearer [REDACTED]').replace(/eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+/g, '[JWT REDACTED]'));
}
api.stdout.on('data', log);
api.stderr.on('data', log);
let calls = 0;
let failures = 0;
let malformed = 0;
let fault = '';
let progressPosts = 0;
let deferredOwner = '';
let deferredPath = '/api/v1/quests/daily';
const deferredResponses = [];
const gateway = createServer((req, res) => {
    const progress = req.url === '/api/v1/quests/progress' && req.method === 'POST';
    if (progress)
        progressPosts++;
    if (progress && fault === 'before') {
        fault = '';
        req.resume();
        // An incomplete HTTP body prevents Chromium from replaying a reset socket.
        res.writeHead(200, { 'Content-Type': 'application/json', 'Content-Length': '1000' });
        res.write('{');
        setTimeout(() => res.destroy(), 25);
        return;
    }
    if (req.url.startsWith('/api/v1/quests/daily') && fault === 'unknown') {
        res.writeHead(200, { 'Content-Type': 'application/json', 'Content-Length': '1000' });
        res.write('{');
        setTimeout(() => res.destroy(), 25);
        return;
    }
    const lose = progress && ['after', 'after-unknown'].includes(fault);
    if (lose)
        fault = fault === 'after-unknown' ? 'unknown' : '';
    const upstream = httpRequest({ hostname: '127.0.0.1', port: 3198, path: req.url, method: req.method, headers: req.headers }, response => {
        if (lose) {
            res.writeHead(response.statusCode, response.headers);
            let sent = false;
            response.on('data', chunk => { if (!sent) {
                res.write(chunk.subarray(0, 1));
                sent = true;
            } });
            response.on('end', () => setTimeout(() => res.destroy(), 25));
            return;
        }
        if (req.url.startsWith(deferredPath) && deferredOwner && req.headers.authorization) {
            const owner = JSON.parse(Buffer.from(req.headers.authorization.split('.')[1], 'base64url').toString()).sub;
            if (owner === deferredOwner) {
                const chunks = [];
                response.on('data', chunk => chunks.push(chunk));
                response.on('end', () => deferredResponses.push(() => { res.writeHead(response.statusCode, response.headers); res.end(Buffer.concat(chunks)); }));
                return;
            }
        }
        res.writeHead(response.statusCode, response.headers);
        response.pipe(res);
    });
    upstream.on('error', () => res.destroy());
    req.pipe(upstream);
});
gateway.listen(3199, '127.0.0.1');
const fixtures = new Map();
const passage = `Negotiated, the agreement helped a small community start a new library. Volunteers collected books and prepared shelves in an empty room. Each learner could borrow a book and discuss it with a neighbour. The organisers wanted to encourage daily habits and make reading accessible to everyone.\n\nAt first, only a few people arrived. A teacher suggested an evening meeting, because many residents worked during the day. Soon the room was full of conversation. People shared stories, practised unfamiliar words in context, and planned the next meeting together. The project showed that a modest change could create a lasting opportunity for the whole community.`;
const glossary = [
    ['negotiate', 'đàm phán, thương lượng', 'To discuss something to reach an agreement.'],
    ['community', 'cộng đồng', 'A group of people who live or work together.'],
    ['learner', 'người học', 'A person who is learning something.'],
    ['habits', 'thói quen', 'Things that someone does regularly.'],
    ['context', 'ngữ cảnh', 'The situation that helps explain a word.'],
].map(([term, meaning_vi, definition]) => ({ term, meaning_vi, definition }));
const questions = [
    ['q1', 'What did the volunteers create?', 'A library'],
    ['q2', 'Why did the teacher suggest an evening meeting?', 'Residents worked during the day'],
    ['q3', 'What does the project show?', 'Small changes can help a community'],
].map(([id, prompt, answer]) => ({ id, prompt, options: { A: answer, B: 'They closed the room', C: 'Nobody wanted books', D: 'They stopped meeting' }, answer: 'A', explanation: 'The passage describes this directly.' }));
const words = [
    ['negotiate', 'To discuss something to reach an agreement.', 'We negotiate a fair price.'],
    ['community', 'People who live or work together.', 'Our community shares books.'],
    ['accessible', 'Easy to reach or use.', 'The library is accessible to everyone.'],
    ['volunteer', 'A person who helps without pay.', 'A volunteer sorts the books.'],
    ['habit', 'Something you do regularly.', 'Reading daily is a useful habit.'],
].map(([term, definition, example]) => ({ term, definition, example }));
const vocabQuestions = words.slice(0, 3).map((word, i) => ({ id: `q${i + 1}`, prompt: `Which word means ${word.definition.toLowerCase()}`, options: { A: word.term, B: 'shelf', C: 'cancel', D: 'empty' }, answer: 'A', explanation: `${word.term} means ${word.definition.toLowerCase()} The studied example demonstrates its use.` }));
const practiceQuestions = [...questions, { id: 'q4', prompt: 'Complete: She ___ books every evening.', options: { A: 'reads', B: 'read', C: 'reading', D: 'have read' }, answer: 'A', explanation: 'A singular subject in the present simple takes reads to describe a regular habit.' }];
const send = (res, status, body) => {
    res.writeHead(status, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify(body));
};
const server = createServer(async (req, res) => {
    try {
        const url = new URL(req.url, 'http://127.0.0.1:3299');
        if (url.pathname === '/ready') {
            const health = await fetch('http://127.0.0.1:3199/healthz').catch(() => null);
            return send(res, health?.ok ? 200 : 503, { ready: !!health?.ok });
        }
        if (url.pathname === '/fixture' && req.method === 'POST') {
            const type = url.searchParams.get('type') || 'reading';
            if (!['reading', 'vocabulary', 'practice'].includes(type))
                return send(res, 400, { error: 'bad_type' });
            const user = randomUUID(), roadmap = randomUUID(), exercise = randomUUID();
            const content = url.searchParams.get('invalid') === '1' ? { type, title: 'Community library', duration_minutes: 10, content_schema: 1, content_level: 'B1', content: { questions: [] } } : url.searchParams.get('legacy') === '1' ? { type, title: 'Community library', duration_minutes: 10, content_origin: 'legacy', content: { passage: 'A volunteer shares books with the community.' } } : { type, title: 'Community library', duration_minutes: 10, content: {} };
            if (url.searchParams.get('keyless') === '1') {
                content.content_origin = 'legacy';
                content.content = { questions: [{ id: 'q1', prompt: 'Choose a library activity.', options: { A: 'Read a book', B: 'Close the library' } }] };
            }
            const historical = url.searchParams.get('historical') === '1';
            const json = JSON.stringify(content).replaceAll("'", "''");
            sql(`INSERT INTO users(id,email,google_id,full_name,cefr_current,target_goal,timezone) VALUES('${user}','${user}@fixture.invalid','${user}','Reading Fixture','B1','Read together','UTC'); INSERT INTO roadmaps(id,user_id,roadmap_json) VALUES('${roadmap}','${user}','{"title":"Fixture roadmap","modules":[]}'); INSERT INTO exercises(id,roadmap_id,day_number,task_type,content_json) VALUES('${exercise}','${roadmap}',1,'${type}','${json}'); ${historical ? `UPDATE exercises SET is_completed=TRUE WHERE id='${exercise}';` : ''}`);
            const now = Math.floor(Date.now() / 1000);
            const encode = value => Buffer.from(JSON.stringify(value)).toString('base64url');
            const unsigned = `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode({ sub: user, iat: now, exp: now + 86400 })}`;
            const token = `${unsigned}.${createHmac('sha256', secret).update(unsigned).digest('base64url')}`;
            execFileSync('docker', [...compose, 'exec', '-T', 'redis', 'redis-cli', '-x', 'SET', `sess:${user}:token`], { input: token, stdio: ['pipe', 'ignore', 'pipe'] });
            redis('EXPIRE', `sess:${user}:token`, '86400');
            fixtures.set(exercise, user);
            return send(res, 200, { exercise, passage, questions: type === 'vocabulary' ? vocabQuestions : type === 'practice' ? practiceQuestions : questions, words, auth: { accessToken: token, expiresAt: (now + 86400) * 1000, user: { id: user, email: `${user}@fixture.invalid`, full_name: 'Reading Fixture', cefr_current: 'B1' } } });
        }
        if (url.pathname === '/renew' && req.method === 'POST') {
            const user = fixtures.get(url.searchParams.get('exercise'));
            if (!user) return send(res, 404, { error: 'unknown_fixture' });
            const now = Math.floor(Date.now() / 1000);
            const encode = value => Buffer.from(JSON.stringify(value)).toString('base64url');
            const unsigned = `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode({ sub: user, iat: now, exp: now + 86400, jti: randomUUID() })}`;
            const token = `${unsigned}.${createHmac('sha256', secret).update(unsigned).digest('base64url')}`;
            execFileSync('docker', [...compose, 'exec', '-T', 'redis', 'redis-cli', '-x', 'SET', `sess:${user}:token`], { input: token, stdio: ['pipe', 'ignore', 'pipe'] });
            redis('EXPIRE', `sess:${user}:token`, '86400');
            return send(res, 200, { access_token: token });
        }
        if (url.pathname === '/deferred') return send(res, 200, { count: deferredResponses.length });
        if (url.pathname === '/release' && req.method === 'POST') { deferredOwner = ''; for (const release of deferredResponses.splice(0)) release(); return send(res, 200, { released: true }); }
        if (url.pathname === '/control' && req.method === 'POST') {
            deferredOwner = url.searchParams.get('defer') || '';
            deferredPath = url.searchParams.get('path') || '/api/v1/quests/daily';
            failures = Number(url.searchParams.get('failures') || 0);
            malformed = Number(url.searchParams.get('malformed') || 0);
            fault = url.searchParams.get('fault') || '';
            if (url.searchParams.get('reset') === '1') {
                calls = 0;
                progressPosts = 0;
            }
            return send(res, 200, { calls, failures });
        }
        if (url.pathname === '/evidence') {
            const exercise = url.searchParams.get('exercise');
            const user = fixtures.get(exercise);
            if (!user)
                return send(res, 404, { error: 'unknown_fixture' });
            const persisted = JSON.parse(sql(`SELECT json_build_object('completed',e.is_completed,'schema',e.content_json->'content_schema','envelope',e.content_json,'result',e.quiz_result,'content',e.content_json->'content','minutes',p.minutes_spent,'date',CURRENT_DATE::text) FROM exercises e LEFT JOIN daily_progress p ON p.user_id='${user}' AND p.date=CURRENT_DATE WHERE e.id='${exercise}'`));
            return send(res, 200, { ...persisted, seconds: Number(redis('GET', `daily:accumulated:${user}:${persisted.date}`) || 0), calls, generations, progressPosts, limiter: redis('GET', `ratelimit:ai:${user}`) });
        }
        if (url.pathname === '/v1/chat/completions' && req.method === 'POST') {
            let raw = '';
            for await (const chunk of req)
                raw += chunk;
            const body = JSON.parse(raw);
            if (body.model !== 'deterministic-reading' || body.response_format?.type !== 'json_object')
                return send(res, 400, { error: 'wrong_provider_contract' });
            calls++;
            const fail = failures > 0;
            const bad = malformed > 0;
            if (bad)
                malformed--;
            const prompt = body.messages.map(message => message.content).join('\n');
            const type = prompt.includes('Ground the questions, answers and explanations in the studied words') ? 'vocabulary' : prompt.includes('Use the actual module focus') ? 'practice' : 'reading';
            providerLog.write(JSON.stringify({ call: calls, type, prompt }) + '\n');
            if (fail)
                failures--;
            providerLog.write(JSON.stringify({ call: calls, model: body.model, fail }) + '\n');
            await new Promise(resolve => setTimeout(resolve, 900));
            if (fail)
                return send(res, 400, { error: 'fixture_generation_failure' });
            return send(res, 200, { choices: [{ message: { content: JSON.stringify(bad ? { questions: [] } : type === 'vocabulary' ? { words, questions: vocabQuestions } : type === 'practice' ? { questions: practiceQuestions } : { passage, questions, glossary }) } }], usage: { prompt_tokens: 100, completion_tokens: 200 } });
        }
        return send(res, 404, { error: 'not_found' });
    }
    catch (error) {
        console.error('reading fixture failed:', error.message);
        send(res, 500, { error: 'fixture_failed' });
    }
});
server.listen(3299, '127.0.0.1');
function stop() {
    server.close();
    gateway.close();
    try {
        api.kill('SIGTERM');
    }
    catch { /* already stopped */ }
    apiLog.end();
    providerLog.end();
    process.exit(0);
}
process.on('SIGTERM', stop);
process.on('SIGINT', stop);
api.on('exit', code => { if (code) {
    console.error(`fixture API exited ${code}`);
    stop();
} });
