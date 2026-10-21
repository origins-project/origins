<script setup lang="ts">
// A line across the whole width of the window, under the bar, in the accent: what a
// reader is to know before anything on the page, as the notice an export was made with.
</script>

<template>
  <div class="banner" role="note"><slot /></div>
</template>

<style scoped>
.banner {
  background: var(--accent);
  color: var(--on-accent);
  font-size: var(--text-md);
  font-weight: var(--weight-medium);
  line-height: var(--leading);
  text-align: center;
  padding: var(--space-2) var(--gutter);
  overflow-wrap: anywhere;
}
@media (max-width: 40em) {
  .banner {
    padding: var(--space-2) var(--gutter-phone);
  }
}
</style>
