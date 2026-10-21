<script setup lang="ts" generic="T">
// Items each under the items they belong to, as deep as they go, each drawn by the
// default slot with how deep it is.
defineProps<{ items: T[]; under: (item: T) => T[]; itemKey: (item: T) => string; label?: string; depth?: number }>()
defineSlots<{ default: (scope: { item: T; depth: number }) => unknown }>()
</script>

<template>
  <ul class="tree" :class="{ nested: depth }" :role="depth ? 'group' : 'tree'" :aria-label="depth ? undefined : label">
    <li v-for="item in items" :key="itemKey(item)" role="treeitem" :aria-level="(depth ?? 0) + 1">
      <div class="item"><slot :item="item" :depth="depth ?? 0" /></div>
      <Tree v-if="under(item).length" :items="under(item)" :under="under" :item-key="itemKey" :depth="(depth ?? 0) + 1">
        <template #default="scope"><slot v-bind="scope" /></template>
      </Tree>
    </li>
  </ul>
</template>

<style scoped>
.tree {
  list-style: none;
  margin: 0;
  padding: 0;
}
.nested {
  margin-inline-start: var(--space-3);
  padding-inline-start: var(--space-4);
  border-inline-start: var(--border) solid var(--line);
}
.item {
  padding: var(--space-1) 0;
}
</style>
