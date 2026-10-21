<script setup lang="ts">
// A side card, or any surface lifted off the page: To research, Details, History. Given
// a title, it has a header with room for an action; given a footer, a footer.
withDefaults(defineProps<{ title?: string; tone?: 'plain' | 'research' | 'warn'; flush?: boolean }>(), {
  tone: 'plain',
})
</script>

<template>
  <section class="card" :class="[tone, { flush }]">
    <header v-if="title || $slots.action" class="head">
      <h3 class="title">{{ title }}</h3>
      <slot name="action" />
    </header>
    <div class="body"><slot /></div>
    <footer v-if="$slots.footer" class="foot"><slot name="footer" /></footer>
  </section>
</template>

<style scoped>
.card {
  background: var(--surface);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow);
  min-width: 0;
}
.head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4) var(--space-1);
}
.title {
  margin: 0;
  flex: 1;
  font: var(--weight-bold) var(--text-xs) var(--sans);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
}
.body {
  padding: var(--space-2) var(--space-4) var(--space-4);
}
.flush > .body {
  padding: 0;
}
.foot {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
  padding: var(--space-3) var(--space-4);
  border-top: var(--border) solid var(--line);
}
.research,
.warn {
  border-color: color-mix(in srgb, var(--warn) 30%, var(--line));
  background: color-mix(in srgb, var(--warn-soft) 45%, var(--surface));
}
.research .title,
.warn .title {
  color: var(--warn);
}
</style>
