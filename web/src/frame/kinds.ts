import type { Kind } from '../api/client'
import type { IconName } from '../ui'

// The icon each kind is shown by, in the lists and wherever a record is found.
export const icons: Record<Kind, IconName> = {
  person: 'person',
  family: 'users',
  place: 'place',
  source: 'book',
  repository: 'archive',
  note: 'note',
  hypothesis: 'flask',
}
