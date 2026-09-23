<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Classified } from '~/utils/content'

const props = defineProps<{ content: Classified, title: string }>()
const emit = defineEmits<{ answer: [questionId: string, option: string], finished: [] }>()

const index = ref(0)
const selected = ref<string | null>(null)

const items = computed(() => props.content.kind === 'words' ? props.content.words : props.content.kind === 'questions' ? props.content.questions : [])
const total = computed(() => items.value.length)
const isLast = computed(() => index.value >= total.value - 1)

const buttonLabel = computed(() => {
  if (props.content.kind === 'questions' && selected.value !== null) return 'Gửi đáp án'
  return isLast.value || props.content.kind === 'raw' ? 'Hoàn thành' : 'Tiếp tục'
})

function next() {
  if (props.content.kind === 'questions' && selected.value !== null) {
    emit('answer', props.content.questions[index.value]!.id, selected.value)
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
    <p v-if="content.kind !== 'raw'" class="text-sm text-mute">
      Câu {{ index + 1 }} / {{ total }}
    </p>

    <template v-if="content.kind === 'words'">
      <p class="font-display text-3xl">
        {{ content.words[index]?.term }}
      </p>
      <p class="text-lg">
        {{ content.words[index]?.definition }}
      </p>
    </template>

    <template v-else-if="content.kind === 'questions'">
      <p class="text-lg">
        "{{ content.questions[index]?.prompt }}"
      </p>
      <div role="radiogroup" class="space-y-2">
        <button
          v-for="(text, key) in content.questions[index]?.options"
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

    <template v-else>
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
