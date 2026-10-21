<script setup lang="ts" generic="T">
import { useVirtualizer, useWindowVirtualizer, type Virtualizer } from '@tanstack/vue-virtual'
import { computed, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import Icon from './Icon.vue'

// A list of records as rows, for as many as a corpus holds: only the rows in view are
// drawn, the rest standing in as two spacers of their height, so a list of thousands
// costs what a screenful does. It scrolls with the page, or in a box of its own when
// given box. A column sorts by its value when pressed, and a row is opened by a click
// or by Enter. A cell draws its value, or the slot named cell-<key> draws it.
export interface Column<R> {
  key: string
  label: string
  value?: (row: R) => string | number | undefined
  // What the column sorts by, where it is not what it shows: a date's year.
  order?: (row: R) => string | number | undefined
  width?: 'narrow' | 'mid' | 'grow'
  align?: 'start' | 'end'
  sortable?: boolean
  mono?: boolean
}
export interface Sort {
  key: string
  descending?: boolean
}

const props = defineProps<{
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  label: string
  box?: boolean
  current?: string
}>()
const sort = defineModel<Sort | undefined>('sort')
const emit = defineEmits<{ open: [row: T] }>()
defineSlots<{ [cell: `cell-${string}`]: (scope: { row: T; value: unknown }) => unknown; empty?: () => unknown }>()

// The row height the virtualizer assumes until it has measured one: a line of text and
// the cell's padding at the root size.
const estimate = 37

const valueOf = (column: Column<T>, row: T): unknown =>
  column.value ? column.value(row) : (row as Record<string, unknown>)[column.key]

const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })
function compare(a: unknown, b: unknown): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  return collator.compare(String(a), String(b))
}

const sorted = computed(() => {
  const by = sort.value
  const column = by && props.columns.find((c) => c.key === by.key)
  if (!by || !column) return props.rows
  const sign = by.descending ? -1 : 1
  return [...props.rows].sort((x, y) => {
    const a = column.order ? column.order(x) : valueOf(column, x)
    const b = column.order ? column.order(y) : valueOf(column, y)
    // Whatever has no value goes last, whichever way the column is sorted.
    if (a === undefined || a === '') return b === undefined || b === '' ? 0 : 1
    if (b === undefined || b === '') return -1
    return sign * compare(a, b)
  })
})

function press(column: Column<T>) {
  if (!column.sortable) return
  const by = sort.value
  if (by?.key !== column.key) sort.value = { key: column.key }
  else if (!by.descending) sort.value = { key: column.key, descending: true }
  else sort.value = undefined
}

function ariaSort(column: Column<T>) {
  if (sort.value?.key !== column.key) return column.sortable ? 'none' : undefined
  return sort.value.descending ? 'descending' : 'ascending'
}

const scroller = ref<HTMLElement | null>(null)
const body = ref<HTMLElement | null>(null)
const margin = ref(0)

const common = () => ({
  count: sorted.value.length,
  estimateSize: () => estimate,
  overscan: 10,
  // A row measured inside the observer's delivery would resize what it observes.
  useAnimationFrameWithResizeObserver: true,
  scrollMargin: margin.value,
  getItemKey: (index: number) => props.rowKey(sorted.value[index] as T),
})
const virtualizer = (
  props.box
    ? useVirtualizer(computed(() => ({ ...common(), getScrollElement: () => scroller.value })))
    : useWindowVirtualizer(computed(common))
) as Ref<Virtualizer<Element | Window, Element>>

// Where the first row starts in what scrolls, which the virtualizer counts from.
function place() {
  const top = body.value?.getBoundingClientRect().top ?? 0
  margin.value = props.box
    ? top - (scroller.value?.getBoundingClientRect().top ?? 0) + (scroller.value?.scrollTop ?? 0)
    : top + window.scrollY
}
// The page's size changing moves the table, and placing it again changes the page's size
// when the rows it draws do, so it waits for the next frame rather than answering inside
// the observer's own delivery, which the browser reports as a loop.
let observer: ResizeObserver | undefined
let frame = 0
onMounted(() => {
  place()
  observer = new ResizeObserver(() => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(place)
  })
  observer.observe(document.body)
})
onBeforeUnmount(() => {
  observer?.disconnect()
  cancelAnimationFrame(frame)
})
watch(() => props.rows, place, { flush: 'post' })

const items = computed(() => virtualizer.value.getVirtualItems())
const before = computed(() => (items.value[0]?.start ?? margin.value) - margin.value)
const after = computed(() => {
  const last = items.value[items.value.length - 1]
  return last ? virtualizer.value.getTotalSize() - (last.end - margin.value) : 0
})

function measure(el: unknown) {
  if (el instanceof Element) virtualizer.value.measureElement(el)
}

function key(event: KeyboardEvent, row: T) {
  if (event.key === 'Enter') emit('open', row)
}
</script>

<template>
  <div ref="scroller" class="table" :class="{ box }">
    <table :aria-label="label" :aria-rowcount="sorted.length + 1">
      <colgroup>
        <col v-for="column in columns" :key="column.key" :class="column.width ?? 'grow'" />
      </colgroup>
      <thead>
        <tr aria-rowindex="1">
          <th
            v-for="column in columns"
            :key="column.key"
            scope="col"
            :class="column.align"
            :aria-sort="ariaSort(column)"
          >
            <button v-if="column.sortable" type="button" class="sort" @click="press(column)">
              {{ column.label }}
              <Icon
                name="chevron"
                class="arrow"
                :class="{ shown: sort?.key === column.key, up: sort?.key === column.key && !sort.descending }"
              />
            </button>
            <template v-else>{{ column.label }}</template>
          </th>
        </tr>
      </thead>
      <tbody ref="body">
        <tr v-if="sorted.length === 0" class="empty">
          <td :colspan="columns.length"><slot name="empty" /></td>
        </tr>
        <tr v-if="before > 0" class="spacer" aria-hidden="true" :style="{ height: `${before}px` }">
          <td :colspan="columns.length" />
        </tr>
        <tr
          v-for="item in items"
          :key="item.key as string"
          :ref="measure"
          :data-index="item.index"
          :aria-rowindex="item.index + 2"
          :aria-current="current !== undefined && current === item.key ? 'true' : undefined"
          tabindex="0"
          class="row"
          @click="emit('open', sorted[item.index] as T)"
          @keydown="key($event, sorted[item.index] as T)"
        >
          <td v-for="column in columns" :key="column.key" :class="[column.align, { mono: column.mono }]">
            <slot
              :name="`cell-${column.key}`"
              :row="sorted[item.index] as T"
              :value="valueOf(column, sorted[item.index] as T)"
            >
              {{ valueOf(column, sorted[item.index] as T) }}
            </slot>
          </td>
        </tr>
        <tr v-if="after > 0" class="spacer" aria-hidden="true" :style="{ height: `${after}px` }">
          <td :colspan="columns.length" />
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table {
  background: var(--surface);
  border: var(--border) solid var(--line);
  border-radius: var(--radius-md);
  /* Clipped rather than scrolled, so that the head sticks under the bar as the page
     scrolls; the columns are fixed and a cell ellipsises rather than widen them. */
  overflow: clip;
}
.box {
  max-height: var(--table-box);
  overflow-y: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
  font-size: var(--text-md);
}
col.narrow {
  width: var(--col-narrow);
}
col.mid {
  width: var(--col);
}
th {
  position: sticky;
  top: 0;
  background: var(--surface);
  text-align: start;
  font: var(--weight-bold) var(--text-2xs) var(--sans);
  letter-spacing: var(--tracking-caps);
  text-transform: uppercase;
  color: var(--faint);
  padding: var(--space-3) var(--space-4);
  border-bottom: var(--border) solid var(--line);
  white-space: nowrap;
}
.table:not(.box) th {
  top: var(--bar);
}
.sort {
  all: unset;
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  cursor: pointer;
}
.sort:hover {
  color: var(--ink);
}
.sort:focus-visible {
  outline: var(--border-strong) solid var(--act);
  outline-offset: var(--border-strong);
}
.arrow {
  opacity: 0;
}
.arrow.shown {
  opacity: 1;
}
.arrow.up {
  transform: rotate(180deg);
}
td {
  padding: var(--space-2) var(--space-4);
  border-bottom: var(--border) solid var(--line);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.end {
  text-align: end;
  font-variant-numeric: tabular-nums;
}
.mono {
  font-family: var(--mono);
  font-size: var(--text-sm);
}
.row {
  cursor: pointer;
}
.row:hover {
  background: var(--surface-2);
}
.row:focus-visible {
  outline-offset: calc(-1 * var(--border-strong));
}
.row[aria-current='true'] {
  background: var(--act-soft);
}
.spacer td,
.empty td {
  border-bottom: 0;
  padding: 0;
}
.empty td {
  padding: var(--space-6) var(--space-4);
  color: var(--muted);
  white-space: normal;
}
tbody tr:last-child td {
  border-bottom: 0;
}
</style>
