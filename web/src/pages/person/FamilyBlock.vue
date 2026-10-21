<script setup lang="ts">
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import { eventLabel, person as said } from '../../labels'
import { initials, years, type Relative, type Relatives } from '../../person'
import { Badge, Block, Button, Card, Grid, PersonCard, Stack, Text } from '../../ui'
import { served, useNaming } from '../../naming'

// Family: the families the person is a child of, with their parents and siblings, the
// half-siblings apart, and each family they are a partner in as a card with its type,
// its events and its children.
const props = defineProps<{ relatives: Relatives; families: Record[] }>()
const { called } = useNaming()

function card(relative: Relative) {
  const portrait = relative.person?.portrait
  return {
    name: relative.name,
    to: relative.ref ? `/${relative.ref}` : undefined,
    lifespan: years(relative.person),
    relation: relative.relation,
    sex: relative.person?.sex,
    src: portrait && served(portrait.record, portrait.file),
    region: portrait?.region,
    initials: initials(relative.name),
    struck: relative.status === 'disproven',
    ghost: !relative.ref,
  }
}

const text = (v: unknown) => (typeof v === 'string' ? v : '')

function events(ref: string) {
  const events = props.families.find((f) => f.ref === ref)?.fields.events
  if (!Array.isArray(events)) return []
  return events.map((e: { [field: string]: unknown }) =>
    [eventLabel(e.type), spoken(text(e.date)), text(e.place) && called('place', text(e.place))].filter(Boolean).join(' · '),
  )
}

function first(union: Relatives['parents'][number]) {
  const f = union.family.first
  if (!f) return ''
  return [eventLabel(f.type), spoken(f.date ?? ''), f.place && called('place', f.place)].filter(Boolean).join(' · ')
}
</script>

<template>
  <Block :title="said.family">
    <Stack gap="4">
      <Card v-for="union in relatives.parents" :key="union.family.ref" :title="said.parents">
        <template #action>
          <Button size="sm" variant="ghost" icon="family" :to="`/${union.family.ref}`">{{ said.familyPage }}</Button>
        </template>
        <Stack gap="3">
          <Stack row wrap gap="2" align="baseline">
            <Badge v-if="union.family.type">{{ union.family.type }}</Badge>
            <Badge v-if="union.status" tone="warn" :struck="union.status === 'disproven'">{{ union.status }}</Badge>
            <Text tone="muted" size="sm">{{ first(union) }}</Text>
          </Stack>
          <Grid fit="card" gap="2">
            <PersonCard v-for="(p, i) in union.partners" :key="i" v-bind="card(p)" />
          </Grid>
          <template v-if="union.children.length">
            <Text caps size="xs" tone="muted">{{ said.siblings }}</Text>
            <Grid fit="card" gap="2">
              <PersonCard v-for="(c, i) in union.children" :key="i" v-bind="card(c)" />
            </Grid>
          </template>
        </Stack>
      </Card>
      <Card v-if="relatives.halfSiblings.length" :title="said.halfSiblings">
        <Grid fit="card" gap="2">
          <PersonCard v-for="(c, i) in relatives.halfSiblings" :key="i" v-bind="card(c)" />
        </Grid>
      </Card>
      <Card v-for="union in relatives.partners" :key="union.family.ref" :title="said.with">
        <template #action>
          <Button size="sm" variant="ghost" icon="family" :to="`/${union.family.ref}`">{{ said.familyPage }}</Button>
        </template>
        <Stack gap="3">
          <Grid v-if="union.partners.length" fit="card" gap="2">
            <PersonCard v-for="(p, i) in union.partners" :key="i" v-bind="card(p)" />
          </Grid>
          <Stack row wrap gap="2" align="baseline">
            <Badge v-if="union.family.type">{{ union.family.type }}</Badge>
            <Text v-for="(line, i) in events(union.family.ref)" :key="i" tone="muted" size="sm">{{ line }}</Text>
          </Stack>
          <template v-if="union.children.length">
            <Text caps size="xs" tone="muted">{{ said.children }}</Text>
            <Grid fit="card" gap="2">
              <PersonCard v-for="(c, i) in union.children" :key="i" v-bind="card(c)" />
            </Grid>
          </template>
        </Stack>
      </Card>
    </Stack>
  </Block>
</template>
