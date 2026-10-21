<script setup lang="ts">
import { useId } from 'vue'
import { Label, SwitchRoot, SwitchThumb } from 'reka-ui'

// Something on or off, its label beside it.
const model = defineModel<boolean>({ default: false })
defineProps<{ label: string; path?: string; disabled?: boolean }>()
const id = useId()
</script>

<template>
  <span class="switch" :data-path="path">
    <SwitchRoot :id="id" v-model="model" class="track" :disabled="disabled">
      <SwitchThumb class="thumb" />
    </SwitchRoot>
    <Label :for="id" class="label">{{ label }}</Label>
  </span>
</template>

<style scoped>
.switch {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--muted);
}
.track {
  all: unset;
  flex: none;
  width: var(--switch-width);
  height: var(--switch-height);
  border-radius: var(--radius-pill);
  background: var(--line-strong);
  position: relative;
  cursor: pointer;
  transition: background var(--quick);
}
.track:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border-strong);
}
.track[data-state='checked'] {
  background: var(--act);
}
.track[data-disabled] {
  opacity: 0.5;
  cursor: default;
}
.thumb {
  display: block;
  position: absolute;
  top: calc((var(--switch-height) - var(--switch-thumb)) / 2);
  left: calc((var(--switch-height) - var(--switch-thumb)) / 2);
  width: var(--switch-thumb);
  height: var(--switch-thumb);
  border-radius: 50%;
  background: var(--surface);
  transition: transform var(--quick);
}
.thumb[data-state='checked'] {
  transform: translateX(calc(var(--switch-width) - var(--switch-height)));
}
.label {
  cursor: pointer;
  user-select: none;
}
</style>
