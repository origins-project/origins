<script setup lang="ts">
import {
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerOverlay,
  DrawerPortal,
  DrawerRoot,
  DrawerTitle,
  DrawerTrigger,
} from 'reka-ui'
import IconButton from './IconButton.vue'
import { actions } from '../labels'
import './overlay.css'
import { usePortal } from './theme'

// An entry of several fields changed where it is shown, from the right, wide for prose
// written beside what it will look like; or the lists, from the left, where the bar has
// no room for them.
const open = defineModel<boolean>('open', { default: false })
withDefaults(defineProps<{ title: string; side?: 'right' | 'left'; path?: string; description?: string; wide?: boolean }>(), {
  side: 'right',
})
const portal = usePortal()
</script>

<template>
  <DrawerRoot v-model:open="open" :swipe-direction="side">
    <DrawerTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DrawerTrigger>
    <DrawerPortal :to="portal">
      <DrawerOverlay class="overlay-scrim" />
      <DrawerContent class="drawer" :class="[side, { wide }]" :data-path="path" v-bind="description ? {} : { 'aria-describedby': undefined }">
        <header class="overlay-head head">
          <div class="heading">
            <DrawerTitle class="overlay-title">{{ title }}</DrawerTitle>
            <p v-if="path" class="overlay-path">{{ path }}</p>
          </div>
          <DrawerClose as-child>
            <IconButton icon="x" :label="actions.close" />
          </DrawerClose>
        </header>
        <DrawerDescription v-if="description" class="description">{{ description }}</DrawerDescription>
        <div class="scroll"><slot /></div>
        <footer v-if="$slots.actions" class="overlay-foot foot"><slot name="actions" /></footer>
      </DrawerContent>
    </DrawerPortal>
  </DrawerRoot>
</template>

<style scoped>
.drawer {
  position: fixed;
  z-index: var(--layer-overlay);
  top: 0;
  bottom: 0;
  width: min(var(--drawer), 100vw);
  display: flex;
  flex-direction: column;
  background: var(--surface);
  color: var(--ink);
  box-shadow: var(--shadow-lg);
  transform: translateX(var(--drawer-swipe-movement-x, 0));
  transition: transform var(--slide) ease;
}
.drawer[data-swiping] {
  transition: none;
}
.right {
  right: 0;
  animation: drawer-from-right var(--slide) ease;
}
.right[data-state='closed'] {
  animation: drawer-from-right var(--slide) ease reverse forwards;
}
.wide {
  width: min(var(--drawer-wide), 100vw);
}
.left {
  left: 0;
  width: min(var(--nav-drawer), 85vw);
  border-right: var(--border) solid var(--line);
  animation: drawer-from-left var(--slide) ease;
}
.left[data-state='closed'] {
  animation: drawer-from-left var(--slide) ease reverse forwards;
}
.head {
  padding: var(--space-4) var(--space-5) var(--space-3);
  border-bottom: var(--border) solid var(--line);
}
.heading {
  flex: 1;
  min-width: 0;
}
.description {
  margin: 0;
  padding: var(--space-2) var(--space-5) 0;
  color: var(--muted);
  font-size: var(--text-md);
}
.scroll {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-5) var(--space-6);
}
.foot {
  padding: var(--space-3) var(--space-5);
  border-top: var(--border) solid var(--line);
}
@keyframes drawer-from-right {
  from {
    transform: translateX(105%);
  }
}
@keyframes drawer-from-left {
  from {
    transform: translateX(-105%);
  }
}
</style>
