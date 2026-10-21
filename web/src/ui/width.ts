import { onScopeDispose, ref, type Ref } from 'vue'

// How much room the window gives: a phone's, under 40em, a narrow window's, under
// 60em, or a wide one's. The two lengths are the breakpoints every component draws at.
export type Width = 'phone' | 'narrow' | 'wide'

export function useWidth(): Ref<Width> {
  const width = ref<Width>('wide')
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return width
  const phone = window.matchMedia('(max-width: 40em)')
  const narrow = window.matchMedia('(max-width: 60em)')
  const read = () => {
    width.value = phone.matches ? 'phone' : narrow.matches ? 'narrow' : 'wide'
  }
  read()
  phone.addEventListener('change', read)
  narrow.addEventListener('change', read)
  onScopeDispose(() => {
    phone.removeEventListener('change', read)
    narrow.removeEventListener('change', read)
  })
  return width
}
