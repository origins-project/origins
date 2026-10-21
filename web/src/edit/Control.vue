<script setup lang="ts">
import { computed } from 'vue'
import { Field, Select, Switch, type Option } from '../ui'
import CitePicker from './CitePicker.vue'
import DateField from './DateField.vue'
import ProseField from './ProseField.vue'
import RecordPicker from './RecordPicker.vue'
import type { Control } from './schema'
import type { Value } from './values'

// One value changed by the control its schema gives it: a date by the date's field, a
// slug by the records of its kind, a flag by a switch, a citation by every one there is,
// prose beside what it will look like.
const model = defineModel<Value>()
const props = defineProps<{ control: Control; label?: string; path: string }>()

const none = '-'
const text = computed({
  get: () => (model.value === undefined || model.value === null ? '' : String(model.value)),
  set: (to: string) => (model.value = to),
})
const flag = computed({
  get: () => model.value === true || model.value === 'true',
  set: (on: boolean) => (model.value = on ? true : undefined),
})
const options = computed<Option[]>(() =>
  props.control.is === 'choice'
    ? [{ value: none, label: '—' }, ...props.control.options.map((o) => ({ value: o, label: o }))]
    : [],
)
</script>

<template>
  <DateField v-if="control.is === 'date'" v-model="text" :label="label" :path="path" />
  <Switch v-else-if="control.is === 'flag'" v-model="flag" :label="label ?? path" :path="path" />
  <Select
    v-else-if="control.is === 'choice'"
    :model-value="text || none"
    :label="label"
    :options="options"
    :path="path"
    @update:model-value="(v) => (text = !v || v === none ? '' : v)"
  />
  <RecordPicker v-else-if="control.is === 'record'" v-model="text" :kind="control.kind" :label="label" :path="path" />
  <CitePicker v-else-if="control.is === 'citation'" v-model="text" :label="label" :path="path" />
  <ProseField v-else-if="control.multiline" v-model="text" :label="label" :path="path" />
  <Field v-else v-model="text" :label="label" :path="path" />
</template>
