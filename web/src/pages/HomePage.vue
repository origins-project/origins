<script setup lang="ts">
import { computed, ref } from 'vue'
import { kindsInOrder, type Drafting, type Record } from '../api/client'
import { useClient } from '../api/use'
import { useCorpus } from '../api/corpus'
import { provideEditing } from '../edit/change'
import EditValue from '../edit/EditValue.vue'
import NewRecord from '../edit/NewRecord.vue'
import { icons } from '../frame/kinds'
import { app, create, fieldLabel, findings as found, frame, home as said, hypothesis, kinds } from '../labels'
import { counted, grouped, surnames } from '../lists'
import { useNaming } from '../naming'
import { listed } from '../search'
import {
  Badge,
  Block,
  Button,
  Card,
  Chip,
  Columns,
  Grid,
  Hero,
  Notice,
  Page,
  PersonCard,
  Stack,
  Stat,
  Text,
} from '../ui'
import HistoryList from './record/HistoryList.vue'

// The project: what it is called and who keeps it, a draft waiting, its records
// counted, the person it starts from, what changed last, its surnames, and beside them
// the bookmarks, the findings counted and the hypotheses still open.
const client = useClient()
const corpus = useCorpus()
const { session, index, findings } = corpus
const { called, card } = useNaming()

const project = ref<Record>()
const draft = ref<Drafting>()
provideEditing({ record: project, kind: computed(() => 'project'), entity: computed(() => 'project') })

async function read() {
  const [page, drafting] = await Promise.allSettled([client.project(), corpus.writes() ? client.draft() : null])
  project.value = page.status === 'fulfilled' ? page.value.record : undefined
  draft.value = (drafting.status === 'fulfilled' && drafting.value) || undefined
}
void read()
corpus.listen((event) => {
  if (event.type === 'moved' || event.type === 'draft') void read()
})

const fields = computed(() => project.value?.fields ?? {})
const text = (value: unknown) => (typeof value === 'string' ? value : '')
const researchers = computed(() =>
  ((fields.value.researchers as { name?: string }[] | undefined) ?? []).map((r) => text(r.name)).filter(Boolean),
)
const main = computed(() => text(index.value?.project['main-person']))
const names = computed(() => surnames(index.value?.people ?? []).slice(0, 40))

const bookmarks = computed(() =>
  ((index.value?.project.bookmarks as string[] | undefined) ?? []).map((mark) => {
    const [first = '', slug = ''] = mark.split('/')
    const kind = kindsInOrder.find((k) => k === first)
    return { mark, kind, slug, to: kind ? `/${kind}/${slug}` : undefined }
  }),
)
const counts = computed(() => {
  const g = grouped(findings.value ?? { findings: [], todo: [] })
  return [
    { key: 'errors', label: found.errors, tone: 'danger', n: counted(g.errors) },
    { key: 'warnings', label: found.warnings, tone: 'warn', n: counted(g.warnings) },
    { key: 'todo', label: found.todo, tone: 'accent', n: counted(g.todo) },
  ] as const
})
const open = computed(() => (index.value?.hypotheses ?? []).filter((h) => h.status === 'open'))

const creating = ref(false)
const discarding = ref(false)
async function discard() {
  discarding.value = true
  try {
    await corpus.change(() => client.discardDraft())
    draft.value = undefined
  } catch (e) {
    corpus.notify(e instanceof Error ? e.message : String(e))
  } finally {
    discarding.value = false
  }
}
</script>

<template>
  <Page>
    <Hero>
      <template #title>
        <EditValue v-if="project" path="name" :label="fieldLabel('name')">{{ fields.name }}</EditValue>
        <template v-else>{{ session?.project || app.title }}</template>
      </template>
      <Stack v-if="project" gap="2">
        <EditValue path="description" :label="fieldLabel('description')">
          <Text tone="muted" size="lg">{{ fields.description }}</Text>
        </EditValue>
        <Stack row wrap gap="3">
          <Text v-if="researchers.length" tone="faint" size="sm">{{ researchers.join(', ') }}</Text>
          <EditValue path="copyright" :label="fieldLabel('copyright')">
            <Text tone="faint" size="sm">{{ fields.copyright }}</Text>
          </EditValue>
        </Stack>
      </Stack>
      <template #actions>
        <Button v-if="corpus.offered.value" icon="gear" to="/settings">{{ frame.settings }}</Button>
        <Button v-else icon="archive" to="/info">{{ frame.info }}</Button>
        <Button v-if="corpus.offered.value" variant="primary" icon="plus" :disabled="!corpus.writes()" data-path="create" @click="creating = true">
          {{ create.new(kinds.person.one) }}
        </Button>
      </template>
    </Hero>

    <Stack gap="6">
      <Notice v-if="draft && !corpus.supposing.value" tone="suppose" icon="flask" data-path="draft">
        <Stack row wrap gap="2" align="baseline">
          <Text strong>{{ said.draft }}</Text>
          <Text>{{ said.edits(draft.edits.length) }}</Text>
          <Text>{{ said.records(draft.records.length) }}</Text>
          <Text tone="muted">{{ said.base }}</Text>
          <Text face="mono" size="sm">{{ draft.base.slice(0, 7) }}</Text>
        </Stack>
        <template #actions>
          <Button size="sm" variant="ghost" :disabled="discarding" @click="discard">{{ said.discard }}</Button>
          <Button size="sm" icon="flask" @click="corpus.setSupposing(true)">{{ said.resume }}</Button>
        </template>
      </Notice>

      <Grid v-if="index" fit="stat" gap="3">
        <Stat
          v-for="kind in kindsInOrder"
          :key="kind"
          :count="index[listed[kind]].length"
          :label="kinds[kind].many"
          :icon="icons[kind]"
          :to="`/${listed[kind]}`"
        />
      </Grid>

      <Columns>
        <Stack gap="6">
          <Block :title="said.mainPerson">
            <template #actions>
              <EditValue v-if="project && corpus.offered.value" path="main-person" :label="said.mainPerson">{{ said.change }}</EditValue>
            </template>
            <PersonCard v-if="main" v-bind="card(main)" />
          </Block>

          <Block v-if="corpus.history.value" :title="said.recent">
            <Card><HistoryList :n="5" /></Card>
          </Block>

          <Block v-if="names.length" :title="said.surnames">
            <Stack row wrap gap="2">
              <Chip v-for="one in names" :key="one.surname" :to="{ path: '/people', query: { q: one.surname } }">
                {{ one.surname }} <Text tone="faint" size="xs">{{ one.count }}</Text>
              </Chip>
            </Stack>
          </Block>
        </Stack>

        <template #side>
          <Card :title="said.bookmarks">
            <template #action><EditValue v-if="project" path="bookmarks" list :label="fieldLabel('bookmarks')" /></template>
            <Stack gap="2" as="ul">
              <Stack v-for="b in bookmarks" :key="b.mark" as="li">
                <Chip v-if="b.kind" :kind="kinds[b.kind].one" :to="b.to">{{ called(b.kind, b.slug) }}</Chip>
                <Text v-else face="mono" size="sm">{{ b.mark }}</Text>
              </Stack>
            </Stack>
          </Card>

          <Card :title="said.findings">
            <template #action><Button size="sm" variant="ghost" to="/findings">{{ said.open }}</Button></template>
            <Stack gap="2">
              <Stack v-for="c in counts" :key="c.key" row gap="2" align="center" :data-severity="c.key">
                <Badge :tone="c.n ? c.tone : 'faint'">{{ c.n }}</Badge>
                <Text size="md">{{ c.label }}</Text>
              </Stack>
            </Stack>
          </Card>

          <Card v-if="open.length" :title="said.hypotheses">
            <Stack gap="3" as="ul">
              <Stack v-for="h in open" :key="h.ref" gap="1" as="li">
                <Chip :to="`/${h.ref}`">{{ called('hypothesis', h.ref.slice(h.ref.indexOf('/') + 1)) }}</Chip>
                <Stack row gap="2">
                  <Text v-if="h.confidence" tone="faint" size="xs">{{ h.confidence }}</Text>
                  <Text v-if="h['proposed-by']" tone="faint" size="xs" data-proposed-by>
                    {{ hypothesis.proposedBy(String(h['proposed-by'])) }}
                  </Text>
                  <Text v-if="h.base" tone="faint" size="xs" face="mono">{{ String(h.base).slice(0, 7) }}</Text>
                </Stack>
              </Stack>
            </Stack>
          </Card>
        </template>
      </Columns>
    </Stack>

    <NewRecord v-model:open="creating" kind="person" />
  </Page>
</template>
