<script setup lang="ts">
import { computed, nextTick, onMounted, reactive } from 'vue'
import RoadmapModuleHeader from '~/components/roadmap/RoadmapModuleHeader.vue'
import RoadmapNode from '~/components/roadmap/RoadmapNode.vue'
import { usePetStore } from '~/stores/pet'
import { useRoadmapStore } from '~/stores/roadmap'
import { roadmapNodes } from '~/utils/roadmap'

const roadmap = useRoadmapStore()
const pet = usePetStore()

const nodes = computed(() => (roadmap.outline ? roadmapNodes(roadmap.outline) : []))
const expanded = reactive<Record<number, boolean>>({})

onMounted(async () => {
  if (!pet.status) void pet.load()
  if (!roadmap.outline && !roadmap.noRoadmap) await roadmap.load()
  if (roadmap.outline) expanded[roadmap.outline.day_number] = true
  await nextTick()
  document.getElementById(`day-${roadmap.outline?.day_number ?? 0}`)?.scrollIntoView({ block: 'center' })
})
</script>

<template>
  <main class="mx-auto max-w-md px-4 pb-8">
    <AppHeader :streak="pet.status?.current_streak ?? null" />

    <template v-if="roadmap.outline">
      <p class="text-xs font-semibold uppercase tracking-wider text-mute">
        Lộ trình học 28 ngày
      </p>
      <h1 class="font-display line-clamp-2 text-[28px] leading-8">
        {{ roadmap.outline.title }}
      </h1>
      <p class="text-mute">
        Trình độ {{ roadmap.outline.cefr_level }} · Đã hoàn thành {{ roadmap.completedDays }}/28 ngày
      </p>
    </template>
    <h1 v-else class="mb-4 font-display text-2xl">
      Lộ trình học 28 ngày
    </h1>

    <AppCard>
      <StateBlock v-if="roadmap.loading && !roadmap.outline" state="loading" />
      <StateBlock v-else-if="roadmap.noRoadmap" state="empty" message="Bạn chưa có lộ trình học." action="Tạo lộ trình 28 ngày" @action="navigateTo('/onboarding')" />
      <StateBlock v-else-if="roadmap.error && !roadmap.outline" state="error" message="Không tải được lộ trình." action="Thử lại" @action="roadmap.load()" />
      <ol v-else-if="roadmap.outline" class="relative ml-3 space-y-4 border-l-2 border-mute/30 pl-7">
        <template v-for="module in roadmap.outline.modules" :key="module.week">
          <RoadmapModuleHeader :module="module" :met="module.days.filter(d => d.is_target_met).length" />
          <RoadmapNode
            v-for="node in nodes.filter(n => n.week === module.week)"
            :key="node.day"
            :node="node"
            :expanded="!!expanded[node.day]"
            @update:expanded="v => (expanded[node.day] = v)"
          />
        </template>
      </ol>
    </AppCard>
  </main>
</template>
