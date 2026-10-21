<script setup lang="ts">
import { computed } from 'vue'
import type { Kind } from '../api/client'
import { useCorpus } from '../api/corpus'
import { listed, names } from '../search'
import { Combobox, type Option } from '../ui'

// A record of a kind, by its slug, chosen from the index by any of its names.
const model = defineModel<string>({ default: '' })
const props = defineProps<{ kind: Kind; label?: string; path?: string; error?: string }>()
const { index } = useCorpus()

const options = computed<Option[]>(() => {
  if (!index.value) return []
  const people = new Map(index.value.people.map((p) => [p.ref, p]))
  return index.value[listed[props.kind]].map((record) => {
    const slug = record.ref.slice(record.ref.indexOf('/') + 1)
    return { value: slug, label: names(props.kind, record, people).name, hint: slug }
  })
})
</script>

<template>
  <Combobox v-model="model" :label="label" :path="path" :options="options" :error="error" mono />
</template>
