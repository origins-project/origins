<script setup lang="ts">
import { icons, type IconName } from './icons'

defineProps<{ name: IconName }>()
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- the paths are our own constants -->
  <svg class="icon" viewBox="0 0 24 24" aria-hidden="true" v-html="icons[name]" />
</template>

<style scoped>
.icon {
  width: var(--icon);
  height: var(--icon);
  flex: none;
  stroke: currentColor;
  fill: none;
  stroke-width: var(--icon-stroke);
  stroke-linecap: round;
  stroke-linejoin: round;
  vertical-align: -0.125em;
}
</style>
