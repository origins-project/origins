// What every form that changes the corpus shares: the record it is changing, pinned by
// the seen it was shown, and what is done once a change lands or is refused.

import { computed, inject, provide, type ComputedRef, type InjectionKey, type Ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Refused, type Drafting, type Kind, type Made, type Record } from '../api/client'
import { useClient } from '../api/use'
import { useCorpus } from '../api/corpus'
import { edit as said, notices } from '../labels'
import { drafts } from '../records'
import { listed } from '../search'

// The record a page lets its reader change.
export interface Editing {
  record: Ref<Record | undefined>
  kind: ComputedRef<Kind | 'project'>
  // the entity an edit names: person/x, or project
  entity: ComputedRef<string>
}

const editingKey: InjectionKey<Editing> = Symbol('editing')

export function provideEditing(editing: Editing) {
  provide(editingKey, editing)
}

export function useEditing(): Editing {
  const editing = inject(editingKey)
  if (editing === undefined) throw new Error('a form was drawn with no record to change')
  return editing
}

// Whether the draft the tab is writing changes what is shown at a path of the record.
export function useDrafted(path: () => string): ComputedRef<boolean> {
  const corpus = useCorpus()
  const { record, entity } = useEditing()
  return computed(() => {
    const draft = corpus.draft.value
    return !!draft && drafts(draft.edits, entity.value, path(), record.value?.fields)
  })
}

// The operations the record allows at a path.
export function opsAt(record: Record | undefined, path: string): string[] {
  return record?.edits.find((e) => e.path === path)?.ops ?? []
}

export function useChange() {
  const client = useClient()
  const corpus = useCorpus()
  const route = useRoute()
  const router = useRouter()
  const writes = computed(() => corpus.writes())
  const offered = corpus.offered

  // Says what a change did, and goes where the record now stands if it went.
  async function landed(result: Made | Drafting) {
    if (!('commit' in result)) {
      corpus.drafted(result)
      const last = result.edits.at(-1)
      corpus.notify(notices.drafted(last ? [last.op, last.entity, last.path].filter(Boolean).join(' ') : ''))
      return
    }
    if (result.commit) corpus.made(result.commit)
    corpus.notify(result.subject, { label: said.undo, run: () => (corpus.undoing.value = true) })
    const [kind, slug] = route.path.slice(1).split('/')
    if (!slug || !(kind! in listed)) return
    const dir = `${listed[kind as Kind]}/${slug}`
    const moved = result.moved.find((m) => m.from === dir)
    if (moved) await router.replace(`/${kind}/${moved.to.slice(moved.to.indexOf('/') + 1)}`)
    else if (result.removed.some((r) => r === dir || r.startsWith(`${dir}/`)))
      await router.replace(result.stands[0] ? `/${result.stands[0]}` : '/')
  }

  // Asks for a change, closes the form it was asked from once answered, and lands it,
  // the whole of it held by the corpus as the tab's own change.
  function make(ask: () => Promise<Made | Drafting>, answered: () => void): Promise<void> {
    return corpus.change(async () => {
      const result = await ask()
      answered()
      await landed(result)
    })
  }

  return { client, writes, offered, make }
}

// What a refusal says, for the form that was refused to show.
export interface Refusal {
  kind: string
  message: string
  findings: { entity: string; path?: string; rule: string; message: string }[]
  problems: string[]
}

export function refusalOf(e: unknown): Refusal {
  if (e instanceof Refused) return { kind: e.kind, message: e.message, findings: e.findings, problems: e.problems }
  return { kind: 'unknown', message: e instanceof Error ? e.message : String(e), findings: [], problems: [] }
}
