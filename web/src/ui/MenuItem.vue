<script setup lang="ts">
import { DropdownMenuCheckboxItem, DropdownMenuItem, DropdownMenuItemIndicator } from 'reka-ui'
import Icon from './Icon.vue'
import type { IconName } from './icons'

// One thing a menu does, or, given checked, one thing it switches.
const checked = defineModel<boolean | undefined>('checked', { default: undefined })
defineProps<{ icon?: IconName; hint?: string; danger?: boolean; disabled?: boolean }>()
defineEmits<{ select: [event: Event] }>()
</script>

<template>
  <DropdownMenuCheckboxItem
    v-if="checked !== undefined"
    v-model="checked"
    class="item"
    :disabled="disabled"
    @select="$emit('select', $event)"
  >
    <Icon v-if="icon" :name="icon" />
    <slot />
    <DropdownMenuItemIndicator class="tick"><Icon name="check" /></DropdownMenuItemIndicator>
  </DropdownMenuCheckboxItem>
  <DropdownMenuItem v-else class="item" :class="{ danger }" :disabled="disabled" @select="$emit('select', $event)">
    <Icon v-if="icon" :name="icon" />
    <slot />
    <span v-if="hint" class="hint">{{ hint }}</span>
  </DropdownMenuItem>
</template>

<style scoped>
.item {
  display: flex;
  width: 100%;
  gap: var(--space-3);
  align-items: center;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: var(--text-md);
  outline: none;
  user-select: none;
}
.item[data-highlighted] {
  background: var(--surface-2);
}
.item[data-disabled] {
  opacity: 0.5;
  pointer-events: none;
}
.danger {
  color: var(--danger);
}
.hint,
.tick {
  margin-left: auto;
}
.hint {
  color: var(--faint);
  font-size: var(--text-xs);
}
.tick {
  color: var(--act);
}
</style>
