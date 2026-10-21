<script setup lang="ts">
import { computed } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import type { IconName } from './icons'
import { actions } from '../labels'

// A small thing that can be pressed: another name with its type, a source cited, a
// filter switched on, a view in force. Pressed, it is a switch; removable, it offers to
// go; given to, it is a link; added, it stands where one could be added.
const props = defineProps<{
  kind?: string
  icon?: IconName
  to?: RouteLocationRaw
  pressed?: boolean
  removable?: boolean
  add?: boolean
  tone?: 'plain' | 'suppose' | 'source'
}>()
defineEmits<{ click: [event: MouseEvent]; remove: [] }>()
const as = computed(() => (props.to !== undefined ? RouterLink : 'button'))
</script>

<template>
  <span class="chip" :class="[tone ?? 'plain', { add }]">
    <component
      :is="as"
      class="press"
      :to="to"
      :type="to === undefined ? 'button' : undefined"
      :aria-pressed="pressed"
      @click="$emit('click', $event)"
    >
      <Icon v-if="icon || add" :name="icon ?? 'plus'" />
      <span v-if="kind" class="kind">{{ kind }}</span>
      <slot />
    </component>
    <button v-if="removable" type="button" class="remove" :aria-label="actions.remove" @click="$emit('remove')">
      <Icon name="x" />
    </button>
  </span>
</template>

<style scoped>
.chip {
  display: inline-flex;
  align-items: center;
  border-radius: var(--radius-pill);
  background: var(--surface);
  border: var(--border) solid var(--line);
  color: var(--ink);
  font-size: var(--text-sm);
  max-width: 100%;
}
.chip:hover {
  border-color: var(--line-strong);
}
.press {
  all: unset;
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: 0 var(--space-3);
  min-height: calc(var(--control-sm) - var(--space-1));
  cursor: pointer;
  border-radius: inherit;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.press:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border-strong);
}
.press[aria-pressed='true'] {
  font-weight: var(--weight-medium);
}
.chip:has(.press[aria-pressed='true']) {
  border-color: var(--ink);
}
.kind {
  color: var(--faint);
  font-size: var(--text-xs);
}
.remove {
  all: unset;
  display: grid;
  place-items: center;
  padding: 0 var(--space-2) 0 0;
  color: var(--faint);
  cursor: pointer;
}
.remove:hover {
  color: var(--ink);
}
.add {
  border-style: dashed;
  border-color: color-mix(in srgb, var(--act) 50%, transparent);
  color: var(--act);
  background: none;
}
.source {
  border-color: transparent;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  color: var(--muted);
  font-size: var(--text-xs);
}
.source:hover {
  color: var(--act);
}
.suppose {
  background: var(--suppose-soft);
  color: var(--suppose);
  border-color: color-mix(in srgb, var(--suppose) 35%, transparent);
}
</style>
