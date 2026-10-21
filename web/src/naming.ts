// What a page looks records up by: each one the index holds, by its ref, and what it is
// called; and where a file a record refers to is kept and served.

import { computed, inject, provide, type InjectionKey, type Ref } from 'vue'
import { address, type Indexed, type Kind, type Record } from './api/client'
import { useCorpus } from './api/corpus'
import { initials, years, type IndexedPerson } from './person'
import { listed, names } from './search'

export function useNaming() {
  const { index } = useCorpus()
  const byRef = computed(() => {
    const all = new Map<string, Indexed>()
    if (!index.value) return all
    for (const kind of Object.keys(listed) as Kind[]) for (const one of index.value[listed[kind]]) all.set(one.ref, one)
    return all
  })
  const people = computed(() => new Map(index.value?.people.map((p) => [p.ref, p]) ?? []))

  // What a record is called, or its slug where the index has no record of it.
  function called(kind: Kind, slug: string): string {
    const found = byRef.value.get(`${kind}/${slug}`)
    if (!found) return slug
    const { name, others } = names(kind, found, people.value)
    return kind === 'source' && others[0] ? others[0] : name
  }

  function person(slug: string): IndexedPerson | undefined {
    return byRef.value.get(`person/${slug}`) as IndexedPerson | undefined
  }

  // Somebody as a card shows them: their name, years and portrait, leading to their
  // record where there is one, or the name a record gave them where there is none.
  function card(slug: string | undefined, name?: string) {
    const found = slug ? person(slug) : undefined
    const shown = (typeof found?.name === 'string' && found.name) || name || slug || ''
    const portrait = found?.portrait
    return {
      name: shown,
      to: found ? `/${found.ref}` : undefined,
      lifespan: years(found),
      sex: found?.sex,
      src: portrait && served(portrait.record, portrait.file),
      region: portrait?.region,
      initials: initials(shown),
      ghost: !found,
    }
  }

  return { index, called, person, card }
}

// The record that keeps a file a record refers to, and its name there: the record
// itself, or another one where the file is written as a path from the record's
// directory into that one's.
export function located(ref: string, file: string): { ref: string; file: string } {
  const kind = ref.slice(0, ref.indexOf('/')) as Kind
  const steps = [listed[kind], ref.slice(ref.indexOf('/') + 1)]
  for (const step of file.split('/')) {
    if (step === '..') steps.pop()
    else if (step !== '.' && step !== '') steps.push(step)
  }
  const [dir = '', slug = '', ...rest] = steps
  const owner = (Object.keys(listed) as Kind[]).find((k) => listed[k] === dir)
  return { ref: `${owner}/${slug}`, file: rest.join('/') }
}

// Where a file a record refers to is served.
export function served(ref: string, file: string): string {
  const at = located(ref, file)
  return address(`/files/${at.ref}/${at.file.split('/').map(encodeURIComponent).join('/')}`)
}

// Whether a file is a picture a page can show, by its extension.
export function picture(file: string): boolean {
  return /\.(jpe?g|png|gif|webp|avif|bmp|svg)$/i.test(file)
}

const keeping: InjectionKey<Ref<Record[]>> = Symbol('keeping')

// The records a page has read, which say what files are kept beside them.
export function provideKept(records: Ref<Record[]>) {
  provide(keeping, records)
}

// Where a picture a record refers to is served, where the record keeping it has it beside
// it: a file not there, or not one the page has read, is not asked for.
export function useKept() {
  const records = inject(keeping, undefined)
  return (ref: string, file: string): string | undefined => {
    if (!picture(file)) return undefined
    const at = located(ref, file)
    const keeper = records?.value.find((r) => r.ref === at.ref)
    return keeper?.attached.includes(at.file) ? served(ref, file) : undefined
  }
}

// A record the page has read, by its reference.
export function useRead() {
  const records = inject(keeping, undefined)
  return (ref: string): Record | undefined => records?.value.find((r) => r.ref === ref)
}
