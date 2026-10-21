<script setup lang="ts">
import { useCorpus } from '../../api/corpus'
import { record as said } from '../../labels'
import { Button, Card } from '../../ui'
import HistoryList from './HistoryList.vue'

// History: the last changes made to the record, and the way to all of them.
defineProps<{ entity: string }>()
defineEmits<{ all: [] }>()
const { hidden } = useCorpus()
</script>

<template>
  <Card v-if="!hidden()" :title="said.history">
    <template #action>
      <Button size="sm" variant="ghost" @click="$emit('all')">{{ said.all }}</Button>
    </template>
    <HistoryList :entity="entity" :n="3" />
  </Card>
</template>
