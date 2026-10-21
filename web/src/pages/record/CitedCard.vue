<script setup lang="ts">
import { computed } from 'vue'
import type { Kind, Record } from '../../api/client'
import { valueAt } from '../../edit/values'
import { eventLabel, fieldLabel, kinds, record as said } from '../../labels'
import { citations } from '../../person'
import { Badge, Card, Chip, Stack, Text } from '../../ui'
import { useNaming } from '../../naming'

// Sources cited: each citation the record makes, once, with every place it is cited
// from and the grade it is given there.
const props = defineProps<{ record: Record }>()
const { called } = useNaming()
const cited = computed(() => citations(props.record.fields))

// What a citation is cited from: the event by its type, and the part of it where the
// citation is deeper in.
function from(path: string): string {
  if (path === '') return kinds[props.record.ref.slice(0, props.record.ref.indexOf('/')) as Kind].one
  const event = /^events\[(\d+)\](?:\.(.*))?$/.exec(path)
  if (!event) return fieldLabel(path) ?? path
  const type = eventLabel((valueAt(props.record.fields, `events[${event[1]}]`) as { type?: unknown } | undefined)?.type)
  return [type, event[2] && (fieldLabel(path) ?? event[2])].filter(Boolean).join(' · ')
}
</script>

<template>
  <Card v-if="cited.length" :title="said.cited">
    <Stack gap="3" as="ul">
      <Stack v-for="c in cited" :key="`${c.source} ${c.anchor}`" gap="1" as="li">
        <Stack row wrap gap="1" align="center">
          <Chip tone="source" icon="book" :to="`/source/${c.source}`">{{ called('source', c.source) }}</Chip>
          <Text v-if="c.anchor" tone="muted" size="sm" face="mono">{{ c.anchor }}</Text>
        </Stack>
        <Stack row wrap gap="2" align="center">
          <Stack v-for="(use, i) in c.uses" :key="i" row gap="1" align="center" as="span">
            <Text tone="muted" size="sm">{{ from(use.path) }}</Text>
            <Badge v-if="use.grade" mono>{{ use.grade }}</Badge>
          </Stack>
        </Stack>
      </Stack>
    </Stack>
  </Card>
</template>
