<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { Drafting } from '../api/client'
import { useCorpus } from '../api/corpus'
import { useClient } from '../api/use'
import { edit as editing, fieldLabel, hypothesis, keeping as said } from '../labels'
import { slugify } from '../lists'
import { Block, Button, Drawer, Field, IconButton, Notice, Select, Stack, TextArea } from '../ui'
import CitePicker from './CitePicker.vue'
import EditsList from './EditsList.vue'
import { refusalOf, type Refusal } from './change'

// The draft kept as an open hypothesis: its edits in words, and the question it puts,
// the slug derived from the question until one is typed, the argument, the confidence
// and the evidence on each side; gone to once it is kept.
const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ draft: Drafting }>()
const client = useClient()
const corpus = useCorpus()
const router = useRouter()

const question = ref('')
const own = ref<string>()
const argument = ref('')
const confidence = ref<string>()
const evidence = ref<{ supports: string[]; contradicts: string[] }>({ supports: [], contradicts: [] })
const refusal = ref<Refusal>()
const busy = ref(false)
watch(open, (now) => {
  if (!now) return
  question.value = argument.value = ''
  own.value = confidence.value = undefined
  evidence.value = { supports: [], contradicts: [] }
  refusal.value = undefined
})

const slug = computed({
  get: () => own.value ?? slugify(question.value),
  set: (typed: string) => (own.value = typed),
})
const confidences = computed(() => Object.entries(said.confidences).map(([value, label]) => ({ value, label })))
const ready = computed(() => slug.value.trim() !== '' && question.value.trim() !== '' && props.draft.edits.length > 0)

async function keep() {
  if (!ready.value || busy.value) return
  busy.value = true
  refusal.value = undefined
  const cited = (list: string[]) => list.filter((c) => c.trim() !== '')
  try {
    const kept = await corpus.change(() =>
      client.keepDraft({
        slug: slug.value.trim(),
        question: question.value.trim(),
        argument: argument.value.trim() || undefined,
        confidence: confidence.value,
        supports: cited(evidence.value.supports),
        contradicts: cited(evidence.value.contradicts),
      }),
    )
    if (kept.commit) corpus.made(kept.commit)
    open.value = false
    corpus.drafted(null)
    corpus.setSupposing(false)
    corpus.notify(kept.subject)
    await router.push(`/hypothesis/${kept.hypothesis}`)
  } catch (e) {
    refusal.value = refusalOf(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Drawer v-model:open="open" :title="said.title" path="draft" wide>
    <Stack gap="5">
      <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">{{ refusal.message }}</Notice>
      <Block :title="hypothesis.edits">
        <EditsList :edits="draft.edits" />
      </Block>
      <Field v-model="question" :label="said.question" path="question" />
      <Field v-model="slug" :label="said.slug" path="slug" mono />
      <TextArea v-model="argument" :label="said.argument" path="body" prose />
      <Select v-model="confidence" :label="fieldLabel('confidence')" :options="confidences" path="confidence" />
      <Stack v-for="side in ['supports', 'contradicts'] as const" :key="side" gap="2">
        <Stack v-for="(_, i) in evidence[side]" :key="i" row gap="2" align="end">
          <CitePicker v-model="evidence[side][i]" :label="fieldLabel(side)" :path="`${side}[${i}]`" />
          <IconButton icon="x" :label="editing.remove" @click="evidence[side].splice(i, 1)" />
        </Stack>
        <Button size="sm" variant="add" icon="plus" :data-path="`${side}[${evidence[side].length}]`" @click="evidence[side].push('')">
          {{ fieldLabel(side) }}
        </Button>
      </Stack>
    </Stack>
    <template #actions>
      <Button size="sm" variant="ghost" @click="open = false">{{ editing.cancel }}</Button>
      <Button size="sm" variant="primary" icon="flask" data-confirm :disabled="busy || !ready" @click="keep">
        {{ said.keep }}
      </Button>
    </template>
  </Drawer>
</template>
