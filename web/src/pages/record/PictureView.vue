<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { spoken } from '../../date'
import EditEntry from '../../edit/EditEntry.vue'
import Editing from '../../edit/Editing.vue'
import RecordPicker from '../../edit/RecordPicker.vue'
import { refusalOf, useChange, type Refusal } from '../../edit/change'
import { edit as editing, fieldLabel, pictures as said } from '../../labels'
import { located, useKept, useNaming, useRead } from '../../naming'
import { depictedOf, entryOf, type Depicted } from '../../records'
import { listed } from '../../search'
import { Button, Chip, Definition, Definitions, Lightbox, Notice, Prose, Stack, Text, Thumb } from '../../ui'

// A picture opened whole from wherever it is shown: what the record keeping it says of
// it, the people in it outlined, a box drawn around somebody to say who they are, and
// the picture made the portrait of somebody in it, opened from the picture small.
// `path` is the entry of the record the page edits that refers to the picture, opened
// from here.
const props = withDefaults(defineProps<{ owner: string; file: string; path?: string; size?: 'md' | 'sm' }>(), {
  size: 'md',
})

const kept = useKept()
const read = useRead()
const { called, person } = useNaming()
const { client, writes, offered, make } = useChange()

const open = ref(false)
const at = computed(() => located(props.owner, props.file))
const keeper = computed(() => read(at.value.ref))
const src = computed(() => kept(props.owner, props.file))
const found = computed(() => (keeper.value ? entryOf(keeper.value.fields, at.value.file) : undefined))
const entry = computed(() => (typeof found.value?.entry === 'object' ? found.value.entry : {}))

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const title = computed(() => text(entry.value.title) || at.value.file.split('/').at(-1) || at.value.file)
const when = computed(() => spoken(text(entry.value.date)))
const place = computed(() => text(entry.value.place))
const where = computed(() => {
  const kind = at.value.ref.slice(0, at.value.ref.indexOf('/')) as keyof typeof listed
  return `${listed[kind]}/${at.value.ref.slice(at.value.ref.indexOf('/') + 1)}/${at.value.file}`
})

const depicted = computed(() => depictedOf(found.value?.entry))
const regions = computed(() =>
  depicted.value.flatMap((d) => (d.region ? [{ key: String(d.at), region: d.region, label: called('person', d.person) }] : [])),
)
const noteOf = (d: Depicted) => (found.value ? keeper.value?.html[`${found.value.path}.depicts[${d.at}].note`] : undefined)

// The person keeping the picture, cropped to where they are in it if they are tagged,
// where that is not already their portrait. A portrait is a file beside the person, and
// somebody with no picture of their own takes one they are tagged in without asking.
const portrayed = computed(() => {
  const ref = at.value.ref
  if (!ref.startsWith('person/')) return undefined
  const slug = ref.slice('person/'.length)
  const indexed = person(slug)
  if (!indexed) return undefined
  const region = depicted.value.find((d) => d.person === slug)?.region
  const now = indexed.portrait
  const same = JSON.stringify(now?.region?.map(Number) ?? null) === JSON.stringify(region ?? null)
  if (now && now.record === ref && now.file === at.value.file && same) return undefined
  return { slug, region }
})

const fraction = (n: number) => String(Math.round(n * 1000) / 1000)

const drawn = ref<number[]>()
const who = ref('')
const refusal = ref<Refusal>()
const busy = ref(false)
// the keeper as the box was drawn, which the change is pinned to
let seen = ''
watch(drawn, (now) => {
  if (!now) return
  seen = keeper.value?.seen ?? ''
  refusal.value = undefined
})
watch(open, (now) => {
  if (now) return
  drawn.value = undefined
  who.value = ''
  refusal.value = undefined
})

async function changing(change: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  refusal.value = undefined
  try {
    await change()
  } catch (e) {
    refusal.value = refusalOf(e)
  } finally {
    busy.value = false
  }
}

// Says who is in the box drawn, in the entry of the record keeping the picture, or in
// a new one where it says nothing of the picture yet.
function tag() {
  const keeping = keeper.value
  if (!keeping || !drawn.value || !who.value || !seen) return
  const one = { person: who.value, region: drawn.value.map(fraction) }
  const f = found.value
  const edit = !f
    ? { op: 'add', entity: keeping.ref, path: 'media', value: { file: at.value.file, depicts: [one] } }
    : typeof f.entry === 'string'
      ? { op: 'set', entity: keeping.ref, path: f.path, value: { file: f.entry, depicts: [one] } }
      : Array.isArray(f.entry.depicts)
        ? { op: 'add', entity: keeping.ref, path: `${f.path}.depicts`, value: one }
        : { op: 'set', entity: keeping.ref, path: `${f.path}.depicts`, value: [one] }
  return changing(() =>
    make(
      () => client.change({ edit, seen }),
      () => {
        drawn.value = undefined
        who.value = ''
      },
    ),
  )
}

// Makes the picture, or the part of it they are in, the portrait of the person keeping it.
function portrait() {
  const keeping = keeper.value
  const chosen = portrayed.value
  if (!keeping || !chosen) return
  const value = chosen.region ? { file: at.value.file, region: chosen.region.map(fraction) } : at.value.file
  return changing(() =>
    make(
      () => client.change({ edit: { op: 'set', entity: keeping.ref, path: 'portrait', value }, seen: keeping.seen }),
      () => undefined,
    ),
  )
}
</script>

<template>
  <Lightbox
    v-model:open="open"
    v-model:drawn="drawn"
    :title="title"
    :src="src"
    :regions="regions"
    :draw="writes && !!keeper && !!src"
    :path="path"
  >
    <template #trigger>
      <Thumb as="button" :size="size" :src="src" :title="title" :regions="regions" :data-path="path" />
    </template>
    <Stack gap="4">
      <Text v-if="when || place" tone="muted">{{ [when, place && called('place', place)].filter(Boolean).join(' · ') }}</Text>
      <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">{{ refusal.message }}</Notice>
      <Stack v-if="drawn" gap="2" data-tagging>
        <RecordPicker v-model="who" kind="person" :label="said.who" />
        <Stack row gap="2">
          <Button size="sm" variant="ghost" @click="drawn = undefined">{{ editing.cancel }}</Button>
          <Button size="sm" variant="primary" :disabled="busy || !who" @click="tag">{{ said.tag }}</Button>
        </Stack>
      </Stack>
      <Definitions>
        <Definition v-if="depicted.length" :term="said.depicts">
          <Stack gap="2">
            <Stack v-for="d in depicted" :key="d.at" gap="1">
              <Chip v-if="person(d.person)" icon="person" :to="`/person/${d.person}`">{{ called('person', d.person) }}</Chip>
              <Text v-else>{{ d.person }}</Text>
              <Prose v-if="noteOf(d)" :html="noteOf(d)!" />
            </Stack>
          </Stack>
        </Definition>
        <Definition :term="said.file">
          <Text face="mono" size="xs">{{ where }}</Text>
        </Definition>
      </Definitions>
      <Stack gap="2">
        <Button v-if="offered && portrayed" size="sm" icon="person" data-portrait :disabled="!writes || busy" @click="portrait">
          {{ said.portraitOf(called('person', portrayed.slug)) }}
        </Button>
        <EditEntry v-if="path" v-slot="{ disabled }" :path="path" :label="fieldLabel(path)">
          <Button size="sm" icon="pen" :data-path="path" :disabled="disabled">{{ said.edit }}</Button>
        </EditEntry>
        <Editing v-else-if="keeper && found" :record="keeper">
          <EditEntry v-slot="{ disabled }" :path="found.path" :label="fieldLabel(found.path)">
            <Button size="sm" icon="pen" :data-path="found.path" :disabled="disabled">{{ said.edit }}</Button>
          </EditEntry>
        </Editing>
      </Stack>
    </Stack>
  </Lightbox>
</template>
