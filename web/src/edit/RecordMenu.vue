<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Kind } from '../api/client'
import { edit as said } from '../labels'
import { Button, Field, IconButton, Menu, MenuItem, MenuSeparator, Stack } from '../ui'
import ChangeDialog from './ChangeDialog.vue'
import RecordPicker from './RecordPicker.vue'
import { useChange, useEditing } from './change'

// What is done to a record as a whole, each from a dialog showing its dry run first.
const { record, kind, entity } = useEditing()
const { client, writes, offered } = useChange()

const slug = computed(() => entity.value.slice(entity.value.indexOf('/') + 1))
const own = computed(() => kind.value as Kind)
const doing = ref<'' | 'rename' | 'merge' | 'split' | 'delete' | 'accept' | 'reject'>('')
const open = (what: typeof doing.value) => computed({
  get: () => doing.value === what,
  set: (on: boolean) => (doing.value = on ? what : ''),
})
const renaming = open('rename')
const merging = open('merge')
const splitting = open('split')
const deleting = open('delete')
const accepting = open('accept')
const rejecting = open('reject')

const to = ref('')
const into = ref('')
const parts = ref<string[]>([])
// the record as the dialog was opened on it, which the change is pinned to
let seen = ''
watch(doing, (now) => {
  seen = record.value?.seen ?? ''
  to.value = slug.value
  into.value = ''
  parts.value = now === 'split' ? [slug.value, ''] : []
})

function change(edit: { op: string; [field: string]: unknown }) {
  return (dry: boolean, sources?: string[]) =>
    client.change({ edit, seen, sources, dry })
}
const settle = (how: 'accept' | 'reject') => (dry: boolean, sources?: string[]) =>
  client[how](slug.value, { seen, sources, dry })

const named = computed(() => parts.value.map((p) => p.trim()).filter(Boolean))
</script>

<template>
  <Menu v-if="offered && kind !== 'project'">
    <template #trigger>
      <IconButton icon="more" :label="said.actions" bordered :disabled="!writes" data-path="actions" />
    </template>
    <template v-if="kind === 'hypothesis'">
      <MenuItem icon="check" @select="doing = 'accept'">{{ said.accept }}</MenuItem>
      <MenuItem icon="x" @select="doing = 'reject'">{{ said.reject }}</MenuItem>
      <MenuSeparator />
    </template>
    <MenuItem @select="doing = 'rename'">{{ said.rename }}</MenuItem>
    <MenuItem @select="doing = 'merge'">{{ said.merge }}</MenuItem>
    <MenuItem @select="doing = 'split'">{{ said.split }}</MenuItem>
    <MenuSeparator />
    <MenuItem danger @select="doing = 'delete'">{{ said.delete }}</MenuItem>
  </Menu>

  <ChangeDialog
    v-model:open="renaming"
    :title="said.rename"
    :confirm="said.rename"
    :make="change({ op: 'rename', entity, to: to.trim() })"
    :ready="!!to.trim() && to.trim() !== slug"
    :asked="to"
    path="rename"
  >
    <Field v-model="to" :label="said.to" mono path="to" />
  </ChangeDialog>

  <ChangeDialog
    v-model:open="merging"
    :title="said.merge"
    :confirm="said.merge"
    :make="change({ op: 'merge', from: entity, into: `${kind}/${into}` })"
    :ready="!!into && into !== slug"
    :asked="into"
    path="merge"
  >
    <RecordPicker v-model="into" :kind="own" :label="said.into" path="into" />
  </ChangeDialog>

  <ChangeDialog
    v-model:open="splitting"
    :title="said.split"
    :confirm="said.split"
    :make="change({ op: 'split', entity, into: named.map((p) => `${kind}/${p}`) })"
    :ready="named.length >= 2"
    :asked="named.join(' ')"
    path="split"
  >
    <Stack gap="2">
      <Field
        v-for="(_, i) in parts"
        :key="i"
        :model-value="parts[i]"
        :label="`${said.into} ${i + 1}`"
        mono
        :path="`into[${i}]`"
        @update:model-value="(p) => (parts[i] = p)"
      />
      <Stack row>
        <Button size="sm" variant="add" icon="plus" @click="parts.push('')">{{ said.add }}</Button>
      </Stack>
    </Stack>
  </ChangeDialog>

  <ChangeDialog
    v-model:open="deleting"
    :title="said.delete"
    :confirm="said.delete"
    :make="change({ op: 'delete', entity })"
    danger
    path="delete"
  />

  <template v-if="kind === 'hypothesis'">
    <ChangeDialog v-model:open="accepting" :title="said.accept" :confirm="said.accept" :make="settle('accept')" />
    <ChangeDialog v-model:open="rejecting" :title="said.reject" :confirm="said.reject" :make="settle('reject')" />
  </template>
</template>
