<script setup lang="ts">
import { computed, toRef } from 'vue'
import type { Kind, Record } from '../api/client'
import { provideEditing } from './change'

// Another record changed from inside a page about somebody: a family's events on its
// partner's timeline. What it holds edits the record it is given rather than the page's.
const props = defineProps<{ record: Record }>()
const ref = computed(() => props.record.ref)
provideEditing({
  record: toRef(props, 'record'),
  kind: computed(() => ref.value.slice(0, ref.value.indexOf('/')) as Kind),
  entity: ref,
})
</script>

<template>
  <div :data-entity="ref"><slot /></div>
</template>
