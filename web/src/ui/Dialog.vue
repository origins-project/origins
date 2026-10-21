<script setup lang="ts">
import {
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogOverlay,
  DialogPortal,
  DialogRoot,
  DialogTitle,
  DialogTrigger,
} from 'reka-ui'
import IconButton from './IconButton.vue'
import { actions } from '../labels'
import './overlay.css'
import { usePortal } from './theme'

// A change to look at before it is made — a delete, a merge, an undo — or a form too
// large for a popover and too small for a drawer.
const open = defineModel<boolean>('open', { default: false })
defineProps<{ title: string; description?: string; path?: string }>()
const portal = usePortal()
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DialogTrigger>
    <DialogPortal :to="portal">
      <DialogOverlay class="overlay-scrim" />
      <DialogContent class="dialog" :data-path="path" v-bind="description ? {} : { 'aria-describedby': undefined }">
        <header class="overlay-head">
          <div class="heading">
            <DialogTitle class="overlay-title">{{ title }}</DialogTitle>
            <p v-if="path" class="overlay-path">{{ path }}</p>
          </div>
          <DialogClose as-child>
            <IconButton icon="x" :label="actions.close" />
          </DialogClose>
        </header>
        <DialogDescription v-if="description" class="description">{{ description }}</DialogDescription>
        <div class="body"><slot /></div>
        <footer v-if="$slots.actions" class="overlay-foot"><slot name="actions" /></footer>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.dialog {
  position: fixed;
  z-index: var(--layer-overlay);
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(var(--dialog), calc(100vw - var(--overlay-gap)));
  max-height: calc(100vh - var(--overlay-gap) * 1.5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-5);
  border-radius: var(--radius-lg);
  background: var(--surface);
  color: var(--ink);
  box-shadow: var(--shadow-lg);
  animation: overlay-fade var(--quick) ease;
}
.heading {
  flex: 1;
  min-width: 0;
}
.description {
  margin: 0;
  color: var(--muted);
  font-size: var(--text-md);
}
.body {
  overflow-y: auto;
  min-height: 0;
}
</style>
