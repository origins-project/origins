<script setup lang="ts">
// A section of a page's main column with its heading, and its actions beside it: Life,
// Family, Notes.
defineProps<{ title: string }>()
</script>

<template>
  <section class="block">
    <header class="head">
      <h2>{{ title }}</h2>
      <div class="actions"><slot name="actions" /></div>
    </header>
    <div class="body"><slot /></div>
  </section>
</template>

<style scoped>
.block {
  margin-bottom: var(--space-10);
  min-width: 0;
}
.head {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
}
.body {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  min-width: 0;
}
.actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
</style>
