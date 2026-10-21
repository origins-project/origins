// A citation as a reference writes it, source#anchor and a grade, in its parts.

export interface Cited {
  ref: string
  source: string
  information: string
  evidence: string
  weight: string
}

// Not assessed, in each position of a grade.
export const unassessed = '-'

export const positions = {
  source: ['O', 'D', 'A'],
  information: ['P', 'S', 'U'],
  evidence: ['D', 'I', 'N'],
  weight: ['0', '1', '2', '3'],
} as const

export function readCitation(text: string): Cited {
  const [ref = '', grade = ''] = text.trim().split(/\s+/)
  const at = (i: number) => (grade[i] && grade[i] !== '-' ? grade[i]! : unassessed)
  return { ref, source: at(0), information: at(1), evidence: at(2), weight: at(3) }
}

export function writeCitation(c: Cited): string {
  if (!c.ref.trim()) return ''
  const weight = c.weight === unassessed ? '' : c.weight
  const grade = `${c.source}${c.information}${c.evidence}${weight}`
  return grade === '---' ? c.ref.trim() : `${c.ref.trim()} ${grade}`
}
