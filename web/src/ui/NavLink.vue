<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import type { IconName } from './icons'

// A link to one of the lists: a word in the bar, or a line of the drawer the lists go
// into when the bar has no room, with its icon and how many the list holds.
withDefaults(defineProps<{ to: RouteLocationRaw; icon?: IconName; count?: number; drawer?: boolean }>(), {
  drawer: false,
})
</script>

<template>
  <RouterLink :to="to" class="link" :class="{ drawer }">
    <Icon v-if="drawer && icon" :name="icon" />
    <slot />
    <span v-if="drawer && count !== undefined" class="count">{{ count }}</span>
  </RouterLink>
</template>

<style scoped>
.link {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  color: var(--muted);
  font-size: var(--text-md);
  text-decoration: none;
  white-space: nowrap;
}
.link:hover {
  color: var(--ink);
  background: var(--surface-2);
}
.link[aria-current] {
  color: var(--ink);
  background: var(--surface-2);
  font-weight: var(--weight-medium);
}
.drawer {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius);
  color: var(--ink);
  font-size: var(--text-base);
}
.drawer[aria-current] {
  color: var(--act);
}
.count {
  margin-left: auto;
  color: var(--faint);
  font-size: var(--text-xs);
  font-variant-numeric: tabular-nums;
}
</style>
