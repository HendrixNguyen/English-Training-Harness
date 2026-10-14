<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { REVIVE_SECONDS, usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { useAuthStore } from '~/stores/auth'
import { daysSince } from '~/utils/plant'
const pet = usePetStore()
const quest = useQuestStore()
const auth = useAuthStore()
const busy = ref(false)
const error = ref<string | null>(null)
const passed = ref(false)
const online = ref(true)
const owner = () => `${auth.user?.id ?? ''}:${auth.sessionEpoch}`
watch(owner, () => { busy.value = false; error.value = null; passed.value = false })
function connection() { online.value = navigator.onLine }
async function load() { await Promise.all([pet.load(), quest.load()]) }
onMounted(() => { connection(); window.addEventListener('online', connection); window.addEventListener('offline', connection); pet.hydrateChallenge(); void load() })
onBeforeUnmount(() => { window.removeEventListener('online', connection); window.removeEventListener('offline', connection) })
const missedDays = computed(() => daysSince(pet.status?.last_practiced_at ?? null))
const missedLine = computed(() => missedDays.value === null ? 'Cậu bỏ tớ một thời gian rồi.' : `Cậu bỏ tớ ${missedDays.value} ngày liền.`)
const today = computed(() => quest.daily?.date ?? null)
const challengeActive = computed(() => !!today.value && pet.challenge?.date === today.value)
const progress = computed(() => challengeActive.value ? pet.challengeProgress(quest.accumulatedSeconds) : 0)
const canCheck = computed(() => progress.value >= REVIVE_SECONDS)
async function revive() {
 if (busy.value || !online.value || !today.value || (challengeActive.value && !canCheck.value)) return
 const ticket = owner(); busy.value = true; error.value = null
 try { const res = await pet.revive(today.value, quest.accumulatedSeconds); if (owner() === ticket && res?.revival_passed) passed.value = true }
 catch { if (owner() === ticket) error.value = 'Không bắt đầu được thử thách. Thử lại.' }
 finally { if (owner() === ticket) busy.value = false }
}
</script>
<template>
  <main class="retro-screen mx-auto max-w-md px-4 pb-8 font-body text-base">
    <NuxtLink to="/" class="flex min-h-12 items-center text-line-lit">‹ Trại</NuxtLink>
    <h1 class="mb-4 font-display text-[28px]">Hồi sinh</h1>
    <p v-if="!online" class="mb-4 text-sm text-ink-1">Cần mạng để hồi sinh. Tiến độ đã lưu vẫn còn.</p>
    <template v-if="passed && pet.status">
      <RetroPanel tone="growth" :announce="false" class="text-center">
        <CompanionSprite :stage="pet.status.stage" :health="pet.status.health_points" react="levelup" />
        <HpBar :value="pet.status.health_points" />
        <p role="status" class="mt-4">{{ pet.status.plant_name }} đã hồi sinh!</p>
      </RetroPanel>
      <div class="retro-bottom-bar"><RetroButton block @click="navigateTo('/')">Về trại</RetroButton></div>
    </template>
    <template v-else-if="pet.notWilted || (pet.status && !pet.isWilted)">
      <RetroPanel :announce="false" class="text-center">
        <template v-if="pet.status"><CompanionSprite :stage="pet.status.stage" :health="pet.status.health_points" /><HpBar :value="pet.status.health_points" /></template>
        <p class="mt-4">Bạn đồng hành của cậu vẫn khỏe.</p>
      </RetroPanel>
      <div class="retro-bottom-bar"><RetroButton block @click="navigateTo('/')">Về trại</RetroButton></div>
    </template>
    <template v-else-if="pet.status">
      <RetroPanel band tone="ember" :announce="false" class="mb-6"><p role="alert">{{ pet.status.plant_name }} đang bị héo rũ!</p></RetroPanel>
      <RetroPanel :speaker="pet.status.plant_name" tone="ember" :announce="false" class="text-center">
        <CompanionSprite :stage="pet.status.stage" :health="pet.status.health_points" react="down" />
        <HpBar :value="pet.status.health_points" />
        <p class="mt-4">{{ missedLine }} Học 15 phút để tớ đứng dậy nhé.</p>
      </RetroPanel>
      <div v-if="challengeActive" class="mt-6"><h2 class="font-display text-xl">Nhiệm vụ hồi sinh</h2><DayBar :value-seconds="progress" :segments="1" :segment-seconds="900" :cells="40" :met="canCheck" /><p class="mt-2 text-sm text-ink-1">{{ canCheck ? 'Đã đủ thời gian. Kiểm tra hồi sinh nhé.' : `Còn ${Math.ceil((REVIVE_SECONDS - progress) / 60)} phút học nữa.` }}</p></div>
      <StateBlock v-if="!today" class="mt-6" :state="quest.error ? 'error' : 'loading'" message="Chưa đọc được thời gian học hôm nay." action="Thử lại" @action="quest.load()" />
      <p v-if="error" role="alert" class="mt-4 text-sm text-ember">{{ error }}</p>
      <div class="retro-bottom-bar">
        <RetroButton v-if="challengeActive && !canCheck" block @click="navigateTo('/')">Vào học ngay</RetroButton>
        <RetroButton v-else block :variant="challengeActive ? 'primary' : 'danger'" :loading="busy" :disabled="!online || !today" @click="revive">{{ challengeActive ? 'Hồi sinh' : 'Hồi sinh (15 phút)' }}</RetroButton>
      </div>
    </template>
    <RetroPanel v-else :announce="false"><StateBlock :state="pet.error ? 'error' : 'loading'" message="Không tải được trạng thái cây. Chưa thể biết cây có héo hay không." action="Thử lại" @action="load" /></RetroPanel>
  </main>
</template>
