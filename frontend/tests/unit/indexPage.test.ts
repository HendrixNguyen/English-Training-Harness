import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick, ref } from 'vue'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import Badge from '~/components/retro/Badge.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import DayBar from '~/components/retro/DayBar.vue'
import HpBar from '~/components/retro/HpBar.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import { useRetroToast } from '~/composables/useRetroToast'
import { useAuthStore } from '~/stores/auth'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'

const api = { get: vi.fn(), post: vi.fn() }
vi.mock('~/composables/useApi', () => ({ useApi: () => api }))
const navigateTo = vi.fn()
vi.stubGlobal('navigateTo', navigateTo)

const { default: IndexPage } = await import('~/pages/index.vue')

const STATUS_BEFORE = { plant_name: 'Mầm Non', health_points: 80, stage: 'sprout', current_streak: 5, last_practiced_at: '2026-09-24T20:00:00Z' }
const STATUS_AFTER = { ...STATUS_BEFORE, health_points: 100, stage: 'sapling', current_streak: 6 }
const WILTED_STATUS = { plant_name: 'Mầm Non', health_points: 0, stage: 'wilted', current_streak: 0, last_practiced_at: null }

const TASKS = [
  { id: 'ex-1', task_type: 'vocabulary', title: 'Từ vựng', duration_minutes: 10, is_completed: false, content_json: {} },
  { id: 'ex-2', task_type: 'reading', title: 'Đọc hiểu', duration_minutes: 10, is_completed: false, content_json: {} },
  { id: 'ex-3', task_type: 'practice', title: 'Viết phản hồi', duration_minutes: 10, is_completed: false, content_json: {} },
]
const DAILY_MET = {
  date: '2026-09-25',
  day_number: 3,
  total_minutes_required: 30,
  accumulated_seconds: 1800,
  is_target_met: true,
  tasks: TASKS.map(t => ({ ...t, is_completed: true })),
}
const DAILY_PARTWAY = {
  ...DAILY_MET,
  accumulated_seconds: 600,
  is_target_met: false,
  tasks: TASKS.map((t, i) => ({ ...t, is_completed: i === 0 })), // task 1 done, task 2 current, task 3 locked
}
/** Met via extra time on task 1, with tasks 2-3 still open (not yet completed). */
const DAILY_MET_TASKS_REMAIN = { ...DAILY_PARTWAY, accumulated_seconds: 1800, is_target_met: true }

function routeGet(status: unknown, daily: unknown) {
  api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? Promise.resolve(status) : Promise.resolve(daily)))
}

function setAuthUser(name = 'Hendrix Nguyen') {
  const auth = useAuthStore()
  auth.user = { id: 'u1', email: 'h@example.com', full_name: name, cefr_current: null }
}

function mountPage(attach = false) {
  return mount(IndexPage, attach ? { attachTo: document.body } : {})
}

beforeEach(() => {
  setActivePinia(createPinia())
  api.get.mockReset()
  api.post.mockReset()
  navigateTo.mockClear()
  vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
  vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { cb(0); return 1 })
  Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
})

afterEach(() => {
  document.body.innerHTML = ''
  vi.useRealTimers()
  // `navigateTo` is stubbed once at module load (mirrors the real global
  // auto-import) — only drop the per-test `requestAnimationFrame` stub, or
  // every test after the first loses `navigateTo` and throws a ReferenceError.
  vi.unstubAllGlobals()
  vi.stubGlobal('navigateTo', navigateTo)
})

describe('/ (design retro-hub.md) layout — acceptance 1, 4', () => {
  it('renders the companion, HP bar, day bar and three quest tiles from pet/status + quests/daily', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const sprite = w.findComponent(CompanionSprite)
    expect(sprite.exists()).toBe(true)
    expect(sprite.props('stage')).toBe('sprout')

    expect(w.findComponent(HpBar).exists()).toBe(true)
    expect(w.text()).toContain('HP 80/100')

    expect(w.findComponent(DayBar).exists()).toBe(true)
    expect(w.findAllComponents(QuestNode)).toHaveLength(3)
  })
})

describe('/ bottom button — acceptance 2', () => {
  it('is absent while the quest region has never loaded', async () => {
    setAuthUser()
    const pet = usePetStore()
    pet.status = { ...STATUS_BEFORE }
    api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? Promise.resolve(STATUS_BEFORE) : new Promise(() => {})))

    const w = mountPage()
    await nextTick()

    expect(w.find('[data-bottom-bar]').exists()).toBe(false)
  })

  it('reads "Vào nhiệm vụ 2 →" for the current task and navigates to it on click', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const bar = w.find('[data-bottom-bar]')
    expect(bar.text()).toContain('Vào nhiệm vụ 2 →')
    await bar.findComponent(RetroButton).trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
  })

  it('reads "Xem hành trình" and navigates to /roadmap when all three tasks are complete', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_MET
    routeGet(STATUS_BEFORE, DAILY_MET)

    const w = mountPage()
    await flushPromises()

    const bar = w.find('[data-bottom-bar]')
    expect(bar.text()).toContain('Xem hành trình')
    await bar.findComponent(RetroButton).trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/roadmap')
  })
})

describe('/ quest tiles — acceptance 3, 5', () => {
  it('maps the next task to "current", keeps the connector lit down to it and dim after', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const nodes = w.findAllComponents(QuestNode)
    expect(nodes[0].props('state')).toBe('done')
    expect(nodes[1].props('state')).toBe('current')
    expect(nodes[2].props('state')).toBe('locked')
    expect(nodes[0].props('connector')).toBe('lit')
    expect(nodes[1].props('connector')).toBe('dim')
    expect(nodes[2].props('connector')).toBe('none')
  })

  it('a locked tile does not navigate; an open/current tile does', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const nodes = w.findAllComponents(QuestNode)
    await nodes[2].find('button').trigger('click') // locked
    expect(navigateTo).not.toHaveBeenCalled()

    await nodes[1].find('button').trigger('click') // current
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
  })

  it('all three complete: every connector lights, remaining (none here) would be "open"', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_MET
    routeGet(STATUS_BEFORE, DAILY_MET)

    const w = mountPage()
    await flushPromises()

    const nodes = w.findAllComponents(QuestNode)
    expect(nodes.map(n => n.props('state'))).toEqual(['done', 'done', 'done'])
    expect(nodes[0].props('connector')).toBe('lit')
    expect(nodes[1].props('connector')).toBe('lit')
  })
})

describe('/ wilted — acceptance 6', () => {
  it('shows the ember panel, the down sprite, the revive button, and keeps the path tappable', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...WILTED_STATUS }
    quest.daily = DAILY_PARTWAY
    routeGet(WILTED_STATUS, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    expect(w.findComponent(RetroPanel).props('tone')).toBe('ember')
    expect(w.findComponent(CompanionSprite).props('react')).toBe('down')
    expect(w.find('[role="meter"]').attributes('aria-valuenow')).toBe('0')
    const revive = w.findComponent(RetroButton)
    expect(revive.text()).toContain('Hồi sinh Mầm Non')
    await revive.trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/revive')

    const nodes = w.findAllComponents(QuestNode)
    await nodes[1].find('button').trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/learn/ex-2')
  })
})

describe('/ empty roadmap — acceptance 1 (states)', () => {
  it('shows the empty-roadmap invitation while the companion panel still renders', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.noRoadmap = true
    api.get.mockImplementation((path: string) =>
      path.endsWith('/pet/status') ? Promise.resolve(STATUS_BEFORE) : Promise.reject(Object.assign(new Error('no roadmap'), { code: 'no_active_roadmap' })))

    const w = mountPage()
    await flushPromises()

    expect(w.findComponent(CompanionSprite).exists()).toBe(true)
    expect(w.text()).toContain('Cậu chưa có hành trình.')
    const btn = w.findAllComponents(RetroButton).filter(b => b.text().includes('Bắt đầu hành trình 28 ngày'))[0]
    await btn.trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/onboarding')
  })
})

describe('/ errors per region', () => {
  it('pet error retries only pet.load', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.error = 'network_error'
    quest.daily = DAILY_PARTWAY
    const petLoad = vi.spyOn(pet, 'load').mockResolvedValue()
    const questLoad = vi.spyOn(quest, 'load').mockResolvedValue()

    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Không tải được Mầm Non. Thử lại.')
    const retry = w.findAllComponents(RetroButton).find(b => b.text().includes('Thử lại'))!
    petLoad.mockClear()
    questLoad.mockClear() // onMounted already called both once — isolate the retry click's own effect
    await retry.trigger('click')
    expect(petLoad).toHaveBeenCalledTimes(1)
    expect(questLoad).not.toHaveBeenCalled()
  })

  it('quest error retries only quest.load', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.error = 'network_error'
    const petLoad = vi.spyOn(pet, 'load').mockResolvedValue()
    const questLoad = vi.spyOn(quest, 'load').mockResolvedValue()

    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Không tải được nhiệm vụ. Thử lại.')
    const retry = w.findAllComponents(RetroButton).find(b => b.text().includes('Thử lại'))!
    petLoad.mockClear()
    questLoad.mockClear()
    await retry.trigger('click')
    expect(questLoad).toHaveBeenCalledTimes(1)
    expect(petLoad).not.toHaveBeenCalled()
  })
})

describe('/ offline — acceptance 7', () => {
  it('renders cached data and shows exactly one toast', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })

    const w = mountPage()
    await flushPromises()

    expect(w.findComponent(CompanionSprite).exists()).toBe(true)
    const { queue } = useRetroToast()
    expect(queue.value).toHaveLength(1)
    expect(queue.value[0].line).toBe('Đang ngoại tuyến — tớ nhớ tiến độ giúp cậu.')
  })
})

describe('/ done — day fully met', () => {
  it('shows the met caption, remaining tiles open, and still points at the next task while any remain', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = { ...DAILY_MET_TASKS_REMAIN }
    routeGet(STATUS_BEFORE, quest.daily)

    const w = mountPage()
    await flushPromises()

    expect(w.text()).toContain('Phòng hôm nay đã xong')
    const nodes = w.findAllComponents(QuestNode)
    expect(nodes[0].props('state')).toBe('done')
    expect(nodes[1].props('state')).toBe('open')
    expect(nodes[2].props('state')).toBe('open')
    expect(w.find('[data-bottom-bar]').text()).toContain('Vào nhiệm vụ 2 →')
  })

  it('reads "Xem hành trình" once every task is complete', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = { ...DAILY_MET }
    routeGet(STATUS_BEFORE, quest.daily)

    const w = mountPage()
    await flushPromises()

    expect(w.find('[data-bottom-bar]').text()).toContain('Xem hành trình')
  })
})

describe('/ status bar — design Addendum H2', () => {
  it('no rack and no spent caption when shields is absent', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    expect(w.findAllComponents(Badge).filter(b => b.props('kind') === 'shield')).toHaveLength(0)
    expect(w.text()).not.toContain('Khiên đã đỡ')
  })

  it('with shields: 1, draws two shield Badges (one earned) with the aria-label', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    const status = { ...STATUS_BEFORE, shields: 1 }
    pet.status = status as typeof STATUS_BEFORE
    quest.daily = DAILY_PARTWAY
    routeGet(status, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const shieldBadges = w.findAllComponents(Badge).filter(b => b.props('kind') === 'shield')
    expect(shieldBadges).toHaveLength(2)
    expect(shieldBadges[0].props('earned')).toBe(true)
    expect(shieldBadges[1].props('earned')).toBe(false)
    expect(w.find('[role="img"][aria-label^="Khiên: 1 trên 2"]').exists()).toBe(true)
  })

  it('shows the spent caption under the HpBar when last_shield_used_on is a few days old', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    const today = new Date()
    const threeDaysAgo = new Date(today.getTime() - 3 * 86_400_000)
    const p = (n: number) => String(n).padStart(2, '0')
    const ymd = `${threeDaysAgo.getFullYear()}-${p(threeDaysAgo.getMonth() + 1)}-${p(threeDaysAgo.getDate())}`
    const status = { ...STATUS_BEFORE, shields: 1, last_shield_used_on: ymd }
    pet.status = status as typeof STATUS_BEFORE
    quest.daily = DAILY_PARTWAY
    routeGet(status, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const dd = p(threeDaysAgo.getDate())
    const mm = p(threeDaysAgo.getMonth() + 1)
    expect(w.text()).toContain(`Khiên đã đỡ cho ngày ${dd}/${mm}.`)
  })

  it('the streak badge click navigates to /roadmap; the name truncates but the full name is in the avatar aria-label', async () => {
    setAuthUser('Nguyễn Đình Gia Huy Rất Là Dài')
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    const nameEl = w.find('span.truncate')
    expect(nameEl.classes()).toContain('truncate')
    expect(w.find('[aria-haspopup="menu"]').attributes('aria-label')).toBe('Tài khoản Nguyễn Đình Gia Huy Rất Là Dài')

    const streakButton = w.findComponent(Badge).element.closest('button')!
    await streakButton.dispatchEvent(new Event('click'))
    expect(navigateTo).toHaveBeenCalledWith('/roadmap')
  })
})

describe('/ avatar menu — design Addendum H4', () => {
  it('opens a role=menu with the two items, focusing the first; Esc closes it and returns focus to the avatar', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage(true)
    await flushPromises()

    const avatar = w.find('[aria-haspopup="menu"]')
    expect(avatar.attributes('aria-expanded')).toBe('false')
    await avatar.trigger('click')
    await nextTick()

    expect(avatar.attributes('aria-expanded')).toBe('true')
    const menu = w.find('[role="menu"]')
    expect(menu.exists()).toBe(true)
    const items = menu.findAll('[role="menuitem"]')
    expect(items).toHaveLength(2)
    expect(items[0].text()).toContain('Cài đặt')
    expect(items[1].text()).toContain('Đăng xuất')
    expect(document.activeElement).toBe(items[0].element)

    await menu.trigger('keydown', { key: 'Escape' })
    await nextTick()
    expect(w.find('[role="menu"]').exists()).toBe(false)
    expect(document.activeElement).toBe(avatar.element)
    w.unmount()
  })

  it('ArrowDown/ArrowUp move focus and wrap', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage(true)
    await flushPromises()
    await w.find('[aria-haspopup="menu"]').trigger('click')
    await nextTick()

    const menu = w.find('[role="menu"]')
    let items = menu.findAll('[role="menuitem"]')
    expect(document.activeElement).toBe(items[0].element)

    await menu.trigger('keydown', { key: 'ArrowDown' })
    await nextTick()
    items = menu.findAll('[role="menuitem"]')
    expect(document.activeElement).toBe(items[1].element)

    await menu.trigger('keydown', { key: 'ArrowDown' }) // wraps back to item 0
    await nextTick()
    items = menu.findAll('[role="menuitem"]')
    expect(document.activeElement).toBe(items[0].element)

    await menu.trigger('keydown', { key: 'ArrowUp' }) // wraps to the last item
    await nextTick()
    items = menu.findAll('[role="menuitem"]')
    expect(document.activeElement).toBe(items[1].element)
    w.unmount()
  })

  it('"Cài đặt" navigates to /settings; "Đăng xuất" signs out then navigates to /login', async () => {
    setAuthUser()
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)
    const auth = useAuthStore()
    const signOut = vi.spyOn(auth, 'signOut').mockResolvedValue()

    const w = mountPage()
    await flushPromises()
    await w.find('[aria-haspopup="menu"]').trigger('click')
    await nextTick()

    const menu = w.find('[role="menu"]')
    const items = menu.findAll('[role="menuitem"]')
    await items[0].trigger('click')
    expect(navigateTo).toHaveBeenCalledWith('/settings')

    navigateTo.mockClear()
    await w.find('[aria-haspopup="menu"]').trigger('click')
    await nextTick()
    const items2 = w.find('[role="menu"]').findAll('[role="menuitem"]')
    await items2[1].trigger('click')
    await flushPromises()
    expect(signOut).toHaveBeenCalledTimes(1)
    expect(navigateTo).toHaveBeenCalledWith('/login')
  })
})

describe('/ reduced motion — acceptance 8', () => {
  afterEach(() => {
    vi.doUnmock('~/composables/useReducedMotion')
    vi.resetModules()
  })

  it('under reduced motion no retro-* animation class runs, and growth-moment chips still show', async () => {
    vi.resetModules()
    vi.doMock('~/composables/useReducedMotion', () => ({ useReducedMotion: () => ref(true) }))
    const { default: IndexPageReduced } = await import('~/pages/index.vue')
    const { setActivePinia: setActivePiniaR, createPinia: createPiniaR } = await import('pinia')
    setActivePiniaR(createPiniaR())
    const { usePetStore: usePetStoreR } = await import('~/stores/pet')
    const { useQuestStore: useQuestStoreR } = await import('~/stores/quest')
    const { useAuthStore: useAuthStoreR } = await import('~/stores/auth')

    const auth = useAuthStoreR()
    auth.user = { id: 'u1', email: 'h@example.com', full_name: 'H', cefr_current: null }
    const pet = usePetStoreR()
    const quest = useQuestStoreR()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    api.get.mockImplementation((path: string) => (path.endsWith('/pet/status') ? Promise.resolve(STATUS_AFTER) : Promise.resolve(DAILY_MET)))

    const w = mount(IndexPageReduced)
    await flushPromises()

    expect(w.html()).not.toMatch(/retro-(breath|hop|shake|flash|dots|blink|rise|badge-pulse)/)
    expect(w.text()).toMatch(/HP|máu/)
  })
})

describe('/ growth moment (adapted from wireframe 7.2)', () => {
  it('shows chest chips and a "hit" reaction after a task that met the target with no stage change', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE, stage: 'sapling' }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet({ ...STATUS_AFTER, stage: 'sapling' }, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(400)
    await nextTick()

    const chipTexts = w.findAll('[data-chip]').map(c => c.text())
    expect(chipTexts).toContain('+20 HP')
    expect(chipTexts).toContain('x6')
    expect(w.findComponent(CompanionSprite).attributes('data-react')).toBe('hit')
  })

  it('reacts "levelup" when the stage changed', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    vi.advanceTimersByTime(400)
    await nextTick()

    expect(w.findComponent(CompanionSprite).attributes('data-react')).toBe('levelup')
  })

  it('shows the after-values once the growth moment settles', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()

    expect(w.find('[role="meter"]').attributes('aria-valuenow')).toBe('100')
    expect(w.findComponent(CompanionSprite).props('stage')).toBe('sapling')
  })

  it('shows no chips and idle react on a plain load', async () => {
    const quest = useQuestStore()
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    expect(w.findAll('[data-chip]')).toHaveLength(0)
    expect(w.findComponent(CompanionSprite).attributes('data-react')).toBe('idle')
    expect(w.text()).toContain('Còn 20 phút nữa là xong phòng hôm nay!')
  })

  it('a task that did not meet the target celebrates nothing', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 80, streak_count: 5, targetMetChanged: false })
    quest.daily = DAILY_PARTWAY
    routeGet(STATUS_BEFORE, DAILY_PARTWAY)

    const w = mountPage()
    await flushPromises()

    expect(w.findAll('[data-chip]')).toHaveLength(0)
    expect(w.findComponent(CompanionSprite).attributes('data-react')).toBe('idle')
  })

  it('is consumed once — a second mount shows nothing', async () => {
    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    await flushPromises()
    w.unmount()

    const w2 = mountPage()
    await flushPromises()
    expect(w2.findAll('[data-chip]')).toHaveLength(0)
  })

  it('HpBar shows the before-value for the first paint, then the live value', async () => {
    let pending: FrameRequestCallback | null = null
    vi.stubGlobal('requestAnimationFrame', (cb: FrameRequestCallback) => { pending = cb; return 1 })

    const pet = usePetStore()
    const quest = useQuestStore()
    pet.status = { ...STATUS_BEFORE }
    pet.applyProgress({ pet_health: 100, streak_count: 6, targetMetChanged: true })
    quest.daily = DAILY_MET
    routeGet(STATUS_AFTER, DAILY_MET)

    const w = mountPage()
    expect(w.findComponent(HpBar).props('value')).toBe(80)
    pending!(0)
    await nextTick()
    expect(w.findComponent(HpBar).props('value')).toBe(100)
  })
})
