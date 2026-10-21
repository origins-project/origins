<script setup lang="ts">
// One name in `Definitions` and what it names.
defineProps<{ term: string; path?: string }>()
</script>

<template>
  <dt class="term">{{ term }}</dt>
  <dd class="named" :data-path="path"><slot /></dd>
</template>

<style scoped>
.term {
  font-size: var(--text-sm);
  color: var(--muted);
}
.named {
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1) var(--space-2);
  align-items: baseline;
}
@media (max-width: 40em) {
  .named {
    margin-bottom: var(--space-2);
  }
}
</style>
