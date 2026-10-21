<script setup lang="ts">
import { computed, ref } from 'vue'
import Prose from './Prose.vue'
import Segmented from './Segmented.vue'
import TextArea from './TextArea.vue'
import { writing as said } from '../labels'

// Prose being written: the markdown, what it will look like as the server renders it,
// or the two side by side, one above the other where there is no room for both.
const model = defineModel<string>({ default: '' })
withDefaults(defineProps<{ html?: string; label?: string; path?: string; rows?: number; disabled?: boolean }>(), {
  html: '',
  rows: 8,
})

const view = ref<'write' | 'split' | 'preview'>('write')
const views = computed(() => [
  { value: 'write', label: said.write },
  { value: 'split', label: said.split },
  { value: 'preview', label: said.preview },
])
const chosen = computed({
  get: () => view.value,
  set: (v: string) => (view.value = v as typeof view.value),
})
</script>

<template>
  <div class="markdown">
    <Segmented v-model="chosen" :options="views" :label="said.view" />
    <div class="panes" :class="view">
      <TextArea v-show="view !== 'preview'" v-model="model" :label="label" :path="path" :rows="rows" :disabled="disabled" prose />
      <div v-if="view !== 'write'" class="preview" data-preview>
        <Prose :html="html" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.markdown {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.panes {
  display: grid;
  gap: var(--space-4);
}
.panes.split {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: start;
}
.preview {
  min-height: var(--textarea);
  padding: var(--space-3) var(--space-4);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  overflow-wrap: anywhere;
}
@container (max-width: 40em) {
  .panes.split {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
