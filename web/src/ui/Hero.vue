<script setup lang="ts">
// The head of a page about somebody: their picture, their name as the page's title,
// what is said of them under it, and the page's own actions.
</script>

<template>
  <header class="hero">
    <div v-if="$slots.portrait" class="portrait"><slot name="portrait" /></div>
    <div class="heading">
      <h1 class="title"><slot name="title" /></h1>
      <slot />
    </div>
    <div v-if="$slots.actions" class="actions"><slot name="actions" /></div>
  </header>
</template>

<style scoped>
.hero {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: var(--space-6);
  align-items: center;
  margin-bottom: var(--space-6);
}
.hero:not(:has(> .portrait)) {
  grid-template-columns: minmax(0, 1fr) auto;
}
.heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}
.title {
  margin: 0;
  font: var(--text-display) / var(--leading-tight) var(--serif);
  letter-spacing: -0.015em;
  overflow-wrap: anywhere;
}
.actions {
  align-self: start;
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}
@media (max-width: 40em) {
  .hero {
    grid-template-columns: auto minmax(0, 1fr);
    gap: var(--space-4);
    align-items: start;
  }
  .actions {
    grid-column: 1 / -1;
  }
}
</style>
