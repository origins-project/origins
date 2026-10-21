<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

// The bar along the top of every page: the brand, the links to the lists, the search
// and the actions. The links stand in the bar while they fit beside the rest with the
// search at its narrowest, and otherwise go, the toggle standing in their place; that
// is measured, not read off the window's width, since what fits depends on the words.
// What is below runs the width of the bar under it; a bar that is still stays where it
// is drawn rather than at the top of the window.
defineProps<{ still?: boolean }>()
const fits = defineModel<boolean>('fits', { default: true })

const bar = ref<HTMLElement | null>(null)
const brand = ref<HTMLElement | null>(null)
const links = ref<HTMLElement | null>(null)
const middle = ref<HTMLElement | null>(null)
const end = ref<HTMLElement | null>(null)

function measure() {
  if (!bar.value || !brand.value || !links.value || !middle.value || !end.value) return
  const style = getComputedStyle(bar.value)
  const gap = parseFloat(style.columnGap) || 0
  const room =
    bar.value.clientWidth -
    (parseFloat(style.paddingLeft) || 0) -
    (parseFloat(style.paddingRight) || 0) -
    brand.value.offsetWidth -
    end.value.offsetWidth -
    (parseFloat(getComputedStyle(middle.value).minWidth) || 0) -
    3 * gap
  fits.value = links.value.scrollWidth <= room
}

let observer: ResizeObserver | undefined
let frame = 0
onMounted(() => {
  if (typeof ResizeObserver === 'undefined') return
  // measured a frame later, since a measure inside the delivery resizes what is observed
  observer = new ResizeObserver(() => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(measure)
  })
  for (const el of [bar, brand, links, end]) if (el.value) observer.observe(el.value)
  void nextTick(measure)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  cancelAnimationFrame(frame)
})
</script>

<template>
  <header class="bar" :class="{ still }">
    <div ref="bar" class="row">
      <div v-if="!fits && $slots.toggle" class="toggle"><slot name="toggle" /></div>
      <div ref="brand" class="brand"><slot name="brand" /></div>
      <nav class="nav" :class="{ away: !fits }" :aria-hidden="!fits || undefined" :inert="!fits || undefined">
        <div ref="links" class="links"><slot name="nav" /></div>
      </nav>
      <div ref="middle" class="middle"><slot /></div>
      <div ref="end" class="end"><slot name="end" /></div>
    </div>
    <div v-if="$slots.below" class="below"><slot name="below" /></div>
  </header>
</template>

<style scoped>
.bar {
  position: sticky;
  top: 0;
  z-index: var(--layer-bar);
  background: color-mix(in srgb, var(--bg) 82%, transparent);
  backdrop-filter: blur(var(--blur));
  border-bottom: var(--border) solid var(--line);
}
.still {
  position: relative;
  z-index: auto;
}
.row {
  height: var(--bar);
  display: flex;
  align-items: center;
  gap: var(--space-4);
  padding: 0 var(--gutter);
  position: relative;
  overflow-x: clip;
}
.below {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  padding: 0 var(--gutter) var(--space-2);
}
.toggle {
  margin-left: calc(-1 * var(--space-2));
  margin-right: calc(-1 * var(--space-2));
}
.brand {
  flex: none;
  min-width: 0;
}
.nav {
  flex: none;
}
/* gone from the line, but laid out still, so that its width can be measured again; the
   row clips it, so that it does not widen the page */
.away {
  position: absolute;
  visibility: hidden;
  pointer-events: none;
}
.links {
  display: flex;
  gap: var(--space-0);
  width: max-content;
}
.middle {
  flex: 0 1 var(--search);
  min-width: var(--search-min);
  margin: 0 auto;
  display: flex;
}
.end {
  flex: none;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
@media (max-width: 40em) {
  .row {
    padding: 0 var(--gutter-phone);
    gap: var(--space-2);
  }
  .below {
    padding: 0 var(--gutter-phone) var(--space-2);
  }
  .brand {
    flex: 0 1 auto;
  }
  .middle {
    flex: none;
    min-width: 0;
    margin: 0 0 0 auto;
  }
}
</style>
