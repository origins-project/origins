<script setup lang="ts">
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import type { IconName } from './icons'

// A button that is only an icon, its label said to whoever cannot see it and shown on
// hover, with a count where it has one; given somewhere to go, it is a link.
defineProps<{
  icon: IconName
  label: string
  badge?: number | string
  bordered?: boolean
  pressed?: boolean
  disabled?: boolean
  to?: RouteLocationRaw
}>()
</script>

<template>
  <RouterLink v-if="to !== undefined" :to="to" class="icon-button" :class="{ bordered }" :aria-label="label" :title="label">
    <Icon :name="icon" />
    <span v-if="badge !== undefined && badge !== 0" class="count">{{ badge }}</span>
  </RouterLink>
  <button
    v-else
    type="button"
    class="icon-button"
    :class="{ bordered }"
    :aria-label="label"
    :title="label"
    :aria-pressed="pressed"
    :disabled="disabled"
  >
    <Icon :name="icon" />
    <span v-if="badge !== undefined && badge !== 0" class="count">{{ badge }}</span>
  </button>
</template>

<style scoped>
.icon-button {
  position: relative;
  width: var(--control);
  height: var(--control);
  display: grid;
  place-items: center;
  border: var(--border) solid transparent;
  border-radius: var(--radius);
  background: none;
  color: var(--muted);
  cursor: pointer;
  font-size: var(--text-lg);
  flex: none;
  text-decoration: none;
}
.icon-button:hover,
.icon-button[aria-pressed='true'],
.icon-button[aria-current] {
  background: var(--surface-2);
  color: var(--ink);
}
.icon-button:disabled {
  opacity: 0.5;
  pointer-events: none;
}
.bordered {
  border-color: var(--line-strong);
  background: var(--surface);
}
.count {
  position: absolute;
  top: calc(-1 * var(--space-1));
  right: calc(-1 * var(--space-1));
  min-width: var(--badge);
  height: var(--badge);
  padding: 0 var(--space-1);
  border-radius: var(--radius-pill);
  background: var(--warn);
  color: var(--on-warn);
  font: var(--weight-bold) var(--text-2xs) / var(--badge) var(--sans);
}
</style>
