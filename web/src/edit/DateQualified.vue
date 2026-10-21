<script setup lang="ts">
import { computed } from 'vue'
import { dates } from '../labels'
import { Grid, Select, Stack, Switch, Text, type Option } from '../ui'
import type { Qualified, Qualifier, Reached } from '../date'
import DateValue from './DateValue.vue'

// A date that is not a period: how it was reached, its qualifier where the shape does
// not give it, and the one value or the two a range runs between.
const date = defineModel<Qualified>({ required: true })
const props = defineProps<{ title?: string; free?: boolean }>()

const stated = '-'
const reachedOptions: Option[] = (Object.entries(dates.reachedAs) as [Reached, string][]).map(([v, label]) => ({
  value: v || stated,
  label,
}))
const qualifiers: Option[] = [
  { value: stated, label: dates.shapes.one },
  { value: 'ABT', label: dates.about },
  { value: 'BEF', label: dates.shapes.before },
  { value: 'AFT', label: dates.shapes.after },
  { value: 'BET', label: dates.shapes.between },
]
const exact = computed(() => date.value.qualifier === '' || date.value.qualifier === 'ABT')
</script>

<template>
  <Stack gap="3">
    <Text v-if="title" caps size="2xs" tone="faint">{{ title }}</Text>
    <Grid :cols="props.free ? 2 : 1" gap="2">
      <Select
        :model-value="date.reached || stated"
        :label="dates.reached"
        :options="reachedOptions"
        @update:model-value="(r) => (date.reached = (r === stated ? '' : r) as Reached)"
      />
      <Select
        v-if="free"
        :model-value="date.qualifier || stated"
        :label="dates.shape"
        :options="qualifiers"
        @update:model-value="(q) => (date.qualifier = (q === stated ? '' : q) as Qualifier)"
      />
    </Grid>
    <Switch
      v-if="!free && exact"
      :model-value="date.qualifier === 'ABT'"
      :label="dates.about"
      @update:model-value="(on) => (date.qualifier = on ? 'ABT' : '')"
    />
    <DateValue v-model="date.value" />
    <template v-if="date.qualifier === 'BET'">
      <Text caps size="2xs" tone="faint">{{ dates.end }}</Text>
      <DateValue v-model="date.end" />
    </template>
  </Stack>
</template>
