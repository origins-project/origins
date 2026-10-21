<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCorpus } from '../api/corpus'
import type { Gap } from '../chart'
import { chart as said, create, edit } from '../labels'
import { familySlug, free, slugify, taken } from '../lists'
import { Field, Segmented, Stack } from '../ui'
import ChangeDialog from './ChangeDialog.vue'
import RecordPicker from './RecordPicker.vue'
import { useChange } from './change'

// A parent nobody has a record of given one: somebody in the tree or somebody new, put
// in the place of the partner unknown, or with the child in a family of their own where
// no family lists the child. Somebody new is made first, and linked as a second change.
const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ gap: Gap }>()
const { client } = useChange()
const corpus = useCorpus()

const mode = ref('tree')
const chosen = ref('')
const given = ref('')
const surname = ref('')
// the family's record as the dialog was opened on it, and somebody new once made
let seen = ''
const made = ref<string>()
watch(open, async (now) => {
  if (!now) return
  mode.value = 'tree'
  chosen.value = given.value = surname.value = ''
  made.value = undefined
  seen = props.gap.family ? (await client.record('family', props.gap.family)).record.seen : ''
})

const modes = [
  { value: 'tree', label: said.inTree },
  { value: 'new', label: said.newPerson },
]
const used = computed(() => taken(corpus.index.value))
const named = computed(() => {
  const parts: { [part: string]: string } = {}
  if (given.value.trim()) parts.given = given.value.trim()
  if (surname.value.trim()) parts.surname = surname.value.trim()
  return parts
})
const newcomer = computed(() => made.value ?? free(slugify(`${surname.value} ${given.value}`), used.value))
const known = computed(() => corpus.index.value?.people.some((p) => p.ref === `person/${chosen.value}`) ?? false)
const ready = computed(() => (mode.value === 'tree' ? known.value : newcomer.value !== ''))

function link(parent: string, dry: boolean, sources?: string[]) {
  const { child, family, partner } = props.gap
  if (family)
    return client.change({
      edit: { op: 'set', entity: `family/${family}`, path: `partners[${partner}].person`, value: parent },
      seen,
      sources,
      dry,
    })
  const partners = [parent, 'unknown']
  return client.change({
    edit: { op: 'create', entity: `family/${free(familySlug(partners), used.value)}`, value: { partners, children: [child] } },
    seen: '',
    sources,
    dry,
  })
}

const make = async (dry: boolean, sources?: string[]) => {
  if (mode.value === 'tree') return link(chosen.value, dry, sources)
  if (made.value) return link(made.value, dry, sources)
  const slug = newcomer.value
  const person = await client.change({ edit: { op: 'create', entity: `person/${slug}`, value: { name: named.value } }, seen: '', dry })
  if (dry) return person
  made.value = slug
  if ('commit' in person && person.commit) corpus.made(person.commit)
  return link(slug, false, sources)
}
</script>

<template>
  <ChangeDialog
    v-model:open="open"
    :title="said.addParent"
    :confirm="edit.add"
    :make="make"
    :ready="ready"
    :asked="JSON.stringify([mode, chosen, named])"
    path="partners"
  >
    <Stack gap="3">
      <Segmented v-model="mode" :options="modes" :label="said.addParent" :disabled="made !== undefined" />
      <RecordPicker v-if="mode === 'tree'" v-model="chosen" kind="person" :label="said.parent" path="partners" />
      <template v-else>
        <Field v-model="given" :label="create.given" path="name.given" />
        <Field v-model="surname" :label="create.surname" path="name.surname" />
      </template>
    </Stack>
  </ChangeDialog>
</template>
