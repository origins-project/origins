<script setup lang="ts">
import { computed } from 'vue'
import { fieldLabel } from '../labels'
import { Button, Definition, Definitions, Stack, Value } from '../ui'
import EditEntry from './EditEntry.vue'
import EditValue from './EditValue.vue'
import { useChange, useEditing } from './change'
import { drawnOf, nodeAt, objectOf, useSchemas } from './schema'
import { empty, shown, type Value as Held } from './values'

// Every field of the record its schema names, each changed where it is shown: a value in
// a popover, an entry of several fields in a drawer, and a field not there yet added by
// the button for it. Given the names of some fields, it draws those, in that order; given
// the names of fields drawn elsewhere, every other one.
const props = defineProps<{ only?: string[]; except?: string[]; compact?: boolean }>()
const { record, kind } = useEditing()
const { client } = useChange()
const schemas = useSchemas(client)

const fields = computed(() => {
  const s = schemas.value
  if (!s || !record.value) return []
  const own = nodeAt(s, kind.value, '')
  const object = own && objectOf(s, own)
  const all = Object.keys((object?.node.properties as object | undefined) ?? {})
  const properties = (props.only ? props.only.filter((name) => all.includes(name)) : all).filter(
    (name) => !props.except?.includes(name),
  )
  return properties.map((name) => {
    const found = nodeAt(s, kind.value, name)!
    const value = record.value!.fields[name]
    return { name, label: fieldLabel(name) ?? name, value, drawn: drawnOf(s, found, value), held: !empty(value) }
  })
})
const held = computed(() => fields.value.filter((f) => f.held))
const absent = computed(() => fields.value.filter((f) => !f.held))
</script>

<template>
  <Stack v-if="record && schemas" gap="5">
    <Definitions v-if="held.length" :compact="compact">
      <Definition v-for="field in held" :key="field.name" :term="field.label" :path="field.name">
        <EditValue v-if="field.drawn === 'value'" :path="field.name" :label="field.label" />
        <template v-else-if="field.drawn === 'values'">
          <EditValue
            v-for="(_, i) in field.value as Held[]"
            :key="i"
            :path="`${field.name}[${i}]`"
            :label="field.label"
          />
          <EditValue :path="field.name" :label="field.label" list />
        </template>
        <template v-else-if="field.drawn === 'entries'">
          <EditEntry
            v-for="(entry, i) in field.value as Held[]"
            :key="i"
            v-slot="{ disabled, marked }"
            :path="`${field.name}[${i}]`"
            :label="field.label"
          >
            <Value :path="`${field.name}[${i}]`" :label="field.label" :disabled="disabled" :marked="marked">{{ shown(entry) }}</Value>
          </EditEntry>
          <EditEntry :path="field.name" :label="field.label" list />
        </template>
        <EditEntry v-else v-slot="{ disabled, marked }" :path="field.name" :label="field.label">
          <Value :path="field.name" :label="field.label" :disabled="disabled" :marked="marked">{{ shown(field.value) }}</Value>
        </EditEntry>
      </Definition>
    </Definitions>
    <Stack v-if="absent.length" row wrap gap="2">
      <template v-for="field in absent" :key="field.name">
        <EditValue v-if="field.drawn === 'value' || field.drawn === 'values'" :path="field.name" :label="field.label" :list="field.drawn === 'values'" />
        <EditEntry v-else-if="field.drawn === 'entries'" :path="field.name" :label="field.label" list />
        <EditEntry v-else v-slot="{ disabled }" :path="field.name" :label="field.label">
          <Button size="sm" variant="add" icon="plus" :data-path="field.name" :disabled="disabled">
            {{ field.label }}
          </Button>
        </EditEntry>
      </template>
    </Stack>
  </Stack>
</template>
