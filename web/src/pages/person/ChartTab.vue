<script setup lang="ts">
import { computed, ref, watchEffect } from 'vue'
import { defaultDistance, draw, maxDistance, shapes, type Gap, type Laid, type Shape } from '../../chart'
import AddParent from '../../edit/AddParent.vue'
import RecordPicker from '../../edit/RecordPicker.vue'
import { useChange } from '../../edit/change'
import { chart as said } from '../../labels'
import { useNaming } from '../../naming'
import { Chart, Chip, Definition, Definitions, Notice, Select, Stack, type ChartBox } from '../../ui'

// A person's charts, drawn from the index in the shape chosen, as far out as the
// distance asks or, for a path, to the person chosen, with a box for each parent
// missing where the tab may change the corpus.
const props = defineProps<{ slug: string }>()
const { index, card } = useNaming()
const { writes } = useChange()

const shape = ref<Shape>('ancestors')
const options = shapes.map((value) => ({ value, label: said.shapes[value] }))
const distance = ref(String(defaultDistance))
const distances = Array.from({ length: maxDistance }, (_, i) => ({ value: String(i + 1), label: String(i + 1) }))
// a path runs to the main person as the index names them now, until somebody is chosen
const chosen = ref<string>()
const to = computed({
  get: () => {
    const main = index.value?.project['main-person'] as string | undefined
    return chosen.value ?? (main && main !== props.slug ? main : '')
  },
  set: (slug: string) => (chosen.value = slug),
})

// the chart last laid out, an answer overtaken by another asked after it dropped
const drawing = ref<Laid>()
let asked = 0
watchEffect(async () => {
  if (!index.value) return
  const one = ++asked
  const ask = { subject: props.slug, shape: shape.value, distance: Number(distance.value), gaps: writes.value, to: to.value }
  const laid = await draw(index.value, ask)
  if (one === asked) drawing.value = laid
})
const boxes = computed<ChartBox[]>(() => {
  if (!drawing.value) return []
  const people = drawing.value.people.map(({ slug, x, y, sector, bar }) => ({
    key: slug,
    x,
    y,
    sector,
    bar,
    ...card(slug),
    subject: slug === props.slug,
  }))
  const gaps = drawing.value.slots.map((s) => ({ key: s.key, x: s.x, y: s.y, name: said.addParent, gap: true }))
  return [...people, ...gaps]
})

const gap = ref<Gap>()
const adding = ref(false)
function press(key: string) {
  gap.value = drawing.value?.slots.find((s) => s.key === key)
  adding.value = gap.value !== undefined
}
const name = computed(() => card(props.slug).name)
const common = computed(() => card(drawing.value?.path?.['common-ancestor']))
</script>

<template>
  <Stack gap="3">
    <Stack row wrap gap="3" align="end">
      <Select v-model="shape" :label="said.shape" :options="options" />
      <RecordPicker v-if="shape === 'path'" v-model="to" kind="person" :label="said.to" />
      <Select v-else v-model="distance" :label="said.distance[shape]" :options="distances" />
      <Notice v-if="drawing?.truncated" tone="warn">{{ said.limited(drawing.limit) }}</Notice>
    </Stack>
    <Definitions v-if="drawing?.path?.related || drawing?.undated" compact>
      <template v-if="drawing.path?.related">
        <Definition :term="said.relation">{{ drawing.path.relation }}</Definition>
        <Definition :term="said.commonAncestor">
          <Chip v-if="common.to" :to="common.to">{{ common.name }}</Chip>
        </Definition>
      </template>
      <Definition v-if="drawing.undated" :term="said.undated">{{ drawing.undated }}</Definition>
    </Definitions>
    <Chart
      v-if="drawing && drawing.shape === shape"
      :boxes="boxes"
      :lines="drawing.lines"
      :width="drawing.width"
      :height="drawing.height"
      :box="drawing.box"
      :ticks="drawing.ticks"
      :label="said.of[shape](name)"
      @press="press"
    />
    <AddParent v-if="gap" v-model:open="adding" :gap="gap" />
  </Stack>
</template>
