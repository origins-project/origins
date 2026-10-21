<script setup lang="ts">
// A value as a page shows it, which opens what changes it on a click or on Enter: a
// popover for one value, a drawer for an entry. It is a button, so it is reached by
// the keyboard as every control is. One a draft changes is marked in the supposing colour.
defineProps<{ path?: string; label?: string; mono?: boolean; disabled?: boolean; marked?: boolean }>()
</script>

<template>
  <button
    type="button"
    class="value"
    :class="{ mono, marked }"
    :data-path="path"
    :data-drafted="marked ? '' : undefined"
    :aria-label="label"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>

<style scoped>
.value {
  all: unset;
  box-sizing: border-box;
  display: inline;
  min-width: 0;
  max-width: 100%;
  overflow-wrap: anywhere;
  border-radius: var(--radius-sm);
  padding: 0 var(--space-0);
  margin: 0 calc(-1 * var(--space-0));
  cursor: pointer;
  transition: background var(--quick);
}
.value:hover,
.value[data-state='open'] {
  background: var(--act-soft);
}
.value:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border);
}
.value:disabled {
  cursor: default;
  background: none;
}
.marked {
  background: var(--suppose-soft);
  box-shadow: inset 0 calc(-1 * var(--border-strong)) 0 var(--suppose);
}
.mono {
  font-family: var(--mono);
  font-size: var(--text-md);
}
</style>
