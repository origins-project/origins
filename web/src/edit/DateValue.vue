<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { dates, edit as said } from '../labels'
import { Button, Field, Grid, Select, Stack, Switch, type Option } from '../ui'
import { calendars, epochal, type Calendar, type Value } from '../date'

// One date of the grammar in its parts, any of which may be empty.
const value = defineModel<Value>({ required: true })
defineProps<{ path?: string }>()

const none = '-'
const calendarOptions: Option[] = (Object.keys(calendars) as Calendar[]).map((c) => ({ value: c, label: c }))
const months = computed<Option[]>(() => [
  { value: none, label: '—' },
  ...calendars[value.value.calendar].map((m) => ({ value: m, label: m })),
])
const common = computed(() => epochal.includes(value.value.calendar))
const more = ref(value.value.dual || value.value.bce || value.value.newYear.day !== '')

// A month of one calendar is no month of another, and only the common era has the rest.
watch(
  () => value.value.calendar,
  (calendar) => {
    const v = value.value
    const list: readonly string[] = calendars[calendar]
    if (v.month && !list.includes(v.month)) v.month = ''
    if (v.newYear.month && !list.includes(v.newYear.month)) v.newYear = { day: '', month: '' }
    if (!epochal.includes(calendar)) Object.assign(v, { dual: false, bce: false, newYear: { day: '', month: '' } })
  },
)
</script>

<template>
  <Stack gap="2" :data-path="path">
    <Grid :cols="5" gap="2">
      <Select v-model="value.calendar" :label="dates.calendar" :options="calendarOptions" />
      <Field v-model="value.day" :label="dates.day" mono />
      <Select
        :model-value="value.month || none"
        :label="dates.month"
        :options="months"
        @update:model-value="(m) => (value.month = !m || m === none ? '' : m)"
      />
      <Field v-model="value.year" :label="dates.year" mono />
      <Field v-model="value.time" :label="dates.time" mono />
    </Grid>
    <Stack v-if="common && more" gap="2">
      <Stack row wrap gap="4">
        <Switch v-model="value.dual" :label="dates.dual" :disabled="value.bce" />
        <Switch v-model="value.bce" :label="dates.bce" :disabled="value.dual" />
      </Stack>
      <Grid :cols="5" gap="2">
        <Field v-model="value.newYear.day" :label="`${dates.newYear} · ${dates.day}`" mono :disabled="value.dual" />
        <Select
          :model-value="value.newYear.month || none"
          :label="`${dates.newYear} · ${dates.month}`"
          :options="months"
          :disabled="value.dual"
          @update:model-value="(m) => (value.newYear.month = !m || m === none ? '' : m)"
        />
      </Grid>
    </Stack>
    <Stack v-else-if="common" row>
      <Button size="sm" variant="ghost" icon="plus" @click="more = true">{{ said.more }}</Button>
    </Stack>
  </Stack>
</template>
