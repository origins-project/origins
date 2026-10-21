<script setup lang="ts">
import { useId } from 'vue'
import { Label } from 'reka-ui'
import './control.css'

// Several lines of text, markdown or a transcription, with its label.
const model = defineModel<string>({ default: '' })
defineProps<{
  label?: string
  placeholder?: string
  path?: string
  hint?: string
  error?: string
  rows?: number
  prose?: boolean
  disabled?: boolean
}>()
const id = useId()
</script>

<template>
  <div class="text-area" :data-path="path">
    <Label v-if="label" :for="id" class="control-label">{{ label }}</Label>
    <textarea
      :id="id"
      v-model="model"
      class="control-input"
      :class="{ prose }"
      :rows="rows"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-invalid="error ? true : undefined"
    />
    <p v-if="error" class="control-error">{{ error }}</p>
    <p v-else-if="hint" class="control-hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.text-area {
  min-width: 0;
}
textarea {
  display: block;
  min-height: var(--textarea);
  resize: vertical;
  font: var(--text-md) / var(--leading) var(--mono);
}
textarea.prose {
  font: var(--text-lg) / var(--leading-prose) var(--serif);
}
</style>
