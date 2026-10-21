// What the component gallery at /ui shows: words and data standing in for a corpus's,
// in development builds only. None of it is a label, so none of it is in labels.ts.
import type { ChartBox, Column, FoundGroup, IconName, Option } from '..'
import { icons } from '../icons'
import { draw, type Shape } from '../../chart'
import { chart as said } from '../../labels'
import { initials } from '../../person'

export interface Person {
  slug: string
  name: string
  born: string
  year?: number
  place: string
}

const given = ['Giovanni', 'Maria', 'Giuseppe', 'Annunziata', 'Pietro', 'Rosa', 'Luigi', 'Teresa']
const surnames = ['Rossi', 'Bianchi', 'Esposito', 'Russo', 'Ferrari', 'Romano', 'Colombo']
export const places = ['Lucca', 'Pisa', 'Thessaloniki', 'Buenos Aires', 'Capannori', 'New York']

// Two thousand people, the same every time, so that the table has more rows than it
// draws.
export const people: Person[] = Array.from({ length: 2000 }, (_, i) => {
  const year = 1700 + ((i * 37) % 300)
  const undated = i % 23 === 0
  return {
    slug: `${surnames[i % surnames.length]!.toLowerCase()}-${given[(i * 3) % given.length]!.toLowerCase()}-${i}`,
    name: `${given[(i * 3) % given.length]} ${surnames[i % surnames.length]}`,
    born: undated ? '' : `ABT ${year}`,
    year: undated ? undefined : year,
    place: places[(i * 5) % places.length]!,
  }
})

export const columns: Column<Person>[] = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'born', label: 'Born', width: 'mid', sortable: true, order: (p) => p.year },
  { key: 'place', label: 'Place', width: 'mid', sortable: true },
  { key: 'slug', label: 'Slug', mono: true },
]

export const shapes: Option[] = [
  { value: 'exact', label: 'Exact' },
  { value: 'about', label: 'About' },
  { value: 'range', label: 'Range' },
  { value: 'period', label: 'Period' },
]

export const sexes: Option[] = [
  { value: 'M', label: 'Male' },
  { value: 'F', label: 'Female' },
  { value: 'X', label: 'Other', hint: 'X' },
  { value: 'U', label: 'Unknown', hint: 'U' },
]

export const tabs: Option[] = [
  { value: 'life', label: 'Life' },
  { value: 'family', label: 'Family' },
  { value: 'sources', label: 'Sources' },
]

export const tags = [
  { name: 'To do', color: '#b5562f' },
  { name: 'Emigrants', color: '#3f6f9a' },
  { name: 'Unstyled' },
]

export interface Region {
  slug: string
  name: string
  under: Region[]
}

// Places within places, for the tree.
export const regions: Region[] = [
  {
    slug: 'italia',
    name: 'Italia',
    under: [
      {
        slug: 'toscana',
        name: 'Toscana',
        under: [
          { slug: 'lucca', name: 'Lucca', under: [{ slug: 'capannori', name: 'Capannori', under: [] }] },
          { slug: 'pisa', name: 'Pisa', under: [] },
        ],
      },
    ],
  },
  { slug: 'argentina', name: 'Argentina', under: [{ slug: 'buenos-aires', name: 'Buenos Aires', under: [] }] },
]

export const yaml = `project:
  name: The Rossi family of Lucca
  main-person: rossi-giovanni
  tags:
    To do:
      color: "#b5562f"
origins:
  format-version: 1
`

export const words = {
  overview: 'Counts, trees and files',
  tree: 'Places',
  file: 'origins.yaml',
  title: 'Components',
  light: 'Light',
  dark: 'Dark',
  supposing: 'Supposing',
  buttons: 'Buttons',
  save: 'Save',
  cancel: 'Cancel',
  remove: 'Delete',
  addEvent: 'Add event',
  more: 'More',
  actions: 'Actions',
  history: 'History',
  link: 'Open the record',
  fields: 'Fields',
  given: 'Given names',
  givenValue: 'Giovanni Battista',
  surname: 'Surname',
  surnameValue: 'Rossi',
  page: 'Page',
  pageHint: 'As the register numbers it',
  slug: 'Slug',
  slugError: 'Another person is called this',
  slugValue: 'rossi-giovanni-1820',
  body: 'Notes',
  bodyValue: 'Born in the *hamlet* of Gragnano, as [the register](sources/lucca-register) says.',
  transcription: 'Transcription',
  date: 'Date',
  dateValue: 'ABT 1820',
  dateReads: 'About 1820',
  dateBad: 'ABUT 1820',
  dateError: 'Not a date origins can read',
  sex: 'Sex',
  choose: 'Choose…',
  living: 'Living',
  private: 'Private',
  shape: 'Shape',
  overlays: 'Overlays',
  popover: 'Edit in place',
  popoverPath: 'events[0].date',
  dialog: 'Delete…',
  dialogTitle: 'Delete Giovanni Rossi?',
  dialogBody: 'His two families keep their other partner, and the one reference to him in prose is taken out with its text.',
  drawer: 'Edit event',
  drawerTitle: 'Birth',
  drawerPath: 'events[0]',
  nav: 'Lists',
  edit: 'Edit',
  rename: 'Rename',
  merge: 'Merge into…',
  show: 'Show private',
  hint: 'R',
  surfaces: 'Surfaces',
  details: 'Details',
  research: 'To research',
  warn: 'Contradicts',
  cardBody: 'Born about 1820 in Lucca, son of Pietro and Maria.',
  researchBody: 'His death is not recorded.',
  warnBody: 'Baptised before he was born.',
  sections: 'Life',
  table: 'Table',
  tableLabel: 'People',
  empty: 'Nobody yet.',
  marks: 'Tags, chips and badges',
  otherName: 'Gianni',
  nickname: 'Nickname',
  filter: 'Emigrants',
  source: 'Lucca register, f. 12',
  supposed: 'Born 1844',
  add: 'Add name',
  open: 'open',
  rejected: 'rejected',
  notices: 'Notices',
  banner: 'This is a read-only demo of Origins. All of its data is generated.',
  info: 'The corpus moved; this page shows what it holds now.',
  warning: 'This hypothesis was made 12 commits ago.',
  danger: 'Somebody changed this record since you opened it.',
  good: 'Accepted: annunziata-born-1844.',
  draft: 'A draft of 3 changes is waiting.',
  toast: 'Renamed Giovanni Rossi.',
  undo: 'Undo',
  keep: 'Keep',
  text: 'Text',
  muted: 'Recorded by the parish priest',
  faint: 'Last changed 3 days ago',
  serif: 'Giovanni Battista Rossi',
  mono: 'persons/rossi-giovanni',
  layout: 'Layout',
  side: 'Beside the main column',
  cell: 'A cell',
  frame: 'Frame',
  origins: 'Origins',
  people: 'People',
  families: 'Families',
  places: 'Places',
  sources: 'Sources',
  search: 'Search',
  searchKeys: 'Ctrl K',
  suppose: 'Suppose',
  findings: 'Findings',
  live: 'Live',
  offline: 'Offline',
  query: 'ross',
  keys: { move: '↑ ↓', choose: 'Enter', close: 'Esc' },
  notify: 'Notify',
  values: 'Values',
  place: 'Place',
  placeValue: 'Lucca',
  occupation: 'Occupation',
  occupationValue: 'Weaver',
  names: 'Other names',
  takeOut: 'Remove',
} as const

export const found: FoundGroup<string>[] = [
  {
    label: 'People',
    found: [
      {
        key: 'person/rossi-giovanni',
        icon: 'person',
        text: [{ text: 'Giovanni ' }, { text: 'Ross', marked: true }, { text: 'i' }],
        sub: [{ text: '1820–1891' }],
        value: 'person/rossi-giovanni',
      },
      {
        key: 'person/bianchi-maria',
        icon: 'person',
        text: [{ text: 'Maria Bianchi' }],
        sub: [{ text: 'Maria ' }, { text: 'Ross', marked: true }, { text: 'i' }],
        value: 'person/bianchi-maria',
      },
    ],
  },
  {
    label: 'Families',
    found: [
      {
        key: 'family/rossi-bianchi',
        icon: 'family',
        text: [{ text: 'Giovanni ' }, { text: 'Ross', marked: true }, { text: 'i & Maria Bianchi' }],
        sub: [{ text: 'married' }],
        value: 'family/rossi-bianchi',
      },
    ],
  },
]

// A picture drawn here rather than fetched, a figure on a ground, so that a portrait and a
// thumbnail have something to crop.
export const picture =
  'data:image/svg+xml,' +
  encodeURIComponent(
    '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 90"><rect width="120" height="90" fill="#cbbfa8"/><circle cx="40" cy="34" r="14" fill="#6b5a45"/><rect x="24" y="50" width="32" height="40" rx="10" fill="#6b5a45"/><circle cx="84" cy="36" r="13" fill="#8a7559"/><rect x="69" y="51" width="30" height="39" rx="10" fill="#8a7559"/></svg>',
  )

// Who is where in the picture above, as fractions of it.
export const depicted = [
  { key: '0', region: [0.18, 0.2, 0.48, 1], label: 'Giovanni Rossi' },
  { key: '1', region: [0.55, 0.22, 0.84, 1], label: 'Maria Bianchi' },
]

// A box being drawn round somebody not yet tagged.
export const untagged = [0.06, 0.06, 0.94, 0.18]

export const person = {
  heading: 'People',
  parts: [
    { text: 'Dott.', soft: true },
    { text: 'Giovanni' },
    { text: 'Battista', call: true },
    { text: 'Titta', nickname: true },
    { text: 'Rossi' },
  ],
  written: 'Giovanni Battista di Antonio Rossi',
  lifespan: '12 March 1834, Borgo San Lorenzo – 3 November 1901, Firenze',
  name: 'Giovanni Battista Rossi',
  initials: 'GR',
  years: '1834 – 1901',
  region: [0, 0, 55, 100],
  partner: 'Maria Bianchi',
  partnerYears: '1840 – 1901',
  child: 'Anna Rossi',
  adopted: 'adopted',
  disproven: 'Annunziata Rossi',
  unnamed: 'Rossi',
  thumb: 'The wedding party',
  prose:
    '<p>He appears as “Gio. Rossi” in the <a href="/source/census-1861">1861 census</a>.</p><ul><li><input type="checkbox" disabled> Find his death in the civil register.</li></ul>',
  moments: [
    { year: '1834', text: 'Birth · 12 March 1834' },
    { year: '', text: 'Baptism · 14 March 1834' },
    { year: '1838', text: 'Birth of his brother Giuseppe', quiet: true },
    { year: '1858', text: 'Weaver · from 1858 to 1880', span: true },
  ],
} as const

// Three generations around one man, one of his sons adopted, drawn as all relatives.
const kindred = [
  { slug: 'giovanni', name: 'Giovanni Battista Rossi', sex: 'M', lifespan: '1834 – 1901' },
  { slug: 'antonio', name: 'Antonio Rossi', sex: 'M', lifespan: '1805 – 1871' },
  { slug: 'lucia', name: 'Lucia Verdi', sex: 'F', lifespan: '1810 – 1850' },
  { slug: 'maria', name: 'Maria Bianchi', sex: 'F', lifespan: '1840 – 1905' },
  { slug: 'anna', name: 'Anna Rossi', sex: 'F', lifespan: '1860 – ' },
  { slug: 'pietro', name: 'Pietro Rossi', lifespan: '1863 – 1941' },
]
const index = {
  people: kindred.map((p) => ({ ref: `person/${p.slug}` })),
  families: [
    { ref: 'family/antonio--lucia', partners: [{ person: 'antonio' }, { person: 'lucia' }], children: [{ person: 'giovanni' }] },
    {
      ref: 'family/giovanni--maria',
      partners: [{ person: 'giovanni' }, { person: 'maria' }],
      children: [{ person: 'anna' }, { person: 'pietro', relations: [{ partner: 'giovanni', relation: 'adoptive' }] }],
    },
  ],
}

async function drawing(subject: string, shape: Shape) {
  const drawn = await draw(index, { subject, shape, distance: 2, gaps: true })
  const people = drawn.people.map(({ slug, x, y }): ChartBox => {
    const p = kindred.find((k) => k.slug === slug)!
    return { key: slug, x, y, name: p.name, sex: p.sex, lifespan: p.lifespan, initials: initials(p.name), to: '/ui', subject: slug === subject }
  })
  const gaps = drawn.slots.map((s): ChartBox => ({ key: s.key, x: s.x, y: s.y, name: said.addParent, gap: true }))
  return { ...drawn, boxes: [...people, ...gaps] }
}

// Giovanni's relatives, and, since Maria Bianchi's parents are nobody's record, her
// daughter's ancestors showing where they would go.
export async function charts() {
  return { chart: await drawing('giovanni', 'relatives'), gapped: await drawing('anna', 'ancestors') }
}

export const iconNames = Object.keys(icons) as IconName[]
