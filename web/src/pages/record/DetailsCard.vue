<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import { spoken } from '../../date'
import EditEntry from '../../edit/EditEntry.vue'
import Fields from '../../edit/Fields.vue'
import { eventLabel, record as said } from '../../labels'
import { placed } from '../../person'
import { Card, Stack, Text, Value } from '../../ui'

// Details: the fields the page draws nowhere else, each changed where it is shown, and,
// where the page has a timeline, the facts held without a date, which it has no place
// for.
const props = defineProps<{ record: Record; only?: string[]; except?: string[]; undated?: boolean }>()
const held = computed(() => (props.undated ? placed(props.record).undated : []))
const text = (v: unknown) => (typeof v === 'string' ? v : '')
</script>

<template>
  <Card :title="said.details">
    <Stack gap="4">
      <Stack v-if="held.length" gap="2">
        <EditEntry
          v-for="one in held"
          :key="one.path"
          v-slot="{ disabled, marked }"
          :path="one.path"
          :label="eventLabel(one.event.type)"
        >
          <Value :path="one.path" :label="eventLabel(one.event.type)" :disabled="disabled" :marked="marked">
            <Stack row wrap gap="2" align="baseline" as="span">
              <Text strong>{{ eventLabel(one.event.type) }}</Text>
              <Text v-if="text(one.event.value)">{{ text(one.event.value) }}</Text>
              <Text v-if="text(one.event.date)" tone="muted">{{ spoken(text(one.event.date)) }}</Text>
            </Stack>
          </Value>
        </EditEntry>
      </Stack>
      <Fields :only="only" :except="except" compact />
    </Stack>
  </Card>
</template>
