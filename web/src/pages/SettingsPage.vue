<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Record, Remotes, Resolution } from '../api/client'
import { useClient } from '../api/use'
import { useCorpus } from '../api/corpus'
import ChangeDialog from '../edit/ChangeDialog.vue'
import EditEntry from '../edit/EditEntry.vue'
import EditValue from '../edit/EditValue.vue'
import Fields from '../edit/Fields.vue'
import { provideEditing, refusalOf, type Refusal } from '../edit/change'
import { edit, fieldLabel, frame, settings as said } from '../labels'
import { useNaming } from '../naming'
import { listed } from '../search'
import {
  Badge,
  Block,
  Button,
  Card,
  Chip,
  Code,
  Definition,
  Definitions,
  Field,
  Grid,
  IconButton,
  Notice,
  Page,
  Segmented,
  Stack,
  Table,
  Tag,
  Text,
  type Column,
} from '../ui'

// What the corpus says of itself and how it is shared: the manifest's fields, its tags
// renamed and styled, the branch and the remotes it is fetched from and pushed to, a
// history to recover, and origins.yaml as it is written. In an export it is Info: the
// same fields read only, and what the export was made from.
const client = useClient()
const corpus = useCorpus()
const { index } = corpus
const { called } = useNaming()

const project = ref<Record>()
provideEditing({ record: project, kind: computed(() => 'project'), entity: computed(() => 'project') })
const remotes = ref<Remotes>()
const resolution = ref<Resolution>()

async function read() {
  const [page, shared] = await Promise.allSettled([client.project(), corpus.history.value ? client.remotes() : undefined])
  project.value = page.status === 'fulfilled' ? page.value.record : undefined
  remotes.value = shared.status === 'fulfilled' ? shared.value : undefined
  if (corpus.writes()) {
    try {
      resolution.value = await client.recover()
    } catch {
      resolution.value = undefined
    }
  }
}
void read()
corpus.listen((event) => {
  if (event.type === 'moved') void read()
})

const writes = computed(() => corpus.writes())

// The tags, each with its style and how many records the index lists carrying it.
interface Styled {
  name: string
  color?: string
  priority?: string
  records: number
}
const tags = computed<Styled[]>(() => {
  const styles = (project.value?.fields.tags ?? {}) as { [tag: string]: { color?: string; priority?: unknown } | null }
  const used = new Map<string, number>()
  if (index.value)
    for (const list of Object.values(listed))
      for (const one of index.value[list]) for (const tag of (one.tags as string[] | undefined) ?? []) used.set(tag, (used.get(tag) ?? 0) + 1)
  const names = new Set([...Object.keys(styles), ...used.keys()])
  return [...names]
    .sort((a, b) => a.localeCompare(b))
    .map((name) => ({
      name,
      color: styles[name]?.color,
      priority: styles[name]?.priority === undefined ? undefined : String(styles[name]!.priority),
      records: used.get(name) ?? 0,
    }))
})
const columns = computed<Column<Styled>[]>(() => [
  { key: 'name', label: said.tag, sortable: true },
  { key: 'color', label: said.color, width: 'mid' },
  { key: 'priority', label: said.priority, width: 'mid' },
  { key: 'records', label: said.records, width: 'narrow', align: 'end', sortable: true },
  ...(corpus.offered.value ? [{ key: 'rename', label: said.rename, width: 'narrow', align: 'end' } as const] : []),
])
const styled = (tag: string) => tag in ((project.value?.fields.tags as object | undefined) ?? {})

const renaming = ref<string>()
const to = ref('')
watch(renaming, (tag) => (to.value = tag ?? ''))
const renamingOpen = computed({
  get: () => renaming.value !== undefined,
  set: (on: boolean) => {
    if (!on) renaming.value = undefined
  },
})
const rename = (dry: boolean, sources?: string[]) =>
  client.change({ edit: { op: 'rename', entity: `tag/${renaming.value}`, to: to.value.trim() }, seen: '', sources, dry })

// Sharing: each command run from its button, what it said shown as a notice and a
// refusal kept beside the buttons until the next one.
const busy = ref('')
const refusal = ref<Refusal>()
async function run(what: string, work: () => Promise<string | undefined>) {
  busy.value = what
  refusal.value = undefined
  try {
    const said = await corpus.change(work)
    if (said) corpus.notify(said)
    await read()
  } catch (e) {
    refusal.value = refusalOf(e)
  } finally {
    busy.value = ''
  }
}
const fetch = () => run('fetch', async () => said.fetched((await client.fetchRemote()).fetched))
const pull = () =>
  run('pull', async () => {
    const taken = await client.pull()
    if (taken.commit) corpus.made(taken.commit)
    return said.pulled(taken.commits, taken.upstream || taken.remote)
  })
const push = () =>
  run('push', async () => {
    const sent = await client.push()
    return said.pushed(sent.branch, sent.remote)
  })
const remoteName = ref('')
const remoteUrl = ref('')
const addRemote = () =>
  run('add', async () => {
    const name = remoteName.value.trim()
    await client.addRemote({ name, url: remoteUrl.value.trim() })
    remoteName.value = remoteUrl.value = ''
    return said.added(name)
  })
const removeRemote = (name: string) =>
  run(`remove:${name}`, async () => {
    await client.removeRemote(name)
    return said.removed(name)
  })

const ways = ['--adopt', '--rewrite', '--reset']
const way = ref('')
watch(resolution, (r) => (way.value = r?.advises || ways[0]!))
const wayOptions = computed(() =>
  ways.map((w) => ({ value: w, label: w === resolution.value?.advises ? `${said.ways[w]} · ${said.recommended}` : said.ways[w]! })),
)
// What an export was made from, its time as the clock of the machine that made it read.
const made = computed(() => corpus.session.value?.export ?? undefined)
const madeAt = computed(() => made.value?.at.slice(0, 16).replace('T', ' '))
const madeAs = computed(() => {
  const view = corpus.session.value?.view
  const hidden = [view?.private && frame.privateHidden, view?.living && frame.livingHidden].filter((v) => !!v)
  return hidden.length ? hidden.join(', ') : said.whole
})
const madeWith = computed(() => {
  const held = new Set(index.value?.hypotheses.map((h) => h.ref))
  return (corpus.session.value?.with ?? []).map(({ hypothesis: slug }) => ({
    slug,
    question: called('hypothesis', slug),
    to: held.has(`hypothesis/${slug}`) ? `/hypothesis/${slug}` : undefined,
  }))
})

const recover = () =>
  run('recover', async () => {
    const r = await client.recover(way.value)
    if (r.commit) corpus.made(r.commit)
    return r.commit ? said.recovered(r.commit.slice(0, 7)) : undefined
  })
</script>

<template>
  <Page :title="corpus.offered.value ? frame.settings : frame.info">
    <Stack v-if="project" gap="8">
      <Block v-if="made" :title="said.export">
        <Card>
          <Definitions data-path="export">
            <Definition :term="said.made" path="export.at"><Text>{{ madeAt }}</Text></Definition>
            <Definition v-if="made.commit" :term="said.commit" path="export.commit">
              <Text face="mono">{{ made.commit.slice(0, 7) }}</Text>
              <Badge v-if="made.changed" tone="warn">{{ said.changed }}</Badge>
            </Definition>
            <Definition :term="said.version" path="export.version"><Text face="mono">{{ made.version }}</Text></Definition>
            <Definition :term="said.view" path="export.view"><Text>{{ madeAs }}</Text></Definition>
            <Definition v-if="madeWith.length" :term="said.supposed" path="export.with">
              <template v-for="one in madeWith" :key="one.slug">
                <Chip v-if="one.to" tone="suppose" icon="flask" :to="one.to">{{ one.question }}</Chip>
                <Text v-else>{{ one.question }}</Text>
              </template>
            </Definition>
            <Definition v-if="made.notice" :term="said.notice" path="export.notice"><Text>{{ made.notice }}</Text></Definition>
          </Definitions>
        </Card>
      </Block>

      <Block :title="said.project">
        <Card><Fields :only="['name', 'description', 'main-person', 'lang', 'copyright']" /></Card>
      </Block>

      <Block :title="said.researchers">
        <Card><Fields :only="['researchers']" /></Card>
      </Block>

      <Block :title="said.bookmarks">
        <Card><Fields :only="['bookmarks']" /></Card>
      </Block>

      <Block :title="said.tags">
        <template #actions><EditEntry path="tags" :label="fieldLabel('tags')" /></template>
        <Card flush>
          <Table :columns="columns" :rows="tags" :row-key="(t) => t.name" :label="said.tags">
            <template #cell-name="{ row }"><Tag :name="row.name" :color="row.color" /></template>
            <template #cell-color="{ row }">
              <EditValue v-if="styled(row.name)" :path="['tags', row.name, 'color']" :label="said.color" />
            </template>
            <template #cell-priority="{ row }">
              <EditValue v-if="styled(row.name)" :path="['tags', row.name, 'priority']" :label="said.priority" />
            </template>
            <template #cell-rename="{ row }">
              <IconButton icon="pen" :label="`${said.rename} ${row.name}`" :disabled="!writes" :data-tag="row.name" @click="renaming = row.name" />
            </template>
          </Table>
        </Card>
      </Block>

      <Block v-if="corpus.history.value" :title="said.git">
        <template #actions>
          <Stack v-if="remotes" row gap="2" align="baseline">
            <Text tone="faint" size="sm">{{ said.branch }}</Text>
            <Text face="mono" size="sm" data-path="branch">{{ remotes.branch }}</Text>
          </Stack>
        </template>
        <Stack gap="4">
          <Notice v-if="refusal" tone="danger" :data-refused="refusal.kind">{{ refusal.message }}</Notice>
          <Card :title="said.remotes">
            <Stack gap="4">
              <Stack v-for="r in remotes?.remotes ?? []" :key="r.name" row gap="3" align="center" justify="between" :data-remote="r.name">
                <Stack gap="0">
                  <Text strong>{{ r.name }}</Text>
                  <Text face="mono" size="sm" tone="muted">{{ r.url }}</Text>
                </Stack>
                <Button size="sm" variant="ghost" :disabled="!writes || !!busy" @click="removeRemote(r.name)">{{ said.remove }}</Button>
              </Stack>
              <Stack row wrap gap="3" align="end">
                <Field v-model="remoteName" :label="said.name" mono path="remote.name" />
                <Field v-model="remoteUrl" :label="said.url" mono path="remote.url" />
                <Button :disabled="!writes || !!busy || !remoteName.trim() || !remoteUrl.trim()" icon="plus" @click="addRemote">
                  {{ said.add }}
                </Button>
              </Stack>
            </Stack>
          </Card>
          <Grid v-if="remotes?.remotes.length" fit="card" gap="4">
            <Card :title="remotes.upstream ? `${said.incoming} · ${remotes.upstream}` : said.incoming" data-path="incoming">
              <template #action>
                <Button size="sm" variant="ghost" :disabled="!!busy" @click="fetch">{{ said.fetch }}</Button>
              </template>
              <Stack gap="3">
                <Text face="serif" size="xl">{{ said.commits(remotes.behind) }}</Text>
                <Stack row><Button :disabled="!writes || !!busy" @click="pull">{{ said.pull }}</Button></Stack>
              </Stack>
            </Card>
            <Card :title="remotes.upstream ? `${said.outgoing} · ${remotes.upstream}` : said.outgoing" data-path="outgoing">
              <Stack gap="3">
                <Text face="serif" size="xl">{{ said.commits(remotes.ahead) }}</Text>
                <Stack row><Button variant="primary" :disabled="!writes || !!busy" @click="push">{{ said.push }}</Button></Stack>
              </Stack>
            </Card>
          </Grid>
          <Card v-if="resolution && !resolution.settled" :title="said.recover" tone="warn" data-path="recover">
            <Stack gap="3">
              <Stack gap="2" as="ul">
                <Stack v-for="c in resolution.foreign ?? []" :key="c.commit" row gap="2" align="baseline" as="li">
                  <Text face="mono" size="xs" tone="faint">{{ c.commit.slice(0, 7) }}</Text>
                  <Text size="sm">{{ c.subject }}</Text>
                </Stack>
              </Stack>
              <Text v-for="file in resolution.dirty ?? []" :key="file" face="mono" size="sm">{{ file }}</Text>
              <Segmented v-model="way" :options="wayOptions" :label="said.recover" path="way" />
              <Stack row>
                <Button variant="primary" :disabled="!writes || !!busy" @click="recover">{{ said.ways[way] }}</Button>
              </Stack>
            </Stack>
          </Card>
        </Stack>
      </Block>

      <Block v-if="corpus.offered.value" :title="said.manifest">
        <Code v-if="project.written" :label="said.manifest">{{ project.written }}</Code>
      </Block>
    </Stack>

    <ChangeDialog
      v-model:open="renamingOpen"
      :title="said.rename"
      :confirm="said.rename"
      :make="rename"
      :ready="!!to.trim() && to.trim() !== renaming"
      :asked="to"
      path="rename"
    >
      <Field v-model="to" :label="edit.to" path="to" />
    </ChangeDialog>
  </Page>
</template>
