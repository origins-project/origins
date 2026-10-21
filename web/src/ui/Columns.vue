<script setup lang="ts">
// A page's two columns: the record in the main one and its cards in the side one,
// which stays in view as the main one scrolls. Under the width a laptop leaves, the
// side one goes below the main one.
</script>

<template>
  <div class="columns" :class="{ single: !$slots.side }">
    <div class="main"><slot /></div>
    <aside v-if="$slots.side" class="side"><slot name="side" /></aside>
  </div>
</template>

<style scoped>
.columns {
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--side);
  gap: var(--space-10);
  align-items: start;
}
.single {
  grid-template-columns: minmax(0, 1fr);
}
.main {
  min-width: 0;
}
.side {
  position: sticky;
  top: calc(var(--bar) + var(--space-6));
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  min-width: 0;
}
/* A breakpoint cannot be a custom property, so the width is written here: 60em, the
   narrowest the two columns fit side by side. */
@media (max-width: 60em) {
  .columns {
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-8);
  }
  .side {
    position: static;
  }
}
</style>
