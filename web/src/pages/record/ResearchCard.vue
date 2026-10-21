<script setup lang="ts">
import { computed } from 'vue'
import { useCorpus } from '../../api/corpus'
import { record as said } from '../../labels'
import { Badge, Card, Stack, Text } from '../../ui'

// To research: what `check` and the research rules found about the record, the tasks
// its prose leaves unticked among them.
const props = defineProps<{ entity: string }>()
const { findings } = useCorpus()

const found = computed(() => {
  const all = [...(findings.value?.findings ?? []), ...(findings.value?.todo ?? [])].filter((f) => f.entity === props.entity)
  const seen = new Set<string>()
  return all.filter((f) => {
    const key = [f.rule, f.path, f.message].join('\u0000')
    if (seen.has(key)) return false
    seen.add(key)
    return true
  })
})
const tone = (severity: string) => (severity === 'error' ? 'danger' : severity === 'warning' ? 'warn' : 'neutral')
</script>

<template>
  <Card v-if="found.length" :title="said.research" tone="research">
    <template #action><Badge>{{ found.length }}</Badge></template>
    <Stack gap="3" as="ul">
      <Stack v-for="(f, i) in found" :key="i" gap="1" as="li" :data-finding="f.rule">
        <Text>{{ f.message }}</Text>
        <Stack row wrap gap="2" align="center">
          <Badge mono :tone="f.rule === 'task' ? 'accent' : tone(f.severity)">{{ f.rule }}</Badge>
          <Text v-if="f.path && f.rule !== 'task'" face="mono" size="xs" tone="faint">{{ f.path }}</Text>
        </Stack>
      </Stack>
    </Stack>
  </Card>
</template>
