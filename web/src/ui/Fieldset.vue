<script setup lang="ts">
// Several fields of one entry, or the entries of one list, under the name they share:
// the people an event names, its sources. Each entry is a line of it, its actions at
// the end of the line.
defineProps<{ legend: string; path?: string }>()
</script>

<template>
  <fieldset class="fieldset" :data-path="path">
    <legend class="legend">{{ legend }}</legend>
    <slot />
    <div v-if="$slots.add" class="add"><slot name="add" /></div>
  </fieldset>
</template>

<style scoped>
.fieldset {
  min-width: 0;
  margin: 0;
  padding: var(--space-2) var(--space-3) var(--space-3);
  border: var(--border) solid var(--line);
  border-radius: var(--radius);
  display: grid;
  gap: var(--space-3);
}
.legend {
  padding: 0 var(--space-1);
  font: var(--weight-bold) var(--text-2xs) var(--sans);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
}
.add {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
</style>
