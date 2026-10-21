<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditButton from '../../edit/EditButton.vue'
import EditEntry from '../../edit/EditEntry.vue'
import EditValue from '../../edit/EditValue.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { eventLabel, fieldLabel, hypothesis as supposed, source as said } from '../../labels'
import { useNaming } from '../../naming'
import { citationsOf, citedBy } from '../../records'
import { Badge, Block, Card, Chip, Hero, Stack, Text } from '../../ui'
import RecordView from '../record/RecordView.vue'
import CitationCard from './CitationCard.vue'

// A source: its title, short title, type and author, then its citations, each with
// whatever cites it, and beside them where it is kept, what it covers and the
// hypotheses that rest on it.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { called } = useNaming()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const anchors = computed(() => Object.keys((props.record.fields.citations as object | undefined) ?? {}))
const citing = computed(() => citedBy(props.around, props.slug))

const kept = computed(() => {
  const list = props.record.fields.repositories
  return (Array.isArray(list) ? list : []).map((one: unknown, i) => {
    const o = (typeof one === 'object' && one !== null ? one : {}) as { [field: string]: unknown }
    return {
      path: `repositories[${i}]`,
      repository: typeof one === 'string' ? one : text(o.repository),
      call: text(o['call-number']),
      medium: text(o.medium),
    }
  })
})

const covers = computed(() => {
  const list = props.record.fields.covers
  return (Array.isArray(list) ? list : []).map((one: { [field: string]: unknown }, i) => ({
    path: `covers[${i}]`,
    events: (Array.isArray(one?.events) ? one.events : []).map(eventLabel).join(', '),
    date: spoken(text(one?.date)),
    place: text(one?.place),
  }))
})

// The hypotheses citing the source as evidence, each citation with whether it supports
// or contradicts it.
const hypotheses = computed(() =>
  props.around
    .filter((r) => r.ref.startsWith('hypothesis/'))
    .flatMap((r) =>
      (['supports', 'contradicts'] as const).flatMap((side) =>
        citationsOf(r.fields[side])
          .filter((c) => c.source === props.slug)
          .map((c) => ({ ref: r.ref, side, ...c })),
      ),
    ),
)
</script>

<template>
  <RecordView
    :record="record"
    :around="around"
    :slug="slug"
    :drawn="['title', 'short-title', 'type', 'author', 'citations', 'repositories', 'covers']"
  >
    <template #hero>
      <Hero>
        <template #title><EditValue path="title" :label="fieldLabel('title')" /></template>
        <Stack row wrap gap="3" align="center">
          <EditValue path="short-title" :label="fieldLabel('short-title')" />
          <EditValue path="type" :label="fieldLabel('type')" />
          <EditValue path="author" :label="fieldLabel('author')" />
        </Stack>
        <template #actions><RecordMenu /></template>
      </Hero>
    </template>

    <Block :title="said.citations">
      <template #actions><EditButton path="citations" :label="fieldLabel('citations')" /></template>
      <Stack gap="4">
        <CitationCard v-for="anchor in anchors" :key="anchor" :record="record" :anchor="anchor" :citing="citing.get(anchor) ?? []" />
      </Stack>
    </Block>

    <template #side>
      <Card :title="said.kept">
        <template #action><EditEntry path="repositories" :label="fieldLabel('repositories')" list /></template>
        <Stack gap="3" as="ul">
          <Stack v-for="k in kept" :key="k.path" gap="1" as="li">
            <Stack row gap="1" align="center" justify="between">
              <Chip icon="archive" :to="`/repository/${k.repository}`">{{ called('repository', k.repository) }}</Chip>
              <EditButton :path="k.path" :label="fieldLabel('repositories')" />
            </Stack>
            <Stack row wrap gap="2" align="center">
              <Text v-if="k.call" face="mono" size="sm">{{ k.call }}</Text>
              <Badge v-if="k.medium" tone="faint">{{ k.medium }}</Badge>
            </Stack>
          </Stack>
        </Stack>
      </Card>
      <Card :title="said.covers">
        <template #action><EditEntry path="covers" :label="fieldLabel('covers')" list /></template>
        <Stack gap="3" as="ul">
          <Stack v-for="c in covers" :key="c.path" row wrap gap="2" align="center" as="li">
            <Text v-if="c.events">{{ c.events }}</Text>
            <Text v-if="c.date" tone="muted" size="sm">{{ c.date }}</Text>
            <Chip v-if="c.place" icon="pin" :to="`/place/${c.place}`">{{ called('place', c.place) }}</Chip>
            <EditButton :path="c.path" :label="fieldLabel('covers')" />
          </Stack>
        </Stack>
      </Card>
      <Card v-if="hypotheses.length" :title="said.hypotheses">
        <Stack gap="2" as="ul">
          <Stack v-for="(h, i) in hypotheses" :key="i" row wrap gap="2" align="center" as="li">
            <Chip icon="flask" tone="suppose" :to="`/${h.ref}`">{{ called('hypothesis', h.ref.slice(h.ref.indexOf('/') + 1)) }}</Chip>
            <Text tone="muted" size="sm">{{ supposed[h.side] }}</Text>
            <Text v-if="h.anchor" face="mono" size="xs" tone="faint">{{ h.anchor }}</Text>
            <Badge v-if="h.grade" mono>{{ h.grade }}</Badge>
          </Stack>
        </Stack>
      </Card>
    </template>
  </RecordView>
</template>
