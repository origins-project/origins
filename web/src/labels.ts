// Every label a page shows, in one place. A page draws data and these, and nothing
// else: a label is a short word or two — a title, a column, a button's verb, a field's
// name — and never a sentence.

import type { Kind } from './api/client'

export const app = {
  title: 'Origins',
}

// What a control says it does, where it says it with no word of the page's own.
export const actions = {
  buildDate: 'Build date',
  close: 'Close',
  dismiss: 'Dismiss',
  remove: 'Remove',
}

// How prose being written is shown: the markdown, what it will look like, or both.
export const writing = {
  view: 'View',
  write: 'Write',
  split: 'Side by side',
  preview: 'Preview',
}

// What a picture opened whole says beside it.
export const pictures = {
  depicts: 'In it',
  file: 'File',
  who: 'Who',
  tag: 'Tag',
  edit: 'Edit',
  portraitOf: (name: string) => `Portrait of ${name}`,
}

// What the frame around every page says: the bar, its menu and the search.
export const frame = {
  home: 'Home',
  lists: 'Lists',
  search: 'Search',
  searchKeys: 'Ctrl K',
  keys: { move: '↑ ↓', choose: 'Enter', close: 'Esc' },
  suppose: 'Suppose',
  findings: 'Findings',
  theme: 'Theme',
  light: 'Light',
  dark: 'Dark',
  live: 'Live',
  offline: 'Offline',
  more: 'More',
  hidePrivate: 'Hide private',
  hideLiving: 'Hide living',
  privateHidden: 'Private hidden',
  livingHidden: 'Living hidden',
  supposed: 'Supposed',
  stopSupposing: 'Stop supposing',
  settings: 'Settings',
  info: 'Info',
  partners: ' & ',
}

// What a change says it did, a verb and its object.
export const notices = {
  moved: (head: string) => `Moved to ${head}`,
  drafted: (subject: string) => `Drafted: ${subject}`,
}

// What the forms that change the corpus say: their buttons, the citation a change
// rests on, the record's menu and what a dry run comes to.
export const edit = {
  save: 'Save',
  add: 'Add',
  cancel: 'Cancel',
  remove: 'Remove',
  more: 'More',
  mapping: 'More fields',
  key: 'Key',
  cite: 'Cite this change',
  citation: 'Citation',
  grade: { source: 'Source', information: 'Information', evidence: 'Evidence', weight: 'Weight' },
  unassessed: '—',
  actions: 'Actions',
  rename: 'Rename',
  merge: 'Merge into',
  split: 'Split',
  delete: 'Delete',
  accept: 'Accept',
  reject: 'Reject',
  undo: 'Undo',
  to: 'New name',
  into: 'Into',
  subject: 'Change',
  written: 'Written',
  moved: 'Moved',
  removed: 'Removed',
  problems: 'Problems',
  findings: 'Findings',
  nothing: 'Nothing changes',
}

const english = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']

// The date dialog's words, a calendar and a month named by their keywords, and a date's
// words where a page shows one.
export const dates = {
  title: 'Date',
  shape: 'Shape',
  shapes: {
    one: 'One date',
    before: 'Before',
    after: 'After',
    between: 'Between',
    period: 'Period',
    words: 'Words only',
  },
  start: 'From',
  end: 'And',
  from: 'From',
  to: 'To',
  reached: 'Reached',
  reachedAs: { '': 'Stated', EST: 'Estimated', CAL: 'Calculated' },
  about: 'About',
  calendar: 'Calendar',
  day: 'Day',
  month: 'Month',
  year: 'Year',
  time: 'Time',
  dual: 'Dual year',
  newYear: 'New year',
  bce: 'Before the common era',
  wording: 'Wording',
  written: 'Written',
  use: 'Use this date',
  sorted: 'Sorted as',
  floating: 'Floating',
  // a date as a reader reads it, each keyword of the grammar in words
  spoken: {
    ABT: 'about',
    BEF: 'before',
    AFT: 'after',
    BET: 'between',
    AND: 'and',
    FROM: 'from',
    TO: 'to',
    EST: 'estimated',
    CAL: 'calculated',
    BCE: 'BCE',
    NY: 'year from',
    calendars: {
      GREGORIAN: 'Gregorian',
      JULIAN: 'Julian',
      HEBREW: 'Hebrew',
      FRENCH_R: 'French Republican',
      PERSIAN: 'Persian',
      ISLAMIC: 'Islamic',
      SWEDISH: 'Swedish',
    },
    months: {
      GREGORIAN: english,
      JULIAN: english,
      SWEDISH: english,
      HEBREW: ['Tishrei', 'Cheshvan', 'Kislev', 'Tevet', 'Shevat', 'Adar I', 'Adar II', 'Nisan', 'Iyar', 'Sivan', 'Tammuz', 'Av', 'Elul'],
      FRENCH_R: ['Vendémiaire', 'Brumaire', 'Frimaire', 'Nivôse', 'Pluviôse', 'Ventôse', 'Germinal', 'Floréal', 'Prairial', 'Messidor', 'Thermidor', 'Fructidor', 'Complémentaires'],
      PERSIAN: ['Farvardin', 'Ordibehesht', 'Khordad', 'Tir', 'Mordad', 'Shahrivar', 'Mehr', 'Aban', 'Azar', 'Dey', 'Bahman', 'Esfand'],
      ISLAMIC: ['Muharram', 'Safar', 'Rabi al-Awwal', 'Rabi al-Thani', 'Jumada al-Awwal', 'Jumada al-Thani', 'Rajab', 'Shaban', 'Ramadan', 'Shawwal', 'Dhu al-Qadah', 'Dhu al-Hijjah'],
    },
  },
}

// What the page of a record of any kind says: its tabs and the cards every page has.
export const record = {
  tabs: { overview: 'Overview', history: 'History' },
  notes: 'Notes',
  research: 'To research',
  details: 'Details',
  cited: 'Sources cited',
  files: 'Files',
  history: 'History',
  all: 'All',
  disk: 'On disk',
  edit: 'Edit',
  events: 'Events',
}

// What a family's page says.
export const family = {
  partners: 'Partners',
  children: 'Children',
}

// What a place's page says.
export const place = {
  names: 'Names',
  within: 'Lies within',
  contains: 'Contains',
  addresses: 'Addresses',
  here: 'Recorded here',
  map: 'Map',
  openMap: 'OpenStreetMap',
}

// What a source's page says.
export const source = {
  citations: 'Citations',
  citedBy: 'Cited by',
  kept: 'Kept at',
  covers: 'Covers',
  hypotheses: 'Hypotheses',
}

// What a repository's page says.
export const repository = {
  sources: 'Sources',
  addresses: 'Addresses',
  links: 'Links',
}

// What a note's page says.
export const note = {
  linked: 'Linked from',
}

// What a hypothesis's page says, an edit's operation among it.
export const hypothesis = {
  question: 'Question',
  argument: 'Argument',
  edits: 'Edits',
  supports: 'Supports',
  contradicts: 'Contradicts',
  changes: 'Records it changes',
  suppose: 'Suppose',
  proposedBy: (who: string) => `Proposed by ${who}`,
  status: { open: 'Open', accepted: 'Accepted', rejected: 'Rejected' } as Record<string, string>,
  ops: {
    set: 'Set',
    unset: 'Unset',
    add: 'Add',
    remove: 'Remove',
    create: 'Create',
    delete: 'Delete',
    rename: 'Rename',
    merge: 'Merge',
    split: 'Split',
  } as Record<string, string>,
}

// What the bar says while the tab is supposing, and the draft kept as a hypothesis.
export const keeping = {
  supposing: 'Supposing',
  edits: (n: number) => `${n} ${n === 1 ? 'edit' : 'edits'}`,
  records: (n: number) => `${n} ${n === 1 ? 'record' : 'records'}`,
  base: 'base',
  since: (n: number) => `${n} ${n === 1 ? 'commit' : 'commits'} since`,
  pop: 'Take back',
  discard: 'Discard',
  stop: 'Stop',
  create: 'Create hypothesis',
  drifted: 'Drifted',
  title: 'Create hypothesis',
  question: 'Question',
  slug: 'Slug',
  argument: 'Argument',
  keep: 'Create',
  confidences: { probable: 'Probable', possible: 'Possible', doubtful: 'Doubtful' } as Record<string, string>,
}

// What a person's page says: its tabs, its blocks and cards, a relative by how they are
// related, and the words an event's type is shown by where origins knows the type.
export const person = {
  tabs: {
    overview: 'Overview',
    charts: 'Charts',
    history: 'History',
  },
  life: 'Life',
  familyEvents: 'Family events',
  family: 'Family',
  parents: 'Parents',
  with: 'With',
  siblings: 'Siblings',
  halfSiblings: 'Half-siblings',
  children: 'Children',
  familyPage: 'Family page',
  aka: 'Also',
  kin: {
    parent: { M: 'father', F: 'mother', other: 'parent' },
    sibling: { M: 'brother', F: 'sister', other: 'sibling' },
    halfSibling: { M: 'half-brother', F: 'half-sister', other: 'half-sibling' },
    partner: { M: 'husband', F: 'wife', other: 'partner' },
    child: { M: 'son', F: 'daughter', other: 'child' },
  },
  sex: { M: 'Male', F: 'Female', X: 'Other', U: 'Unknown' } as Record<string, string>,
  yes: 'Yes',
  no: 'No',
}

// What a chart says: its shape, how far it reaches, what it is of, its controls, and
// where a walk stopped short.
export const chart = {
  shape: 'Chart',
  shapes: {
    ancestors: 'Ancestors',
    descendants: 'Descendants',
    both: 'Both',
    relatives: 'All relatives',
    'ancestor-tree': 'Ancestor tree',
    'descendant-tree': 'Descendant tree',
    fan: 'Fan',
    path: 'Relationship',
    timeline: 'Timeline',
  },
  distance: {
    ancestors: 'Generations',
    descendants: 'Generations',
    both: 'Generations',
    relatives: 'Steps',
    'ancestor-tree': 'Generations',
    'descendant-tree': 'Generations',
    fan: 'Generations',
    path: 'Steps',
    timeline: 'Steps',
  },
  of: {
    ancestors: (name: string) => `Ancestors of ${name}`,
    descendants: (name: string) => `Descendants of ${name}`,
    both: (name: string) => `Ancestors and descendants of ${name}`,
    relatives: (name: string) => `Relatives of ${name}`,
    'ancestor-tree': (name: string) => `Ancestor tree of ${name}`,
    'descendant-tree': (name: string) => `Descendant tree of ${name}`,
    fan: (name: string) => `Fan of ${name}`,
    path: (name: string) => `Relationship of ${name}`,
    timeline: (name: string) => `Timeline of the relatives of ${name}`,
  },
  to: 'To',
  relation: 'Relation',
  commonAncestor: 'Common ancestor',
  undated: 'Undated',
  limited: (n: number) => `First ${n} people`,
  addParent: 'Add a parent',
  parent: 'Parent',
  inTree: 'In the tree',
  newPerson: 'Somebody new',
  zoomIn: 'Zoom in',
  zoomOut: 'Zoom out',
  fit: 'Fit',
}

// An event's type as a page shows it, where origins knows the word; any other is shown
// as the record wrote it.
export const eventTypes: Record<string, string> = {
  adoption: 'Adoption',
  annulment: 'Annulment',
  baptism: 'Baptism',
  'bar-mitzvah': 'Bar mitzvah',
  'bat-mitzvah': 'Bat mitzvah',
  birth: 'Birth',
  blessing: 'Blessing',
  burial: 'Burial',
  census: 'Census',
  christening: 'Christening',
  confirmation: 'Confirmation',
  cremation: 'Cremation',
  death: 'Death',
  divorce: 'Divorce',
  education: 'Education',
  emigration: 'Emigration',
  engagement: 'Engagement',
  graduation: 'Graduation',
  immigration: 'Immigration',
  marriage: 'Marriage',
  'marriage-banns': 'Banns',
  military: 'Military service',
  naturalization: 'Naturalization',
  occupation: 'Occupation',
  ordination: 'Ordination',
  probate: 'Probate',
  religion: 'Religion',
  residence: 'Residence',
  retirement: 'Retirement',
  title: 'Title',
  will: 'Will',
}

// An event's type as a page shows it.
export function eventLabel(type: unknown): string {
  const word = typeof type === 'string' ? type : ''
  return eventTypes[word.toLowerCase()] ?? word
}

// What home says: the project, its records counted and what waits.
export const home = {
  mainPerson: 'Main person',
  recent: 'Recent changes',
  surnames: 'Surnames',
  bookmarks: 'Bookmarks',
  findings: 'Findings',
  open: 'Open',
  hypotheses: 'Open hypotheses',
  draft: 'Draft',
  edits: (n: number) => `${n} ${n === 1 ? 'edit' : 'edits'}`,
  records: (n: number) => `${n} ${n === 1 ? 'record' : 'records'}`,
  base: 'base',
  resume: 'Resume',
  discard: 'Discard',
  change: 'Change',
}

// What a list of records says: its columns, its filters and how many it shows.
export const lists = {
  filter: 'Filter',
  shown: (n: number, of: number) => `${n} of ${of}`,
  nothing: 'Nothing matches',
  living: 'living',
  all: 'All',
  livingOnly: 'Living',
  privateOnly: 'Private',
  open: 'Open',
  list: 'List',
  tree: 'Tree',
  until: 'until',
  from: 'from',
  columns: {
    name: 'Name',
    born: 'Born',
    place: 'Place',
    died: 'Died',
    tags: 'Tags',
    partners: 'Partners',
    type: 'Type',
    first: 'First event',
    children: 'Children',
    within: 'Within',
    references: 'References',
    title: 'Title',
    author: 'Author',
    kept: 'Kept at',
    cited: 'Cited',
    sources: 'Sources',
    events: 'Events',
    question: 'Question',
    status: 'Status',
    confidence: 'Confidence',
    proposedBy: 'Proposed by',
    base: 'Base',
  },
}

// What the findings say: the three kinds of them, and what each leads to.
export const findings = {
  all: 'All',
  errors: 'Errors',
  warnings: 'Warnings',
  todo: 'To do',
  filter: 'Filter',
  open: 'Open',
  nothing: 'Nothing found',
  more: (n: number) => `${n} more`,
}

// What Settings says: its sections and what sharing the corpus does.
export const settings = {
  project: 'Project',
  researchers: 'Researchers',
  bookmarks: 'Bookmarks',
  tags: 'Tags',
  tag: 'Tag',
  color: 'Colour',
  priority: 'Priority',
  records: 'Records',
  rename: 'Rename',
  git: 'Git',
  branch: 'branch',
  remotes: 'Remotes',
  remote: 'Remote',
  name: 'Name',
  url: 'URL',
  add: 'Add',
  remove: 'Remove',
  incoming: 'Incoming',
  outgoing: 'Outgoing',
  commits: (n: number) => `${n} ${n === 1 ? 'commit' : 'commits'}`,
  fetch: 'Fetch',
  pull: 'Pull',
  push: 'Push',
  recover: 'Recover',
  recommended: 'recommended',
  ways: { '--adopt': 'Adopt', '--rewrite': 'Rewrite', '--reset': 'Reset' } as Record<string, string>,
  manifest: 'origins.yaml',
  export: 'Export',
  made: 'Made',
  commit: 'Commit',
  changed: 'Uncommitted changes',
  version: 'Version',
  view: 'View',
  whole: 'Everything',
  supposed: 'Supposed',
  notice: 'Notice',
  fetched: (names: string[]) => `Fetched ${names.join(', ')}`,
  pulled: (n: number, from: string) => `Pulled ${n} ${n === 1 ? 'commit' : 'commits'} from ${from}`,
  pushed: (branch: string, to: string) => `Pushed ${branch} to ${to}`,
  recovered: (commit: string) => `Recorded ${commit}`,
  added: (name: string) => `Added ${name}`,
  removed: (name: string) => `Removed ${name}`,
}

// What making a record says.
export const create = {
  new: (kind: string) => `New ${kind.toLowerCase()}`,
  slug: 'Slug',
  given: 'Given names',
  surname: 'Surname',
  name: 'Name',
  title: 'Title',
  partner: 'Partner',
  create: 'Create',
}

// The page that asks the passphrase off the loopback.
export const login = {
  passphrase: 'Passphrase',
  signIn: 'Sign in',
  insecure: 'This connection is not encrypted: the passphrase is sent in clear.',
}

// What each kind is called, one of it and a list of it.
export const kinds: Record<Kind, { one: string; many: string }> = {
  person: { one: 'Person', many: 'People' },
  family: { one: 'Family', many: 'Families' },
  place: { one: 'Place', many: 'Places' },
  source: { one: 'Source', many: 'Sources' },
  repository: { one: 'Repository', many: 'Repositories' },
  note: { one: 'Note', many: 'Notes' },
  hypothesis: { one: 'Hypothesis', many: 'Hypotheses' },
}

// A field's name, keyed by the pattern of its path: the path with every index written
// [] and every key of a mapping *, or any end of it. A path takes the label of the
// longest end of it that has one, so a field named alike everywhere is labelled once,
// as `date`, and one that means something else somewhere is labelled there too, as
// `citations.*.date`.
export const fields: Record<string, string> = {
  address: 'Address',
  addresses: 'Addresses',
  age: 'Age',
  'also-known-as': 'Other names',
  attributes: 'Attributes',
  author: 'Author',
  base: 'Base',
  body: 'Notes',
  bookmarks: 'Bookmarks',
  call: 'Call name',
  'call-number': 'Call number',
  cause: 'Cause',
  children: 'Children',
  citations: 'Citations',
  'citations.*.date': 'Entry date',
  city: 'City',
  color: 'Colour',
  confidence: 'Confidence',
  connector: 'Connector',
  contradicts: 'Contradicts',
  coordinates: 'Coordinates',
  copyright: 'Copyright',
  country: 'Country',
  county: 'County',
  covers: 'Covers',
  date: 'Date',
  depicts: 'Depicts',
  description: 'Description',
  display: 'Written as',
  edits: 'Edits',
  entity: 'Record',
  event: 'Event',
  events: 'Events',
  family: 'Family',
  'family-nickname': 'Family nickname',
  file: 'File',
  from: 'From',
  given: 'Given names',
  group: 'Filed with',
  id: 'Identifier',
  ids: 'Identifiers',
  images: 'Images',
  into: 'Into',
  lang: 'Language',
  lines: 'Lines',
  links: 'Links',
  living: 'Living',
  locality: 'Locality',
  'main-person': 'Main person',
  media: 'Pictures',
  medium: 'Medium',
  name: 'Name',
  nickname: 'Nickname',
  not: 'Did not happen',
  note: 'Note',
  op: 'Operation',
  origin: 'Origin',
  page: 'Page',
  parish: 'Parish',
  partners: 'Partners',
  path: 'Path',
  person: 'Person',
  place: 'Place',
  portrait: 'Portrait',
  postal: 'Postal code',
  prefix: 'Prefix',
  primary: 'Primary',
  priority: 'Priority',
  private: 'Private',
  'proposed-by': 'Proposed by',
  publication: 'Publication',
  region: 'Region',
  relation: 'Relation',
  repositories: 'Kept in',
  repository: 'Repository',
  researchers: 'Researchers',
  role: 'Role',
  'same-as': 'Same as',
  sex: 'Sex',
  'short-title': 'Short title',
  sort: 'Sorted as',
  'sort-date': 'Sort date',
  sources: 'Sources',
  state: 'State',
  status: 'Status',
  street: 'Street',
  suffix: 'Suffix',
  supports: 'Supports',
  surname: 'Surname',
  'surname-prefix': 'Surname prefix',
  tags: 'Tags',
  text: 'Text',
  title: 'Title',
  to: 'To',
  type: 'Type',
  until: 'Until',
  url: 'URL',
  value: 'Value',
  with: 'With',
  within: 'Within',
}

// pattern is a path as the labels are keyed: events[2].date is events[].date, and
// a key of a mapping a page names is put as * by whoever knows it is one.
export function pattern(path: string): string {
  return path.replace(/\[\d+\]/g, '[]')
}

// fieldLabel is the label of the longest end of a path's pattern that has one.
export function fieldLabel(path: string): string | undefined {
  const parts = pattern(path).split('.')
  for (let start = 0; start < parts.length; start++) {
    const end = parts.slice(start).join('.')
    const label = fields[end] ?? fields[end.replace(/\[\]$/, '')]
    if (label !== undefined) return label
  }
  return undefined
}
