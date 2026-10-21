<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { edit as said, fieldLabel } from '../labels'
import { Button, Drawer, Notice, Stack, Text } from '../ui'
import CitePicker from './CitePicker.vue'
import FormNode from './FormNode.vue'
import { refusalOf, useChange, useDrafted, useEditing, type Refusal } from './change'
import { leadOf, nodeAt, objectOf, scalarOf, useSchemas } from './schema'
import { copy, empty, pruned, valueAt, type Value } from './values'

// An entry of several fields changed in a drawer whose form is drawn from the schema at
// its path: an event, a name, a citation; or, for a list, a new entry added to it.
const props = defineProps<{ path: string; list?: boolean; label?: string }>()

const { record, kind, entity } = useEditing()
const { client, writes, offered, make } = useChange()
const schemas = useSchemas(client)

const open = ref(false)
const listed = computed(() => {
  const list = valueAt(record.value?.fields, props.path)
  return Array.isArray(list) ? list.length : 0
})
const at = computed(() => (props.list ? `${props.path}[${listed.value}]` : props.path))
const found = computed(() => schemas.value && nodeAt(schemas.value, kind.value, at.value))
const held = computed(() => (props.list ? undefined : valueAt(record.value?.fields, props.path)))
const marked = useDrafted(() => props.path)
const named = computed(() => props.label ?? fieldLabel(at.value) ?? props.path)

const value = ref<Value>()
const cite = ref('')
const refusal = ref<Refusal>()
const busy = ref(false)
// the record as the form was opened on it, which the change is pinned to
let seen = ''

// The entry as its form holds it: a value written short is opened out into its fields.
function start(): Value {
  const given = copy(held.value)
  const s = schemas.value
  if (!s || !found.value) return given
  const object = objectOf(s, found.value)
  const lead = object && leadOf(object.node)
  if (object && lead && (given === undefined || typeof given !== 'object')) return empty(given) ? {} : { [lead]: given }
  if (given === undefined && !scalarOf(s, found.value)) return {}
  return given
}

watch(open, (now) => {
  if (!now) return
  seen = record.value?.seen ?? ''
  value.value = start()
  cite.value = ''
  refusal.value = undefined
})

async function change(op: 'set' | 'unset' | 'add') {
  if (!seen || busy.value) return
  busy.value = true
  try {
    const edit = { op, entity: entity.value, path: props.path, ...(op === 'unset' ? {} : { value: pruned(value.value) }) }
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
  const kept = pruned(value.value)
  if (props.list) return kept === undefined ? undefined : change('add')
  return change(kept === undefined ? 'unset' : 'set')
}
</script>

<template>
  <Drawer v-model:open="open" :title="named" :path="at">
    <template #trigger>
      <slot v-if="!list" :disabled="!writes" :marked="marked" />
      <Button v-else-if="offered" size="sm" variant="add" icon="plus" :data-path="at" :disabled="!writes">{{ named }}</Button>
    </template>
    <Stack gap="4">
      <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">
        <Stack gap="1">
          <Text>{{ refusal.message }}</Text>
          <Text v-for="(f, i) in refusal.findings" :key="i" size="sm">
            {{ [f.entity, f.path, f.rule].filter(Boolean).join(' · ') }} — {{ f.message }}
          </Text>
        </Stack>
      </Notice>
      <FormNode v-if="schemas && found" v-model="value" :schemas="schemas" :found="found" :path="at" :kind="kind" />
      <CitePicker v-model="cite" :label="said.cite" />
    </Stack>
    <template #actions>
      <Button v-if="!list && !empty(held)" variant="danger" :disabled="busy" @click="change('unset')">{{ said.remove }}</Button>
      <Button variant="ghost" @click="open = false">{{ said.cancel }}</Button>
      <Button variant="primary" :disabled="busy" @click="save">{{ list ? said.add : said.save }}</Button>
    </template>
  </Drawer>
</template>
