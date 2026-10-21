# Dates

A date is written as the record gives it, and each part is optional. `1834` and
`1 JAN 1834` are different dates: the first says only the year, and Origins never
reads it as the first of January.

```
1834                    the year only
MAR 1834                the month and the year
12 MAR 1834             the full date
12 MAR 1834 09:20       with the time, from a modern certificate
3 MAR                   the day and the month, with the year unknown
NOV                     the month only
3 ? 1850                the day and the year, with the month unknown
ABT 1834                about 1834
EST 1834                estimated from other evidence
CAL 1834                calculated, for example from an age
EST ABT 1834            estimated, and approximate
BEF 1900                before 1900
AFT 1850                after 1850
BET 1830 AND 1835       some time between 1830 and 1835
FROM 1830 TO 1860       the whole of the years from 1830 to 1860
FROM 12 MAR 1834        a period with no recorded end
TO 1860                 a period with no recorded start
1750/51                 a dual year
15 FEB 1700 NY 25 MAR   a year counted from 25 March
JULIAN 14 FEB 1799      in the Julian calendar
44 BCE                  before the common era
ABT 1854 (the winter of the cholera)
(before the war)
```

The months are `JAN`, `FEB`, `MAR`, `APR`, `MAY`, `JUN`, `JUL`, `AUG`, `SEP`, `OCT`,
`NOV` and `DEC`. A number on its own is a year, so a day always comes with its month,
and a day whose month you don't know has `?` in the month's place.

This is the date grammar of GEDCOM 7, with a few additions. A GEDCOM date must have a
year, so Origins exports a date without one as a date phrase.

## Ranges and periods

`BET 1845 AND 1851` is one moment, somewhere in those years, such as a birth that you
can only place between two censuses. `FROM 1830 TO 1860` is the whole of the years, for
a fact that lasted, such as an occupation or a residence. Each end of a range or a
period is a date of its own, with its own qualifier and calendar.

`EST` and `CAL` say how you reached the date, and they go before any other form:
`EST ABT 1850`, `CAL BEF 1850`, `EST BET 1840 AND 1850`.

## Dates without a year

A date without a year, such as `3 MAR` from a worn gravestone, is kept, shown and
searched like any other. Origins leaves it out of everything that compares dates, such
as ages, the order of events and the checks that `origins check` makes, and it doesn't
report the date as incomplete.

## The record's own words

The words that the record uses go in parentheses after the date:
`ABT 1854 (the winter of the cholera)`. When the record gives nothing that a date can
be made of, the words stand alone: `(before the war)`. Origins shows the words and
doesn't compute anything from them, so a date with only words is treated like a date
without a year.

## Calendars

A date in a calendar other than the Gregorian starts with the calendar's name. Origins
keeps the date in the calendar it was written in, and converts it only to compare it
with another date.

| Calendar | Months |
|---|---|
| `JULIAN` | as the Gregorian |
| `HEBREW` | `TSH` `CSH` `KSL` `TVT` `SHV` `ADR` `ADS` `NSN` `IYR` `SVN` `TMZ` `AAV` `ELL` |
| `FRENCH_R` | `VEND` `BRUM` `FRIM` `NIVO` `PLUV` `VENT` `GERM` `FLOR` `PRAI` `MESS` `THER` `FRUC` `COMP` |
| `PERSIAN` | `FARVA` `ORDIB` `KHORD` `TIR` `MORDA` `SHAHR` `MEHR` `ABAN` `AZAR` `DEY` `BAHMA` `ESFAN` |
| `ISLAMIC` | `MUHAR` `SAFAR` `RABIA` `RABIT` `JUMAA` `JUMAT` `RAJAB` `SHAAB` `RAMAD` `SHAWW` `DHUAQ` `DHUAH` |
| `SWEDISH` | as the Gregorian |

```
HEBREW 5 NSN 5594
FRENCH_R 18 BRUM 8
PERSIAN 12 FARVA 1300
ISLAMIC 1 RAMAD 1250
SWEDISH 30 FEB 1712
BET JULIAN 1 JAN 1700 AND 1 JAN 1701
```

Each end of a range can be in a different calendar.

- `FRENCH_R` is the calendar of the French Republic, from year 1 to year 14, and
  Origins refuses a date outside those years.
- `SWEDISH` is the calendar that Sweden kept from March 1700 to February 1712, and
  Origins refuses a date outside those years too.
- `ISLAMIC` is the tabular Hijri calendar. A month that began with a sighting of the
  moon can differ from the table by a day.
- `PERSIAN` is the Solar Hijri calendar.
- In the Hebrew calendar, `ADR` is the first Adar, which only a leap year has. Origins
  writes a date in `ADR` of a common year as `ADS`.

Only Gregorian and Julian dates take a dual year or `BCE`.

## Years that began on another day

A dual year, `15 FEB 1700/01`, gives the year as the record wrote it and the year
counted from 1 January. Use it for a year that began on 25 March and was named for that
day, as in Florence and in England before 1752.

`NY` and a day after the date say that the record counted its years from that day,
and named each year for the 1 January it contains, as the Pisan reckoning did:
`15 FEB 1700 NY 25 MAR`.
