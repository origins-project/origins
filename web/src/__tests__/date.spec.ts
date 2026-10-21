import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import {
  calendars,
  emptyDate,
  emptyQualified,
  emptyValue,
  parseDate,
  spoken,
  writeDate,
  type Built,
  type Calendar,
  type Qualified,
  type Value,
} from '../date'

// the tests run in web/, whatever directory npm was started in
const model = readFileSync(join(process.cwd(), '..', 'internal', 'model', 'date_test.go'), 'utf8')

// Every canonical form the model's own test holds the grammar to.
const canonical = [...model.slice(model.indexOf('var canonical'), model.indexOf('\n}\n', model.indexOf('var canonical'))).matchAll(/^\t"(.*)",$/gm)].map(
  (m) => m[1]!,
)

function value(parts: Partial<Value>): Value {
  return { ...emptyValue(parts.calendar), ...parts }
}

function qualified(parts: Partial<Qualified>): Qualified {
  return { ...emptyQualified(), ...parts }
}

function built(parts: Partial<Built>): Built {
  return { ...emptyDate(), ...parts }
}

describe('the date grammar', () => {
  it('finds the canonical forms', () => {
    expect(canonical.length).toBeGreaterThan(50)
  })

  for (const text of canonical) {
    it(`reads ${text} and writes it back`, () => {
      const read = parseDate(text)
      expect(read).toBeDefined()
      expect(writeDate(read!)).toBe(text)
    })
  }

  it('reads what is typed loosely as the corpus writes it', () => {
    expect(writeDate(parseDate('  abt 03 mar   1834  ( so  the priest wrote )')!)).toBe('ABT 3 MAR 1834 (so the priest wrote)')
    expect(writeDate(parseDate('julian 1750/51')!)).toBe('JULIAN 1750/51')
    expect(writeDate(parseDate('GREGORIAN 1834')!)).toBe('1834')
  })

  it('has no reading of what the grammar does not say', () => {
    for (const text of ['', '()', 'MAR MAR', '1 2 3 4', '1750/52', 'BET 1830', '12:3', '1834 (open', 'FROM', 'NY 1 SEP', '1834 NY 25'])
      expect(parseDate(text), text).toBeUndefined()
  })
})

describe('the date dialog', () => {
  const every: [string, Built][] = [
    ['1834', built({ moment: qualified({ value: value({ year: '1834' }) }) })],
    ['12 MAR 1834 09:20', built({ moment: qualified({ value: value({ day: '12', month: 'MAR', year: '1834', time: '09:20' }) }) })],
    ['3 ?', built({ moment: qualified({ value: value({ day: '3' }) }) })],
    ['NOV', built({ moment: qualified({ value: value({ month: 'NOV' }) }) })],
    ['EST ABT 1834', built({ moment: qualified({ reached: 'EST', qualifier: 'ABT', value: value({ year: '1834' }) }) })],
    ['CAL BEF 1850', built({ shape: 'before', moment: qualified({ reached: 'CAL', qualifier: 'BEF', value: value({ year: '1850' }) }) })],
    ['AFT 44 BCE', built({ shape: 'after', moment: qualified({ qualifier: 'AFT', value: value({ year: '44', bce: true }) }) })],
    [
      'BET JULIAN 1 JAN 1700 AND 12 FEB 1700/01',
      built({
        shape: 'between',
        moment: qualified({
          qualifier: 'BET',
          value: value({ calendar: 'JULIAN', day: '1', month: 'JAN', year: '1700' }),
          end: value({ day: '12', month: 'FEB', year: '1700', dual: true }),
        }),
      }),
    ],
    [
      'JULIAN 15 FEB 1700 NY 25 MAR',
      built({ moment: qualified({ value: value({ calendar: 'JULIAN', day: '15', month: 'FEB', year: '1700', newYear: { day: '25', month: 'MAR' } }) }) }),
    ],
    [
      'FROM EST 1830 TO CAL BEF 1860',
      built({
        shape: 'period',
        from: qualified({ reached: 'EST', value: value({ year: '1830' }) }),
        to: qualified({ reached: 'CAL', qualifier: 'BEF', value: value({ year: '1860' }) }),
      }),
    ],
    ['FROM 12 MAR 1834', built({ shape: 'period', from: qualified({ value: value({ day: '12', month: 'MAR', year: '1834' }) }) })],
    ['TO PERSIAN DEY 1357', built({ shape: 'period', to: qualified({ value: value({ calendar: 'PERSIAN', month: 'DEY', year: '1357' }) }) })],
    ['(before the war)', built({ shape: 'words', wording: 'before the war' })],
    ['HEBREW 5 NSN 5594 (on the eve)', built({ moment: qualified({ value: value({ calendar: 'HEBREW', day: '5', month: 'NSN', year: '5594' }) }), wording: 'on the eve' })],
  ]

  for (const [text, date] of every) {
    it(`builds ${text} and reads it back`, () => {
      expect(writeDate(date)).toBe(text)
      expect(parseDate(text)).toEqual(date)
    })
  }

  it('builds a month of every calendar in its own words', () => {
    for (const calendar of Object.keys(calendars) as Calendar[]) {
      for (const month of calendars[calendar]) {
        const date = built({ moment: qualified({ value: value({ calendar, day: '1', month, year: '1712' }) }) })
        const text = writeDate(date)
        expect(parseDate(text), text).toEqual(date)
      }
    }
  })

  it('writes nothing for a date with nothing in it', () => {
    expect(writeDate(emptyDate())).toBe('')
    expect(writeDate(built({ shape: 'period' }))).toBe('')
  })
})

describe('a date as a page says it', () => {
  it('says a moment in words', () => {
    expect(spoken('12 MAR 1834')).toBe('12 March 1834')
    expect(spoken('ABT 1834')).toBe('about 1834')
    expect(spoken('EST BET 1840 AND 1850')).toBe('estimated between 1840 and 1850')
    expect(spoken('3 ? 1850')).toBe('3 ? 1850')
    expect(spoken('44 BCE')).toBe('44 BCE')
  })

  it('says a period from its two ends', () => {
    expect(spoken('FROM 1830 TO 1860')).toBe('from 1830 to 1860')
    expect(spoken('TO 1860')).toBe('to 1860')
  })

  it('names a calendar other than the Gregorian, and the day a year was counted from', () => {
    expect(spoken('JULIAN 15 FEB 1700 NY 25 MAR')).toBe('15 February 1700 year from 25 March Julian')
    expect(spoken('HEBREW 5 NSN 5594')).toBe('5 Nisan 5594 Hebrew')
    expect(spoken('12 FEB 1750/51')).toBe('12 February 1750/51')
  })

  it('leaves what it cannot read as it was written', () => {
    expect(spoken('the spring after the flood')).toBe('the spring after the flood')
  })
})
