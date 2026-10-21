<script setup lang="ts">
import { computed } from 'vue'
import type { Kind, Record } from '../../api/client'
import { useCorpus } from '../../api/corpus'
import EditButton from '../../edit/EditButton.vue'
import EditEntry from '../../edit/EditEntry.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import EditsList from '../../edit/EditsList.vue'
import { fieldLabel, hypothesis as said, kinds } from '../../labels'
import { useNaming } from '../../naming'
import { citationsOf, concerned } from '../../records'
import { Badge, Block, Card, Chip, Hero, Stack, SupposeButton, Text } from '../../ui'
import NotesBlock from '../record/NotesBlock.vue'
import RecordView from '../record/RecordView.vue'

// A hypothesis: its question and status, the switch reading every page with it supposed,
// then its argument, its edits and the records they change, and beside them the evidence
// it rests on and goes against, each with its grade.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { index, called } = useNaming()
const corpus = useCorpus()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const question = computed(() => called('hypothesis', props.slug))
const status = computed(() => text(props.record.fields.status))
const tone = computed(() => (status.value === 'accepted' ? 'good' : status.value === 'rejected' ? 'faint' : 'suppose'))
const confidence = computed(() => text(props.record.fields.confidence))
const proposedBy = computed(() => text(props.record.fields['proposed-by']))
// The argument without the question heading it, which the page's heading already is.
const argument = computed(() => (props.record.html.body ?? '').replace(/^\s*<h1[^>]*>[\s\S]*?<\/h1>\s*/, ''))

const supposing = computed(() => corpus.supposed.value.includes(props.slug))
function suppose() {
  const others = corpus.supposed.value.filter((s) => s !== props.slug)
  corpus.setSupposed(supposing.value ? others : [...others, props.slug])
}

const list = computed(() => (Array.isArray(props.record.fields.edits) ? props.record.fields.edits : []))
const known = computed(() => {
  const all = new Set<string>()
  if (index.value) for (const list of Object.values(index.value)) if (Array.isArray(list)) for (const one of list) all.add(one.ref)
  return all
})
const changes = computed(() => concerned(props.record.fields.edits))
const kindOf = (ref: string) => ref.slice(0, ref.indexOf('/')) as Kind
const slugOf = (ref: string) => ref.slice(ref.indexOf('/') + 1)
const evidence = (side: 'supports' | 'contradicts') => citationsOf(props.record.fields[side])
</script>

<template>
  <RecordView :record="record" :around="around" :slug="slug" :drawn="['edits', 'supports', 'contradicts', 'body']">
    <template #hero>
      <Hero>
        <template #title>{{ question }}</template>
        <Stack row wrap gap="2" align="center">
          <Badge v-if="status" :tone="tone">{{ said.status[status] ?? status }}</Badge>
          <Text v-if="confidence" tone="muted">{{ confidence }}</Text>
          <Text v-if="proposedBy" tone="muted" data-proposed-by>{{ said.proposedBy(proposedBy) }}</Text>
          <Text v-if="record.fields.base" face="mono" size="sm" tone="faint">{{ text(record.fields.base) }}</Text>
        </Stack>
        <template #actions>
          <SupposeButton v-if="corpus.offered.value" :label="said.suppose" :pressed="supposing" data-path="suppose" @click="suppose" />
          <RecordMenu />
        </template>
      </Hero>
    </template>

    <NotesBlock :record="record" :title="said.argument" :html="argument" />

    <Block :title="said.edits">
      <template #actions><EditEntry path="edits" :label="fieldLabel('edits')" list /></template>
      <EditsList v-slot="{ at }" :edits="list">
        <EditButton :path="`edits[${at}]`" :label="fieldLabel('edits')" />
      </EditsList>
    </Block>

    <Block :title="said.changes">
      <Stack row wrap gap="2">
        <template v-for="ref in changes" :key="ref">
          <Chip v-if="known.has(ref)" :kind="kinds[kindOf(ref)]?.one" :to="`/${ref}`">{{ called(kindOf(ref), slugOf(ref)) }}</Chip>
          <Chip v-else :kind="kinds[kindOf(ref)]?.one">{{ slugOf(ref) }}</Chip>
        </template>
      </Stack>
    </Block>

    <template #side>
      <Card v-for="side in ['supports', 'contradicts'] as const" :key="side" :title="said[side]">
        <template #action><EditEntry :path="side" :label="fieldLabel(side)" list /></template>
        <Stack gap="2" as="ul">
          <Stack v-for="(c, i) in evidence(side)" :key="i" row wrap gap="2" align="center" as="li">
            <Chip tone="source" icon="book" :to="`/source/${c.source}${c.anchor ? `#${c.anchor}` : ''}`">
              {{ called('source', c.source) }}
            </Chip>
            <Text v-if="c.anchor" face="mono" size="xs" tone="faint">{{ c.anchor }}</Text>
            <Badge v-if="c.grade" mono>{{ c.grade }}</Badge>
            <EditButton :path="`${side}[${i}]`" :label="fieldLabel(side)" />
          </Stack>
        </Stack>
      </Card>
    </template>
  </RecordView>
</template>
