<script setup lang="ts">
import { computed, useId } from 'vue'
import {
  ComboboxAnchor,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxPortal,
  ComboboxRoot,
  ComboboxViewport,
  Label,
} from 'reka-ui'
import { fold } from '../search'
import type { Option } from './Select.vue'
import './control.css'
import { usePortal } from './theme'

// Text typed, or one of a list chosen for it as it is typed: a record's slug, a
// citation. What is typed stands whether or not the list has it.
const model = defineModel<string>({ default: '' })
const props = withDefaults(
  defineProps<{
    label?: string
    options: Option[]
    path?: string
    placeholder?: string
    error?: string
    mono?: boolean
    disabled?: boolean
    // how many of the options matching what is typed are listed
    shown?: number
  }>(),
  { shown: 30 },
)
const id = useId()
const portal = usePortal()

const matching = computed(() => {
  const words = fold(model.value).text.split(/\s+/).filter(Boolean)
  return props.options
    .filter((o) => {
      const text = fold(`${o.label} ${o.value} ${o.hint ?? ''}`).text
      return words.every((w) => text.includes(w))
    })
    .slice(0, props.shown)
})

function choose(value: unknown) {
  if (typeof value === 'string') model.value = value
}
</script>

<template>
  <div class="combobox" :data-path="path">
    <Label v-if="label" :for="id" class="control-label">{{ label }}</Label>
    <ComboboxRoot
      :model-value="model"
      ignore-filter
      :reset-search-term-on-blur="false"
      :reset-search-term-on-select="false"
      open-on-click
      :disabled="disabled"
      @update:model-value="choose"
    >
      <ComboboxAnchor class="anchor">
        <ComboboxInput
          :id="id"
          v-model="model"
          class="control-input"
          :class="{ mono }"
          :placeholder="placeholder"
          spellcheck="false"
          autocomplete="off"
          :aria-invalid="error ? true : undefined"
        />
      </ComboboxAnchor>
      <ComboboxPortal :to="portal">
        <ComboboxContent v-if="matching.length" class="content" position="popper" :side-offset="4">
          <ComboboxViewport>
            <ComboboxItem v-for="option in matching" :key="option.value" :value="option.value" class="item">
              <span class="name">{{ option.label }}</span>
              <span v-if="option.hint" class="hint">{{ option.hint }}</span>
            </ComboboxItem>
          </ComboboxViewport>
        </ComboboxContent>
      </ComboboxPortal>
    </ComboboxRoot>
    <p v-if="error" class="control-error">{{ error }}</p>
  </div>
</template>

<style scoped>
.combobox {
  min-width: 0;
}
.mono {
  font-family: var(--mono);
}
/* Reka draws the content inside a positioned wrapper, which takes this scope. */
.content,
:deep(.content) {
  z-index: var(--layer-overlay);
  width: var(--reka-combobox-trigger-width);
  max-height: min(var(--palette-list), var(--reka-combobox-content-available-height));
  overflow-y: auto;
  background: var(--surface);
  color: var(--ink);
  border: var(--border) solid var(--line);
  border-radius: var(--radius);
  box-shadow: var(--shadow-lg);
  padding: var(--space-1);
}
.item {
  display: flex;
  align-items: baseline;
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
.name {
  min-width: 0;
  overflow-wrap: anywhere;
}
.hint {
  color: var(--faint);
  font: var(--text-2xs) var(--mono);
  margin-left: auto;
  flex: none;
}
</style>
