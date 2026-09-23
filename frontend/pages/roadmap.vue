<script setup lang="ts">
import { computed, nextTick, onMounted } from 'vue'
import { usePetStore } from '~/stores/pet'
import { useQuestStore } from '~/stores/quest'
import { roadmapNodes } from '~/utils/roadmap'

const quest = useQuestStore()
const pet = usePetStore()

onMounted(async () => {
  if (!pet.status) void pet.load()
  if (!quest.daily && !quest.noRoadmap) await quest.load()
  await nextTick()
  document.getElementById(`day-${quest.daily?.day_number ?? 0}`)?.scrollIntoView({ block: 'center' })
})

const nodes = computed(() => (quest.daily ? roadmapNodes(quest.daily.day_number) : []))
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" />
    <h1 class="mb-4 font-display text-2xl">
      Lộ trình học 28 ngày
    </h1>

    <AppCard>
      <StateBlock v-if="quest.loading && !quest.daily" state="loading" />
      <StateBlock v-else-if="quest.noRoadmap" state="empty" message="Bạn chưa có lộ trình học." action="Tạo lộ trình 28 ngày" @action="navigateTo('/onboarding')" />
      <StateBlock v-else-if="quest.error && !quest.daily" state="error" message="Không tải được lộ trình." action="Thử lại" @action="quest.load()" />
      <ol v-else class="relative space-y-3">
        <template v-for="(node, i) in nodes" :key="node.day">
          <li v-if="i % 7 === 0" class="pt-2 text-xs font-semibold uppercase tracking-wider text-mute" aria-hidden="true">
            Tuần {{ node.week }}
          </li>
          <li class="flex" :class="i % 2 === 0 ? 'justify-start pl-2' : 'justify-end pr-2'">
            <RoadmapNode :node="node" />
          </li>
          <li v-if="i < nodes.length - 1 && (i + 1) % 7 !== 0" class="h-4 border-mute/30" :class="i % 2 === 0 ? 'ml-[40%] border-l' : 'mr-[40%] border-r'" aria-hidden="true" />
        </template>
      </ol>
    </AppCard>
  </main>
</template>
