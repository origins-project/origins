<script setup lang="ts">
import { computed } from 'vue'
import type { Record } from '../../api/client'
import EditValue from '../../edit/EditValue.vue'
import { fieldLabel, record as said } from '../../labels'
import { Block, Icon, Prose } from '../../ui'

// Notes: the record's body, changed from the button beside its heading.
const props = defineProps<{ record: Record; title?: string; html?: string }>()
const body = computed(() => props.html ?? props.record.html.body ?? '')
</script>

<template>
  <Block :title="title ?? said.notes">
    <template #actions>
      <EditValue path="body" :label="fieldLabel('body')"><Icon name="note" /></EditValue>
    </template>
    <Prose v-if="body" :html="body" />
  </Block>
</template>
