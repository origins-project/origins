<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditEntry from '../../edit/EditEntry.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { shown } from '../../edit/values'
import { fieldLabel, person as said } from '../../labels'
import { initials, nameParts, type IndexedPerson } from '../../person'
import { Badge, Hero, Name, Portrait, Stack, Text, Value } from '../../ui'
import { served, useNaming } from '../../naming'

// The head of a person's page: the portrait, the name in its parts and any title held
// beside it, the lifespan, and the other names with their types and dates.
const props = defineProps<{ record: Record; indexed?: IndexedPerson }>()
const { called } = useNaming()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const parts = computed(() => nameParts(props.record.fields.name))
const written = computed(() => {
  const name = props.record.fields.name
  return typeof name === 'object' && name !== null ? text((name as { text?: unknown }).text) : ''
})
const plain = computed(
  () => text(props.indexed?.name) || parts.value.map((p) => p.text).join(' ') || props.record.ref.slice(props.record.ref.indexOf('/') + 1),
)
const portrait = computed(() => props.indexed?.portrait)
const titles = computed(() => {
  const events = props.record.fields.events
  if (!Array.isArray(events)) return []
  return events.filter((e: { type?: unknown }) => text(e.type).toLowerCase() === 'title').map((e: { value?: unknown }) => text(e.value)).filter(Boolean)
})

function end(dated: IndexedPerson['birth']) {
  if (!dated) return ''
  return [spoken(dated.date ?? ''), dated.place && called('place', dated.place)].filter(Boolean).join(', ')
}
const born = computed(() => end(props.indexed?.birth))
const died = computed(() => end(props.indexed?.death))

const others = computed(() => {
  const list = props.record.fields['also-known-as']
  if (!Array.isArray(list)) return []
  return list.map((n: unknown) => {
    const o = (typeof n === 'object' && n !== null ? n : {}) as { [field: string]: unknown }
    const parts = nameParts(n).map((p) => p.text).join(' ')
    const dates = [text(o.from), text(o.until)].filter(Boolean).map(spoken).join(' – ')
    return { name: text(o.text) || parts || shown(n), type: text(o.type), dates }
  })
})
</script>

<template>
  <Hero>
    <template #portrait>
      <EditEntry v-slot="{ disabled, marked }" path="portrait" :label="fieldLabel('portrait')">
        <Portrait
          as="button"
          size="lg"
          data-path="portrait"
          :disabled="disabled"
          :marked="marked"
          :label="fieldLabel('portrait')"
          :src="portrait && served(portrait.record, portrait.file)"
          :region="portrait?.region"
          :initials="initials(plain)"
          :sex="indexed?.sex"
        />
      </EditEntry>
    </template>
    <template #title>
      <EditEntry v-slot="{ disabled, marked }" path="name" :label="fieldLabel('name')">
        <Value path="name" :label="fieldLabel('name')" :disabled="disabled" :marked="marked">
          <Name v-if="parts.length" :parts="parts" />
          <bdi v-else>{{ plain }}</bdi>
        </Value>
      </EditEntry>
    </template>
    <Stack gap="2">
      <Text v-if="written && written !== plain" italic tone="muted">{{ written }}</Text>
      <Stack v-if="titles.length" row wrap gap="2">
        <Text v-for="(title, i) in titles" :key="i" face="serif" size="lg">{{ title }}</Text>
      </Stack>
      <Stack v-if="born || died" row wrap gap="2" align="baseline">
        <Text tone="muted">{{ born }}</Text>
        <Text tone="faint">–</Text>
        <Text tone="muted">{{ died }}</Text>
      </Stack>
      <Stack row wrap gap="2" align="center">
        <Text v-if="others.length" caps size="xs" tone="faint">{{ said.aka }}</Text>
        <EditEntry
          v-for="(other, i) in others"
          :key="i"
          v-slot="{ disabled, marked }"
          :path="`also-known-as[${i}]`"
          :label="fieldLabel('also-known-as')"
        >
          <Value :path="`also-known-as[${i}]`" :label="fieldLabel('also-known-as')" :disabled="disabled" :marked="marked">
            <Stack row gap="1" align="baseline" as="span">
              <Text>{{ other.name }}</Text>
              <Badge v-if="other.type" tone="faint">{{ other.type }}</Badge>
              <Text v-if="other.dates" tone="faint" size="sm">{{ other.dates }}</Text>
            </Stack>
          </Value>
        </EditEntry>
        <EditEntry path="also-known-as" :label="fieldLabel('also-known-as')" list />
      </Stack>
    </Stack>
    <template #actions><RecordMenu /></template>
  </Hero>
</template>
