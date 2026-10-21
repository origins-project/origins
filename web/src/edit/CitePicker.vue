<script setup lang="ts">
import { computed } from 'vue'
import { useCorpus } from '../api/corpus'
import { edit as said } from '../labels'
import { Combobox, Grid, Select, Stack, type Option } from '../ui'
import { positions, readCitation, unassessed, writeCitation, type Cited } from './citation'

// A citation, chosen from every source#anchor the corpus has, and the grade this use of
// it is given.
const model = defineModel<string>({ default: '' })
defineProps<{ label?: string; path?: string }>()
const { index } = useCorpus()

const options = computed<Option[]>(() =>
  (index.value?.sources ?? []).flatMap((source) => {
    const slug = source.ref.slice(source.ref.indexOf('/') + 1)
    const anchors = Array.isArray(source.anchors) ? (source.anchors as string[]) : []
    return anchors.map((anchor) => ({
      value: `${slug}#${anchor}`,
      label: `${typeof source.name === 'string' ? source.name : slug} · ${anchor}`,
    }))
  }),
)

const cited = computed(() => readCitation(model.value))
function change(part: keyof Cited, value: string | undefined) {
  model.value = writeCitation({ ...cited.value, [part]: value ?? unassessed })
}

const choices = (letters: readonly string[]): Option[] => [
  { value: unassessed, label: said.unassessed },
  ...letters.map((l) => ({ value: l, label: l })),
]
const grades = [
  { part: 'source', label: said.grade.source, options: choices(positions.source) },
  { part: 'information', label: said.grade.information, options: choices(positions.information) },
  { part: 'evidence', label: said.grade.evidence, options: choices(positions.evidence) },
  { part: 'weight', label: said.grade.weight, options: choices(positions.weight) },
] as const
</script>

<template>
  <Stack gap="2" :data-path="path">
    <Combobox
      :model-value="cited.ref"
      :label="label ?? said.citation"
      :options="options"
      mono
      @update:model-value="(ref) => change('ref', ref)"
    />
    <Grid :cols="4" gap="2">
      <Select
        v-for="grade in grades"
        :key="grade.part"
        :label="grade.label"
        :options="grade.options"
        :model-value="cited[grade.part]"
        :disabled="!cited.ref"
        @update:model-value="(v) => change(grade.part, v)"
      />
    </Grid>
  </Stack>
</template>
