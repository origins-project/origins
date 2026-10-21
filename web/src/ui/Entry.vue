<script setup lang="ts">
// One entry of a list in a form, what it holds beside the actions on it.
defineProps<{ path?: string }>()
</script>

<template>
  <div class="entry" :data-path="path">
    <div class="holds"><slot /></div>
    <div v-if="$slots.actions" class="actions"><slot name="actions" /></div>
  </div>
</template>

<style scoped>
.entry {
  display: flex;
  gap: var(--space-2);
  align-items: end;
  min-width: 0;
}
.holds {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: var(--space-2);
}
.actions {
  flex: none;
  display: flex;
  gap: var(--space-1);
}
</style>
