<script setup lang="ts">
import { computed } from 'vue'
import type { Fit, Space } from './layout'

// Cells in rows: a number of equal columns, or as many as fit, each at least one of
// the widths the tokens name. Equal columns fall to one on a phone. Packed, the cells
// fill as many columns as fit one under another, each as tall as what it holds, so
// that a long one beside short ones leaves no row as tall as itself.
const props = withDefaults(defineProps<{ cols?: number; fit?: Fit; gap?: Space; packed?: boolean }>(), { gap: '4' })
const style = computed(() => ({
  gap: `var(--space-${props.gap})`,
  '--gap': `var(--space-${props.gap})`,
  '--width': props.fit ? `var(--fit-${props.fit})` : undefined,
  '--columns': props.fit
    ? `repeat(auto-fill, minmax(min(var(--fit-${props.fit}), 100%), 1fr))`
    : `repeat(${props.cols ?? 2}, minmax(0, 1fr))`,
}))
</script>

<template>
  <div class="grid" :class="{ fixed: !fit, packed }" :style="style"><slot /></div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: var(--columns);
  min-width: 0;
}
.packed {
  display: block;
  columns: var(--width, var(--fit-wide)) auto;
}
.packed > :slotted(*) {
  break-inside: avoid;
  margin-block-end: var(--gap);
}
/* A breakpoint cannot be a custom property, so the width is written here: 40em, a
   phone's. */
@media (max-width: 40em) {
  .fixed {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
