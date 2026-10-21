// What the server reads a typed date as, asked again a moment after the typing stops.

import { onScopeDispose, ref, watch, type Ref } from 'vue'
import type { DateRead } from '../api/client'
import { useClient } from '../api/use'
import { dates } from '../labels'
import { refusalOf } from './change'

export function useDateReading(text: Ref<string>, pause = 250) {
  const client = useClient()
  const read = ref<DateRead>()
  const error = ref('')
  let timer: ReturnType<typeof setTimeout> | undefined
  let asked = 0

  async function ask(said: string) {
    const mine = ++asked
    if (!said.trim()) {
      read.value = undefined
      error.value = ''
      return
    }
    try {
      const answer = await client.date(said)
      if (mine !== asked) return
      read.value = answer
      error.value = ''
    } catch (e) {
      if (mine !== asked) return
      read.value = undefined
      error.value = refusalOf(e).message
    }
  }

  watch(
    text,
    (said) => {
      clearTimeout(timer)
      timer = setTimeout(() => ask(said), pause)
    },
    { immediate: true },
  )
  onScopeDispose(() => clearTimeout(timer))

  // One line saying how the date sorts, or that it floats.
  const reads = () => {
    const r = read.value
    if (!r) return ''
    return r.anchored ? `${dates.sorted} ${r.sort ?? ''}`.trim() : dates.floating
  }
  return { read, error, reads }
}
