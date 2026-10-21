<script setup lang="ts">
import Icon from './Icon.vue'

// Suppose: the switch between the corpus and the tab's draft, filled in the supposing
// colour while it is on.
defineProps<{ label: string; pressed?: boolean; disabled?: boolean }>()
</script>

<template>
  <button type="button" class="suppose" :aria-pressed="pressed" :disabled="disabled">
    <Icon name="flask" />
    <span>{{ label }}</span>
  </button>
</template>

<style scoped>
.suppose {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--control);
  padding: 0 var(--space-3);
  border-radius: var(--radius-pill);
  border: var(--border) solid var(--line);
  background: var(--surface);
  color: var(--muted);
  cursor: pointer;
  font-size: var(--text-md);
  white-space: nowrap;
}
.suppose:hover {
  color: var(--suppose);
  border-color: color-mix(in srgb, var(--suppose) 40%, var(--line));
}
.suppose[aria-pressed='true'] {
  background: var(--suppose);
  border-color: var(--suppose);
  color: var(--on-suppose);
}
.suppose:disabled {
  opacity: 0.5;
  pointer-events: none;
}
</style>
