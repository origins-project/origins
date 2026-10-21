<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import EditEntry from '../../edit/EditEntry.vue'
import { fieldLabel, record as said } from '../../labels'
import { isPeriod, placed, yearOf } from '../../person'
import { Block, Moment, Timeline } from '../../ui'
import EventCard from './EventCard.vue'

// Events: every event of the record, those a date puts in order first and in that order,
// a period drawn as a span, and those nothing dates after them.
const props = defineProps<{ record: Record }>()

const items = computed(() => {
  const { dated, undated } = placed(props.record)
  let last = ''
  return [...dated, ...undated].map((one) => {
    const year = yearOf(one.sort)
    const shown = year === last ? '' : year
    last = year
    return { one, year: shown }
  })
})
</script>

<template>
  <Block :title="said.events">
    <template #actions><EditEntry path="events" :label="fieldLabel('events')" list /></template>
    <Timeline v-if="items.length">
      <Moment v-for="{ one, year } in items" :key="one.path" :year="year" :span="isPeriod(one.event.date)" :path="one.path">
        <EventCard :owner="record.ref" :path="one.path" :event="one.event" :html="record.html" />
      </Moment>
    </Timeline>
  </Block>
</template>
