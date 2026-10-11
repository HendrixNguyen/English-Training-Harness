<script setup lang="ts">
import { computed, ref, useId } from 'vue'
import ItemPassage from '~/components/learn/ItemPassage.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import type { Classified } from '~/utils/content'

const props = defineProps<{ content: Classified, title: string }>()
const emit = defineEmits<{ answer: [questionId: string, option: string], finished: [] }>()

const index = ref(props.content.kind === 'reading' ? -1 : 0)
const passageOpen = ref(false)
const passageRef = ref<InstanceType<typeof ItemPassage> | null>(null)
const sheetRef = ref<InstanceType<typeof ItemPassage> | null>(null)
const sheetId = useId()
const passageItem = computed(() => props.content.kind === 'reading' && index.value === -1)
const questions = computed(() => props.content.kind === 'questions' || props.content.kind === 'reading' ? props.content.questions : [])
const selected = ref<string | null>(null)

const items = computed(() => props.content.kind === 'words' ? props.content.words : questions.value)
const total = computed(() => items.value.length)
const isLast = computed(() => index.value >= total.value - 1)

const buttonLabel = computed(() => {
  if (passageItem.value) return 'Đọc xong →'
  if (questions.value.length && selected.value !== null) return 'Gửi đáp án'
  return isLast.value || props.content.kind === 'raw' ? 'Hoàn thành' : 'Tiếp tục'
})

function togglePassage() {
  sheetRef.value?.closeGloss()
  passageOpen.value = !passageOpen.value
}

function next() {
  passageRef.value?.closeGloss()
  sheetRef.value?.closeGloss()
  if (!passageItem.value && questions.value.length && selected.value !== null) {
    emit('answer', questions.value[index.value]!.id, selected.value)
  }
  selected.value = null
  if (props.content.kind === 'raw' || isLast.value) {
    emit('finished')
    return
  }
  index.value += 1
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="content.kind !== 'raw' && !passageItem" class="flex h-11 items-center justify-between">
      <p class="text-sm text-mute">Câu {{ index + 1 }} / {{ total }}</p>
      <slot name="eyebrow-action" />
    </div>

    <ItemPassage v-if="passageItem && content.kind === 'reading'" ref="passageRef" :passage="content.passage" :glossary="content.glossary">
      <template #eyebrow-action><slot name="eyebrow-action" /></template>
    </ItemPassage>

    <template v-if="content.kind === 'words'">
      <p class="font-display text-3xl">
        {{ content.words[index]?.term }}
      </p>
      <p class="text-lg">
        {{ content.words[index]?.definition }}
      </p>
    </template>

    <template v-else-if="!passageItem && (content.kind === 'questions' || content.kind === 'reading')">
      <template v-if="content.kind === 'reading'">
        <RetroButton variant="secondary" block :aria-expanded="passageOpen" :aria-controls="sheetId" @click="togglePassage">{{ passageOpen ? 'Ẩn đoạn văn' : 'Xem lại đoạn' }}</RetroButton>
        <ItemPassage v-if="passageOpen" :id="sheetId" ref="sheetRef" mode="sheet" :passage="content.passage" :glossary="content.glossary">
          <template #eyebrow-action><slot name="eyebrow-action" /></template>
        </ItemPassage>
      </template>
      <p class="text-lg">
        "{{ questions[index]?.prompt }}"
      </p>
      <div role="radiogroup" class="space-y-2">
        <button
          v-for="(text, key) in questions[index]?.options"
          :key="key"
          type="button"
          role="radio"
          :aria-checked="selected === key"
          class="block w-full rounded-btn border px-4 py-3 text-left"
          :class="selected === key ? 'border-growth bg-growth/10' : 'border-ink/15 dark:border-paper/15'"
          @click="selected = String(key)"
        >
          ({{ key }}) {{ text }}
        </button>
      </div>
    </template>

    <template v-else-if="content.kind === 'raw'">
      <h2 class="font-display text-2xl">
        {{ title }}
      </h2>
      <pre class="overflow-x-auto rounded-card bg-ink/5 p-3 text-xs dark:bg-paper/10">{{ content.text }}</pre>
    </template>

    <AppButton block @click="next">
      {{ buttonLabel }}
    </AppButton>
  </div>
</template>
