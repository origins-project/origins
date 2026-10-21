<script setup lang="ts" generic="T">
import {
  DialogContent,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  ListboxContent,
  ListboxFilter,
  ListboxGroup,
  ListboxGroupLabel,
  ListboxItem,
  ListboxRoot,
  VisuallyHidden,
} from 'reka-ui'
import Icon from './Icon.vue'
import type { IconName } from './icons'
import './overlay.css'
import { usePortal } from './theme'

// A piece of words, marked where it is what was asked for.
export interface Segment {
  text: string
  marked?: boolean
}

// One thing found, with the words it is shown by and what it stands for.
export interface Found<T> {
  key: string
  icon: IconName
  text: Segment[]
  sub?: Segment[]
  value: T
}

export interface FoundGroup<T> {
  label: string
  found: Found<T>[]
}

// The search: a field over what it finds, in groups, the list following the arrow keys
// and kept in view, Enter or a click choosing.
const open = defineModel<boolean>('open', { default: false })
const query = defineModel<string>('query', { default: '' })
defineProps<{ label: string; groups: FoundGroup<T>[]; keys?: { move: string; choose: string; close: string } }>()
const emit = defineEmits<{ choose: [value: T] }>()
const portal = usePortal()

function choose(value: T) {
  open.value = false
  emit('choose', value)
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogPortal :to="portal">
      <DialogOverlay class="overlay-scrim" />
      <DialogContent class="palette" :aria-describedby="undefined">
        <VisuallyHidden><DialogTitle>{{ label }}</DialogTitle></VisuallyHidden>
        <ListboxRoot class="root" highlight-on-hover :selection-behavior="'replace'">
          <div class="query">
            <Icon name="search" />
            <ListboxFilter v-model="query" class="input" auto-focus :aria-label="label" :placeholder="label" />
            <kbd v-if="keys">{{ keys.close }}</kbd>
          </div>
          <ListboxContent class="results">
            <ListboxGroup v-for="group in groups" :key="group.label" class="group">
              <ListboxGroupLabel class="group-label">{{ group.label }}</ListboxGroupLabel>
              <ListboxItem
                v-for="found in group.found"
                :key="found.key"
                :value="found.key"
                class="found"
                @select="choose(found.value)"
              >
                <span class="kind"><Icon :name="found.icon" /></span>
                <span class="text">
                  <template v-for="(segment, i) in found.text" :key="i">
                    <mark v-if="segment.marked">{{ segment.text }}</mark>
                    <template v-else>{{ segment.text }}</template>
                  </template>
                </span>
                <span v-if="found.sub?.length" class="sub">
                  <template v-for="(segment, i) in found.sub" :key="i">
                    <mark v-if="segment.marked">{{ segment.text }}</mark>
                    <template v-else>{{ segment.text }}</template>
                  </template>
                </span>
              </ListboxItem>
            </ListboxGroup>
          </ListboxContent>
          <footer v-if="keys" class="keys">
            <span><kbd>{{ keys.move }}</kbd></span>
            <span><kbd>{{ keys.choose }}</kbd></span>
          </footer>
        </ListboxRoot>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.palette {
  position: fixed;
  z-index: var(--layer-palette);
  top: var(--palette-top);
  left: 50%;
  transform: translateX(-50%);
  width: min(var(--palette), calc(100vw - var(--overlay-gap)));
  background: var(--surface);
  color: var(--ink);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
  animation: overlay-fade var(--quick) ease;
}
.root {
  display: flex;
  flex-direction: column;
}
.query {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-bottom: var(--border) solid var(--line);
  color: var(--faint);
}
.input {
  flex: 1;
  min-width: 0;
  border: 0;
  padding: 0;
  background: none;
  color: var(--ink);
  font-size: var(--text-lg);
  outline: none;
}
.results {
  max-height: var(--palette-list);
  overflow-y: auto;
  padding: var(--space-1);
}
.results:empty {
  display: none;
}
.group-label {
  padding: var(--space-2) var(--space-2) var(--space-1);
  font-size: var(--text-2xs);
  font-weight: var(--weight-bold);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
}
.found {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2);
  border-radius: var(--radius);
  cursor: pointer;
  outline: none;
  min-width: 0;
}
.found[data-highlighted] {
  background: var(--act-soft);
}
.kind {
  width: var(--kind);
  height: var(--kind);
  flex: none;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  background: var(--surface-2);
  color: var(--muted);
}
.text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sub {
  margin-left: auto;
  padding-left: var(--space-2);
  color: var(--muted);
  font-size: var(--text-sm);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
  flex-shrink: 1;
}
mark {
  background: none;
  color: var(--act);
  font-weight: var(--weight-bold);
}
.keys {
  display: flex;
  gap: var(--space-4);
  padding: var(--space-2) var(--space-4);
  border-top: var(--border) solid var(--line);
  font-size: var(--text-xs);
  color: var(--faint);
}
@media (max-width: 40em) {
  .keys {
    display: none;
  }
}
</style>
