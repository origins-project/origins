<script setup lang="ts">
import { computed } from 'vue'
import type { Kind, Record } from '../../api/client'
import { spoken } from '../../date'
import EditButton from '../../edit/EditButton.vue'
import { eventLabel, fieldLabel, kinds, source as said } from '../../labels'
import { useKept, useNaming } from '../../naming'
import type { Citing } from '../../records'
import { Badge, Card, Chip, Prose, Scan, Stack, Text } from '../../ui'
import { valueAt } from '../../edit/values'

// One citation of a source: its anchor, the page and date it gives and the event it
// records, its transcription and note, who it names, its scans with the entry marked,
// and every record citing it with the grade each gives it.
const props = defineProps<{ record: Record; anchor: string; citing: Citing[] }>()
const { called } = useNaming()
const kept = useKept()

const path = computed(() => `citations.${props.anchor}`)
const citation = computed(() => (props.record.fields.citations as { [anchor: string]: { [field: string]: unknown } })[props.anchor] ?? {})
const text = (v: unknown) => (typeof v === 'string' ? v : '')
const stated = computed(() => {
  const c = citation.value
  return [text(c.page), spoken(text(c.date)), c.event ? eventLabel(c.event) : ''].filter(Boolean)
})
const transcription = computed(() => props.record.html[`${path.value}.text`] ?? '')
const note = computed(() => props.record.html[`${path.value}.note`] ?? '')

const named = computed(() => {
  const list = citation.value.with
  return Array.isArray(list)
    ? list.map((w: { [field: string]: unknown }) => {
        const n = w.name
        const name = typeof n === 'string' ? n : [text((n as { given?: unknown })?.given), text((n as { surname?: unknown })?.surname)].filter(Boolean).join(' ')
        return { slug: text(w.person), name, role: text(w.role) }
      })
    : []
})
const scans = computed(() => {
  const list = citation.value.images
  return Array.isArray(list)
    ? list.map((m: unknown) => {
        const o = (typeof m === 'object' && m !== null ? m : {}) as { file?: unknown; title?: unknown; region?: unknown }
        const file = typeof m === 'string' ? m : text(o.file)
        return { file, title: text(o.title) || file, region: Array.isArray(o.region) ? (o.region as number[]) : undefined }
      })
    : []
})

const kindOf = (ref: string) => ref.slice(0, ref.indexOf('/')) as Kind
const slugOf = (ref: string) => ref.slice(ref.indexOf('/') + 1)
// Where a record cites it: the event by its type, or the record as a whole.
function from(one: Citing): string {
  const event = /^events\[(\d+)\]/.exec(one.path)
  if (event) return eventLabel((valueAt(one.record.fields, `events[${event[1]}]`) as { type?: unknown } | undefined)?.type)
  return one.path === '' ? kinds[kindOf(one.record.ref)].one : (fieldLabel(one.path) ?? one.path)
}
</script>

<template>
  <Card :title="anchor" :id="anchor">
    <template #action><EditButton :path="path" :label="fieldLabel(path)" /></template>
    <Stack gap="3">
      <Stack v-if="stated.length" row wrap gap="2">
        <Text v-for="(words, i) in stated" :key="i" tone="muted" size="sm">{{ words }}</Text>
      </Stack>
      <Prose v-if="transcription" :html="transcription" />
      <Prose v-if="note" :html="note" />
      <Stack v-if="named.length" row wrap gap="2">
        <template v-for="(n, i) in named" :key="i">
          <Chip v-if="n.slug && n.slug !== 'unknown'" :kind="n.role" :to="`/person/${n.slug}`">{{ called('person', n.slug) }}</Chip>
          <Chip v-else :kind="n.role">{{ n.name || n.slug }}</Chip>
        </template>
      </Stack>
      <Stack v-if="scans.length" row wrap gap="3">
        <Scan v-for="s in scans" :key="s.file" :src="kept(record.ref, s.file)" :region="s.region" :title="s.title" />
      </Stack>
      <Stack v-if="citing.length" gap="1">
        <Text caps size="xs" tone="muted">{{ said.citedBy }}</Text>
        <Stack row wrap gap="2" align="center">
          <Stack v-for="(one, i) in citing" :key="i" row gap="1" align="center" as="span">
            <Chip :kind="from(one)" :to="`/${one.record.ref}`">{{ called(kindOf(one.record.ref), slugOf(one.record.ref)) }}</Chip>
            <Badge v-if="one.grade" mono>{{ one.grade }}</Badge>
          </Stack>
        </Stack>
      </Stack>
    </Stack>
  </Card>
</template>
