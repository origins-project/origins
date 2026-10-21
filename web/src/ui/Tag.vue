<script setup lang="ts">
import { computed } from 'vue'

// One of the researcher's words, with the colour the manifest gives it, a colour that
// is data rather than style.
const props = defineProps<{ name: string; color?: string }>()
const dot = computed(() => (props.color ? { '--tag': props.color } : undefined))
</script>

<template>
  <span class="tag" :style="dot">{{ name }}</span>
</template>

<style scoped>
.tag {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-xs);
  padding: 0 var(--space-2);
  border-radius: var(--radius-pill);
  background: var(--surface-2);
  color: var(--muted);
  white-space: nowrap;
}
.tag::before {
  content: '';
  width: var(--dot);
  height: var(--dot);
  border-radius: 50%;
  background: var(--tag, var(--faint));
}
</style>
