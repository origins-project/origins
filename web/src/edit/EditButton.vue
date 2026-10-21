<script setup lang="ts">
import { fieldLabel } from '../labels'
import { IconButton } from '../ui'
import EditEntry from './EditEntry.vue'

// An entry changed in a drawer from a button beside what shows it, where what shows it
// leads somewhere else: a partner's card, a place it lies within.
const props = defineProps<{ path: string; label?: string }>()
const named = () => props.label ?? fieldLabel(props.path) ?? props.path
</script>

<template>
  <EditEntry v-slot="{ disabled }" :path="path" :label="named()">
    <IconButton icon="pen" :label="named()" :disabled="disabled" :data-path="path" />
  </EditEntry>
</template>
