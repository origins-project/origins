<script setup lang="ts">
import { computed } from 'vue'
import Icon from './Icon.vue'
import IconButton from './IconButton.vue'
import type { IconName } from './icons'
import { actions } from '../labels'

// A line set apart in its tone: a refusal in the dialog it was made in, a draft
// waiting, a warning. A notice that floats is the one a change leaves behind, a verb and
// its object, at the corner of the window until it is dismissed.
const props = withDefaults(
  defineProps<{
    tone?: 'info' | 'warn' | 'danger' | 'good' | 'suppose' | 'ink'
    icon?: IconName
    dismissible?: boolean
  }>(),
  { tone: 'info' },
)
defineEmits<{ dismiss: [] }>()
const shown = computed<IconName | undefined>(
  () => props.icon ?? (props.tone === 'danger' || props.tone === 'warn' ? 'alert' : undefined),
)
</script>

<template>
  <div class="notice" :class="tone" :role="tone === 'danger' ? 'alert' : 'status'">
    <Icon v-if="shown" :name="shown" class="mark" />
    <div class="words"><slot /></div>
    <div v-if="$slots.actions" class="actions"><slot name="actions" /></div>
    <IconButton v-if="dismissible" icon="x" :label="actions.dismiss" class="dismiss" @click="$emit('dismiss')" />
  </div>
</template>

<style scoped>
.notice {
  display: flex;
  gap: var(--space-3);
  align-items: start;
  padding: var(--space-3);
  border-radius: var(--radius);
  font-size: var(--text-md);
}
.mark {
  margin-top: var(--space-1);
  flex: none;
}
.words {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}
.dismiss {
  width: var(--control-sm);
  height: var(--control-sm);
  margin: calc(-1 * var(--space-1)) calc(-1 * var(--space-1)) 0 0;
  color: inherit;
}
.info {
  background: var(--surface-2);
  color: var(--muted);
}
.warn {
  background: var(--warn-soft);
  color: var(--warn);
}
.danger {
  background: var(--danger-soft);
  color: var(--danger);
}
.good {
  background: var(--good-soft);
  color: var(--good);
}
.suppose {
  background: var(--suppose-soft);
  color: var(--suppose);
}
.ink {
  background: var(--ink);
  color: var(--bg);
  box-shadow: var(--shadow-lg);
}
.ink .dismiss:hover {
  background: none;
  color: inherit;
}
</style>
