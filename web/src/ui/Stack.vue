<script setup lang="ts">
import { computed } from 'vue'
import type { Space } from './layout'

// Things one after another, down the page or across it, a step of the spacing scale
// apart. A row may wrap; either may align and spread its things.
const props = withDefaults(
  defineProps<{
    row?: boolean
    gap?: Space
    align?: 'start' | 'center' | 'end' | 'stretch' | 'baseline'
    justify?: 'start' | 'center' | 'end' | 'between'
    wrap?: boolean
    grow?: boolean
    as?: string
  }>(),
  { gap: '4', as: 'div' },
)
const style = computed(() => ({ gap: `var(--space-${props.gap})` }))
</script>

<template>
  <component
    :is="as"
    class="stack"
    :class="[row ? 'row' : 'column', align && `align-${align}`, justify && `justify-${justify}`, { wrap, grow }]"
    :style="style"
  >
    <slot />
  </component>
</template>

<style scoped>
.stack {
  display: flex;
  min-width: 0;
}
ul.stack,
ol.stack {
  list-style: none;
  margin: 0;
  padding: 0;
}
.column {
  flex-direction: column;
}
.row {
  flex-direction: row;
  align-items: center;
}
.wrap {
  flex-wrap: wrap;
}
.grow {
  flex: 1;
}
.align-start {
  align-items: flex-start;
}
.align-center {
  align-items: center;
}
.align-end {
  align-items: flex-end;
}
.align-stretch {
  align-items: stretch;
}
.align-baseline {
  align-items: baseline;
}
.justify-start {
  justify-content: flex-start;
}
.justify-center {
  justify-content: center;
}
.justify-end {
  justify-content: flex-end;
}
.justify-between {
  justify-content: space-between;
}
</style>
