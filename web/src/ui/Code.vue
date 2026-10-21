<script setup lang="ts">
// Text as a file holds it, kept to its lines and set in the monospaced face.
defineProps<{ label?: string }>()
</script>

<template>
  <pre class="code" :aria-label="label" tabindex="0"><slot /></pre>
</template>

<style scoped>
.code {
  margin: 0;
  padding: var(--space-3) var(--space-4);
  overflow: auto;
  background: var(--surface-2);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-sm);
  font: var(--text-sm) / var(--leading) var(--mono);
  color: var(--ink);
  white-space: pre;
}
</style>
