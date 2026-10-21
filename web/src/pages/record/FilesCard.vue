<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import EditEntry from '../../edit/EditEntry.vue'
import { fieldLabel, record as said } from '../../labels'
import { Card, Stack } from '../../ui'
import PictureView from './PictureView.vue'

// Files: the record's pictures, the files beside it that nothing in it names, and, for
// a person, the pictures of them that other records keep, each opening whole.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
type Entry = { file: string }
const own = computed(() => {
  const list = props.record.fields.media
  return Array.isArray(list) ? list.map((m: unknown) => (typeof m === 'string' ? m : text((m as Entry).file))) : []
})
// Every file the record names as a picture, its own or an event's or a citation's.
function named(value: unknown, into: Set<string>, under = ''): Set<string> {
  if (Array.isArray(value)) {
    for (const v of value) {
      if (under === 'media' || under === 'images') into.add(typeof v === 'string' ? v : text((v as Entry | undefined)?.file))
      else named(v, into)
    }
  } else if (value !== null && typeof value === 'object') {
    for (const [key, v] of Object.entries(value)) named(v, into, key)
  }
  return into
}
const loose = computed(() => {
  const all = named(props.record.fields, new Set())
  return props.record.attached.filter((file) => !all.has(file))
})
const theirs = computed(() =>
  !props.record.ref.startsWith('person/') ? [] : props.around.flatMap((other) => {
    const list = other.fields.media
    if (!Array.isArray(list)) return []
    return list.flatMap((m: unknown) => {
      const depicts = (m as { depicts?: unknown }).depicts
      if (!Array.isArray(depicts)) return []
      const shows = depicts.some((d) => d === props.slug || (d as { person?: unknown })?.person === props.slug)
      const file = text((m as Entry).file)
      return shows ? [{ owner: other.ref, file }] : []
    })
  }),
)
</script>

<template>
  <Card :title="said.files">
    <template #action><EditEntry path="media" :label="fieldLabel('media')" list /></template>
    <Stack row wrap gap="2">
      <PictureView v-for="(file, i) in own" :key="`media[${i}]`" :owner="record.ref" :file="file" :path="`media[${i}]`" />
      <PictureView v-for="file in loose" :key="file" :owner="record.ref" :file="file" />
      <PictureView v-for="t in theirs" :key="`${t.owner}/${t.file}`" :owner="t.owner" :file="t.file" />
    </Stack>
  </Card>
</template>
