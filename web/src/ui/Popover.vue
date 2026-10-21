<script setup lang="ts">
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import './overlay.css'
import { usePortal } from './theme'

// One value changed where it is shown: the trigger is the value, the popover its form.
const open = defineModel<boolean>('open', { default: false })
withDefaults(defineProps<{ path?: string; side?: 'top' | 'bottom' | 'left' | 'right'; label?: string }>(), {
  side: 'bottom',
})
const portal = usePortal()
</script>

<template>
  <PopoverRoot v-model:open="open">
    <PopoverTrigger as-child>
      <slot name="trigger" />
    </PopoverTrigger>
    <PopoverPortal :to="portal">
      <PopoverContent
        class="popover"
        :side="side"
        align="start"
        :side-offset="6"
        :collision-padding="16"
        :data-path="path"
      >
        <!-- Reka names the popover by its trigger; the label says what is changed. -->
        <p v-if="label" class="label">{{ label }}</p>
        <p v-if="path" class="overlay-path">{{ path }}</p>
        <slot />
        <div v-if="$slots.actions" class="overlay-foot actions"><slot name="actions" /></div>
      </PopoverContent>
    </PopoverPortal>
  </PopoverRoot>
</template>

<style scoped>
/* Reka draws the content inside a positioned wrapper, which takes this scope. */
.popover,
:deep(.popover) {
  z-index: var(--layer-overlay);
  width: var(--popover);
  max-width: calc(100vw - var(--overlay-gap));
  max-height: var(--reka-popover-content-available-height);
  overflow-y: auto;
  background: var(--surface);
  color: var(--ink);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  padding: var(--space-4);
  display: grid;
  gap: var(--space-3);
  animation: overlay-fade var(--quick) ease;
}
.label {
  font: var(--weight-bold) var(--text-2xs) var(--sans);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
}
.actions {
  margin-top: var(--space-1);
}
</style>
