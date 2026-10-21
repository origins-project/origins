<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import { Primitive } from 'reka-ui'
import Icon from './Icon.vue'
import type { IconName } from './icons'

// A button, or a link drawn as one where it goes somewhere. An add button is the dashed
// one that stands where something could be added.
const props = withDefaults(
  defineProps<{
    variant?: 'default' | 'primary' | 'ghost' | 'danger' | 'add'
    size?: 'md' | 'sm'
    icon?: IconName
    to?: RouteLocationRaw
    href?: string
    type?: 'button' | 'submit'
    disabled?: boolean
    solid?: boolean
  }>(),
  { variant: 'default', size: 'md', type: 'button' },
)

const as = computed(() => (props.to !== undefined ? RouterLink : props.href !== undefined ? 'a' : 'button'))
const attrs = computed(() =>
  props.to !== undefined
    ? { to: props.to }
    : props.href !== undefined
      ? { href: props.href }
      : { type: props.type, disabled: props.disabled },
)
</script>

<template>
  <Primitive
    :as="as"
    v-bind="attrs"
    class="button"
    :class="[variant, size, { solid }]"
  >
    <Icon v-if="icon" :name="icon" />
    <span v-if="$slots.default" class="label"><slot /></span>
  </Primitive>
</template>

<style scoped>
.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  height: var(--control);
  padding: 0 var(--space-4);
  border-radius: var(--radius);
  border: var(--border) solid var(--line-strong);
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  font-size: var(--text-md);
  font-weight: var(--weight-medium);
  white-space: nowrap;
  text-decoration: none;
  transition: background var(--quick);
  min-width: 0;
  max-width: 100%;
}
.label {
  overflow: hidden;
  text-overflow: ellipsis;
}
.button:hover {
  background: var(--surface-2);
  color: var(--ink);
}
.button:disabled {
  opacity: 0.5;
  cursor: default;
  pointer-events: none;
}
.sm {
  height: var(--control-sm);
  padding: 0 var(--space-3);
  font-size: var(--text-sm);
}
.primary {
  background: var(--act);
  border-color: var(--act);
  color: var(--on-act);
}
.primary:hover {
  background: var(--act-hover);
  border-color: var(--act-hover);
  color: var(--on-act);
}
.ghost {
  border-color: transparent;
  background: none;
  color: var(--muted);
}
.ghost:hover {
  background: var(--surface-2);
  color: var(--ink);
}
.danger {
  border-color: transparent;
  background: none;
  color: var(--danger);
}
.danger:hover {
  background: var(--danger-soft);
  color: var(--danger);
}
.danger.solid,
.danger.solid:hover {
  background: var(--danger);
  color: var(--on-danger);
}
.add {
  height: var(--control-sm);
  padding: 0 var(--space-3);
  border: var(--border) dashed color-mix(in srgb, var(--act) 50%, transparent);
  background: none;
  color: var(--act);
  font-size: var(--text-sm);
}
.add:hover {
  background: var(--act-soft);
  border-style: solid;
  color: var(--act);
}
</style>
