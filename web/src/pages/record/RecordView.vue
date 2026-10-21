<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCorpus } from '../../api/corpus'
import type { Record } from '../../api/client'
import { record as said } from '../../labels'
import { provideKept } from '../../naming'
import { Columns, Icon, Stack, Tabs, Text } from '../../ui'
import CitedCard from './CitedCard.vue'
import DetailsCard from './DetailsCard.vue'
import FilesCard from './FilesCard.vue'
import HistoryCard from './HistoryCard.vue'
import HistoryList from './HistoryList.vue'
import NotesBlock from './NotesBlock.vue'
import ResearchCard from './ResearchCard.vue'

// A record's page other than a person's: the heading, then the overview — what the
// page draws of the record, its notes, and the cards beside them, the record's other
// fields among them — and the history of the record.
const props = defineProps<{ record: Record; around: Record[]; slug: string; drawn: string[] }>()
defineSlots<{ hero(): unknown; default(): unknown; side?(): unknown }>()

const corpus = useCorpus()
const tab = ref('overview')
const every = [
  { value: 'overview', label: said.tabs.overview },
  { value: 'history', label: said.tabs.history },
]
const tabs = computed(() => (corpus.history.value ? every : every.filter((t) => t.value !== 'history')))
watch(tabs, (now) => {
  if (!now.some((t) => t.value === tab.value)) tab.value = 'overview'
})
provideKept(computed(() => [props.record, ...props.around]))
const except = computed(() => [...props.drawn, 'body', 'media'])
</script>

<template>
  <Stack gap="6">
    <slot name="hero" />
    <Tabs v-model="tab" :tabs="tabs" :label="said.tabs.overview">
      <template #overview>
        <Columns>
          <Stack gap="8">
            <slot />
            <NotesBlock v-if="!drawn.includes('body')" :record="record" />
          </Stack>
          <template #side>
            <Stack gap="4">
              <ResearchCard :entity="record.ref" />
              <slot name="side" />
              <DetailsCard :record="record" :except="except" />
              <CitedCard :record="record" />
              <FilesCard :record="record" :around="around" :slug="slug" />
              <HistoryCard v-if="corpus.history.value" :entity="record.ref" @all="tab = 'history'" />
              <Stack row gap="2">
                <Text tone="faint"><Icon name="file" /></Text>
                <Text tone="faint" face="mono" size="xs" :title="said.disk">{{ record.file }}</Text>
              </Stack>
            </Stack>
          </template>
        </Columns>
      </template>
      <template #history>
        <HistoryList :entity="record.ref" />
      </template>
    </Tabs>
  </Stack>
</template>
