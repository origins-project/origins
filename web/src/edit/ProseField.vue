<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { Markdown } from '../ui'
import { useChange, useEditing } from './change'

// Prose changed beside what it will look like, rendered by the server for the record
// it is written on, so that the preview is the page.
const model = defineModel<string>({ default: '' })
defineProps<{ label?: string; path?: string }>()

const { entity } = useEditing()
const { client } = useChange()

const html = ref('')
let asked = 0
let timer: ReturnType<typeof setTimeout> | undefined
async function render(prose: string) {
  const mine = ++asked
  try {
    const rendered = prose.trim() ? (await client.render(entity.value, prose)).html : ''
    if (mine === asked) html.value = rendered
  } catch {
    // the preview stays as it was; the prose is what is saved
  }
}
watch(
  model,
  (prose) => {
    clearTimeout(timer)
    timer = setTimeout(() => void render(prose), 250)
  },
  { immediate: true },
)
onBeforeUnmount(() => clearTimeout(timer))
</script>

<template>
  <Markdown v-model="model" :html="html" :label="label" :path="path" />
</template>
