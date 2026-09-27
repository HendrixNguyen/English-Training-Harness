<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { REVIVE_SECONDS, usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { daysSince } from '~/utils/plant'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import HpBar from '~/components/retro/HpBar.vue'
import DayBar from '~/components/retro/DayBar.vue'
import RetroToast from '~/components/retro/RetroToast.vue'
import StateBlock from '~/components/ui/StateBlock.vue'
import { useRetroToast } from '~/composables/useRetroToast'

/**
 * `/revive` — the KO screen (design harness/designs/retro-revive.md, kit
 * v2). Deviation from the plan's own Step 1 example prose: the missed-days
 * line uses the design §7 wording verbatim ("... ngày liền.") — the design
 * doc, not the plan's paraphrase, is the copy source of truth.
 */

const pet = usePetStore()
const quest = useQuestStore()
const toast = useRetroToast()

const busy = ref(false)
const reviveError = ref<string | null>(null)
const passed = ref(false)
/** `levelup` plays once on a pass, then the sprite settles to `idle` on
 * `@reacted` (design §2, §3: down -> levelup -> sprout). */
const reactState = ref<'levelup' | 'idle'>('levelup')

const online = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
function setOnline() { online.value = navigator.onLine }

onMounted(async () => {
  pet.hydrateChallenge()
  window.addEventListener('online', setOnline)
  window.addEventListener('offline', setOnline)
  await Promise.all([pet.status ? Promise.resolve() : pet.load(), quest.daily ? Promise.resolve() : quest.load()])
})

onBeforeUnmount(() => {
  window.removeEventListener('online', setOnline)
  window.removeEventListener('offline', setOnline)
})

const plantName = computed(() => pet.status?.plant_name || 'Mầm Non')
const health = computed(() => pet.status?.health_points ?? 100)
const spriteStage = computed(() => pet.status?.stage ?? 'sprout')

const missedDays = computed(() => daysSince(pet.status?.last_practiced_at ?? null))
const missedLine = computed(() =>
  missedDays.value === null
    ? 'Cậu bỏ tớ lâu quá. Học 15 phút để tớ đứng dậy nhé.'
    : `Cậu bỏ tớ ${missedDays.value} ngày liền. Học 15 phút để tớ đứng dậy nhé.`)

const today = computed(() => quest.daily?.date ?? new Date().toISOString().slice(0, 10))
const challengeActive = computed(() => pet.challenge !== null && pet.challenge.date === today.value)
const progress = computed(() => pet.challengeProgress(quest.accumulatedSeconds))
const canCheck = computed(() => progress.value >= REVIVE_SECONDS)
const remainingMinutes = computed(() => Math.max(0, Math.ceil((REVIVE_SECONDS - progress.value) / 60)))

// Executor trap (design §8): the alarm only on real wilted data — never on
// a failed load with nothing cached, and never guessed from `notWilted`.
const isWiltedReal = computed(() => pet.status !== null && pet.isWilted)
// `!passed`: a fresh healthy load (design's "Not wilted" state) and "just
// revived" both read as pet.isWilted === false — passed.value is what tells
// them apart, so a revive doesn't fall through to the plain healthy button
// once the store's own health/stage catch up.
const isHealthy = computed(() => !passed.value && (pet.notWilted || (pet.status !== null && !pet.isWilted)))
const isUnknown = computed(() => pet.status === null && pet.error !== null)

const showBand = computed(() => isWiltedReal.value && !passed.value)

const primaryLabel = computed(() => {
  if (passed.value) return 'Về trại'
  if (!challengeActive.value) return 'Hồi sinh (15 phút)'
  return canCheck.value ? 'Hồi sinh' : 'Vào học ngay'
})
const primaryVariant = computed(() => (challengeActive.value || passed.value ? 'primary' : 'danger'))
/** `home` covers both "Vào học ngay" and "Về trại" — both just navigate to
 * '/'; only the label differs. `navigateTo` stays a template-only call (like
 * every other kit page) so the Vitest `global.mocks` stub reaches it. */
const primaryAction = computed<'revive' | 'home'>(() => {
  if (passed.value) return 'home'
  if (!challengeActive.value) return 'revive'
  return canCheck.value ? 'revive' : 'home'
})

async function revive() {
  if (!online.value) {
    toast.show('Cần mạng để hồi sinh. Tớ vẫn đếm phút cho cậu.', 'ember')
    return
  }
  busy.value = true
  reviveError.value = null
  try {
    const res = await pet.revive(today.value, quest.accumulatedSeconds)
    if (res?.revival_passed) {
      passed.value = true
      reactState.value = 'levelup'
    }
  } catch {
    reviveError.value = 'Chưa bắt đầu được nhiệm vụ hồi sinh. Thử lại.'
  } finally {
    busy.value = false
  }
}

function onReacted() {
  reactState.value = 'idle'
}
</script>

<template>
  <main class="min-h-screen bg-ground-0 mx-auto max-w-md px-4 pb-8">
    <RetroPanel v-if="showBand" band tone="ember" data-band class="mt-4 text-center">
      <p class="font-display text-[22px] uppercase leading-6 text-ink-0">
        {{ plantName }} đã gục
      </p>
    </RetroPanel>

    <template v-if="passed">
      <RetroPanel tone="growth" :speaker="plantName" class="mt-4">
        <template #portrait>
          <CompanionSprite :stage="spriteStage" :health="health" :size="128" :react="reactState" @reacted="onReacted" />
        </template>
        <HpBar :value="health" class="mt-2" />
        <p class="mt-2 font-body text-[17px] text-ink-0">
          Tớ dậy rồi. Cảm ơn cậu.
        </p>
      </RetroPanel>
    </template>

    <template v-else-if="isHealthy">
      <RetroPanel tone="growth" :speaker="plantName" class="mt-4">
        <template #portrait>
          <CompanionSprite :stage="spriteStage" :health="health" :size="128" react="idle" />
        </template>
        <HpBar :value="health" class="mt-2" />
        <p class="mt-2 font-body text-[17px] text-ink-0">
          {{ plantName }} vẫn khoẻ.
        </p>
      </RetroPanel>
    </template>

    <template v-else-if="isWiltedReal">
      <RetroPanel tone="ember" :speaker="plantName" class="mt-4">
        <template #portrait>
          <CompanionSprite stage="wilted" :health="health" :size="128" react="down" />
        </template>
        <HpBar :value="health" class="mt-2" />
        <p class="mt-2 font-body text-[17px] text-ink-0">
          {{ missedLine }}
        </p>
      </RetroPanel>

      <div v-if="challengeActive" class="mt-4">
        <div class="flex items-baseline justify-between">
          <span class="font-display text-xl uppercase leading-6 text-ink-1">Nhiệm vụ hồi sinh</span>
        </div>
        <DayBar class="mt-1" :value-seconds="progress" :segments="1" :segment-seconds="REVIVE_SECONDS" :met="canCheck" />
        <p class="mt-1 text-right text-sm" :class="canCheck ? 'text-growth' : 'text-ink-1'">
          {{ canCheck ? 'Đủ 15 phút rồi' : `Còn ${remainingMinutes} phút học nữa` }}
        </p>
      </div>

      <p v-if="reviveError" role="alert" class="mt-3 font-body text-sm text-ember">
        {{ reviveError }}
      </p>
    </template>

    <template v-else-if="isUnknown">
      <StateBlock
        class="mt-4"
        state="error"
        :message="`Không tải được trạng thái của ${plantName}. Chưa biết bạn ấy có gục không.`"
        action="Thử lại"
        @action="pet.load()"
      />
    </template>

    <template v-else>
      <StateBlock class="mt-4" state="loading" />
    </template>

    <div
      v-if="isWiltedReal || passed"
      class="sticky bottom-0 mt-4 border-t-2 border-line-lit bg-ground-1 p-4"
      :style="{ paddingBottom: 'calc(1rem + env(safe-area-inset-bottom))' }"
    >
      <RetroButton
        block
        :variant="primaryVariant"
        :loading="primaryAction === 'revive' && busy"
        @click="primaryAction === 'revive' ? revive() : navigateTo('/')"
      >
        {{ primaryLabel }}
      </RetroButton>
    </div>

    <RetroButton v-if="isHealthy" block class="mt-4" @click="navigateTo('/')">
      Về trại
    </RetroButton>

    <RetroToast />
  </main>
</template>
