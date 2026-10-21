import { computed, inject, type InjectionKey, type Ref } from 'vue'

// Where an overlay opened inside a Theme is put, so that it is drawn in that theme
// rather than in the root's: a dialog opened from the dark half of the gallery is dark.
export const portalKey: InjectionKey<Ref<HTMLElement | null>> = Symbol('portal')

export function usePortal() {
  const target = inject(portalKey, undefined)
  return computed(() => target?.value ?? 'body')
}
