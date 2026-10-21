<script setup lang="ts">
import Button from './Button.vue'
import Notice from './Notice.vue'
import './overlay.css'

// What a change leaves behind, each a verb and its object and what can be done about
// it, stacked at the corner of the window until dismissed.
defineProps<{ toasts: { key: string | number; text: string; action?: string }[] }>()
defineEmits<{ dismiss: [key: string | number]; act: [key: string | number] }>()
</script>

<template>
  <div class="toasts" aria-live="polite">
    <Notice v-for="toast in toasts" :key="toast.key" tone="ink" dismissible class="toast" @dismiss="$emit('dismiss', toast.key)">
      {{ toast.text }}
      <template v-if="toast.action" #actions>
        <Button size="sm" variant="ghost" class="act" @click="$emit('act', toast.key)">{{ toast.action }}</Button>
      </template>
    </Notice>
  </div>
</template>

<style scoped>
.toasts {
  position: fixed;
  z-index: var(--layer-toast);
  right: var(--gutter);
  bottom: var(--gutter);
  width: min(var(--toast), calc(100vw - 2 * var(--gutter-phone)));
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  pointer-events: none;
}
.toast {
  pointer-events: auto;
  align-items: center;
  animation: overlay-fade var(--quick) ease;
}
.act,
.act:hover {
  color: inherit;
}
@media (max-width: 40em) {
  .toasts {
    right: var(--gutter-phone);
    bottom: var(--gutter-phone);
  }
}
</style>
