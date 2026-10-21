<script setup lang="ts">
import { computed, ref } from 'vue'
import { kindsInOrder, type Kind } from '../api/client'
import { useCorpus } from '../api/corpus'
import { findings as said, frame, kinds } from '../labels'
import { counted, grouped, type Rule } from '../lists'
import { useNaming } from '../naming'
import { Badge, Button, Card, Chip, Field, Grid, Page, Segmented, Stack, Text } from '../ui'

// Everything check found and research leaves to do, by severity and then by rule, each
// leading to the record it is about.
const { findings, offered } = useCorpus()
const { called } = useNaming()

// How many of a rule a card shows until asked for the rest.
const first = 50
const opened = ref(new Set<string>())
const query = ref('')
const severity = ref('all')
const groups = computed(() => grouped(findings.value ?? { findings: [], todo: [] }, query.value))
const sections = computed(() =>
  (
    [
      { key: 'errors', title: said.errors, tone: 'danger' },
      { key: 'warnings', title: said.warnings, tone: 'warn' },
      { key: 'todo', title: said.todo, tone: 'accent' },
    ] as const
  ).map((s) => ({ ...s, rules: groups.value[s.key] as Rule[] })),
)
const options = computed(() => [
  { value: 'all', label: said.all },
  ...sections.value.map((s) => ({ value: s.key, label: `${s.title} ${counted(s.rules)}` })),
])
const shown = computed(() =>
  sections.value.filter((s) => (severity.value === 'all' || severity.value === s.key) && s.rules.length),
)

function kindOf(entity: string): Kind | undefined {
  const kind = entity.slice(0, entity.indexOf('/')) as Kind
  return kindsInOrder.includes(kind) ? kind : undefined
}
const slugOf = (entity: string) => entity.slice(entity.indexOf('/') + 1)
// a finding about the project is about what Settings edits, or what Info shows of it
const linkOf = (entity: string) => (kindOf(entity) ? `/${entity}` : offered.value ? '/settings' : '/info')
const nameOf = (entity: string) => {
  const kind = kindOf(entity)
  return kind ? called(kind, slugOf(entity)) : offered.value ? frame.settings : frame.info
}
</script>

<template>
  <Page :title="frame.findings">
    <Stack gap="5">
      <Stack row wrap gap="3" align="end">
        <Field v-model="query" type="search" :label="said.filter" path="filter" />
        <Segmented v-model="severity" :options="options" :label="frame.findings" path="severity" />
      </Stack>
      <Text v-if="!shown.length" tone="muted">{{ said.nothing }}</Text>
      <Stack v-for="section in shown" :key="section.key" gap="3" :data-severity="section.key">
        <Stack row gap="2" align="center">
          <Text as="h2" size="lg" strong>{{ section.title }}</Text>
          <Badge :tone="section.tone">{{ counted(section.rules) }}</Badge>
        </Stack>
        <Grid fit="wide" gap="4" packed>
          <Card v-for="rule in section.rules" :key="rule.rule" :title="rule.rule" :data-rule="rule.rule">
            <template #action><Badge>{{ rule.findings.length }}</Badge></template>
            <Stack gap="3" as="ul">
              <Stack v-for="(f, i) in opened.has(rule.rule) ? rule.findings : rule.findings.slice(0, first)" :key="i" gap="1" as="li" :data-finding="f.entity">
                <Stack row wrap gap="2" align="center">
                  <Chip :kind="kindOf(f.entity) ? kinds[kindOf(f.entity)!].one : undefined" :to="linkOf(f.entity)">
                    {{ nameOf(f.entity) }}
                  </Chip>
                  <Text v-if="f.path && f.rule !== 'task'" face="mono" size="xs" tone="faint">{{ f.path }}</Text>
                </Stack>
                <Text size="md">{{ f.message }}</Text>
              </Stack>
            </Stack>
            <template v-if="rule.findings.length > first && !opened.has(rule.rule)" #footer>
              <Button size="sm" variant="ghost" @click="opened = new Set([...opened, rule.rule])">
                {{ said.more(rule.findings.length - first) }}
              </Button>
            </template>
          </Card>
        </Grid>
      </Stack>
    </Stack>
  </Page>
</template>
