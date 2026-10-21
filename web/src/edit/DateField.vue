<script setup lang="ts">
import { ref } from 'vue'
import { DateInput } from '../ui'
import DateDialog from './DateDialog.vue'
import { useDateReading } from './reading'

// A date typed, read back by the server as it is typed, or built in the dialog.
const model = defineModel<string>({ default: '' })
defineProps<{ label?: string; path?: string; disabled?: boolean }>()

const building = ref(false)
const { error, reads } = useDateReading(model)
</script>

<template>
  <DateInput
    v-model="model"
    :label="label"
    :path="path"
    :reads="reads()"
    :error="error"
    :disabled="disabled"
    @build="building = true"
  />
  <DateDialog v-model:open="building" :text="model" @use="(text) => (model = text)" />
</template>
