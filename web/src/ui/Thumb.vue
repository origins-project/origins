<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from './Icon.vue'
import Regions, { type Region } from './Regions.vue'

// A picture small, with its title along its foot and the people in it outlined: a
// record's file, a photograph of somebody another record keeps. A file that is not a
// picture is its icon; as a button, it opens whatever says what it is.
const props = withDefaults(
  defineProps<{ src?: string; title?: string; size?: 'md' | 'sm'; as?: 'span' | 'button'; regions?: readonly Region[] }>(),
  { size: 'md', as: 'span', regions: () => [] },
)

// The picture fills the square and is cut to it, so the outlines are laid over the
// whole of it, once its shape is known.
const ratio = ref(0)
const covered = computed(() => {
  if (!ratio.value) return undefined
  return ratio.value >= 1
    ? { height: '100%', aspectRatio: String(ratio.value) }
    : { width: '100%', aspectRatio: String(ratio.value) }
})
const outlined = computed(() => props.size === 'md' && props.regions.length > 0)
function loaded(e: Event) {
  const img = e.target as HTMLImageElement
  if (img.naturalHeight) ratio.value = img.naturalWidth / img.naturalHeight
}
</script>

<template>
  <component
    :is="as"
    class="thumb"
    :class="size"
    :type="as === 'button' ? 'button' : undefined"
    :aria-label="title"
    :title="title"
    :role="as === 'span' && title ? 'img' : undefined"
  >
    <span v-if="src && outlined" class="whole" :style="covered">
      <img :src="src" alt="" loading="lazy" @load="loaded" />
      <Regions v-if="covered" :regions="regions" />
    </span>
    <img v-else-if="src" :src="src" alt="" loading="lazy" />
    <Icon v-else name="file" />
    <span v-if="title && size === 'md'" class="title" aria-hidden="true">{{ title }}</span>
  </component>
</template>

<style scoped>
.thumb {
  all: unset;
  box-sizing: border-box;
  position: relative;
  display: grid;
  place-items: center;
  flex: none;
  overflow: hidden;
  width: var(--thumb);
  height: var(--thumb);
  border-radius: var(--radius);
  border: var(--border) solid var(--line);
  background: var(--surface-2);
  color: var(--faint);
  font-size: var(--text-xl);
}
.sm {
  width: var(--thumb-sm);
  height: var(--thumb-sm);
}
button.thumb {
  cursor: pointer;
}
button.thumb:hover {
  border-color: var(--line-strong);
}
button.thumb:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border);
}
img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.whole {
  position: absolute;
  inset: 0;
}
.whole[style] {
  inset: auto;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
}
.title {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 0 var(--space-1);
  font-size: var(--text-2xs);
  line-height: var(--leading);
  background: color-mix(in srgb, var(--surface) 85%, transparent);
  color: var(--ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
