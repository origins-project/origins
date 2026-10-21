<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { Kind } from '../api/client'
import { create as said, kinds } from '../labels'
import { familySlug, slugify } from '../lists'
import { Field, Stack } from '../ui'
import ChangeDialog from './ChangeDialog.vue'
import RecordPicker from './RecordPicker.vue'
import { useChange } from './change'

// A record made from its name, its slug derived from the name as an import derives it
// until somebody writes one of their own, and gone to once it is made.
const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ kind: Exclude<Kind, 'hypothesis'> }>()
const { client } = useChange()
const router = useRouter()

const given = ref('')
const surname = ref('')
const name = ref('')
const partners = ref(['', ''])
const own = ref<string>()
watch(open, (now) => {
  if (!now) return
  given.value = surname.value = name.value = ''
  partners.value = ['', '']
  own.value = undefined
})

const titled = computed(() => props.kind === 'source' || props.kind === 'note')
const chosen = computed(() => partners.value.filter(Boolean))
const value = computed<{ [field: string]: unknown }>(() => {
  switch (props.kind) {
    case 'person': {
      const parts: { [part: string]: string } = {}
      if (given.value.trim()) parts.given = given.value.trim()
      if (surname.value.trim()) parts.surname = surname.value.trim()
      return { name: parts }
    }
    case 'family':
      return { partners: chosen.value }
    default:
      return { [titled.value ? 'title' : 'name']: name.value.trim() }
  }
})
const derived = computed(() => {
  if (props.kind === 'person') return slugify(`${surname.value} ${given.value}`)
  if (props.kind === 'family') return familySlug(chosen.value)
  return slugify(name.value)
})
const slug = computed({
  get: () => own.value ?? derived.value,
  set: (typed: string) => (own.value = typed),
})
const ready = computed(() => slug.value.trim() !== '' && Object.values(value.value).some((v) => (Array.isArray(v) ? v.length : typeof v === 'object' ? Object.keys(v as object).length : v)))

const make = async (dry: boolean, sources?: string[]) => {
  const entity = `${props.kind}/${slug.value.trim()}`
  const made = await client.change({ edit: { op: 'create', entity, value: value.value }, seen: '', sources, dry })
  if (!dry) await router.push(`/${entity}`)
  return made
}
</script>

<template>
  <ChangeDialog
    v-model:open="open"
    :title="said.new(kinds[kind].one)"
    :confirm="said.create"
    :make="make"
    :ready="ready"
    :asked="JSON.stringify([value, slug])"
    path="create"
  >
    <Stack gap="3">
      <template v-if="kind === 'person'">
        <Field v-model="given" :label="said.given" path="name.given" />
        <Field v-model="surname" :label="said.surname" path="name.surname" />
      </template>
      <template v-else-if="kind === 'family'">
        <RecordPicker v-model="partners[0]" kind="person" :label="said.partner" path="partners[0]" />
        <RecordPicker v-model="partners[1]" kind="person" :label="said.partner" path="partners[1]" />
      </template>
      <Field v-else v-model="name" :label="titled ? said.title : said.name" :path="titled ? 'title' : 'name'" />
      <Field v-model="slug" :label="said.slug" mono path="slug" />
    </Stack>
  </ChangeDialog>
</template>
