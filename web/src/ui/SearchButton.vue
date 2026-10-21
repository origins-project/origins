<script setup lang="ts">
import Icon from './Icon.vue'

// What opens the search: drawn as a field holding its label and the keys that open it
// too, and on a phone as its icon alone.
defineProps<{ label: string; keys?: string }>()
</script>

<template>
  <button type="button" class="search" :aria-label="label" :aria-keyshortcuts="keys?.replace(' ', '+')">
    <Icon name="search" />
    <span class="label">{{ label }}</span>
    <kbd v-if="keys" class="keys">{{ keys }}</kbd>
  </button>
</template>

<style scoped>
.search {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  height: var(--control);
  padding: 0 var(--space-3);
  border: var(--border) solid var(--line);
  border-radius: var(--radius);
  background: var(--surface);
  color: var(--faint);
  cursor: text;
  text-align: left;
  font-size: var(--text-md);
}
.search:hover {
  border-color: var(--line-strong);
}
.label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (max-width: 40em) {
  .search {
    flex: none;
    width: var(--control);
    justify-content: center;
    padding: 0;
    border-color: transparent;
    background: none;
    color: var(--muted);
    cursor: pointer;
    font-size: var(--text-lg);
  }
  .label,
  .keys {
    display: none;
  }
}
</style>
