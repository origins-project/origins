<script setup lang="ts">
import { useId } from 'vue'
import { Label } from 'reka-ui'
import IconButton from './IconButton.vue'
import { actions } from '../labels'
import './control.css'

// A date as origins writes it, typed, beside the button that opens the dialog building
// one; what the text reads as is said under it by whoever parsed it.
const model = defineModel<string>({ default: '' })
defineProps<{
  label?: string
  path?: string
  reads?: string
  error?: string
  disabled?: boolean
}>()
defineEmits<{ build: [] }>()
const id = useId()
</script>

<template>
  <div class="date-input" :data-path="path">
    <Label v-if="label" :for="id" class="control-label">{{ label }}</Label>
    <div class="row">
      <input
        :id="id"
        v-model="model"
        class="control-input"
        type="text"
        spellcheck="false"
        :disabled="disabled"
        :aria-invalid="error ? true : undefined"
      />
      <IconButton icon="calendar" :label="actions.buildDate" bordered :disabled="disabled" @click="$emit('build')" />
    </div>
    <p v-if="error" class="control-error">{{ error }}</p>
    <p v-else-if="reads" class="control-hint">{{ reads }}</p>
  </div>
</template>

<style scoped>
.date-input {
  min-width: 0;
}
.row {
  display: flex;
  gap: var(--space-2);
}
input {
  font-family: var(--mono);
}
</style>
