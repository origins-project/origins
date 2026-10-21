<script setup lang="ts">
import { DropdownMenuContent, DropdownMenuPortal, DropdownMenuRoot, DropdownMenuTrigger } from 'reka-ui'
import { usePortal } from './theme'

// A record's menu and the "…" menu: the trigger, and MenuItems under it.
const open = defineModel<boolean>('open', { default: false })
withDefaults(defineProps<{ align?: 'start' | 'end' }>(), { align: 'end' })
const portal = usePortal()
</script>

<template>
  <DropdownMenuRoot v-model:open="open">
    <DropdownMenuTrigger as-child>
      <slot name="trigger" />
    </DropdownMenuTrigger>
    <DropdownMenuPortal :to="portal">
      <DropdownMenuContent class="menu" :align="align" :side-offset="6" :collision-padding="16">
        <slot />
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>

<style scoped>
/* Reka draws the content inside a positioned wrapper, which takes this scope. */
.menu,
:deep(.menu) {
  z-index: var(--layer-overlay);
  min-width: var(--menu);
  background: var(--surface);
  color: var(--ink);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: var(--space-1);
  animation: overlay-fade var(--quick) ease;
}
</style>
