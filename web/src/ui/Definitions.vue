<script setup lang="ts">
// Names and what each names, one under the other: a record's fields, what a change
// comes to. Each `Definition` is a pair.
defineProps<{ compact?: boolean }>()
</script>

<template>
  <dl class="definitions" :class="{ compact }"><slot /></dl>
</template>

<style scoped>
.definitions {
  margin: 0;
  display: grid;
  grid-template-columns: minmax(0, max-content) minmax(0, 1fr);
  gap: var(--space-2) var(--space-4);
  align-items: baseline;
}
.compact {
  gap: var(--space-1) var(--space-3);
  font-size: var(--text-md);
}
@media (max-width: 40em) {
  .definitions {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-0);
  }
}
</style>
