<script setup lang="ts">
import { useId } from 'vue'
import { Label } from 'reka-ui'
import './control.css'

// One line of text with its label, carrying the path of what it changes.
const model = defineModel<string>({ default: '' })
withDefaults(
  defineProps<{
    label?: string
    type?: 'text' | 'password' | 'search'
    placeholder?: string
    path?: string
    hint?: string
    error?: string
    mono?: boolean
    disabled?: boolean
  }>(),
  { type: 'text' },
)
const id = useId()
</script>

<template>
  <div class="field" :data-path="path">
    <Label v-if="label" :for="id" class="control-label">{{ label }}</Label>
    <input
      :id="id"
      v-model="model"
      class="control-input"
      :class="{ mono }"
      :type="type"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-invalid="error ? true : undefined"
      :aria-describedby="hint || error ? `${id}-said` : undefined"
    />
    <p v-if="error" :id="`${id}-said`" class="control-error">{{ error }}</p>
    <p v-else-if="hint" :id="`${id}-said`" class="control-hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
.field {
  min-width: 0;
}
.mono {
  font-family: var(--mono);
}
</style>
