<script setup lang="ts">
import { ref } from 'vue'
import { DialogClose, DialogContent, DialogOverlay, DialogPortal, DialogRoot, DialogTitle, DialogTrigger } from 'reka-ui'
import Icon from './Icon.vue'
import IconButton from './IconButton.vue'
import Regions, { type Region } from './Regions.vue'
import Theme from './Theme.vue'
import { actions } from '../labels'
import './overlay.css'
import { usePortal } from './theme'

// A picture whole, dark around it, with the people in it outlined and named, and beside
// it what is said of it. Where it may be drawn on, a box dragged across it is the part
// of it somebody is in, left in `drawn` as four fractions until it is taken or cleared.
const open = defineModel<boolean>('open', { default: false })
const drawn = defineModel<number[] | undefined>('drawn')
const props = withDefaults(
  defineProps<{ title: string; src?: string; regions?: readonly Region[]; draw?: boolean; path?: string }>(),
  { regions: () => [] },
)
const portal = usePortal()

const picture = ref<HTMLElement | null>(null)
const drawing = ref<number[]>()
let from: [number, number] | undefined

function at(e: PointerEvent): [number, number] {
  const box = picture.value!.getBoundingClientRect()
  const clamp = (v: number) => Math.min(1, Math.max(0, v))
  return [clamp((e.clientX - box.left) / box.width), clamp((e.clientY - box.top) / box.height)]
}
function down(e: PointerEvent) {
  if (!props.draw || e.button !== 0) return
  e.preventDefault()
  ;(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId)
  from = at(e)
  drawing.value = [...from, ...from]
}
function move(e: PointerEvent) {
  if (!from) return
  const [x, y] = at(e)
  drawing.value = [Math.min(x, from[0]), Math.min(y, from[1]), Math.max(x, from[0]), Math.max(y, from[1])]
}
function up() {
  const box = drawing.value
  cancel()
  if (box && box[2]! - box[0]! > 0.02 && box[3]! - box[1]! > 0.02) drawn.value = box
}
function cancel() {
  from = undefined
  drawing.value = undefined
}
</script>

<template>
  <DialogRoot v-model:open="open">
    <DialogTrigger v-if="$slots.trigger" as-child>
      <slot name="trigger" />
    </DialogTrigger>
    <DialogPortal :to="portal">
      <DialogOverlay class="overlay-scrim" />
      <DialogContent class="lightbox" :data-path="path" :aria-describedby="undefined">
        <Theme theme="dark" class="frame">
          <div class="stage">
            <span
              v-if="src"
              ref="picture"
              class="picture"
              :class="{ draw }"
              data-picture
              @pointerdown="down"
              @pointermove="move"
              @pointerup="up"
              @pointercancel="cancel"
            >
              <img :src="src" alt="" draggable="false" />
              <Regions :regions="regions" :drawing="drawing ?? drawn" labelled />
            </span>
            <Icon v-else name="file" class="missing" />
          </div>
          <aside class="side">
            <header class="overlay-head">
              <DialogTitle class="overlay-title">{{ title }}</DialogTitle>
              <DialogClose as-child>
                <IconButton icon="x" :label="actions.close" />
              </DialogClose>
            </header>
            <div class="said"><slot /></div>
          </aside>
        </Theme>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>

<style scoped>
.lightbox {
  position: fixed;
  z-index: var(--layer-overlay);
  inset: 0;
  animation: overlay-fade var(--quick) ease;
}
.frame {
  height: 100%;
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--lightbox-side);
}
.stage {
  display: grid;
  place-items: center;
  padding: var(--space-6);
  min-height: 0;
  overflow: hidden;
}
.picture {
  position: relative;
  display: inline-block;
  line-height: 0;
  user-select: none;
  touch-action: none;
}
.picture.draw {
  cursor: crosshair;
}
img {
  display: block;
  max-width: 100%;
  max-height: calc(100vh - var(--space-6) * 2);
  border-radius: var(--radius-sm);
}
.missing {
  font-size: var(--text-xl);
  color: var(--faint);
}
.side {
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--surface);
  border-left: var(--border) solid var(--line);
}
.side .overlay-head {
  padding: var(--space-4) var(--space-5) var(--space-3);
}
.said {
  flex: 1;
  overflow-y: auto;
  padding: 0 var(--space-5) var(--space-5);
}
@media (max-width: 40em) {
  .frame {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto auto;
    overflow-y: auto;
  }
  .stage {
    padding: var(--space-4);
  }
  img {
    max-height: 70vh;
  }
  .side {
    border-left: 0;
    border-top: var(--border) solid var(--line);
  }
}
</style>
