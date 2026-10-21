<script setup lang="ts">
// A life in order, each `Moment` on a line down the page with its year beside it.
</script>

<template>
  <ol class="timeline"><slot /></ol>
</template>

<style scoped>
.timeline {
  position: relative;
  margin: 0;
  padding: 0;
  list-style: none;
  min-width: 0;
}
.timeline::before {
  content: '';
  position: absolute;
  left: calc(var(--year) + var(--space-6) / 2 + var(--space-1));
  top: var(--space-3);
  bottom: var(--space-3);
  width: var(--border-strong);
  background: var(--line);
}
@media (max-width: 40em) {
  .timeline::before {
    left: calc(var(--year-phone) + var(--space-5) / 2 + var(--space-1));
  }
}
</style>
