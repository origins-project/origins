<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { Index } from '../api/client'
import { frame, kinds } from '../labels'
import { search } from '../search'
import { Palette, SearchButton, type FoundGroup } from '../ui'
import { icons } from './kinds'

// The search: the button in the bar, and the palette it, Ctrl K and / open.
const props = defineProps<{ index?: Index }>()
const router = useRouter()
const open = ref(false)
const query = ref('')

const groups = computed<FoundGroup<string>[]>(() =>
  props.index
    ? search(props.index, query.value).map(({ kind, hits }) => ({
        label: kinds[kind].many,
        found: hits.map((hit) => ({ key: hit.ref, icon: icons[kind], text: hit.text, sub: hit.sub, value: `/${hit.ref}` })),
      }))
    : [],
)

watch(open, (now) => {
  if (!now) query.value = ''
})

function typing(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    (target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName))
  )
}

function key(event: KeyboardEvent) {
  const k = event.key.toLowerCase()
  if (k === 'k' && (event.ctrlKey || event.metaKey) && !event.altKey && !event.shiftKey) {
    event.preventDefault()
    open.value = true
  } else if (k === '/' && !event.ctrlKey && !event.metaKey && !event.altKey && !typing(event.target)) {
    event.preventDefault()
    open.value = true
  }
}

onMounted(() => window.addEventListener('keydown', key))
onBeforeUnmount(() => window.removeEventListener('keydown', key))
</script>

<template>
  <SearchButton :label="frame.search" :keys="frame.searchKeys" @click="open = true" />
  <Palette
    v-model:open="open"
    v-model:query="query"
    :label="frame.search"
    :groups="groups"
    :keys="frame.keys"
    @choose="(to) => router.push(to)"
  />
</template>
