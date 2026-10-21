<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { dates, edit as said } from '../labels'
import { Button, Definition, Definitions, Dialog, Field, Notice, Segmented, Stack, Text, type Option } from '../ui'
import { emptyDate, emptyQualified, parseDate, qualifierFor, writeDate, type Built, type Shape } from '../date'
import DateQualified from './DateQualified.vue'
import { useDateReading } from './reading'

// A date built from its parts, showing the text it will write as it is built and what
// the server reads that text as; a date it cannot read back from the text starts empty.
const open = defineModel<boolean>('open', { default: false })
const props = defineProps<{ text: string; path?: string }>()
const emit = defineEmits<{ use: [text: string] }>()

const built = ref<Built>(emptyDate())

watch(open, (now) => {
  if (!now) return
  const read = parseDate(props.text) ?? emptyDate()
  read.from ??= emptyQualified()
  read.to ??= emptyQualified()
  built.value = read
})

const shapes: Option[] = (Object.entries(dates.shapes) as [Shape, string][]).map(([value, label]) => ({ value, label }))
const shape = computed({
  get: () => built.value.shape,
  set: (to: Shape) => {
    built.value.shape = to
    built.value.moment.qualifier = qualifierFor(to, built.value.moment.qualifier)
  },
})

const written = computed(() => writeDate(built.value))
const { read, error } = useDateReading(written)

function use() {
  emit('use', written.value)
  open.value = false
}
</script>

<template>
  <Dialog v-model:open="open" :title="dates.title" :path="path">
    <Stack gap="4">
      <Segmented
        :model-value="shape"
        :label="dates.shape"
        :options="shapes"
        @update:model-value="(s) => (shape = s as Shape)"
      />
      <template v-if="built.shape === 'period'">
        <DateQualified v-if="built.from" v-model="built.from" :title="dates.from" free />
        <DateQualified v-if="built.to" v-model="built.to" :title="dates.to" free />
      </template>
      <DateQualified v-else-if="built.shape !== 'words'" v-model="built.moment" />
      <Field v-model="built.wording" :label="dates.wording" />
      <Definitions compact>
        <Definition :term="dates.written">
          <Text face="mono" data-written>{{ written }}</Text>
        </Definition>
        <Definition v-if="read" :term="dates.sorted">
          <Text face="mono">{{ read.anchored ? read.sort : dates.floating }}</Text>
        </Definition>
      </Definitions>
      <Notice v-if="error" tone="danger">{{ error }}</Notice>
    </Stack>
    <template #actions>
      <Button variant="ghost" @click="open = false">{{ said.cancel }}</Button>
      <Button variant="primary" :disabled="!!error || !written" @click="use">{{ dates.use }}</Button>
    </template>
  </Dialog>
</template>
