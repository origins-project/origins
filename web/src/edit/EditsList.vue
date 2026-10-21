<script setup lang="ts">
import { computed } from 'vue'
import type { Kind } from '../api/client'
import { fieldLabel, hypothesis as said, kinds } from '../labels'
import { useNaming } from '../naming'
import { editOf } from '../records'
import { Badge, Chip, Stack, Text } from '../ui'
import { shown, type Value } from './values'

// Edits in words, as a hypothesis states them: what is done, to which records, at which
// field, with what value. Whatever acts on one is given its position.
const props = defineProps<{ edits: unknown[] }>()
defineSlots<{ default?(props: { at: number }): unknown }>()
const { index, called } = useNaming()

const stated = computed(() => props.edits.map((e) => editOf((e ?? {}) as Parameters<typeof editOf>[0])))
const known = computed(() => {
  const all = new Set<string>()
  if (index.value) for (const list of Object.values(index.value)) if (Array.isArray(list)) for (const one of list) all.add(one.ref)
  return all
})
const kindOf = (ref: string) => ref.slice(0, ref.indexOf('/')) as Kind
const slugOf = (ref: string) => ref.slice(ref.indexOf('/') + 1)
</script>

<template>
  <Stack gap="3" as="ol">
    <Stack v-for="(e, i) in stated" :key="i" row wrap gap="2" align="center" as="li">
      <Badge tone="suppose">{{ said.ops[e.op] ?? e.op }}</Badge>
      <template v-for="ref in e.entities" :key="ref">
        <Chip v-if="known.has(ref)" :kind="kinds[kindOf(ref)]?.one" :to="`/${ref}`">{{ called(kindOf(ref), slugOf(ref)) }}</Chip>
        <Text v-else face="mono" size="sm">{{ ref }}</Text>
      </template>
      <Text v-if="e.path" tone="muted">{{ fieldLabel(e.path) ?? e.path }}</Text>
      <Text v-if="e.path" face="mono" size="xs" tone="faint">{{ e.path }}</Text>
      <Text v-if="e.value !== undefined">{{ shown(e.value as Value) }}</Text>
      <slot :at="i" />
    </Stack>
  </Stack>
</template>
