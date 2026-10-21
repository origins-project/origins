<script setup lang="ts">
import { computed } from 'vue'
import type { Kind, Record } from '../../api/client'
import EditValue from '../../edit/EditValue.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { fieldLabel, kinds, note as said } from '../../labels'
import { useNaming } from '../../naming'
import { refersTo } from '../../records'
import { Block, Chip, Hero, Stack } from '../../ui'
import EventsBlock from '../record/EventsBlock.vue'
import NotesBlock from '../record/NotesBlock.vue'
import RecordView from '../record/RecordView.vue'

// A note: its title, its body, the events it holds and the records linking to it.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { called } = useNaming()

const kindOf = (ref: string) => ref.slice(0, ref.indexOf('/')) as Kind
const linked = computed(() => props.around.filter((r) => refersTo(r, 'note', props.slug)).map((r) => r.ref))
</script>

<template>
  <RecordView :record="record" :around="around" :slug="slug" :drawn="['title', 'events', 'body']">
    <template #hero>
      <Hero>
        <template #title><EditValue path="title" :label="fieldLabel('title')" /></template>
        <template #actions><RecordMenu /></template>
      </Hero>
    </template>
    <NotesBlock :record="record" />
    <EventsBlock :record="record" />
    <Block v-if="linked.length" :title="said.linked">
      <Stack row wrap gap="2">
        <Chip v-for="ref in linked" :key="ref" :kind="kinds[kindOf(ref)].one" :to="`/${ref}`">
          {{ called(kindOf(ref), ref.slice(ref.indexOf('/') + 1)) }}
        </Chip>
      </Stack>
    </Block>
  </RecordView>
</template>
