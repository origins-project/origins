<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import type { IconName } from './icons'

// How many there are of something, leading to the list of them.
defineProps<{ count: number; label: string; to: RouteLocationRaw; icon?: IconName }>()
</script>

<template>
  <RouterLink :to="to" class="stat">
    <span class="count">{{ count }}</span>
    <span class="label"><Icon v-if="icon" :name="icon" />{{ label }}</span>
  </RouterLink>
</template>

<style scoped>
.stat {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding: var(--space-3) var(--space-4);
  background: var(--surface);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  color: var(--ink);
  text-decoration: none;
}
.stat:hover {
  border-color: var(--line-strong);
}
.stat:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border-strong);
}
.count {
  font: var(--text-xl) / var(--leading-tight) var(--serif);
}
.label {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  color: var(--muted);
  font-size: var(--text-sm);
}
</style>
