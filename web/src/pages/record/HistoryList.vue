<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Logged } from '../../api/client'
import { useCorpus } from '../../api/corpus'
import { useChange } from '../../edit/change'
import { edit as said, hypothesis } from '../../labels'
import { IconButton, Stack, Text } from '../../ui'

// The changes made to a record, or to the corpus where no record is named, the last
// first, the last of the whole corpus offering to be taken back. A hidden view has no
// history to show.
const props = defineProps<{ entity?: string; n?: number }>()
const corpus = useCorpus()
const { client, writes } = useChange()

const entries = ref<Logged[]>([])
const head = ref('')
let asked = 0
async function read() {
  const mine = ++asked
  if (corpus.hidden()) {
    entries.value = []
    return
  }
  try {
    const [own, last] = await Promise.all([client.log(props.entity, props.n), client.log(undefined, 1)])
    if (mine !== asked) return
    entries.value = own.entries
    head.value = last.entries[0]?.commit ?? ''
  } catch {
    if (mine !== asked) return
    entries.value = []
  }
}
watch(() => [props.entity, props.n, corpus.view.value], read, { immediate: true })
corpus.listen((event) => {
  if (event.type === 'moved') void read()
})

const undoable = computed(() => writes.value && !corpus.supposing.value)
</script>

<template>
  <Stack gap="3" as="ol">
    <Stack v-for="entry in entries" :key="entry.commit" row gap="2" align="start" justify="between" as="li" :data-commit="entry.commit">
      <Stack gap="0">
        <Text size="sm">{{ entry.subject }}</Text>
        <Stack row wrap gap="2">
          <Text tone="faint" size="xs">{{ entry.date.slice(0, 10) }}</Text>
          <Text tone="faint" size="xs" face="mono">{{ entry.commit.slice(0, 7) }}</Text>
          <Text tone="faint" size="xs">{{ entry.author }}</Text>
          <Text v-if="entry['proposed-by']" tone="faint" size="xs" data-proposed-by>
            {{ hypothesis.proposedBy(entry['proposed-by']) }}
          </Text>
        </Stack>
      </Stack>
      <IconButton
        v-if="undoable && entry.commit === head"
        icon="undo"
        :label="said.undo"
        @click="corpus.undoing.value = true"
      />
    </Stack>
  </Stack>
</template>
