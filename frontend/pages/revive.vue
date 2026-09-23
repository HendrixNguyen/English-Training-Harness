<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { REVIVE_SECONDS, usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { daysSince } from '~/utils/plant'

const pet = usePetStore()
const quest = useQuestStore()

const busy = ref(false)
const error = ref<string | null>(null)
const passed = ref(false)

onMounted(async () => {
  pet.hydrateChallenge()
  await Promise.all([pet.status ? Promise.resolve() : pet.load(), quest.daily ? Promise.resolve() : quest.load()])
})

const missedDays = computed(() => daysSince(pet.status?.last_practiced_at ?? null))
const today = computed(() => quest.daily?.date ?? new Date().toISOString().slice(0, 10))
const challengeActive = computed(() => pet.challenge !== null && pet.challenge.date === today.value)
const progress = computed(() => pet.challengeProgress(quest.accumulatedSeconds))
const canCheck = computed(() => progress.value >= REVIVE_SECONDS)

async function revive() {
  busy.value = true
  error.value = null
  try {
    const res = await pet.revive(today.value, quest.accumulatedSeconds)
    if (res?.revival_passed) passed.value = true
  } catch {
    error.value = 'Không bắt đầu được thử thách. Thử lại.'
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <template v-if="pet.loading && !pet.status">
      <AppCard class="mt-4">
        <StateBlock state="loading" />
      </AppCard>
    </template>

    <template v-else-if="passed">
      <AppCard class="mt-4 text-center">
        <PlantSvg :stage="pet.status?.stage ?? 'sprout'" :health="pet.status?.health_points ?? 50" />
        <p class="mt-3 font-display text-2xl text-growth">
          Cây đã hồi sinh!
        </p>
        <p class="text-mute">
          Máu cây: {{ pet.status?.health_points ?? 50 }}%
        </p>
        <AppButton class="mt-4" block @click="navigateTo('/')">
          Về trang chính
        </AppButton>
      </AppCard>
    </template>

    <template v-else-if="pet.notWilted || (pet.status && !pet.isWilted)">
      <AppCard class="mt-4 text-center">
        <PlantSvg :stage="pet.status?.stage ?? 'sprout'" :health="pet.status?.health_points ?? 100" />
        <p class="mt-3 font-display text-2xl">
          Cây của bạn vẫn khỏe 🌱
        </p>
        <AppButton class="mt-4" block @click="navigateTo('/')">
          Về trang chính
        </AppButton>
      </AppCard>
    </template>

    <template v-else>
      <div class="mt-4 rounded-card bg-alert px-4 py-3 text-center font-semibold uppercase tracking-wide text-white" role="alert">
        ⚠️ Cây xanh đang bị héo rũ!
      </div>

      <AppCard class="mt-4 text-center">
        <PlantSvg stage="wilted" :health="0" />
        <p class="text-sm text-mute">
          Cây héo - 0%
        </p>
      </AppCard>

      <AppCard v-if="!challengeActive" class="mt-4">
        <p class="font-display text-lg">
          "Bạn đã bỏ học<template v-if="missedDays !== null"> {{ missedDays }} ngày liên tiếp</template>. Hãy hoàn thành Bài kiểm tra Cứu Cây 15 phút để hồi sinh!"
        </p>
        <AppButton class="mt-4" variant="danger" block :loading="busy" @click="revive">
          🚨 Cứu cây ngay (Quiz 15 phút)
        </AppButton>
      </AppCard>

      <AppCard v-else class="mt-4">
        <SegmentedProgress :value-seconds="progress" label="Học 15 phút để hồi sinh" :segments="1" :segment-seconds="900" :met="canCheck" />
        <div class="mt-4 flex flex-col gap-2">
          <AppButton block @click="navigateTo('/')">
            Vào học ngay
          </AppButton>
          <AppButton variant="ghost" block :loading="busy" @click="revive">
            Kiểm tra hồi sinh
          </AppButton>
        </div>
      </AppCard>

      <p v-if="error" class="mt-3 rounded-card border border-alert/40 bg-alert/10 px-4 py-3 text-sm text-alert" role="alert">
        {{ error }}
      </p>
    </template>
  </main>
</template>
