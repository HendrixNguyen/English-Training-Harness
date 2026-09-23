import { vi } from 'vitest'

/**
 * Map-backed stand-in for the browser CacheStorage — Node and happy-dom expose
 * none. Real semantics for what utils/session.ts and the tests use, no spies:
 * a test proves a cache is gone by asking `has()`, not by counting calls.
 */
class FakeCache {
  private readonly entries = new Map<string, Response>()
  async put(request: RequestInfo | URL, response: Response): Promise<void> {
    this.entries.set(keyOf(request), response)
  }
  async match(request: RequestInfo | URL): Promise<Response | undefined> {
    return this.entries.get(keyOf(request))
  }
}

function keyOf(request: RequestInfo | URL): string {
  return typeof request === 'string' ? request : request instanceof URL ? request.href : request.url
}

export class FakeCacheStorage {
  private readonly stores = new Map<string, FakeCache>()
  async open(name: string): Promise<FakeCache> {
    let c = this.stores.get(name)
    if (!c) {
      c = new FakeCache()
      this.stores.set(name, c)
    }
    return c
  }
  async has(name: string): Promise<boolean> {
    return this.stores.has(name)
  }
  async delete(name: string): Promise<boolean> {
    return this.stores.delete(name)
  }
  async keys(): Promise<string[]> {
    return [...this.stores.keys()]
  }
}

/** Installs a fresh fake as `caches` with one per-user entry and one asset entry seeded. */
export async function installSeededCaches(apiStateName: string): Promise<FakeCacheStorage> {
  const fake = new FakeCacheStorage()
  vi.stubGlobal('caches', fake)
  await (await fake.open(apiStateName)).put('http://api.test/api/v1/quests/daily', new Response('{"day_number":3}'))
  await (await fake.open('assets')).put('http://app.test/_nuxt/entry.js', new Response('// js'))
  return fake
}
