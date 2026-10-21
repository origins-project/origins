<script setup lang="ts">
import { provide, ref } from 'vue'
import { portalKey } from './theme'

// A part of the page drawn in one theme whatever the rest is drawn in, with the
// overlays opened from it; supposing, its controls act in the supposing colour.
defineProps<{ theme: 'light' | 'dark'; supposing?: boolean }>()
const root = ref<HTMLElement | null>(null)
provide(portalKey, root)
</script>

<template>
  <div ref="root" class="theme" :data-theme="theme" :data-supposing="supposing || undefined"><slot /></div>
</template>

<style scoped>
.theme {
  background: var(--bg);
  color: var(--ink);
}
</style>
