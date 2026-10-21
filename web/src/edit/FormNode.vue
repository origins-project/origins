<script setup lang="ts">
import { computed } from 'vue'
import { edit as said, fieldLabel } from '../labels'
import { Button, Entry, Field, Fieldset, IconButton, Stack } from '../ui'
import Control from './Control.vue'
import {
  arrayOf,
  controlOf,
  leadOf,
  mapOf,
  objectOf,
  resolve,
  scalarOf,
  steps,
  type Found,
  type Node,
  type Schemas,
} from './schema'
import { empty, type Value } from './values'

// A value drawn from its schema: a scalar as its control, a mapping as its fields in the
// schema's order, a list as its entries, each with what adds to it and takes from it.
// Every change hands a new value up, so the whole entry is what its form holds.
const model = defineModel<Value>()
const props = defineProps<{ schemas: Schemas; found: Found; path: string; kind: string }>()

type Mapping = { [key: string]: Value }
const isMapping = (v: Value): v is Mapping => v !== null && typeof v === 'object' && !Array.isArray(v)

const object = computed(() => objectOf(props.schemas, props.found))
const array = computed(() => arrayOf(props.schemas, props.found))
const map = computed(() => mapOf(props.schemas, props.found))
const scalar = computed(() => scalarOf(props.schemas, props.found))

const shape = computed(() => {
  const v = model.value
  if (Array.isArray(v) && array.value) return 'array'
  if (isMapping(v) && object.value) return 'object'
  if (isMapping(v) && map.value) return 'map'
  if (scalar.value) return 'scalar'
  if (object.value) return 'object'
  if (array.value) return 'array'
  if (map.value) return 'map'
  return 'scalar'
})

const label = (path: string) => fieldLabel(path) ?? String(steps(path).at(-1) ?? path)
const own = computed(() => props.kind as Parameters<typeof controlOf>[2])
const control = computed(() => scalar.value && controlOf(scalar.value, props.path, own.value))

const within = (parent: Found | undefined, node: unknown): Found =>
  resolve(props.schemas, { node: node as Node, refs: parent?.refs ?? [], from: parent?.from ?? props.found.from })

// What a new entry of a node starts as: text where it may be text, else what holds fields.
function blank(found: Found): Value {
  if (scalarOf(props.schemas, found)) return ''
  if (arrayOf(props.schemas, found)) return []
  return {}
}

// A scalar that may also be written out in full, made what it is short for: the list it
// is the one entry of, where it may be a list, or else the mapping it is the lead of.
const lead = computed(() => (object.value ? leadOf(object.value.node) : undefined))
const expanded = computed(() => (array.value ? `${props.path}[0]` : lead.value && `${props.path}.${lead.value}`))
function expand() {
  if (array.value) model.value = empty(model.value) ? [] : [model.value]
  else if (lead.value) model.value = empty(model.value) ? {} : { [lead.value]: model.value }
}

const mapping = computed<Mapping>(() => (isMapping(model.value) ? model.value : {}))
const properties = computed(() =>
  Object.entries((object.value?.node.properties as Node | undefined) ?? {}).map(([name, node]) => {
    const found = within(object.value, node)
    const path = props.path ? `${props.path}.${name}` : name
    const value = mapping.value[name]
    const held = !(scalarOf(props.schemas, found) === undefined && empty(value))
    return { name, found, path, value, held }
  }),
)
function setField(name: string, to: Value) {
  const next = { ...mapping.value }
  if (to === undefined) delete next[name]
  else next[name] = to
  model.value = next
}
function addField(name: string, found: Found) {
  const list = arrayOf(props.schemas, found)
  setField(name, list ? [blank(within(list, list.node.items))] : {})
}

const entries = computed<Value[]>(() => (Array.isArray(model.value) ? model.value : []))
const item = computed(() => array.value && within(array.value, array.value.node.items))
function setEntry(i: number, to: Value) {
  model.value = entries.value.map((e, at) => (at === i ? to : e))
}
function removeEntry(i: number) {
  model.value = entries.value.filter((_, at) => at !== i)
}
function addEntry() {
  if (item.value) model.value = [...entries.value, blank(item.value)]
}

const keyed = computed(() => Object.entries(mapping.value))
const valued = computed(() => map.value && within(map.value, map.value.node.additionalProperties))
function rekey(i: number, key: string) {
  model.value = Object.fromEntries(keyed.value.map(([k, v], at) => [at === i ? key : k, v]))
}
function setKeyed(i: number, to: Value) {
  model.value = Object.fromEntries(keyed.value.map(([k, v], at) => [k, at === i ? to : v]))
}
function removeKeyed(i: number) {
  model.value = Object.fromEntries(keyed.value.filter((_, at) => at !== i))
}
function addKeyed() {
  if (valued.value && !('' in mapping.value)) model.value = { ...mapping.value, '': blank(valued.value) }
}
</script>

<template>
  <Stack v-if="shape === 'scalar' && control" gap="2" align="stretch">
    <Control v-model="model" :control="control" :label="label(path)" :path="path" />
    <Stack v-if="expanded" row>
      <Button size="sm" variant="ghost" icon="plus" :data-path="expanded" @click="expand">
        {{ said.mapping }}
      </Button>
    </Stack>
  </Stack>

  <Stack v-else-if="shape === 'object'" gap="3" :data-path="path || undefined">
    <template v-for="field in properties" :key="field.name">
      <Fieldset
        v-if="field.held && !scalarOf(schemas, field.found) && !Array.isArray(field.value)"
        :legend="label(field.path)"
        :path="field.path"
      >
        <FormNode
          :model-value="field.value"
          :schemas="schemas"
          :found="field.found"
          :path="field.path"
          :kind="kind"
          @update:model-value="(to) => setField(field.name, to)"
        />
      </Fieldset>
      <FormNode
        v-else-if="field.held"
        :model-value="field.value"
        :schemas="schemas"
        :found="field.found"
        :path="field.path"
        :kind="kind"
        @update:model-value="(to) => setField(field.name, to)"
      />
    </template>
    <Stack v-if="properties.some((f) => !f.held)" row wrap gap="2">
      <template v-for="field in properties" :key="field.name">
        <Button
          v-if="!field.held"
          size="sm"
          variant="add"
          icon="plus"
          :data-path="field.path"
          @click="addField(field.name, field.found)"
        >
          {{ label(field.path) }}
        </Button>
      </template>
    </Stack>
  </Stack>

  <Fieldset v-else-if="shape === 'array' && item" :legend="label(path)" :path="path">
    <Entry v-for="(entry, i) in entries" :key="i" :path="`${path}[${i}]`">
      <FormNode
        :model-value="entry"
        :schemas="schemas"
        :found="item"
        :path="`${path}[${i}]`"
        :kind="kind"
        @update:model-value="(to) => setEntry(i, to)"
      />
      <template #actions>
        <IconButton icon="x" :label="said.remove" @click="removeEntry(i)" />
      </template>
    </Entry>
    <template #add>
      <Button size="sm" variant="add" icon="plus" :data-path="`${path}[${entries.length}]`" @click="addEntry">
        {{ said.add }}
      </Button>
    </template>
  </Fieldset>

  <Fieldset v-else-if="shape === 'map' && valued" :legend="label(path)" :path="path">
    <Entry v-for="([key, entry], i) in keyed" :key="i" :path="`${path}.${key}`">
      <Stack gap="2">
        <Field :model-value="key" :label="said.key" mono @update:model-value="(k) => rekey(i, k)" />
        <FormNode
          :model-value="entry"
          :schemas="schemas"
          :found="valued"
          :path="`${path}.${key}`"
          :kind="kind"
          @update:model-value="(to) => setKeyed(i, to)"
        />
      </Stack>
      <template #actions>
        <IconButton icon="x" :label="said.remove" @click="removeKeyed(i)" />
      </template>
    </Entry>
    <template #add>
      <Button size="sm" variant="add" icon="plus" :data-path="`${path}.*`" @click="addKeyed">{{ said.add }}</Button>
    </template>
  </Fieldset>
</template>
