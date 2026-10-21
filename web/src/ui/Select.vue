<script setup lang="ts">
import { useId } from 'vue'
import {
  Label,
  SelectContent,
  SelectItem,
  SelectItemIndicator,
  SelectItemText,
  SelectPortal,
  SelectRoot,
  SelectTrigger,
  SelectValue,
  SelectViewport,
} from 'reka-ui'
import Icon from './Icon.vue'
import './control.css'
import { usePortal } from './theme'

export interface Option {
  value: string
  label: string
  hint?: string
  disabled?: boolean
}

// One of a few words, chosen from a list; for a word of an open vocabulary, the field.
const model = defineModel<string>()
defineProps<{
  label?: string
  options: Option[]
  placeholder?: string
  path?: string
  disabled?: boolean
}>()
const id = useId()
const portal = usePortal()
</script>

<template>
  <div class="select" :data-path="path">
    <Label v-if="label" :for="id" class="control-label">{{ label }}</Label>
    <SelectRoot v-model="model" :disabled="disabled">
      <SelectTrigger :id="id" class="control-input trigger">
        <SelectValue :placeholder="placeholder" />
        <Icon name="chevron" />
      </SelectTrigger>
      <SelectPortal :to="portal">
        <SelectContent class="content" position="popper" :side-offset="4">
          <SelectViewport>
            <SelectItem v-for="option in options" :key="option.value" :value="option.value" class="item">
              <SelectItemText>{{ option.label }}</SelectItemText>
              <span v-if="option.hint" class="hint">{{ option.hint }}</span>
              <SelectItemIndicator class="tick"><Icon name="check" /></SelectItemIndicator>
            </SelectItem>
          </SelectViewport>
        </SelectContent>
      </SelectPortal>
    </SelectRoot>
  </div>
</template>

<style scoped>
.select {
  min-width: 0;
}
.trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  text-align: left;
  cursor: pointer;
}
.trigger[data-placeholder] {
  color: var(--faint);
}
/* Reka draws the content inside a positioned wrapper, which takes this scope. */
.content,
:deep(.content) {
  z-index: var(--layer-overlay);
  min-width: var(--reka-select-trigger-width);
  max-height: var(--reka-select-content-available-height);
  background: var(--surface);
  color: var(--ink);
  border: var(--border) solid var(--line);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  padding: var(--space-1);
}
.item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-size: var(--text-md);
  cursor: pointer;
  outline: none;
}
.item[data-highlighted] {
  background: var(--surface-2);
}
.hint {
  color: var(--faint);
  font-size: var(--text-xs);
  margin-left: auto;
}
.tick {
  color: var(--act);
  margin-left: auto;
}
.hint + .tick {
  margin-left: 0;
}
</style>
