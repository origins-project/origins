<script setup lang="ts">
import { computed } from 'vue'
import { useCorpus } from '../api/corpus'
import { frame as said } from '../labels'
import { useNaming } from '../naming'
import { Button, Chip, Stack, Text } from '../ui'

// The hypotheses the pages are read with supposed true: those the server was started
// with, which stay, and those the tab supposes, each of which it can stop supposing.
// A hypothesis is a link to its page where the view holds one.
const corpus = useCorpus()
const { called, index } = useNaming()

const shown = computed(() => {
  const locked = (corpus.session.value?.with ?? []).map((s) => s.hypothesis)
  const own = corpus.supposed.value.filter((slug) => !locked.includes(slug))
  const held = new Set(index.value?.hypotheses.map((h) => h.ref))
  return [...locked.map((slug) => ({ slug, locked: true })), ...own.map((slug) => ({ slug, locked: false }))].map((one) => ({
    ...one,
    question: called('hypothesis', one.slug),
    to: held.has(`hypothesis/${one.slug}`) ? `/hypothesis/${one.slug}` : undefined,
  }))
})

const stop = (slug: string) => corpus.setSupposed(corpus.supposed.value.filter((s) => s !== slug))
</script>

<template>
  <Stack row wrap gap="3" align="center" data-supposed>
    <Text strong tone="suppose">{{ said.supposed }}</Text>
    <Stack v-for="one in shown" :key="one.slug" row gap="1" align="center" :data-hypothesis="one.slug">
      <Chip v-if="one.to" tone="suppose" icon="flask" :to="one.to">{{ one.question }}</Chip>
      <Text v-else>{{ one.question }}</Text>
      <Button v-if="!one.locked" size="sm" variant="ghost" @click="stop(one.slug)">{{ said.stopSupposing }}</Button>
    </Stack>
  </Stack>
</template>
