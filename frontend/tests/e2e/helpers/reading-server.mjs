// Test-only infrastructure. No application route, OAuth stub or auth bypass.
import { createServer } from 'node:http'
import { createHmac, randomBytes, randomUUID } from 'node:crypto'
import { execFileSync, spawn } from 'node:child_process'
import { createWriteStream } from 'node:fs'
import { fileURLToPath } from 'node:url'

const backend = fileURLToPath(new URL('../../../../backend/', import.meta.url))
const compose = ['compose', '-p', 'reading-flow', '-f', `${backend}docker-compose.yml`]
function docker(args) {
  return execFileSync('docker', [...compose, ...args], { encoding: 'utf8' }).trim()
}
// Refuse to seed or query anything except the nominated disposable services.
if (!docker(['port', 'postgres', '5432']).endsWith(':55432')
  || !docker(['port', 'redis', '6379']).endsWith(':56379')) {
  throw new Error('reading-flow requires disposable PostgreSQL :55432 and Redis :56379')
}
const sql = query => docker(['exec', '-T', 'postgres', 'psql', '-U', 'english', '-d', 'english', '-At', '-v', 'ON_ERROR_STOP=1', '-c', query])
const redis = (...args) => docker(['exec', '-T', 'redis', 'redis-cli', '--raw', ...args])
const secret = randomBytes(32).toString('hex')
const apiLog = createWriteStream('/tmp/reading-flow-task4-api.log')
const providerLog = createWriteStream('/tmp/reading-flow-task4-provider.log')
const api = spawn('/tmp/reading-flow-task4-api', [], {
  cwd: backend,
  env: {
    ...process.env,
    DATABASE_URL: 'postgres://english:english@localhost:55432/english?sslmode=disable',
    REDIS_URL: 'redis://localhost:56379/0',
    PORT: '3199', FRONTEND_ORIGIN: 'http://127.0.0.1:3100',
    GOOGLE_CLIENT_ID: 'fixture-only', GOOGLE_CLIENT_SECRET: 'fixture-only',
    JWT_SECRET: secret, ENCRYPTION_SECRET_KEY: randomBytes(32).toString('hex'),
    GEMINI_API_KEY: '', DEEPSEEK_API_KEY: '',
    OPENAI_API_KEY: 'local-fixture-only', OPENAI_BASE_URL: 'http://127.0.0.1:3299/v1',
    OPENAI_MODEL: 'deterministic-reading', VAPID_PUBLIC_KEY: '', VAPID_PRIVATE_KEY: '',
  },
  stdio: ['ignore', 'pipe', 'pipe'],
})
api.stdout.pipe(apiLog)
api.stderr.pipe(apiLog)
let calls = 0
let failures = 0
const fixtures = new Map()
const passage = `Negotiated, the agreement helped a small community start a new library. Volunteers collected books and prepared shelves in an empty room. Each learner could borrow a book and discuss it with a neighbour. The organisers wanted to encourage daily habits and make reading accessible to everyone.\n\nAt first, only a few people arrived. A teacher suggested an evening meeting, because many residents worked during the day. Soon the room was full of conversation. People shared stories, practised unfamiliar words in context, and planned the next meeting together. The project showed that a modest change could create a lasting opportunity for the whole community.`
const glossary = [
  ['negotiate', 'đàm phán, thương lượng', 'To discuss something to reach an agreement.'],
  ['community', 'cộng đồng', 'A group of people who live or work together.'],
  ['learner', 'người học', 'A person who is learning something.'],
  ['habits', 'thói quen', 'Things that someone does regularly.'],
  ['context', 'ngữ cảnh', 'The situation that helps explain a word.'],
].map(([term, meaning_vi, definition]) => ({ term, meaning_vi, definition }))
const questions = [
  ['q1', 'What did the volunteers create?', 'A library'],
  ['q2', 'Why did the teacher suggest an evening meeting?', 'Residents worked during the day'],
  ['q3', 'What does the project show?', 'Small changes can help a community'],
].map(([id, prompt, answer]) => ({ id, prompt, options: { A: answer, B: 'They closed the room', C: 'Nobody wanted books', D: 'They stopped meeting' }, answer: 'A', explanation: 'The passage describes this directly.' }))

const send = (res, status, body) => {
  res.writeHead(status, { 'Content-Type': 'application/json' })
  res.end(JSON.stringify(body))
}
const server = createServer(async (req, res) => {
  try {
    const url = new URL(req.url, 'http://127.0.0.1:3299')
    if (url.pathname === '/ready') {
      const health = await fetch('http://127.0.0.1:3199/healthz').catch(() => null)
      return send(res, health?.ok ? 200 : 503, { ready: !!health?.ok })
    }
    if (url.pathname === '/fixture' && req.method === 'POST') {
      const user = randomUUID(), roadmap = randomUUID(), exercise = randomUUID()
      sql(`INSERT INTO users(id,email,google_id,full_name,cefr_current,target_goal,timezone) VALUES('${user}','${user}@fixture.invalid','${user}','Reading Fixture','B1','Read together','UTC'); INSERT INTO roadmaps(id,user_id,roadmap_json) VALUES('${roadmap}','${user}','{"title":"Fixture roadmap","modules":[]}'); INSERT INTO exercises(id,roadmap_id,day_number,task_type,content_json) VALUES('${exercise}','${roadmap}',1,'reading','{"type":"reading","title":"Community library","duration_minutes":10,"content":{}}');`)
      const now = Math.floor(Date.now() / 1000)
      const encode = value => Buffer.from(JSON.stringify(value)).toString('base64url')
      const unsigned = `${encode({ alg: 'HS256', typ: 'JWT' })}.${encode({ sub: user, iat: now, exp: now + 86400 })}`
      const token = `${unsigned}.${createHmac('sha256', secret).update(unsigned).digest('base64url')}`
      redis('SET', `sess:${user}:token`, token, 'EX', '86400')
      fixtures.set(exercise, user)
      return send(res, 200, { exercise, passage, questions, auth: { accessToken: token, expiresAt: (now + 86400) * 1000, user: { id: user, email: `${user}@fixture.invalid`, full_name: 'Reading Fixture', cefr_current: 'B1' } } })
    }
    if (url.pathname === '/control' && req.method === 'POST') {
      failures = Number(url.searchParams.get('failures') || 0)
      calls = 0
      return send(res, 200, { calls, failures })
    }
    if (url.pathname === '/evidence') {
      const exercise = url.searchParams.get('exercise')
      const user = fixtures.get(exercise)
      if (!user) return send(res, 404, { error: 'unknown_fixture' })
      const persisted = JSON.parse(sql(`SELECT json_build_object('completed',e.is_completed,'schema',e.content_json->'content_schema','content',e.content_json->'content','minutes',p.minutes_spent,'date',CURRENT_DATE::text) FROM exercises e LEFT JOIN daily_progress p ON p.user_id='${user}' AND p.date=CURRENT_DATE WHERE e.id='${exercise}'`))
      return send(res, 200, { ...persisted, seconds: Number(redis('GET', `daily:accumulated:${user}:${persisted.date}`) || 0), calls })
    }
    if (url.pathname === '/v1/chat/completions' && req.method === 'POST') {
      let raw = ''
      for await (const chunk of req) raw += chunk
      const body = JSON.parse(raw)
      if (body.model !== 'deterministic-reading' || body.response_format?.type !== 'json_object') return send(res, 400, { error: 'wrong_provider_contract' })
      calls++
      const fail = failures > 0
      if (fail) failures--
      providerLog.write(JSON.stringify({ call: calls, model: body.model, fail }) + '\n')
      await new Promise(resolve => setTimeout(resolve, 900))
      if (fail) return send(res, 400, { error: 'fixture_generation_failure' })
      return send(res, 200, { choices: [{ message: { content: JSON.stringify({ passage, questions, glossary }) } }], usage: { prompt_tokens: 100, completion_tokens: 200 } })
    }
    return send(res, 404, { error: 'not_found' })
  } catch (error) {
    console.error('reading fixture failed:', error.message)
    send(res, 500, { error: 'fixture_failed' })
  }
})
server.listen(3299, '127.0.0.1')
function stop() {
  server.close()
  try { api.kill('SIGTERM') } catch { /* already stopped */ }
  apiLog.end()
  providerLog.end()
  process.exit(0)
}
process.on('SIGTERM', stop)
process.on('SIGINT', stop)
api.on('exit', code => { if (code) { console.error(`fixture API exited ${code}`); stop() } })
