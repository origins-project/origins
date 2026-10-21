<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditButton from '../../edit/EditButton.vue'
import EditEntry from '../../edit/EditEntry.vue'
import EditValue from '../../edit/EditValue.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { eventLabel, family as said, fieldLabel } from '../../labels'
import { useNaming } from '../../naming'
import type { IndexedFamily } from '../../person'
import { Block, Grid, Hero, PersonCard, Stack, Text } from '../../ui'
import EventsBlock from '../record/EventsBlock.vue'
import RecordView from '../record/RecordView.vue'

// A family: its partners and what it is, then the partners as cards, the children with
// their relations to each partner, the events and the notes.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { index, called, card } = useNaming()

const indexed = computed(() => index.value?.families.find((f) => f.ref === props.record.ref) as IndexedFamily | undefined)
const title = computed(() => called('family', props.slug))
const first = computed(() => {
  const f = indexed.value?.first
  if (!f) return ''
  return [eventLabel(f.type), spoken(f.date ?? ''), f.place && called('place', f.place)].filter(Boolean).join(' · ')
})

const partners = computed(() => (indexed.value?.partners ?? []).map((p) => card(p.person, p.name)))
const children = computed(() =>
  (indexed.value?.children ?? []).map((c) => {
    const relations = [...new Set((c.relations ?? []).map((r) => r.relation).filter((r) => r !== 'birth'))]
    return {
      ...card(c.person, c.name),
      relation: [...relations, c.status].filter(Boolean).join(' · '),
      struck: c.status === 'disproven',
    }
  }),
)
</script>

<template>
  <RecordView :record="record" :around="around" :slug="slug" :drawn="['partners', 'type', 'children', 'events']">
    <template #hero>
      <Hero>
        <template #title>{{ title }}</template>
        <Stack row wrap gap="3" align="center">
          <EditValue path="type" :label="fieldLabel('type')" />
          <Text v-if="first" tone="muted">{{ first }}</Text>
        </Stack>
        <template #actions><RecordMenu /></template>
      </Hero>
    </template>
    <Block :title="said.partners">
      <template #actions><EditEntry path="partners" :label="fieldLabel('partners')" list /></template>
      <Grid fit="card" gap="2">
        <Stack v-for="(p, i) in partners" :key="i" row gap="1" align="center">
          <Stack grow><PersonCard v-bind="p" /></Stack>
          <EditButton :path="`partners[${i}]`" />
        </Stack>
      </Grid>
    </Block>
    <Block :title="said.children">
      <template #actions><EditEntry path="children" :label="fieldLabel('children')" list /></template>
      <Grid v-if="children.length" fit="card" gap="2">
        <Stack v-for="(c, i) in children" :key="i" row gap="1" align="center">
          <Stack grow><PersonCard v-bind="c" /></Stack>
          <EditButton :path="`children[${i}]`" />
        </Stack>
      </Grid>
    </Block>
    <EventsBlock :record="record" />
  </RecordView>
</template>
