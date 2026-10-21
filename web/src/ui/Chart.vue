<script setup lang="ts">
import { computed, onMounted, ref, useId, watch } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'
import Icon from './Icon.vue'
import IconButton from './IconButton.vue'
import Portrait from './Portrait.vue'
import { chart as labels } from '../labels'

// A chart of people: a box for each, where the drawing puts it, and the lines between
// them, a tie not by birth dashed and a partnership quieter, and a dashed box pressed to
// fill a gap where somebody is missing. Somebody may instead be a piece of a ring, its
// words turned and kept within it, or a bar over the years of an axis, an end with no
// date dashed. It pans by dragging, zooms with the buttons, a pinch or the wheel held
// with Ctrl, and opens fitted to its frame, or at its own size around the subject where
// fitting would make it unreadable.
export interface ChartBox {
  key: string
  x: number
  y: number
  name: string
  lifespan?: string
  to?: RouteLocationRaw
  sex?: string
  src?: string
  region?: readonly number[]
  initials?: string
  subject?: boolean
  // a box standing for somebody missing, pressed rather than followed
  gap?: boolean
  sector?: { path: string; label: { x: number; y: number }; turn: number }
  bar?: { x: number; width: number; born: boolean; ended: boolean }
}

export interface ChartTick {
  year: number
  x: number
}

export interface ChartLine {
  path: string
  tie: 'birth' | 'other' | 'partner' | 'gap'
}

const props = defineProps<{
  boxes: ChartBox[]
  lines: ChartLine[]
  width: number
  height: number
  box: { width: number; height: number }
  label: string
  ticks?: ChartTick[]
}>()
const id = useId()
const emit = defineEmits<{ press: [key: string] }>()

const frame = ref<HTMLElement>()
const scale = ref(1)
const at = ref({ x: 0, y: 0 })
const smallest = 0.2
const largest = 2.5
const readable = 0.6

const clamp = (s: number) => Math.min(largest, Math.max(smallest, s))

function size() {
  const r = frame.value?.getBoundingClientRect()
  return { width: r?.width || props.width, height: r?.height || props.height }
}

function fitted() {
  const { width, height } = size()
  return Math.min(1, width / props.width, height / props.height)
}

// Fitted where that is readable, else at full size with the subject in view.
function open() {
  const { width, height } = size()
  const fit = fitted()
  if (fit >= readable) {
    scale.value = fit
    at.value = { x: (width - props.width * fit) / 2, y: (height - props.height * fit) / 2 }
    return
  }
  scale.value = 1
  const subject = props.boxes.find((b) => b.subject) ?? props.boxes[0]
  at.value = subject
    ? { x: Math.min(0, width / 2 - subject.x - props.box.width / 2), y: height / 2 - subject.y - props.box.height / 2 }
    : { x: 0, y: 0 }
}

function fit() {
  const { width, height } = size()
  const s = clamp(fitted())
  scale.value = s
  at.value = { x: (width - props.width * s) / 2, y: (height - props.height * s) / 2 }
}

// Zooms by a factor keeping the point under (cx, cy), in the frame's pixels, still.
function zoom(factor: number, cx?: number, cy?: number) {
  const { width, height } = size()
  const x = cx ?? width / 2
  const y = cy ?? height / 2
  const next = clamp(scale.value * factor)
  const k = next / scale.value
  at.value = { x: x - (x - at.value.x) * k, y: y - (y - at.value.y) * k }
  scale.value = next
}

function wheel(e: WheelEvent) {
  if (!e.ctrlKey && !e.metaKey) return
  e.preventDefault()
  const r = frame.value!.getBoundingClientRect()
  zoom(Math.exp(-e.deltaY / 300), e.clientX - r.left, e.clientY - r.top)
}

// Every pointer down, where it is now; one drags, two pinch.
const pointers = new Map<number, { x: number; y: number }>()
let travelled = 0

function down(e: PointerEvent) {
  if (e.button !== 0) return
  pointers.set(e.pointerId, { x: e.clientX, y: e.clientY })
  travelled = 0
}

function move(e: PointerEvent) {
  const was = pointers.get(e.pointerId)
  if (!was) return
  // let go outside the frame, where its pointerup went to somebody else
  if (e.buttons === 0) {
    pointers.delete(e.pointerId)
    return
  }
  if (pointers.size === 1) {
    const dx = e.clientX - was.x
    const dy = e.clientY - was.y
    travelled += Math.abs(dx) + Math.abs(dy)
    if (travelled > 4) frame.value?.setPointerCapture(e.pointerId)
    at.value = { x: at.value.x + dx, y: at.value.y + dy }
  } else if (pointers.size === 2) {
    const other = [...pointers.entries()].find(([id]) => id !== e.pointerId)![1]
    const before = Math.hypot(was.x - other.x, was.y - other.y)
    const after = Math.hypot(e.clientX - other.x, e.clientY - other.y)
    const r = frame.value!.getBoundingClientRect()
    travelled += 10
    if (before > 0) zoom(after / before, (e.clientX + other.x) / 2 - r.left, (e.clientY + other.y) / 2 - r.top)
  }
  pointers.set(e.pointerId, { x: e.clientX, y: e.clientY })
}

function up(e: PointerEvent) {
  pointers.delete(e.pointerId)
}

// A drag that ends on a box is not a click on it.
function click(e: MouseEvent) {
  if (travelled > 4) {
    e.preventDefault()
    e.stopPropagation()
  }
}

const transform = computed(() => `translate(${at.value.x} ${at.value.y}) scale(${scale.value})`)
// the top of a timeline's rows, which its years are marked above
const axis = computed(() => Math.min(...props.boxes.map((b) => b.y), props.height))
const tone = (sex?: string) => ({ M: 'male', F: 'female' })[sex ?? ''] ?? 'unknown'
// a name set apart from the years beside it in an SVG text, which takes no dir, so that
// a name written right to left does not carry the years into its own direction
const isolated = (name: string) => `\u2068${name}\u2069`

onMounted(open)
watch(() => [props.width, props.height, props.boxes], open)
</script>

<template>
  <div class="chart">
    <div
      ref="frame"
      class="frame"
      @wheel="wheel"
      @pointerdown="down"
      @pointermove="move"
      @pointerup="up"
      @pointercancel="up"
      @click.capture="click"
    >
      <svg role="img" :aria-label="label" :data-scale="scale.toFixed(2)">
        <g :transform="transform">
          <g v-for="tick in ticks" :key="tick.year" class="tick">
            <line :x1="tick.x" :x2="tick.x" :y1="axis" :y2="height" />
            <text :x="tick.x" :y="axis - 6">{{ tick.year }}</text>
          </g>
          <path v-for="(line, i) in lines" :key="i" :d="line.path" class="line" :class="line.tie" />
          <template v-for="(b, i) in boxes" :key="b.key">
            <g v-if="b.sector" :data-person="b.key">
              <clipPath :id="`${id}-${i}`">
                <path :d="b.sector.path" />
              </clipPath>
              <RouterLink v-slot="{ href, navigate }" :to="b.to ?? ''" custom>
                <a :href="href" class="sector" :class="tone(b.sex)" :clip-path="`url(#${id}-${i})`" @click="navigate">
                  <path :d="b.sector.path" class="slice" />
                  <text :transform="`translate(${b.sector.label.x} ${b.sector.label.y}) rotate(${b.sector.turn})`">
                    <tspan x="0" :dy="b.lifespan ? '-0.2em' : '0.35em'" class="name">
                      {{ isolated(b.name) }}
                    </tspan>
                    <tspan v-if="b.lifespan" x="0" dy="1.25em" class="lifespan">
                      {{ b.lifespan }}
                    </tspan>
                  </text>
                </a>
              </RouterLink>
            </g>
            <g v-else-if="b.bar" :data-person="b.key">
              <RouterLink v-slot="{ href, navigate }" :to="b.to ?? ''" custom>
                <a :href="href" class="life" @click="navigate">
                  <text :x="b.bar.x" :y="b.y + 12" class="name">
                    {{ isolated(b.name) }}
                    <tspan v-if="b.lifespan" class="lifespan">
                      {{ b.lifespan }}
                    </tspan>
                  </text>
                  <rect
                    :x="b.bar.x"
                    :y="b.y + 17"
                    :width="b.bar.width"
                    height="10"
                    rx="3"
                    class="bar"
                    :class="[tone(b.sex), { subject: b.subject, open: !b.bar.born || !b.bar.ended }]"
                  />
                </a>
              </RouterLink>
            </g>
            <foreignObject
              v-else
              :x="b.x"
              :y="b.y"
              :width="box.width"
              :height="box.height"
              :data-person="b.gap ? undefined : b.key"
              :data-gap="b.gap ? b.key : undefined"
            >
              <button v-if="b.gap" type="button" class="box gap" draggable="false" @click="emit('press', b.key)">
                <Icon name="plus" />
                <span class="name">{{ b.name }}</span>
              </button>
              <component
                :is="b.to !== undefined ? RouterLink : 'span'"
                v-else
                :to="b.to"
                class="box"
                :class="[tone(b.sex), { subject: b.subject }]"
                draggable="false"
              >
                <Portrait :src="b.src" :region="b.region" :initials="b.initials" :sex="b.sex" />
                <span class="words">
                  <span class="name" dir="auto">{{ b.name }}</span>
                  <span v-if="b.lifespan" class="lifespan">{{ b.lifespan }}</span>
                </span>
              </component>
            </foreignObject>
          </template>
        </g>
      </svg>
    </div>
    <div class="controls">
      <IconButton icon="plus" :label="labels.zoomIn" bordered @click="zoom(1.25)" />
      <IconButton icon="minus" :label="labels.zoomOut" bordered @click="zoom(0.8)" />
      <IconButton icon="fit" :label="labels.fit" bordered @click="fit" />
    </div>
  </div>
</template>

<style scoped>
.chart {
  position: relative;
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  background: var(--surface);
  overflow: hidden;
}
.frame {
  height: var(--chart-height);
  touch-action: none;
  cursor: grab;
  user-select: none;
}
.frame:active {
  cursor: grabbing;
}
svg {
  display: block;
  width: 100%;
  height: 100%;
}
.line {
  fill: none;
  stroke: var(--line-strong);
  stroke-width: 1.5;
}
.line.other {
  stroke-dasharray: 5 4;
}
.line.partner {
  stroke: var(--line);
}
.line.gap {
  stroke: var(--line);
  stroke-dasharray: 5 4;
}
.box {
  box-sizing: border-box;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  height: 100%;
  padding: 0 var(--space-2);
  border-radius: var(--radius-md);
  border: var(--border) solid transparent;
  color: var(--ink);
  text-decoration: none;
  -webkit-user-drag: none;
}
.male {
  background: var(--male);
}
.female {
  background: var(--female);
}
.unknown {
  background: var(--unknown);
}
.subject {
  border-color: var(--act);
  border-width: var(--border-strong);
}
a.box:hover {
  border-color: var(--line-strong);
}
a.box.subject:hover {
  border-color: var(--act);
}
.gap {
  font: inherit;
  color: var(--muted);
  background: var(--unknown);
  border: var(--border) dashed var(--line-strong);
  cursor: pointer;
}
.gap:hover {
  color: var(--ink);
  border-color: var(--act);
}
.gap .name {
  font-weight: var(--weight-medium);
}
a.box:focus-visible,
.gap:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: calc(-1 * var(--border-strong));
}
.words {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.name {
  font-weight: var(--weight-bold);
  font-size: var(--text-sm);
  line-height: var(--leading-tight);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.lifespan {
  font-size: var(--text-xs);
  color: var(--muted);
}
.tick line {
  stroke: var(--line);
}
.tick text {
  font-size: var(--text-xs);
  fill: var(--muted);
  text-anchor: middle;
}
.slice {
  stroke: var(--surface);
  stroke-width: 2;
}
.sector.male .slice {
  fill: var(--male);
}
.sector.female .slice {
  fill: var(--female);
}
.sector.unknown .slice {
  fill: var(--unknown);
}
.sector:hover .slice {
  stroke: var(--line-strong);
}
.sector text {
  text-anchor: middle;
  fill: var(--ink);
}
.sector .name,
.life .name {
  font-weight: var(--weight-bold);
  font-size: var(--text-xs);
}
.sector .lifespan,
.life .lifespan {
  font-size: var(--text-xs);
  fill: var(--muted);
  font-weight: normal;
}
.life text {
  fill: var(--ink);
}
.bar.male {
  fill: var(--male);
}
.bar.female {
  fill: var(--female);
}
.bar.unknown {
  fill: var(--unknown);
}
.bar {
  stroke: var(--line-strong);
  stroke-width: 1;
}
.bar.open {
  stroke-dasharray: 4 3;
}
.bar.subject {
  stroke: var(--act);
  stroke-width: 2;
}
.life:hover .bar {
  stroke: var(--act);
}
.sector:focus-visible,
.life:focus-visible {
  outline: var(--border-strong) solid var(--act);
}
.controls {
  position: absolute;
  top: var(--space-2);
  right: var(--space-2);
  display: flex;
  gap: var(--space-1);
  background: var(--surface);
  border-radius: var(--radius);
}
</style>
