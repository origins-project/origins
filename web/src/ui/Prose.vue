<script setup lang="ts">
// A record's prose as the server rendered it: markdown made HTML, every link it holds
// already the route of what it leads to, and every tag but the few it keeps shown as
// text, so that what is drawn here is never the corpus's own markup.
import { computed } from 'vue'
import { address, exported } from '../api/client'

const props = defineProps<{ html: string }>()

// In an export every route and file a link names is where the export wrote it.
const shown = computed(() =>
  exported ? props.html.replace(/(href|src)="(\/[^"]*)"/g, (_, attr: string, at: string) => `${attr}="${address(at)}"`) : props.html,
)
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- rendered and cleaned by the server -->
  <div class="prose" v-html="shown" />
</template>

<style scoped>
.prose {
  font: var(--text-lg) / var(--leading-prose) var(--serif);
  overflow-wrap: anywhere;
  min-width: 0;
}
.prose :deep(:first-child) {
  margin-top: 0;
}
.prose :deep(:last-child) {
  margin-bottom: 0;
}
.prose :deep(p),
.prose :deep(ul),
.prose :deep(ol),
.prose :deep(blockquote),
.prose :deep(pre),
.prose :deep(table) {
  margin: 0 0 var(--space-3);
}
.prose :deep(h1),
.prose :deep(h2),
.prose :deep(h3),
.prose :deep(h4) {
  font: var(--weight-bold) var(--text-base) var(--sans);
  margin: var(--space-4) 0 var(--space-2);
}
.prose :deep(a) {
  color: var(--accent);
}
.prose :deep(blockquote) {
  padding-left: var(--space-4);
  border-left: var(--border-strong) solid var(--line-strong);
  color: var(--muted);
}
.prose :deep(code),
.prose :deep(pre) {
  font: var(--text-md) var(--mono);
}
.prose :deep(pre) {
  padding: var(--space-3);
  background: var(--surface-2);
  border-radius: var(--radius);
  overflow-x: auto;
}
.prose :deep(li:has(> input[type='checkbox'])) {
  list-style: none;
}
.prose :deep(input[type='checkbox']) {
  margin: 0 var(--space-2) 0 calc(-1 * var(--space-5));
  accent-color: var(--accent);
}
.prose :deep(img) {
  max-width: 100%;
}
.prose :deep(table) {
  border-collapse: collapse;
  font: var(--text-md) var(--sans);
}
.prose :deep(th),
.prose :deep(td) {
  border-bottom: var(--border) solid var(--line);
  padding: var(--space-1) var(--space-2);
  text-align: left;
}
</style>
