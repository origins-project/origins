<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditButton from '../../edit/EditButton.vue'
import EditEntry from '../../edit/EditEntry.vue'
import EditValue from '../../edit/EditValue.vue'
import Editing from '../../edit/Editing.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { fieldLabel, kinds, place as said } from '../../labels'
import { useNaming } from '../../naming'
import { isPeriod, yearOf } from '../../person'
import { addressOf, anglesOf, eventsAt, within } from '../../records'
import { Badge, Block, Button, Card, Chip, Hero, Moment, Stack, Text, Timeline, Value } from '../../ui'
import EventCard from '../record/EventCard.vue'
import RecordView from '../record/RecordView.vue'

// A place: its name and type, then its names in their languages, what it lay within and
// when, its addresses, the events recorded there and the places within it, and beside
// them where it is on a map.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { index, called } = useNaming()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const span = (from: unknown, until: unknown) => [text(from), text(until)].filter(Boolean).map(spoken).join(' – ')

interface Named {
  path: string
  name: string
  lang: string
  type: string
  dates: string
}
function named(path: string, value: unknown): Named {
  const o = (typeof value === 'object' && value !== null ? value : {}) as { [field: string]: unknown }
  return { path, name: typeof value === 'string' ? value : text(o.name), lang: text(o.lang), type: text(o.type), dates: span(o.from, o.until) }
}
const name = computed(() => named('name', props.record.fields.name))
const others = computed(() => {
  const list = props.record.fields['also-known-as']
  return Array.isArray(list) ? list.map((n, i) => named(`also-known-as[${i}]`, n)) : []
})
const lived = computed(() => span(props.record.fields.from, props.record.fields.until))

const lies = computed(() => {
  const w = props.record.fields.within
  const list = Array.isArray(w) ? w.map((one, i) => ({ one, path: `within[${i}]` })) : w === undefined ? [] : [{ one: w, path: 'within' }]
  return list.map(({ one, path }) => {
    const o = (typeof one === 'object' && one !== null ? one : {}) as { [field: string]: unknown }
    return { path, place: typeof one === 'string' ? one : text(o.place), dates: span(o.from, o.until) }
  })
})
const listed = computed(() => props.record.fields.within === undefined || Array.isArray(props.record.fields.within))
const contains = computed(() => within(index.value, props.slug).map((p) => p.ref.slice(p.ref.indexOf('/') + 1)))

const addresses = computed(() => {
  const list = props.record.fields.addresses
  return Array.isArray(list) ? list.map(addressOf) : []
})

const angles = computed(() => anglesOf(props.record.fields.coordinates))
const map = computed(() => {
  const a = angles.value
  return a && `https://www.openstreetmap.org/?mlat=${a.latitude}&mlon=${a.longitude}#map=13/${a.latitude}/${a.longitude}`
})

const here = computed(() => {
  let last = ''
  return eventsAt(props.around, props.slug).map((one) => {
    const year = yearOf(one.sort)
    const shown = year === last ? '' : year
    last = year
    return { one, year: shown, kind: one.record.ref.slice(0, one.record.ref.indexOf('/')) as keyof typeof kinds }
  })
})
const slugOf = (ref: string) => ref.slice(ref.indexOf('/') + 1)
</script>

<template>
  <RecordView
    :record="record"
    :around="around"
    :slug="slug"
    :drawn="['name', 'also-known-as', 'type', 'within', 'coordinates', 'addresses']"
  >
    <template #hero>
      <Hero>
        <template #title>
          <EditEntry v-slot="{ disabled, marked }" path="name" :label="fieldLabel('name')">
            <Value path="name" :label="fieldLabel('name')" :disabled="disabled" :marked="marked">{{ name.name || slug }}</Value>
          </EditEntry>
        </template>
        <Stack row wrap gap="3" align="center">
          <EditValue path="type" :label="fieldLabel('type')" />
          <Text v-if="lived" tone="muted">{{ lived }}</Text>
        </Stack>
        <template #actions><RecordMenu /></template>
      </Hero>
    </template>

    <Block :title="said.names">
      <template #actions><EditEntry path="also-known-as" :label="fieldLabel('also-known-as')" list /></template>
      <Stack gap="2" as="ul">
        <Stack v-for="n in [name, ...others]" :key="n.path" row wrap gap="2" align="baseline" as="li">
          <EditEntry v-slot="{ disabled, marked }" :path="n.path" :label="fieldLabel(n.path)">
            <Value :path="n.path" :label="fieldLabel(n.path)" :disabled="disabled" :marked="marked">
              <Text face="serif" size="lg">{{ n.name }}</Text>
            </Value>
          </EditEntry>
          <Badge v-if="n.lang" mono>{{ n.lang }}</Badge>
          <Badge v-if="n.type" tone="faint">{{ n.type }}</Badge>
          <Text v-if="n.dates" tone="muted" size="sm">{{ n.dates }}</Text>
        </Stack>
      </Stack>
    </Block>

    <Block :title="said.within">
      <template #actions>
        <EditEntry v-if="listed" path="within" :label="fieldLabel('within')" list />
      </template>
      <Stack gap="2" as="ul">
        <Stack v-for="w in lies" :key="w.path" row wrap gap="2" align="center" as="li">
          <Chip icon="place" :to="`/place/${w.place}`">{{ called('place', w.place) }}</Chip>
          <Text v-if="w.dates" tone="muted" size="sm">{{ w.dates }}</Text>
          <EditButton :path="w.path" :label="fieldLabel('within')" />
        </Stack>
      </Stack>
    </Block>

    <Block v-if="contains.length" :title="said.contains">
      <Stack row wrap gap="2">
        <Chip v-for="p in contains" :key="p" icon="place" :to="`/place/${p}`">{{ called('place', p) }}</Chip>
      </Stack>
    </Block>

    <Block :title="said.addresses">
      <template #actions><EditEntry path="addresses" :label="fieldLabel('addresses')" list /></template>
      <Stack gap="2">
        <EditEntry
          v-for="(a, i) in addresses"
          :key="i"
          v-slot="{ disabled, marked }"
          :path="`addresses[${i}]`"
          :label="fieldLabel('addresses')"
        >
          <Value :path="`addresses[${i}]`" :label="fieldLabel('addresses')" :disabled="disabled" :marked="marked">{{ a }}</Value>
        </EditEntry>
      </Stack>
    </Block>

    <Block v-if="here.length" :title="said.here">
      <Timeline>
        <Moment v-for="{ one, year, kind } in here" :key="`${one.record.ref}:${one.path}`" :year="year" :span="isPeriod(one.event.date)">
          <Stack gap="2">
            <Stack row>
              <Chip :kind="kinds[kind].one" :to="`/${one.record.ref}`">{{ called(kind, slugOf(one.record.ref)) }}</Chip>
            </Stack>
            <Editing :record="one.record">
              <EventCard :owner="one.record.ref" :path="one.path" :event="one.event" :html="one.record.html" />
            </Editing>
          </Stack>
        </Moment>
      </Timeline>
    </Block>

    <template #side>
      <Card :title="said.map">
        <template v-if="map" #action>
          <Button size="sm" variant="ghost" icon="pin" :href="map">{{ said.openMap }}</Button>
        </template>
        <EditValue path="coordinates" :label="fieldLabel('coordinates')" />
      </Card>
    </template>
  </RecordView>
</template>
