<script setup lang="ts">
import { computed, ref } from 'vue'
import { useCorpus } from '../api/corpus'
import { useClient } from '../api/use'
import KeepDraft from '../edit/KeepDraft.vue'
import { keeping as said } from '../labels'
import { Button, Notice, Stack, Text } from '../ui'

// What the tab is supposing: the edits and the records the draft holds, the commit it
// was begun at and how far the corpus has come since, what no longer carries, and the
// draft taken back an edit, thrown away, kept as a hypothesis or left for later.
const corpus = useCorpus()
const client = useClient()
const keeping = ref(false)
const busy = ref(false)

const draft = computed(() => corpus.draft.value)
const edits = computed(() => draft.value?.edits.length ?? 0)

async function run(work: () => Promise<void>) {
  if (busy.value) return
  busy.value = true
  try {
    await corpus.change(work)
  } catch (e) {
    corpus.notify(e instanceof Error ? e.message : String(e))
  } finally {
    busy.value = false
  }
}
const pop = () => run(async () => corpus.drafted(await client.popDraft()))
const discard = () => run(async () => {
  await client.discardDraft()
  corpus.drafted(null)
})
</script>

<template>
  <Stack gap="2" data-draft>
    <Stack row wrap gap="3" align="center">
      <Text strong tone="suppose">{{ said.supposing }}</Text>
      <Text data-edits>{{ said.edits(edits) }}</Text>
      <Text>{{ said.records(draft?.records.length ?? 0) }}</Text>
      <template v-if="draft">
        <Text tone="muted">{{ said.base }}</Text>
        <Text face="mono" size="sm">{{ draft.base.slice(0, 7) }}</Text>
        <Text v-if="draft.since" tone="muted" data-since>{{ said.since(draft.since) }}</Text>
      </template>
      <Stack row wrap gap="2">
        <Button size="sm" variant="ghost" icon="undo" :disabled="busy || !edits" @click="pop">{{ said.pop }}</Button>
        <Button size="sm" variant="ghost" :disabled="busy || !draft" @click="discard">{{ said.discard }}</Button>
        <Button size="sm" variant="ghost" @click="corpus.setSupposing(false)">{{ said.stop }}</Button>
        <Button size="sm" icon="flask" :disabled="busy || !edits || !!draft?.drifted.length" data-path="draft" @click="keeping = true">
          {{ said.create }}
        </Button>
      </Stack>
    </Stack>
    <Notice v-if="draft?.drifted.length" tone="danger" data-drifted>
      <Stack gap="1">
        <Text strong>{{ said.drifted }}</Text>
        <Text v-for="(reason, i) in draft.drifted" :key="i" size="sm">{{ reason }}</Text>
      </Stack>
    </Notice>
    <KeepDraft v-if="draft" v-model:open="keeping" :draft="draft" />
  </Stack>
</template>
