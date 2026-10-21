<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { edit as said, fieldLabel } from '../labels'
import { Button, Drawer, Notice, Popover, Stack, Text, Value } from '../ui'
import Control from './Control.vue'
import CitePicker from './CitePicker.vue'
import { refusalOf, useChange, useDrafted, useEditing, type Refusal } from './change'
import { addressable, controlOf, nodeAt, scalarOf, steps, useSchemas, written, type Path } from './schema'
import { empty, placed, shown, valueAt, type Value as Held } from './values'

// One value of a record changed where it is shown, in a popover, or prose in a drawer
// wide enough to show it beside what it will look like: set, taken out, or, for a list,
// added to. The value is the trigger; one not there yet is an add button.
const props = defineProps<{ path: Path; list?: boolean; label?: string }>()

const { record, kind, entity } = useEditing()
const { client, writes, offered, make } = useChange()
const schemas = useSchemas(client)

const open = ref(false)
const path = computed(() => written(props.path))
const held = computed(() => (props.list ? undefined : valueAt(record.value?.fields, props.path)))
const route = computed(() => (props.list ? [...steps(props.path), listed.value] : steps(props.path)))
const at = computed(() => written(route.value))
const listed = computed(() => {
  const list = valueAt(record.value?.fields, props.path)
  return Array.isArray(list) ? list.length : 0
})
const control = computed(() => {
  if (!schemas.value) return undefined
  const found = nodeAt(schemas.value, kind.value, route.value)
  const scalar = found && scalarOf(schemas.value, found)
  return scalar && controlOf(scalar, route.value, kind.value)
})
const marked = useDrafted(() => path.value)
const named = computed(() => props.label ?? fieldLabel(at.value) ?? path.value)
const wide = computed(() => control.value?.is === 'text' && !!control.value.multiline)
const shell = computed(() => (wide.value ? Drawer : Popover))

const value = ref<Held>()
const cite = ref('')
const refusal = ref<Refusal>()
const busy = ref(false)
// the record as the form was opened on it, which the change is pinned to
let seen = ''
watch(open, (now) => {
  if (!now) return
  seen = record.value?.seen ?? ''
  value.value = held.value
  cite.value = ''
  refusal.value = undefined
})

// The edit, made at the path, or where a key on it is no step of a path, as a set of
// the field above that key whole with the change made in it.
function edited(op: 'set' | 'unset' | 'add') {
  const cut = route.value.findIndex((s) => !addressable(s))
  if (cut < 0) return { op, entity: entity.value, path: path.value, ...(op === 'unset' ? {} : { value: value.value }) }
  const above = route.value.slice(0, cut)
  const whole = placed(valueAt(record.value?.fields, above), route.value.slice(cut), op === 'unset' ? undefined : value.value)
  return { op: 'set', entity: entity.value, path: written(above), value: whole }
}

async function change(op: 'set' | 'unset' | 'add') {
  if (!seen || busy.value) return
  busy.value = true
  const edit = edited(op)
  try {
    await make(
      () => client.change({ edit, seen, sources: cite.value ? [cite.value] : undefined }),
      () => (open.value = false),
    )
  } catch (e) {
    refusal.value = refusalOf(e)
  } finally {
    busy.value = false
  }
}

function save() {
  if (props.list) return empty(value.value) ? undefined : change('add')
  return change(empty(value.value) ? 'unset' : 'set')
}

// Enter saves, unless it is choosing from a list that is open.
function entered(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  if (target.tagName === 'TEXTAREA' || target.getAttribute('aria-expanded') === 'true') return
  e.preventDefault()
  void save()
}
</script>

<template>
  <component :is="shell" v-model:open="open" :path="at" v-bind="wide ? { title: named, wide: true } : {}">
    <template #trigger>
      <Value v-if="!list && !empty(held)" :path="path" :label="named" :disabled="!writes" :marked="marked">
        <slot>{{ shown(held) }}</slot>
      </Value>
      <Button v-else-if="offered" size="sm" variant="add" icon="plus" :data-path="at" :disabled="!writes">{{ named }}</Button>
    </template>
    <Stack gap="3" @keydown.enter="entered">
      <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">{{ refusal.message }}</Notice>
      <Control v-if="control" v-model="value" :control="control" :label="named" :path="at" />
      <Text v-else tone="faint">{{ shown(held) }}</Text>
      <CitePicker v-model="cite" :label="said.cite" />
    </Stack>
    <template #actions>
      <Button v-if="!list && !empty(held)" variant="danger" size="sm" :disabled="busy" @click="change('unset')">
        {{ said.remove }}
      </Button>
      <Button size="sm" variant="ghost" @click="open = false">{{ said.cancel }}</Button>
      <Button size="sm" variant="primary" :disabled="busy || !control" @click="save">
        {{ list ? said.add : said.save }}
      </Button>
    </template>
  </component>
</template>
