<script setup lang="ts">
// A page's body: as wide as the tokens allow and centred, with a head of a title, the
// line under it and the page's own actions, before whatever the page holds; a narrow
// one holds a single form, centred in the window below the bar.
defineProps<{ title?: string; narrow?: boolean }>()
</script>

<template>
  <main class="page" :class="{ narrow }">
    <header v-if="title || $slots.head || $slots.actions" class="head">
      <div class="heading">
        <h1 v-if="title">{{ title }}</h1>
        <slot name="head" />
      </div>
      <div v-if="$slots.actions" class="actions"><slot name="actions" /></div>
    </header>
    <slot />
  </main>
</template>

<style scoped>
.page {
  max-width: var(--page);
  margin: 0 auto;
  padding: var(--space-8) var(--gutter) var(--space-12);
  min-width: 0;
}
.narrow {
  max-width: calc(var(--popover) + 2 * var(--gutter));
  min-height: calc(100dvh - var(--bar));
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
  margin-bottom: var(--space-8);
}
.heading {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}
.actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}
@media (max-width: 40em) {
  .page {
    padding: var(--space-5) var(--gutter-phone) var(--space-10);
  }
}
</style>
