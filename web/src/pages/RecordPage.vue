<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { Kind, Record } from '../api/client'
import { useCorpus } from '../api/corpus'
import { useClient } from '../api/use'
import { provideEditing } from '../edit/change'
import FamilyPage from './family/FamilyPage.vue'
import HypothesisPage from './hypothesis/HypothesisPage.vue'
import NotePage from './note/NotePage.vue'
import PersonPage from './person/PersonPage.vue'
import PlacePage from './place/PlacePage.vue'
import RepositoryPage from './repository/RepositoryPage.vue'
import SourcePage from './source/SourcePage.vue'
import { Notice, Page } from '../ui'

const props = defineProps<{ kind: Kind; slug: string }>()
const client = useClient()
const corpus = useCorpus()

const record = ref<Record>()
const around = ref<Record[]>([])
const failed = ref('')
let asked = 0
async function read() {
  const mine = ++asked
  try {
    const page = await client.record(props.kind, props.slug)
    if (mine !== asked) return
    record.value = page.record
    around.value = page.around
    failed.value = ''
  } catch (e) {
    if (mine !== asked) return
    record.value = undefined
    failed.value = e instanceof Error ? e.message : String(e)
  }
}
watch(
  () => [props.kind, props.slug],
  () => {
    record.value = undefined
    void read()
  },
  { immediate: true },
)
corpus.listen((event) => {
  if (event.type === 'moved' || event.type === 'draft') void read()
})

provideEditing({ record, kind: computed(() => props.kind), entity: computed(() => `${props.kind}/${props.slug}`) })

const pages = {
  person: PersonPage,
  family: FamilyPage,
  place: PlacePage,
  source: SourcePage,
  repository: RepositoryPage,
  note: NotePage,
  hypothesis: HypothesisPage,
}
</script>

<template>
  <Page>
    <Notice v-if="failed" tone="danger">{{ failed }}</Notice>
    <component :is="pages[kind]" v-if="record" :key="record.ref" :record="record" :around="around" :slug="slug" />
  </Page>
</template>
