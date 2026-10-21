<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Drafting, Made } from '../api/client'
import { edit as said } from '../labels'
import { Button, Definition, Definitions, Dialog, Notice, Stack, Text } from '../ui'
import CitePicker from './CitePicker.vue'
import { refusalOf, useChange, type Refusal } from './change'

// A change that reaches past one value — a rename, a merge, a split, a delete, an undo,
// settling a hypothesis — made only from what its dry run said it would do, and asked
// again whenever what it is asked with changes.
const open = defineModel<boolean>('open', { default: false })
const props = withDefaults(
  defineProps<{
    title: string
    make: (dry: boolean, sources?: string[]) => Promise<Made | Drafting>
    confirm: string
    danger?: boolean
    // whether what the change is asked with is complete enough to ask
    ready?: boolean
    // what the change is asked with, so that the dry run is asked again when it changes
    asked?: string
    path?: string
  }>(),
  { ready: true, asked: '' },
)

const { make: change } = useChange()
const preview = ref<Made | Drafting>()
const refusal = ref<Refusal>()
const busy = ref(false)
const cite = ref('')
let asking = 0

async function dry() {
  const mine = ++asking
  preview.value = undefined
  refusal.value = undefined
  if (!props.ready) return
  try {
    const answer = await props.make(true)
    if (mine === asking) preview.value = answer
  } catch (e) {
    if (mine === asking) refusal.value = refusalOf(e)
  }
}

watch(
  () => [open.value, props.asked, props.ready] as const,
  ([now]) => {
    if (now) void dry()
  },
)
watch(open, (now) => {
  if (now) cite.value = ''
})

async function proceed() {
  busy.value = true
  try {
    await change(
      () => props.make(false, cite.value ? [cite.value] : undefined),
      () => (open.value = false),
    )
  } catch (e) {
    refusal.value = refusalOf(e)
    preview.value = undefined
  } finally {
    busy.value = false
  }
}

const made = (p: Made | Drafting | undefined): Made | undefined => (p && 'commit' in p ? p : undefined)
</script>

<template>
  <Dialog v-model:open="open" :title="title" :path="path">
    <Stack gap="4">
      <slot />
      <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">
        <Stack gap="1">
          <Text>{{ refusal.message }}</Text>
          <Text v-for="(f, i) in refusal.findings" :key="i" size="sm">
            {{ [f.entity, f.path, f.rule].filter(Boolean).join(' · ') }} — {{ f.message }}
          </Text>
          <Text v-for="(p, i) in refusal.problems" :key="`p${i}`" size="sm">{{ p }}</Text>
        </Stack>
      </Notice>
      <Definitions v-if="made(preview)" compact data-preview>
        <Definition :term="said.subject">{{ made(preview)!.subject }}</Definition>
        <Definition v-if="made(preview)!.written.length" :term="said.written">
          <Stack gap="1">
            <Text v-for="w in made(preview)!.written" :key="w" face="mono" size="sm">{{ w }}</Text>
          </Stack>
        </Definition>
        <Definition v-if="made(preview)!.moved.length" :term="said.moved">
          <Stack gap="1">
            <Text v-for="m in made(preview)!.moved" :key="m.from" face="mono" size="sm">{{ m.from }} → {{ m.to }}</Text>
          </Stack>
        </Definition>
        <Definition v-if="made(preview)!.removed.length" :term="said.removed">
          <Stack gap="1">
            <Text v-for="r in made(preview)!.removed" :key="r" face="mono" size="sm">{{ r }}</Text>
          </Stack>
        </Definition>
        <Definition v-if="made(preview)!.problems.length" :term="said.problems">
          <Stack gap="1">
            <Text v-for="(p, i) in made(preview)!.problems" :key="i" tone="warn" size="sm">{{ p }}</Text>
          </Stack>
        </Definition>
        <Definition v-if="made(preview)!.empty" :term="said.subject">{{ said.nothing }}</Definition>
      </Definitions>
      <CitePicker v-model="cite" :label="said.cite" />
    </Stack>
    <template #actions>
      <Button variant="ghost" @click="open = false">{{ said.cancel }}</Button>
      <Button
        :variant="danger ? 'danger' : 'primary'"
        solid
        :disabled="busy || !ready || !preview"
        data-confirm
        @click="proceed"
      >
        {{ confirm }}
      </Button>
    </template>
  </Dialog>
</template>
