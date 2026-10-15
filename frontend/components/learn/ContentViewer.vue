<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from 'vue'
import ItemPassage from '~/components/learn/ItemPassage.vue'
import CompanionSprite from '~/components/retro/CompanionSprite.vue'
import RetroButton from '~/components/retro/RetroButton.vue'
import RetroPanel from '~/components/retro/RetroPanel.vue'
import { questionsOf, type Classified } from '~/utils/content'
import { initialSession, type LearningSession } from '~/utils/learningSession'
const props = defineProps<{ content: Classified, title: string, session?: LearningSession, speaker?: string, companion?: { stage: string, health: number } }>()
const emit = defineEmits<{ answer: [questionId: string, option: string], 'session-change': [session: LearningSession], finished: [] }>()
const state = ref<LearningSession>(props.session ?? initialSession(props.content))
watch(() => props.session, value => { if (value) state.value = value })
const passageRef = ref<InstanceType<typeof ItemPassage> | null>(null)
const sheetRef = ref<InstanceType<typeof ItemPassage> | null>(null)
const heading = ref<HTMLElement | null>(null)
const feedbackHeading = ref<HTMLElement | null>(null)
const sheetId = useId()
const questions = computed(() => questionsOf(props.content))
const question = computed(() => questions.value[state.value.cursor])
const passageItem = computed(() => props.content.kind === 'reading' && state.value.phase === 'study')
const wordItem = computed(() => props.content.kind === 'words' && state.value.phase === 'study')
const feedback = computed(() => state.value.phase === 'feedback')
const correct = computed(() => feedback.value && question.value?.answer ? state.value.pending === question.value.answer : null)
const label = computed(() => passageItem.value ? 'Đọc xong →' : wordItem.value && props.content.kind === 'words' && state.value.cursor === props.content.words.length - 1 && questions.value.length ? 'Làm câu hỏi' : wordItem.value || feedback.value ? 'Tiếp tục' : 'Trả lời')
function update(patch: Partial<LearningSession>) { state.value = { ...state.value, ...patch }; emit('session-change', state.value) }
function choose(key: string) { if (state.value.phase === 'selection' && question.value && Object.hasOwn(question.value.options, key)) update({ pending: key }) }
function keys(event: KeyboardEvent, key: string) {
  if (event.repeat || state.value.phase !== 'selection') { event.preventDefault(); return }
  const options = Object.keys(question.value?.options ?? {})
  if (['ArrowDown', 'ArrowRight', 'ArrowUp', 'ArrowLeft', 'Home', 'End'].includes(event.key)) {
    event.preventDefault()
    const index = options.indexOf(key)
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? options.length - 1 : (index + (['ArrowDown', 'ArrowRight'].includes(event.key) ? 1 : -1) + options.length) % options.length
    choose(options[next]!)
    const group = (event.currentTarget as HTMLElement).parentElement
    ;(group?.querySelectorAll<HTMLButtonElement>('[role="radio"]')[next])?.focus()
  }
}
function togglePassage() { sheetRef.value?.closeGloss(); update({ passageOpen: !state.value.passageOpen }) }
async function next() {
  if (props.content.kind === 'raw' || state.value.phase === 'summary') return
  if (state.value.phase === 'selection') {
    const q = question.value; const option = state.value.pending
    if (!q || !option || !Object.hasOwn(q.options, option)) return
    update({ phase: 'feedback', answers: { ...state.value.answers, [q.id]: option } })
    emit('answer', q.id, option)
    await nextTick(); feedbackHeading.value?.focus()
    return
  }
  passageRef.value?.closeGloss(); sheetRef.value?.closeGloss()
  if (wordItem.value && props.content.kind === 'words' && state.value.cursor < props.content.words.length - 1) update({ cursor: state.value.cursor + 1 })
  else if (state.value.phase === 'study' && questions.value.length) update({ phase: 'selection', cursor: 0 })
  else if (feedback.value && state.value.cursor < questions.value.length - 1) update({ phase: 'selection', cursor: state.value.cursor + 1, pending: null })
  else { update({ phase: 'summary', cursor: questions.value.length, pending: null }); emit('finished') }
  await nextTick(); heading.value?.focus()
}
</script>

<template>
  <div class="space-y-4">
    <div v-if="!passageItem && content.kind !== 'raw'" class="flex min-h-11 items-center justify-between">
      <h2 ref="heading" tabindex="-1" class="font-display text-[22px]">
        {{ wordItem && content.kind === 'words' ? `Từ ${state.cursor + 1} / ${content.words.length}` : `Câu ${state.cursor + 1} / ${questions.length}` }}
      </h2>
      <slot name="eyebrow-action" />
    </div>
    <ItemPassage v-if="passageItem && content.kind === 'reading'" ref="passageRef" :passage="content.passage" :glossary="content.glossary">
      <template #eyebrow-action><slot name="eyebrow-action" /></template>
    </ItemPassage>
    <RetroPanel v-if="wordItem && content.kind === 'words'" :speaker="speaker" :announce="false">
        <template v-if="companion" #portrait><CompanionSprite :stage="companion.stage" :health="companion.health" :name="speaker" :size="48" crop="face" :reduced="true" /></template>
      <div class="space-y-3 font-body text-[17px] leading-[26px]">
        <p class="font-body text-[24px] font-bold leading-8">{{ content.words[state.cursor]?.term }}</p>
        <p>{{ content.words[state.cursor]?.definition }}</p>
        <p v-if="content.words[state.cursor]?.example">{{ content.words[state.cursor]?.example }}</p>
      </div>
    </RetroPanel>
    <template v-else-if="!passageItem && question">
      <template v-if="content.kind === 'reading'">
        <RetroButton variant="secondary" block :aria-expanded="state.passageOpen" :aria-controls="sheetId" @click="togglePassage">{{ state.passageOpen ? 'Ẩn đoạn văn' : 'Xem lại đoạn' }}</RetroButton>
        <ItemPassage v-if="state.passageOpen" :id="sheetId" ref="sheetRef" mode="sheet" :passage="content.passage" :glossary="content.glossary">
          <template #eyebrow-action><slot name="eyebrow-action" /></template>
        </ItemPassage>
      </template>
      <RetroPanel :speaker="speaker" :announce="false" :tone="feedback && correct !== null ? correct ? 'growth' : 'ember' : 'plain'">
        <template v-if="companion" #portrait><CompanionSprite :stage="companion.stage" :health="companion.health" :name="speaker" :size="48" crop="face" :reduced="true" /></template>
        <p class="mb-3 font-display text-[22px]">Đã trả lời {{ Object.keys(state.answers).length }} / {{ questions.length }}</p>
        <p class="mb-4 font-body text-[17px] leading-[26px]">{{ question.prompt }}</p>
        <div role="radiogroup" aria-label="Chọn đáp án" class="space-y-2">
          <button
            v-for="(text, key) in question.options" :key="key" type="button" role="radio"
            :tabindex="state.pending === key || (!state.pending && key === Object.keys(question.options)[0]) ? 0 : -1" :aria-checked="state.pending === key" :disabled="feedback" :aria-disabled="feedback"
            class="block min-h-14 w-full rounded-sm border-2 px-4 py-3 text-left font-body text-[17px] leading-[26px] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-torch"
            :class="feedback && question.answer === key ? 'border-growth bg-ground-2' : feedback && state.pending === key && correct === false ? 'border-ember bg-ground-2' : state.pending === key ? 'border-line-lit bg-ground-2' : 'border-line-dim bg-ground-1'"
            @click="choose(String(key))" @keydown="keys($event, String(key))"
          >
            <span v-if="state.pending === key" aria-hidden="true">▶ </span>({{ key }}) {{ text }}
            <span v-if="feedback && state.pending === key" class="block text-sm">Bạn chọn</span>
            <span v-if="feedback && question.answer === key" class="block text-sm text-growth">✔ Đáp án đúng</span>
          </button>
        </div>
        <p v-if="state.phase === 'selection' && !state.pending" class="mt-3 font-body text-sm text-ink-1">Chọn một đáp án rồi trả lời.</p>
        <div v-if="feedback" role="status" aria-live="polite" class="mt-4 border-t border-line-lit pt-4 font-body text-[17px] leading-[26px]">
          <h3 ref="feedbackHeading" tabindex="-1" class="font-bold">{{ correct === null ? 'Đã ghi nhận lựa chọn' : correct ? 'Trúng!' : 'Trượt…' }}</h3>
          <p v-if="correct === null">Câu này chưa có đáp án để chấm.</p>
          <p v-if="question.answer && question.explanation">{{ question.explanation }}</p>
        </div>
      </RetroPanel>
    </template>
    <div v-if="content.kind !== 'raw' && state.phase !== 'summary'" class="retro-bottom-bar">
      <RetroButton :key="state.phase" block :disabled="state.phase === 'selection' && !state.pending" @keydown="($event: KeyboardEvent) => { if ($event.repeat) $event.preventDefault() }" @click="next">{{ label }}</RetroButton>
    </div>
  </div>
</template>
