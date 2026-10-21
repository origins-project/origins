<script setup lang="ts">
import { computed } from 'vue'
import { spoken } from '../../date'
import EditEntry from '../../edit/EditEntry.vue'
import { eventLabel, fieldLabel } from '../../labels'
import { Badge, Chip, Prose, Stack, Text, Value } from '../../ui'
import { useNaming } from '../../naming'
import PictureView from './PictureView.vue'
import { citationsOf } from '../../records'

// One event of a record: what it was and when, which opens it in a drawer, and beside
// it where it was, who it names, its pictures and its sources, each leading to its own
// record.
const props = defineProps<{
  owner: string
  path: string
  event: { [field: string]: unknown }
  html?: { [path: string]: string }
}>()
const { called } = useNaming()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const type = computed(() => eventLabel(props.event.type))
const when = computed(() => spoken(text(props.event.date)))
const what = computed(() => text(props.event.value))
const place = computed(() => text(props.event.place))
const address = computed(() => {
  const a = props.event.address
  if (typeof a === 'string') return a
  const lines = (a as { lines?: unknown } | undefined)?.lines
  return Array.isArray(lines) ? lines.map(text).join(', ') : ''
})

interface Named {
  role: string
  slug: string
  name: string
  sources: string[]
}
const people = computed<Named[]>(() => {
  const list = Array.isArray(props.event.with) ? props.event.with : []
  return list.map((w: { [field: string]: unknown }) => {
    const n = w.name
    const name = typeof n === 'string' ? n : text((n as { text?: unknown } | undefined)?.text) || text((n as { given?: unknown })?.given)
    return { role: text(w.role), slug: text(w.person), name, sources: Array.isArray(w.sources) ? w.sources.map(text) : [] }
  })
})
const stated = computed(() =>
  [props.event.age, props.event.cause]
    .map((v) => (typeof v === 'string' ? v : text((v as { value?: unknown } | undefined)?.value)))
    .filter(Boolean),
)
const media = computed(() => {
  const list = Array.isArray(props.event.media) ? props.event.media : []
  return list.map((m: unknown) => (typeof m === 'string' ? m : text((m as { file?: unknown }).file)))
})
const sources = computed(() => citationsOf(props.event.sources))
const notes = computed(() => Object.entries(props.html ?? {}).filter(([path]) => path.startsWith(`${props.path}.`)))
</script>

<template>
  <Stack gap="2">
    <EditEntry v-slot="{ disabled, marked }" :path="path" :label="type || fieldLabel(path)">
      <Value :path="path" :label="type" :disabled="disabled" :marked="marked">
        <Stack row wrap gap="2" align="baseline" as="span">
          <Text strong>{{ type }}</Text>
          <Text v-if="what">{{ what }}</Text>
          <Text v-if="when" tone="muted">{{ when }}</Text>
        </Stack>
      </Value>
    </EditEntry>
    <Stack v-if="place || address" row wrap gap="2" align="center">
      <Chip v-if="place" icon="pin" :to="`/place/${place}`">{{ called('place', place) }}</Chip>
      <Text v-if="address" tone="muted" size="sm">{{ address }}</Text>
    </Stack>
    <Stack v-if="people.length" row wrap gap="2">
      <template v-for="(named, i) in people" :key="i">
        <Chip v-if="named.slug && named.slug !== 'unknown'" :kind="named.role" :to="`/person/${named.slug}`">
          {{ called('person', named.slug) }}
        </Chip>
        <Chip v-else :kind="named.role">{{ named.name || named.slug }}</Chip>
      </template>
    </Stack>
    <Stack v-if="stated.length" row wrap gap="3">
      <Text v-for="(words, i) in stated" :key="i" tone="muted" size="sm">{{ words }}</Text>
    </Stack>
    <Stack v-if="media.length" row wrap gap="2">
      <PictureView v-for="file in media" :key="file" :owner="owner" :file="file" size="sm" />
    </Stack>
    <Prose v-for="[at, rendered] in notes" :key="at" :html="rendered" />
    <Stack v-if="sources.length" row wrap gap="2" align="center">
      <template v-for="(citation, i) in sources" :key="i">
        <Chip tone="source" icon="book" :to="`/source/${citation.source}`">
          {{ called('source', citation.source) }}
          <Text v-if="citation.anchor" tone="muted">· {{ citation.anchor }}</Text>
        </Chip>
        <Badge v-if="citation.grade" mono>{{ citation.grade }}</Badge>
      </template>
    </Stack>
  </Stack>
</template>
