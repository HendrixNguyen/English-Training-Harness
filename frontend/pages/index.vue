<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue'
import Badge from '~/components/retro/Badge.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import DayBar from '~/components/retro/DayBar.vue'
import HpBar from '~/components/retro/HpBar.vue'
import QuestNode from '~/components/retro/QuestNode.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroToast from '~/components/retro/RetroToast.vue'
import SpeechBox from '~/components/retro/SpeechBox.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { useGrowthMoment } from '~/composables/useGrowthMoment'
import { useRetroToast } from '~/composables/useRetroToast'
import { useAuthStore } from '~/stores/auth'
import { usePetStore } from '~/stores/pet'
import type { QuestTask } from '~/stores/quest'
import { useQuestStore } from '~/stores/quest'
import { shieldSpentLine, speechLine } from '~/utils/plant'

const quest = useQuestStore()
const pet = usePetStore()
const auth = useAuthStore()
const { start, chips, react, pulse, displayHealth, displayStage } = useGrowthMoment()
const { show: showToast } = useRetroToast()

// Pinia state survives the SPA navigation from /learn/:id, so pet.status
// already holds the *after* values by this component's very first render —
// consumeDelta() must run before that render (not in onMounted, which fires
// after it) or the first paint shows the after-value, not the before-value
// the growth moment is built on. The connector only lights once the fresh
// GET /pet/status + /quests/daily land in onMounted, never optimistically.
start(pet.consumeDelta())

onMounted(() => {
  void Promise.all([pet.load(), quest.load()])
  if (typeof navigator !== 'undefined' && navigator.onLine === false) {
    showToast('Đang ngoại tuyến — tớ nhớ tiến độ giúp cậu.')
  }
})

const plantName = computed(() => pet.status?.plant_name?.trim() || 'Mầm Non')

/**
 * design Addendum H2 / plan "Order constraint": the streak-shield branch
 * (`pet.status.shields` / `last_shield_used_on`) has not merged into
 * `PetStatus`, so the rack binds to this local, structural guard instead —
 * it renders the moment the field starts arriving from the API, and the
 * follow-up commit on that branch swaps this for a typed field.
 */
const shields = computed(() => {
  const s = (pet.status as { shields?: unknown } | null)?.shields
  return typeof s === 'number' ? s : null
})
const lastShieldUsedOn = computed(() => {
  const s = (pet.status as { last_shield_used_on?: unknown } | null)?.last_shield_used_on
  return typeof s === 'string' ? s : null
})
const spentLine = computed(() => (lastShieldUsedOn.value ? shieldSpentLine(lastShieldUsedOn.value) : null))
const spentDateLabel = computed(() => {
  const [, mm, dd] = (lastShieldUsedOn.value ?? '').split('-')
  return mm && dd ? `${dd}/${mm}` : null
})
const shieldsAriaLabel = computed(() => {
  if (shields.value === null) return ''
  const base = `Khiên: ${shields.value} trên 2`
  return spentLine.value && spentDateLabel.value ? `${base}, một chiếc vừa đỡ cho ngày ${spentDateLabel.value}` : base
})

const bubble = computed(() => (pet.status
  ? speechLine({
      stage: pet.status.stage,
      health: displayHealth(pet.status.health_points),
      targetMet: quest.targetMet,
      accumulatedSeconds: quest.accumulatedSeconds,
      streak: pet.status.current_streak,
      lastPracticedAt: pet.status.last_practiced_at,
      shields: shields.value ?? undefined,
      dayNumber: quest.daily?.day_number,
      name: plantName.value,
    })
  : ''))

function rowState(taskId: string, completed: boolean): 'done' | 'next' | 'locked' | 'open' {
  if (completed) return 'done'
  if (quest.targetMet) return 'open' // extra study is allowed once the day is met
  return taskId === quest.nextTaskId ? 'next' : 'locked'
}

/** `QuestNode` has no `next` state of its own — the current tile IS `next`. */
function nodeState(task: QuestTask): 'done' | 'current' | 'open' | 'locked' {
  const s = rowState(task.id, task.is_completed)
  return s === 'next' ? 'current' : s
}

/** design §1/§3: lit down to the current tile, dim after; the last tile has no trailing connector. */
function connectorFor(index: number): 'lit' | 'dim' | 'none' {
  const tasks = quest.sortedTasks
  if (index >= tasks.length - 1) return 'none'
  return tasks[index].is_completed ? 'lit' : 'dim'
}

function onEnter(taskId: string) {
  navigateTo(`/learn/${taskId}`)
}

// Vue's template compiler resolves a bare identifier through `_ctx`, not the
// module scope, so calling the auto-imported `navigateTo`/`useApi`-style
// global directly from a template attribute silently resolves to nothing —
// every navigation is routed through a plain script function instead.
function goToRoadmap() {
  navigateTo('/roadmap')
}
function goToRevive() {
  navigateTo('/revive')
}
function goToOnboarding() {
  navigateTo('/onboarding')
}

const allTasksDone = computed(() => quest.sortedTasks.length > 0 && quest.sortedTasks.every(t => t.is_completed))
const currentTaskNumber = computed(() => {
  const idx = quest.sortedTasks.findIndex(t => t.id === quest.nextTaskId)
  return idx === -1 ? quest.sortedTasks.length : idx + 1
})
const bottomLabel = computed(() => (allTasksDone.value ? 'Xem hành trình' : `Vào nhiệm vụ ${currentTaskNumber.value} →`))
/** design §5 acceptance 2: hidden only while the quest region has never loaded — a cached `daily` still shows it. */
const showBottomBar = computed(() => quest.daily !== null)

function onBottomButton() {
  if (allTasksDone.value) navigateTo('/roadmap')
  else if (quest.nextTaskId) navigateTo(`/learn/${quest.nextTaskId}`)
}

// --- H4 avatar menu (stopgap until plan 6b's Inn settings screen) ---
const menuOpen = ref(false)
const menuFocusIndex = ref(0)
const avatarEl = ref<HTMLButtonElement | null>(null)
const menuItemEls = ref<(HTMLButtonElement | null)[]>([null, null])

function setMenuItemRef(el: Element | null, i: number) {
  menuItemEls.value[i] = el as HTMLButtonElement | null
}

function focusMenuItem(i: number) {
  const n = menuItemEls.value.length
  const idx = ((i % n) + n) % n
  menuFocusIndex.value = idx
  menuItemEls.value[idx]?.focus()
}

function openMenu() {
  menuOpen.value = true
  menuFocusIndex.value = 0
  void nextTick(() => focusMenuItem(0))
}

function closeMenu(refocusAvatar = true) {
  menuOpen.value = false
  if (refocusAvatar) void nextTick(() => avatarEl.value?.focus())
}

function toggleMenu() {
  if (menuOpen.value) closeMenu()
  else openMenu()
}

function onMenuKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') { e.preventDefault(); closeMenu() }
  else if (e.key === 'ArrowDown') { e.preventDefault(); focusMenuItem(menuFocusIndex.value + 1) }
  else if (e.key === 'ArrowUp') { e.preventDefault(); focusMenuItem(menuFocusIndex.value - 1) }
}

function onMenuFocusOut(e: FocusEvent) {
  const related = e.relatedTarget as Node | null
  const container = e.currentTarget as HTMLElement
  if (!related || !container.contains(related)) closeMenu(false)
}

async function selectSettings() {
  closeMenu(false)
  await navigateTo('/settings')
}

async function selectSignOut() {
  closeMenu(false)
  await auth.signOut()
  await navigateTo('/login')
}
</script>

<template>
  <main class="relative mx-auto min-h-screen max-w-md bg-ground-0 px-4 pb-28 pt-4 text-ink-0">
    <div class="relative flex h-12 items-center justify-between gap-2">
      <div class="flex min-w-0 items-center gap-2">
        <button
          ref="avatarEl"
          type="button"
          class="flex h-10 w-10 shrink-0 items-center justify-center border-2 border-line-lit bg-ground-2 font-display text-xl leading-6 text-ink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch"
          aria-haspopup="menu"
          :aria-expanded="menuOpen"
          :aria-label="`Tài khoản ${auth.user?.full_name ?? ''}`"
          @click="toggleMenu"
        >
          {{ auth.initial }}
        </button>
        <span class="truncate font-display text-xl leading-6 text-ink-0">{{ auth.user?.full_name }}</span>

        <div
          v-if="menuOpen"
          role="menu"
          class="absolute left-0 top-12 z-20 w-44 border-2 border-line-lit bg-ground-1"
          @keydown="onMenuKeydown"
          @focusout="onMenuFocusOut"
        >
          <button
            :ref="(el) => setMenuItemRef(el as Element | null, 0)"
            type="button"
            role="menuitem"
            class="flex h-12 w-full items-center gap-2 border-b-2 border-line-dim px-3 font-display text-xl leading-6 text-ink-0"
            @click="selectSettings"
          >
            <span class="w-3 text-torch" aria-hidden="true">{{ menuFocusIndex === 0 ? '▶' : '' }}</span>
            Cài đặt
          </button>
          <button
            :ref="(el) => setMenuItemRef(el as Element | null, 1)"
            type="button"
            role="menuitem"
            class="flex h-12 w-full items-center gap-2 px-3 font-display text-xl leading-6 text-ink-0"
            @click="selectSignOut"
          >
            <span class="w-3 text-torch" aria-hidden="true">{{ menuFocusIndex === 1 ? '▶' : '' }}</span>
            Đăng xuất
          </button>
        </div>
      </div>

      <div class="flex shrink-0 items-center gap-2">
        <button
          type="button"
          class="focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch"
          @click="goToRoadmap"
        >
          <Badge kind="streak" :count="pet.status?.current_streak ?? 0" :pulse="pulse" />
        </button>
        <span v-if="shields !== null" class="flex items-center gap-2" role="img" :aria-label="shieldsAriaLabel">
          <Badge kind="shield" :earned="shields > 0" />
          <Badge kind="shield" :earned="shields > 1" />
        </span>
      </div>
    </div>

    <RetroPanel v-if="pet.loading && !pet.status" class="mt-4">
      <StateBlock state="loading" />
    </RetroPanel>
    <StateBlock
      v-else-if="pet.error && !pet.status"
      class="mt-4"
      state="error"
      :message="`Không tải được ${plantName}. Thử lại.`"
      action="Thử lại"
      @action="pet.load()"
    />
    <RetroPanel v-else-if="pet.status" class="mt-4" :speaker="plantName" :tone="pet.isWilted ? 'ember' : 'plain'">
      <template #portrait>
        <CompanionSprite
          :stage="displayStage(pet.status.stage)"
          :health="displayHealth(pet.status.health_points)"
          :size="128"
          :react="pet.isWilted ? 'down' : react"
        />
      </template>
      <div aria-live="polite" class="flex flex-wrap gap-2">
        <span
          v-for="c in chips"
          :key="c.tone"
          data-chip
          class="retro-rise border-2 border-line-dim bg-ground-2 px-2 font-display text-xl leading-6"
          :class="c.tone === 'growth' ? 'text-growth' : 'text-torch'"
        >{{ c.text }}</span>
      </div>
      <HpBar class="mt-2" :value="displayHealth(pet.status.health_points)" :max="100" />
      <p v-if="spentLine" class="mt-1 font-body text-sm text-ink-1">{{ spentLine }}</p>
      <SpeechBox
        v-if="!pet.isWilted"
        class="mt-2"
        :line="bubble"
        :name="plantName"
        :stage="displayStage(pet.status.stage)"
        :health="displayHealth(pet.status.health_points)"
      />
      <RetroButton v-else class="mt-3" variant="danger" block @click="goToRevive">
        Hồi sinh {{ plantName }}
      </RetroButton>
    </RetroPanel>

    <template v-if="quest.noRoadmap">
      <RetroPanel class="mt-4">
        <StateBlock state="empty" message="Cậu chưa có hành trình." action="Bắt đầu hành trình 28 ngày" @action="goToOnboarding" />
      </RetroPanel>
    </template>
    <template v-else>
      <RetroPanel v-if="quest.loading && !quest.daily" class="mt-4">
        <StateBlock state="loading" />
      </RetroPanel>
      <StateBlock
        v-else-if="quest.error && !quest.daily"
        class="mt-4"
        state="error"
        message="Không tải được nhiệm vụ. Thử lại."
        action="Thử lại"
        @action="quest.load()"
      />
      <template v-else-if="quest.daily">
        <p class="mt-4 font-display text-base uppercase leading-5 tracking-[0.05em] text-ink-1">
          Phòng hôm nay
        </p>
        <DayBar :value-seconds="quest.accumulatedSeconds" :met="quest.targetMet" />

        <ul class="mt-4">
          <QuestNode
            v-for="(task, i) in quest.sortedTasks"
            :key="task.id"
            :task="task"
            :index="i"
            :state="nodeState(task)"
            :connector="connectorFor(i)"
            @enter="onEnter"
          />
        </ul>
      </template>
    </template>

    <div
      v-if="showBottomBar"
      data-bottom-bar
      class="fixed inset-x-0 bottom-0 z-10 mx-auto max-w-md border-t-2 border-line-dim bg-ground-0 p-4"
      style="padding-bottom: calc(16px + env(safe-area-inset-bottom))"
    >
      <RetroButton block @click="onBottomButton">
        {{ bottomLabel }}
      </RetroButton>
    </div>

    <RetroToast />
  </main>
</template>
