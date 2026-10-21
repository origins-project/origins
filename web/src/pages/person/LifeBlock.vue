<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditEntry from '../../edit/EditEntry.vue'
import Editing from '../../edit/Editing.vue'
import { eventLabel, fieldLabel, person as said } from '../../labels'
import { isPeriod, kinWord, placed, quiet, yearOf, type IndexedPerson, type Quiet, type Relatives } from '../../person'
import { Block, Chip, Moment, Stack, Switch, Text, Timeline } from '../../ui'
import EventCard from '../record/EventCard.vue'

// Life: every dated event of the person in order, a period drawn as a span, the events
// of the families they are a partner in among them, and their relatives' births and
// deaths in a quieter style, which one switch turns off.
const props = defineProps<{ record: Record; indexed?: IndexedPerson; families: Record[]; relatives: Relatives }>()
const others = ref(true)

type Item =
  | { key: string; sort: string; kind: 'own' | 'family'; record: Record; path: string; event: { [field: string]: unknown } }
  | { key: string; sort: string; kind: 'quiet'; quiet: Quiet }

const items = computed(() => {
  const all: Item[] = []
  for (const one of placed(props.record).dated)
    all.push({ key: `${props.record.ref}:${one.path}`, sort: one.sort!, kind: 'own', record: props.record, path: one.path, event: one.event })
  for (const family of props.families)
    for (const one of placed(family).dated)
      all.push({ key: `${family.ref}:${one.path}`, sort: one.sort!, kind: 'family', record: family, path: one.path, event: one.event })
  if (others.value)
    quiet(props.relatives, props.indexed).forEach((q, i) => all.push({ key: `quiet:${i}`, sort: q.sort, kind: 'quiet', quiet: q }))
  const rank = { own: 0, family: 1, quiet: 2 }
  all.sort((a, b) => (a.sort < b.sort ? -1 : a.sort > b.sort ? 1 : rank[a.kind] - rank[b.kind]))
  let last = ''
  return all.map((item) => {
    const year = yearOf(item.sort)
    const shown = year === last ? '' : year
    last = year
    return { item, year: shown }
  })
})
</script>

<template>
  <Block :title="said.life">
    <template #actions>
      <Switch v-model="others" :label="said.familyEvents" />
      <EditEntry path="events" :label="fieldLabel('events')" list />
    </template>
    <Timeline v-if="items.length">
      <template v-for="{ item, year } in items" :key="item.key">
        <Moment v-if="item.kind === 'quiet'" :year="year" quiet>
          <Stack row wrap gap="2" align="baseline">
            <Text>{{ eventLabel(item.quiet.event.type) }}</Text>
            <Text tone="faint">{{ kinWord(said.kin, item.quiet.relative) }}</Text>
            <Chip v-if="item.quiet.relative.ref" :to="`/${item.quiet.relative.ref}`">{{ item.quiet.relative.name }}</Chip>
            <Text v-else>{{ item.quiet.relative.name }}</Text>
            <Text tone="faint">{{ spoken(item.quiet.event.date ?? '') }}</Text>
          </Stack>
        </Moment>
        <Moment v-else-if="item.kind === 'own'" :year="year" :span="isPeriod(item.event.date)" :path="item.path">
          <EventCard :owner="record.ref" :path="item.path" :event="item.event" :html="record.html" />
        </Moment>
        <Moment v-else :year="year" :span="isPeriod(item.event.date)">
          <Editing :record="item.record">
            <EventCard :owner="item.record.ref" :path="item.path" :event="item.event" :html="item.record.html" />
          </Editing>
        </Moment>
      </template>
    </Timeline>
  </Block>
</template>
