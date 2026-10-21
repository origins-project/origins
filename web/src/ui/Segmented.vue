<script setup lang="ts">
import { ToggleGroupItem, ToggleGroupRoot } from 'reka-ui'
import type { Option } from './Select.vue'

// One of a few choices shown side by side: a date's shape, a calendar, a theme.
const model = defineModel<string>()
defineProps<{ options: Option[]; label: string; path?: string; disabled?: boolean }>()

// Reka hands back no value when the pressed item is pressed again; the choice stays.
function choose(value: unknown) {
  if (typeof value === 'string' && value !== '') model.value = value
}
</script>

<template>
  <ToggleGroupRoot
    type="single"
    class="segmented"
    :model-value="model"
    :aria-label="label"
    :data-path="path"
    :disabled="disabled"
    @update:model-value="choose"
  >
    <ToggleGroupItem v-for="option in options" :key="option.value" :value="option.value" class="item">
      {{ option.label }}
    </ToggleGroupItem>
  </ToggleGroupRoot>
</template>

<style scoped>
.segmented {
  display: inline-flex;
  flex-wrap: wrap;
  justify-self: start;
  align-self: start;
  max-width: 100%;
  border: var(--border) solid var(--line-strong);
  border-radius: var(--radius);
  overflow: hidden;
}
.item {
  border: 0;
  background: var(--surface);
  padding: var(--space-1) var(--space-3);
  font-size: var(--text-sm);
  cursor: pointer;
  color: var(--muted);
}
.item + .item {
  border-left: var(--border) solid var(--line);
}
.item:hover {
  color: var(--ink);
}
.item[data-state='on'] {
  background: var(--act-soft);
  color: var(--act);
  font-weight: var(--weight-medium);
}
.item:focus-visible {
  outline-offset: calc(-1 * var(--border-strong));
}
</style>
