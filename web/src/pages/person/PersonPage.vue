<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCorpus } from '../../api/corpus'
import type { Record } from '../../api/client'
import EditValue from '../../edit/EditValue.vue'
import { fieldLabel, person as said, record as shown } from '../../labels'
import { relatives as related, type IndexedPerson } from '../../person'
import { Block, Columns, Icon, Prose, Stack, Tabs, Text } from '../../ui'
import ChartTab from './ChartTab.vue'
import CitedCard from '../record/CitedCard.vue'
import DetailsCard from '../record/DetailsCard.vue'
import FamilyBlock from './FamilyBlock.vue'
import FilesCard from '../record/FilesCard.vue'
import HistoryCard from '../record/HistoryCard.vue'
import HistoryList from '../record/HistoryList.vue'
import LifeBlock from './LifeBlock.vue'
import PersonHero from './PersonHero.vue'
import ResearchCard from '../record/ResearchCard.vue'
import { provideKept, useNaming } from '../../naming'

// A person: the heading, then the overview — their life in order, their family, their
// notes and the cards beside them — the charts, and the history of the record.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { index, person } = useNaming()

const corpus = useCorpus()
const tab = ref('overview')
const every = [
  { value: 'overview', label: said.tabs.overview },
  { value: 'charts', label: said.tabs.charts },
  { value: 'history', label: said.tabs.history },
]
const tabs = computed(() => (corpus.history.value ? every : every.filter((t) => t.value !== 'history')))
watch(tabs, (now) => {
  if (!now.some((t) => t.value === tab.value)) tab.value = 'overview'
})

const indexed = computed<IndexedPerson | undefined>(() => person(props.slug))
const relatives = computed(() =>
  index.value ? related(index.value, props.slug) : { parents: [], partners: [], halfSiblings: [] },
)
const families = computed(() => {
  const mine = new Set(relatives.value.partners.map((u) => u.family.ref))
  return props.around.filter((r) => mine.has(r.ref))
})
provideKept(computed(() => [props.record, ...props.around]))
const body = computed(() => props.record.html.body ?? '')
</script>

<template>
  <Stack gap="6">
    <PersonHero :record="record" :indexed="indexed" />
    <Tabs v-model="tab" :tabs="tabs" :label="said.tabs.overview">
      <template #overview>
        <Columns>
          <Stack gap="8">
            <LifeBlock :record="record" :indexed="indexed" :families="families" :relatives="relatives" />
            <FamilyBlock :relatives="relatives" :families="families" />
            <Block :title="shown.notes">
              <template #actions>
                <EditValue path="body" :label="fieldLabel('body')"><Icon name="note" /></EditValue>
              </template>
              <Prose v-if="body" :html="body" />
            </Block>
          </Stack>
          <template #side>
            <Stack gap="4">
              <ResearchCard :entity="record.ref" />
              <DetailsCard
                :record="record"
                :only="['sex', 'living', 'private', 'ids', 'same-as', 'links', 'tags', 'not', 'sources']"
                undated
              />
              <CitedCard :record="record" />
              <FilesCard :record="record" :around="around" :slug="slug" />
              <HistoryCard v-if="corpus.history.value" :entity="record.ref" @all="tab = 'history'" />
              <Stack row gap="2">
                <Text tone="faint"><Icon name="file" /></Text>
                <Text tone="faint" face="mono" size="xs" :title="shown.disk">{{ record.file }}</Text>
              </Stack>
            </Stack>
          </template>
        </Columns>
      </template>
      <template #charts>
        <ChartTab :slug="slug" />
      </template>
      <template #history>
        <HistoryList :entity="record.ref" />
      </template>
    </Tabs>
  </Stack>
</template>
