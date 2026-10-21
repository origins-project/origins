<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import type { Indexed, Kind } from '../api/client'
import { useCorpus } from '../api/corpus'
import { spoken } from '../date'
import NewRecord from '../edit/NewRecord.vue'
import { create, frame, kinds, lists as said } from '../labels'
import { answers, placeTree, type Branch, type Within } from '../lists'
import { useNaming } from '../naming'
import type { IndexedPerson } from '../person'
import { listed, names } from '../search'
import { Badge, Button, Card, Chip, Field, Page, Segmented, Stack, Table, Tag, Text, Tree, type Column, type Sort } from '../ui'

// Every record of a kind, from the index: a column for each thing that tells them
// apart, filtered by any word of any of them and sorted by any column.
const props = defineProps<{ kind: Kind }>()
const corpus = useCorpus()
const { index } = corpus
const { called } = useNaming()
const route = useRoute()
const router = useRouter()

const query = ref(typeof route.query.q === 'string' ? route.query.q : '')
watch(query, (q) => void router.replace({ query: q ? { q } : {} }))
const shape = ref('all')
const sort = ref<Sort>()
const creating = ref(false)
watch(
  () => props.kind,
  () => {
    query.value = typeof route.query.q === 'string' ? route.query.q : ''
    shape.value = props.kind === 'place' ? 'list' : 'all'
    sort.value = undefined
  },
  { immediate: true },
)

const text = (value: unknown) => (typeof value === 'string' ? value : '')
const slugOf = (ref: string) => ref.slice(ref.indexOf('/') + 1)
const people = computed(() => new Map(index.value?.people.map((p) => [p.ref, p]) ?? []))
const nameOf = (row: Indexed) => names(props.kind, row, people.value).name
const dated = (d: unknown) => spoken(text((d as { date?: unknown } | undefined)?.date))
const sortOf = (d: unknown) => text((d as { sort?: unknown } | undefined)?.sort) || undefined
const placeOf = (d: unknown) => {
  const place = text((d as { place?: unknown } | undefined)?.place)
  return place ? called('place', place) : ''
}
const count = (list: unknown) => (Array.isArray(list) ? list.length : 0)
const tagsOf = (row: Indexed) => (Array.isArray(row.tags) ? (row.tags as string[]) : [])
const styles = computed(() => (index.value?.project.tags ?? {}) as { [tag: string]: { color?: string } })

const columns = computed<Column<Indexed>[]>(() => {
  const c = said.columns
  const name: Column<Indexed> = { key: 'name', label: c.name, value: nameOf, sortable: true }
  const tags: Column<Indexed> = { key: 'tags', label: c.tags, value: (r) => tagsOf(r).join(' '), width: 'mid', sortable: true }
  const type: Column<Indexed> = { key: 'type', label: c.type, value: (r) => text(r.type), width: 'mid', sortable: true }
  switch (props.kind) {
    case 'person':
      return [
        name,
        { key: 'born', label: c.born, value: (r) => dated(r.birth), order: (r) => sortOf(r.birth), width: 'mid', sortable: true },
        { key: 'place', label: c.place, value: (r) => placeOf(r.birth), width: 'mid', sortable: true },
        { key: 'died', label: c.died, value: (r) => dated(r.death), order: (r) => sortOf(r.death), width: 'mid', sortable: true },
        tags,
      ]
    case 'family':
      return [
        { ...name, label: c.partners },
        type,
        { key: 'first', label: c.first, value: (r) => dated(r.first), order: (r) => sortOf(r.first), width: 'mid', sortable: true },
        { key: 'children', label: c.children, value: (r) => count(r.children), width: 'narrow', align: 'end', sortable: true },
        tags,
      ]
    case 'place':
      return [
        name,
        type,
        { key: 'within', label: c.within, value: (r) => withinOf(r).map((w) => called('place', w.place)).join(', '), sortable: true },
        { key: 'references', label: c.references, value: (r) => Number(r.references ?? 0), width: 'narrow', align: 'end', sortable: true },
        tags,
      ]
    case 'source':
      return [
        { ...name, label: c.title },
        { key: 'author', label: c.author, value: (r) => text(r.author), width: 'mid', sortable: true },
        type,
        { key: 'kept', label: c.kept, value: (r) => keptOf(r).join(', '), sortable: true },
        { key: 'cited', label: c.cited, value: (r) => Number(r.citations ?? 0), width: 'narrow', align: 'end', sortable: true },
      ]
    case 'repository':
      return [
        name,
        type,
        { key: 'sources', label: c.sources, value: (r) => count(r.sources), width: 'narrow', align: 'end', sortable: true },
        tags,
      ]
    case 'note':
      return [
        { ...name, label: c.title },
        { key: 'events', label: c.events, value: (r) => count(r.events), width: 'narrow', align: 'end', sortable: true },
        tags,
      ]
    case 'hypothesis':
      return [
        { ...name, label: c.question },
        { key: 'status', label: c.status, value: (r) => text(r.status), width: 'mid', sortable: true },
        { key: 'confidence', label: c.confidence, value: (r) => text(r.confidence), width: 'mid', sortable: true },
        { key: 'proposed-by', label: c.proposedBy, value: (r) => text(r['proposed-by']), width: 'mid', sortable: true },
        { key: 'base', label: c.base, value: (r) => text(r.base).slice(0, 7), width: 'narrow', mono: true },
      ]
  }
  return [name]
})

const withinOf = (row: Indexed) => (row.within as Within[] | undefined) ?? []
const keptOf = (row: Indexed) =>
  ((row.kept as { repository: string }[] | undefined) ?? []).map((k) => called('repository', k.repository))

const options = computed(() => {
  switch (props.kind) {
    case 'person':
      return [
        { value: 'all', label: said.all },
        { value: 'living', label: said.livingOnly },
        { value: 'private', label: said.privateOnly },
      ]
    case 'place':
      return [
        { value: 'list', label: said.list },
        { value: 'tree', label: said.tree },
      ]
    case 'hypothesis':
      return [
        { value: 'all', label: said.all },
        { value: 'open', label: said.open },
      ]
  }
  return []
})

const all = computed<Indexed[]>(() => index.value?.[listed[props.kind]] ?? [])
const rows = computed(() =>
  all.value.filter((row) => {
    if (shape.value === 'living' && !(row as IndexedPerson).living) return false
    if (shape.value === 'private' && !row.private) return false
    if (shape.value === 'open' && row.status !== 'open') return false
    const { name, others } = names(props.kind, row, people.value)
    return answers(query.value, [name, others, slugOf(row.ref), columns.value.map((c) => String(c.value?.(row) ?? ''))])
  }),
)
const tree = computed(() => {
  const shown = new Set(rows.value.map((r) => r.ref))
  const prune = (branches: Branch[]): Branch[] =>
    branches.flatMap((b) => {
      const under = prune(b.under)
      return shown.has(b.place.ref) || under.length ? [{ ...b, under }] : []
    })
  return prune(placeTree(all.value))
})

const open = (row: Indexed) => void router.push(`/${row.ref}`)
</script>

<template>
  <Page :title="kinds[kind].many">
    <template v-if="corpus.offered.value" #actions>
      <Button v-if="kind === 'hypothesis'" icon="flask" :disabled="!corpus.writes()" @click="corpus.setSupposing(true)">
        {{ frame.suppose }}
      </Button>
      <Button v-else variant="primary" icon="plus" :disabled="!corpus.writes()" data-path="create" @click="creating = true">
        {{ create.new(kinds[kind].one) }}
      </Button>
    </template>
    <Stack gap="4">
      <Stack row wrap gap="3" align="end">
        <Field v-model="query" type="search" :label="said.filter" path="filter" />
        <Segmented v-if="options.length" v-model="shape" :options="options" :label="said.filter" path="shape" />
        <Text tone="muted" size="sm" data-path="shown">{{ said.shown(rows.length, all.length) }}</Text>
      </Stack>
      <Card v-if="kind === 'place' && shape === 'tree'">
        <Tree :items="tree" :under="(b) => b.under" :item-key="(b) => b.place.ref" :label="kinds.place.many">
          <template #default="{ item }">
            <Stack row gap="2" align="baseline" wrap>
              <Chip :to="`/${item.place.ref}`">{{ nameOf(item.place) }}</Chip>
              <Text v-if="item.place.type" tone="faint" size="sm">{{ item.place.type }}</Text>
              <Text v-if="item.from || item.until" tone="muted" size="sm">
                {{ [item.from && `${said.from} ${item.from}`, item.until && `${said.until} ${item.until}`].filter(Boolean).join(' ') }}
              </Text>
            </Stack>
          </template>
        </Tree>
        <Text v-if="!tree.length" tone="muted">{{ said.nothing }}</Text>
      </Card>
      <Card v-else flush>
        <Table v-model:sort="sort" :columns="columns" :rows="rows" :row-key="(r) => r.ref" :label="kinds[kind].many" @open="open">
          <template #cell-name="{ row, value }">
            <Stack row gap="2" align="baseline">
              <Text strong>{{ value }}</Text>
              <Badge v-if="kind === 'person' && (row as IndexedPerson).living" tone="good">{{ said.living }}</Badge>
              <Badge v-if="row.private" tone="danger">{{ said.privateOnly }}</Badge>
            </Stack>
          </template>
          <template #cell-tags="{ row }">
            <Stack row gap="1" wrap>
              <Tag v-for="tag in tagsOf(row)" :key="tag" :name="tag" :color="styles[tag]?.color" />
            </Stack>
          </template>
          <template #cell-status="{ value }">
            <Badge :tone="value === 'open' ? 'warn' : value === 'accepted' ? 'good' : 'faint'">{{ value }}</Badge>
          </template>
          <template #empty>{{ said.nothing }}</template>
        </Table>
      </Card>
    </Stack>
    <NewRecord v-if="kind !== 'hypothesis'" v-model:open="creating" :kind="kind" />
  </Page>
</template>
