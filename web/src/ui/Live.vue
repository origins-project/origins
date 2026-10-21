<script setup lang="ts">
// Whether the page hears the server: a dot, green while the socket is open, with its
// word.
defineProps<{ up: boolean }>()
</script>

<template>
  <span class="live" :class="{ down: !up }" role="status"><slot /></span>
</template>

<style scoped>
.live {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--space-1);
  font-size: var(--text-xs);
  color: var(--faint);
  white-space: nowrap;
}
.live::before {
  content: '';
  width: var(--dot);
  height: var(--dot);
  border-radius: 50%;
  background: var(--good);
  box-shadow: 0 0 0 var(--halo) var(--good-soft);
}
.down::before {
  background: var(--warn);
  box-shadow: 0 0 0 var(--halo) var(--warn-soft);
}
</style>
