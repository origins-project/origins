<script setup lang="ts">
import Icon from './Icon.vue'
import Regions from './Regions.vue'

// A picture of a page whole, with the part of it that matters outlined where that is
// marked, and its title under it: a citation's scan and the entry on it. What the slot
// holds is laid over the picture.
defineProps<{ src?: string; region?: readonly number[]; title?: string }>()
</script>

<template>
  <figure class="scan">
    <span class="frame">
      <img v-if="src" :src="src" alt="" loading="lazy" />
      <Icon v-else name="file" />
      <Regions v-if="src && region && region.length === 4" :regions="[{ key: 'entry', region }]" />
      <slot v-if="src" />
    </span>
    <figcaption v-if="title" class="title">{{ title }}</figcaption>
  </figure>
</template>

<style scoped>
.scan {
  margin: 0;
  display: inline-flex;
  flex-direction: column;
  gap: var(--space-1);
  max-width: 100%;
}
.frame {
  position: relative;
  display: grid;
  place-items: center;
  min-width: var(--thumb);
  min-height: var(--thumb);
  border-radius: var(--radius);
  border: var(--border) solid var(--line);
  background: var(--surface-2);
  color: var(--faint);
  font-size: var(--text-xl);
  overflow: hidden;
}
img {
  display: block;
  max-width: 100%;
  max-height: var(--scan);
}
.title {
  font-size: var(--text-xs);
  color: var(--muted);
  overflow-wrap: anywhere;
}
</style>
