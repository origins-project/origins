<script setup lang="ts">
import { computed, ref, watch } from 'vue'

// A person's picture, cropped to the part of it that is them where that is marked, or
// their initials on the colour of their recorded sex where there is no picture. Large,
// it heads their page; small, it stands beside their name wherever they are listed. As
// a button, it opens whatever changes it, and one a draft changes is marked in the
// supposing colour.
const props = withDefaults(
  defineProps<{
    src?: string
    region?: readonly number[]
    initials?: string
    sex?: string
    size?: 'lg' | 'sm'
    label?: string
    as?: 'span' | 'button'
    marked?: boolean
  }>(),
  { size: 'sm', as: 'span' },
)

const natural = ref<{ width: number; height: number }>()
watch(
  () => props.src,
  () => (natural.value = undefined),
)
function loaded(e: Event) {
  const img = e.target as HTMLImageElement
  natural.value = { width: img.naturalWidth, height: img.naturalHeight }
}

// The picture scaled so that the region covers the square and centred on it, in
// fractions of the square, which needs the picture's own size to know its shape.
const crop = computed(() => {
  const r = props.region
  const n = natural.value
  if (!r || r.length !== 4 || !n?.width || !n.height) return undefined
  const [left, top, right, bottom] = r as [number, number, number, number]
  const width = (right - left) * n.width
  const height = (bottom - top) * n.height
  if (width <= 0 || height <= 0) return undefined
  const scale = Math.max(1 / width, 1 / height)
  const percent = (f: number) => `${f * 100}%`
  return {
    width: percent(n.width * scale),
    height: percent(n.height * scale),
    left: percent((1 - width * scale) / 2 - left * n.width * scale),
    top: percent((1 - height * scale) / 2 - top * n.height * scale),
  }
})
// hidden until its shape is known, then shown whole where the region cannot crop it
const waiting = computed(() => !!props.region && !natural.value)
const tone = computed(() => ({ M: 'male', F: 'female' })[props.sex ?? ''] ?? 'unknown')
</script>

<template>
  <component
    :is="as"
    class="portrait"
    :class="[size, tone, { cropped: crop, waiting, marked }]"
    :data-drafted="marked ? '' : undefined"
    :type="as === 'button' ? 'button' : undefined"
    :aria-label="label"
    :role="as === 'span' && label ? 'img' : undefined"
  >
    <img v-if="src" :src="src" alt="" :style="crop" @load="loaded" />
    <span v-else class="initials" aria-hidden="true">{{ initials }}</span>
  </component>
</template>

<style scoped>
.portrait {
  all: unset;
  box-sizing: border-box;
  position: relative;
  display: grid;
  place-items: center;
  flex: none;
  overflow: hidden;
  width: var(--avatar);
  height: var(--avatar);
  border-radius: var(--radius-pill);
  font: var(--weight-medium) var(--text-sm) var(--sans);
}
.lg {
  width: var(--portrait);
  height: var(--portrait);
  border-radius: var(--radius-lg);
  border: var(--border) solid var(--line);
  box-shadow: var(--shadow);
  font: var(--text-2xl) var(--serif);
}
button.portrait {
  cursor: pointer;
}
button.portrait:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border-strong);
}
button.portrait:disabled {
  cursor: default;
}
.marked {
  outline: var(--border-strong) solid var(--suppose);
  outline-offset: var(--border-strong);
}
.male {
  background: var(--male);
  color: var(--male-ink);
}
.female {
  background: var(--female);
  color: var(--female-ink);
}
.unknown {
  background: var(--unknown);
  color: var(--unknown-ink);
}
img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.cropped img {
  position: absolute;
  max-width: none;
  object-fit: fill;
}
.waiting img {
  visibility: hidden;
}
@media (max-width: 40em) {
  .lg {
    width: var(--portrait-phone);
    height: var(--portrait-phone);
    font-size: var(--text-xl);
  }
}
</style>
