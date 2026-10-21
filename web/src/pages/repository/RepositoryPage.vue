<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import EditButton from '../../edit/EditButton.vue'
import EditEntry from '../../edit/EditEntry.vue'
import EditValue from '../../edit/EditValue.vue'
import RecordMenu from '../../edit/RecordMenu.vue'
import { fieldLabel, repository as said } from '../../labels'
import { useNaming } from '../../naming'
import { addressOf, linkOf } from '../../records'
import { Badge, Block, Button, Card, Chip, Hero, Stack, Text, Value } from '../../ui'
import RecordView from '../record/RecordView.vue'

// A repository: its name and type, then the sources kept there with their call numbers,
// and beside them its addresses and links.
const props = defineProps<{ record: Record; around: Record[]; slug: string }>()
const { called } = useNaming()

const text = (v: unknown) => (typeof v === 'string' ? v : '')
const sources = computed(() =>
  props.around
    .filter((r) => r.ref.startsWith('source/'))
    .map((r) => {
      const list = Array.isArray(r.fields.repositories) ? r.fields.repositories : []
      const here = list
        .map((one: unknown) => (typeof one === 'object' && one !== null ? (one as { [field: string]: unknown }) : { repository: one }))
        .filter((one) => one.repository === props.slug)
        .map((one) => ({ call: text(one['call-number']), medium: text(one.medium) }))
      return { slug: r.ref.slice(r.ref.indexOf('/') + 1), here }
    })
    .filter((s) => s.here.length),
)
const addresses = computed(() => (Array.isArray(props.record.fields.addresses) ? props.record.fields.addresses.map(addressOf) : []))
const links = computed(() => (Array.isArray(props.record.fields.links) ? props.record.fields.links.map(linkOf) : []))
</script>

<template>
  <RecordView :record="record" :around="around" :slug="slug" :drawn="['name', 'type', 'addresses', 'links']">
    <template #hero>
      <Hero>
        <template #title><EditValue path="name" :label="fieldLabel('name')" /></template>
        <Stack row wrap gap="3" align="center">
          <EditValue path="type" :label="fieldLabel('type')" />
        </Stack>
        <template #actions><RecordMenu /></template>
      </Hero>
    </template>

    <Block :title="said.sources">
      <Stack gap="3" as="ul">
        <Stack v-for="s in sources" :key="s.slug" gap="1" as="li">
          <Stack row>
            <Chip tone="source" icon="book" :to="`/source/${s.slug}`">{{ called('source', s.slug) }}</Chip>
          </Stack>
          <Stack v-for="(k, i) in s.here" :key="i" row wrap gap="2" align="center">
            <Text v-if="k.call" face="mono" size="sm">{{ k.call }}</Text>
            <Badge v-if="k.medium" tone="faint">{{ k.medium }}</Badge>
          </Stack>
        </Stack>
      </Stack>
    </Block>

    <template #side>
      <Card :title="said.addresses">
        <template #action><EditEntry path="addresses" :label="fieldLabel('addresses')" list /></template>
        <Stack gap="2">
          <EditEntry
            v-for="(a, i) in addresses"
            :key="i"
            v-slot="{ disabled, marked }"
            :path="`addresses[${i}]`"
            :label="fieldLabel('addresses')"
          >
            <Value :path="`addresses[${i}]`" :label="fieldLabel('addresses')" :disabled="disabled" :marked="marked">{{ a }}</Value>
          </EditEntry>
        </Stack>
      </Card>
      <Card :title="said.links">
        <template #action><EditEntry path="links" :label="fieldLabel('links')" list /></template>
        <Stack gap="2" as="ul">
          <Stack v-for="(l, i) in links" :key="i" row gap="1" align="center" justify="between" as="li">
            <Button size="sm" variant="ghost" icon="link" :href="l.url">{{ l.title }}</Button>
            <EditButton :path="`links[${i}]`" :label="fieldLabel('links')" />
          </Stack>
        </Stack>
      </Card>
    </template>
  </RecordView>
</template>
