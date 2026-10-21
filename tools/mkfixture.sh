#!/usr/bin/env bash
set -euo pipefail
T="$(cd "$(dirname "$0")/.." && pwd)/test"
rm -rf "$T"; mkdir -p "$T"; cd "$T"
w() { mkdir -p "$(dirname "$1")"; cat > "$1"; }

w origins.yaml <<'XX'
project:
  name: Marchena Sólveigarson family
  description: Descendants of Per Andersson and of Bartolomeo Rossi
  main-person: verdier-jean-baptiste
  lang: en
  copyright: © 2024 Ingrid de Vries
  researchers:
    - name: Ingrid de Vries
      addresses:
        - |-
          Lindengracht 112
          1015 KM Amsterdam
      links: [mailto:ingrid@example.org]
    - name: Tomasz Nowak
      note: >-
        Compiled the Papadopoulou branch, 2015–2019, from the registers of
        Trieste.
  bookmarks:
    - people/verdier-jean-baptiste
    - sources/parish-san-lorenzo-baptisms
    - notes/a-london-wedding-of-1859
  tags:
    Emigrants:
      color: '#2a7'
    To do:
      color: '#d33'
      priority: 1
origins:
  format-version: 1
XX

w .gitattributes <<'XX'
/origins.yaml text eol=lf
/people/**/*.md text eol=lf
/families/**/*.md text eol=lf
/places/**/*.md text eol=lf
/sources/**/*.md text eol=lf
/repositories/**/*.md text eol=lf
/notes/**/*.md text eol=lf
/hypotheses/**/*.md text eol=lf
XX

# ---------------------------------------------------------------- generation -3

w people/andersson-per/README.md <<'XX'
---
name:
  given: Per
  surname:
    - name: Andersson
      origin: patronymic
also-known-as:
  - given: Per
    surname: Rapp
    type: soldier
    from: 1704
    sources: [upplands-regemente-rolls#1704-rapp]
sex: M
events:
  - type: birth
    date: EST BET 1670 AND 1685
  - type: residence
    sort-date: 1700
    note: On the farm at Väsby, which the rolls give as where he came from
  - type: military-service
    date: FROM SWEDISH 1704 TO SWEDISH 30 FEB 1712
    note: Upplands regemente, in the Great Northern War
    sources: [upplands-regemente-rolls#1704-rapp]
---

The muster rolls of the Uppland regiment take him on in 1704 as Per Rapp, the name the
company gave the man holding its place, where a farmhand would otherwise be written by
the farm he came from. The rolls were kept in the calendar Sweden followed from 1700 to
1712, and his last pay is dated 30 February 1712, a day no other calendar has, so the
period ends on it.

The rolls take no man under twenty and keep none much past fifty, so his birth is an
estimate between the two. Where he came from they say but not when he left, so Väsby is
placed before the regiment and dated nothing.

He stayed on at Stade when the garrison there was given up, and his daughter Karin was
born there in his fifties or sixties, late but not past belief.
XX

# ---------------------------------------------------------------- generation -2

w people/persdotter-karin/README.md <<'XX'
---
name:
  given: Karin
  surname:
    - name: Persdotter
      origin: patronymic
sex: F
events:
  - type: birth
    date: ABT 1735
    place: stade
  - type: death
    date: 1791
    place: stade
---

Her surname says whose daughter she was, and nothing about a family: Per's son would
have been Persson, and Per himself was Anders's son. She shares no surname with her
father, as she shares none with her own son.
XX

w people/rossi-bartolomeo/README.md <<'XX'
---
name:
  given: Bartolomeo
  surname: Rossi
sex: M
events:
  - type: birth
    date: 12 FEB 1750/51
    place: pian-del-mugnone
  - type: will
    date: 20 APR 1799 NY 25 MAR
  - type: death
    date: 1799
    place: pian-del-mugnone
---

The baptismal entry is dated 12 February 1750 in the Florentine style, where the
year turned on 25 March. In modern reckoning that is 1751, so the record is written
here as a dual date rather than silently converted. Converting it would destroy the
evidence that the original said 1750.

His will was drawn up by a notary from Pisa, who dated it in the Pisan style: the
year 1799 began there on 25 March 1798, so 20 April 1799 is a spring earlier than it
reads, and the will comes before his death as a will has to. The year is kept as
the notary wrote it, with the day it was counted from.
XX

w people/sarti-angela/README.md <<'XX'
---
name:
  given: Angela
  surname: Sarti
sex: F
events:
  - type: birth
    date: EST 1755
  - type: death
    date: 3 MAR
    place: pian-del-mugnone
    sources: [cimitero-san-lorenzo-gravestones#sarti-angela O--]
---

Her death date is the awkward case this fixture exists for. The gravestone gives
"3 marzo" clearly, but the year is worn away entirely — only the first digit, a 1,
survives. The day and month are solid evidence and must be recorded; the year is
genuinely unknown and must not be guessed.

The birth estimate is inferred from Bartolomeo's, on the assumption she was of
comparable age. It is an estimate, not a calculation from any record, which is why
it is EST rather than CAL.
XX

# ---------------------------------------------------------------- generation -1

w people/rossi-caterina/README.md <<'XX'
---
name:
  given: Caterina
  surname: Rossi
also-known-as:
  - given: Katharina
    surname: Lang
    type: married
    from: 1802
sex: F
events:
  - type: birth
    date: 3 APR 1779
    place: pian-del-mugnone
    sources: [parish-san-lorenzo-baptisms#1779-042]
  - type: baptism
    date: 5 APR 1779
    place: firenze
    sources: [parish-san-lorenzo-baptisms#1779-042 OPD]
  - type: death
    date: 1840
    place: hamburg
  - type: burial
    date: 22 ? 1840
    place: hamburg
---

Born at Pian del Mugnone and carried down to San Lorenzo in the city to be baptised,
where her father had his workshop that spring. The register names her and her father,
and the entry is the oldest record of the family in this tree.

She married a Hamburg merchant at Florence in 1802 and went north with him, and every
German record after that calls her Katharina Lang. The married name is dated from the
wedding, so that either spelling finds her and each record shows the name she bore when
it was written.

The burial register's margin, where the month was written, is torn away. The day
survives, and is kept without a month rather than given one.
XX

w people/lang-georg/README.md <<'XX'
---
name:
  given: Johann Georg
  call: Georg
  surname: Lang
sex: M
events:
  - type: birth
    date: CAL 1772
    place: stade
  - type: death
    date: 19 OCT 1841
    place: hamburg
---

Georg was the name he went by, the second of his two given names, as the Rufname
usually was. His birth year is calculated, not estimated: his burial entry states his
age as 69, which fixes the year within one. CAL and EST are different claims and the
distinction is worth keeping — one is arithmetic on a record, the other is a guess.

Stade's register names his mother and gives his father as Hinrich Lang, of whom nothing
else is known. He traded in Florentine silk from Hamburg, which is how he came to marry
there, and he crossed to London in 1834 to stand godfather to his grandson.
XX

# ---------------------------------------------------------------- generation 0

w people/lang-anna-maria/README.md <<'XX'
---
name:
  given: Anna Maria
  surname: Lang
also-known-as:
  - given: Anna Maria
    surname: Verdier
    type: married
    from: 1832
  - given: Anna Maria
    surname: Ershov
    type: married
    from: 1853
sex: F
events:
  - type: birth
    date: ABT 1812
    place: hamburg
    sources: [hamburg-st-michaelis-baptisms#1812-017]
  - type: occupation
    value: dressmaker
    date: FROM 1830 TO 1860
    place: london
    note: In Spitalfields, for a court dressmaker
  - type: death
    date: 14 JUL 1878
    place: london
    age:
      value: 66
      note: Given by Michael, who registered the death.
---

The St Michaelis register survives for these years only as a copy made after the fire
of 1842, which lists her among the children baptised in 1811 and 1812 without a day.
The estimate rests on that and on her stated age of 66 on the 1878 death record, which
is the only document that gives an age at all.

She married twice. Louis died between 1845 and 1851, and she married Ivan Ershov in
1853 — a remarriage after bereavement, not after divorce, which matters when reading
the household in the 1861 census.

She worked as a dressmaker in London from 1830 until 1860, when the order books stop
naming her. That is thirty years held, not one moment somewhere inside them, so the
occupation is dated as a period and not as a range.
XX

w people/lang-joseph/README.md <<'XX'
---
name:
  given: Joseph
  surname: Lang
sex: M
events:
  - type: birth
    date: 2 JAN 1810
    place: hamburg
    sources: [hamburg-st-michaelis-baptisms#1810-003]
  - type: death
    date: 5 JAN 1810
    place: hamburg
---

Died at three days old. The register records a Nottaufe, a baptism at home on the day
of birth by whoever was there, which was given to a child not expected to live.
XX

w people/verdier-louis/README.md <<'XX'
---
name:
  given: Louis
  surname: Verdier
sex: M
events:
  - type: birth
    date: FRENCH_R 12 FLOR 7
    place: lyon
  - type: occupation
    value: silk weaver
    place: london
  - type: death
    date: BET 1845 AND 1851
    place: london
---

His birth was registered at Lyon in the Republican calendar, which is what the
registers of the year VII used. The entry is kept in the reckoning it was written in:
writing 1 May 1799 instead would put a day in the record that the record does not
give. It still sorts before his death, because the ordering is derived and the date
is not.

Louis is named as living in the parish rate book of 1845 and as deceased on the 1851
census return, which brackets his death. No burial entry has been found. A bracketed
range is the honest record here: writing ABT 1848 would invent a precision the evidence
does not support.
XX

w people/ershov-ivan/README.md <<'XX'
---
name:
  given: Иван
  surname:
    - name: Петрович
      origin: patronymic
    - name: Ершов
      primary: true
  lang: ru
also-known-as:
  - given: Ivan
    surname:
      - name: Petrovich
        origin: patronymic
      - name: Ershov
        primary: true
    lang: ru-Latn
sex: M
events:
  - type: birth
    date: JULIAN 3 MAR 1818 (3/15 марта 1818)
    place: nizhny-tagil
  - type: census
    date: 1834
    place: nizhny-tagil
    sources: [nizhny-tagil-revision-1834#ershov-ivan]
  - type: residence
    date: FROM 1851
    place: london
  - type: death
    date: 1870
    place: london
---

The parish book at Nizhny Tagil writes both days, as Russian records did for the
foreign reader: the 3rd by the Julian calendar the Church kept, the 15th by the
Gregorian. The date is kept in the reckoning it was written in, with the book's own
words beside it.

The revision list of 1834, the count of souls owing the poll tax, has him at the
works as Петров сын, Peter's son, which is his patronymic and no surname. He came to
London in 1851 as a clerk to a Russian iron agent.
XX

# ---------------------------------------------------------------- generation 1

w people/verdier-jean-baptiste/README.md <<'XX'
---
name:
  text: Jean Baptiste Verdier, fils de Louis
  given: Jean Baptiste
  call: Jean
  surname: Verdier
  sources: [french-church-london-baptisms#1834-042 OPD]
also-known-as:
  - J. B. Verdier
  - given: John
    surname: Verdia
    type: aka
    sources: [london-marriages-1858#p112 ---1]
sex: M
portrait: photos/1890-portrait.jpg
events:
  - type: birth
    date: 12 MAR 1834
    place: london
    sources: [french-church-london-baptisms#1834-042 OPD3]
  - type: baptism
    date: 14 MAR 1834
    place: london
    with:
      - person: lang-georg
        role: godfather
        sources: [french-church-london-baptisms#1834-042 OPD]
      - name: Mary Ann Collins
        role: godmother
        note: No other entry in the register names her.
    sources: [french-church-london-baptisms#1834-042]
  - type: residence
    date: 1861
    place: london
    address: 14 Fournier Street, Spitalfields
    sources: [census-1861-england#f088 OPD]
    ids:
      gramps: E0007
  - type: death
    date: 3 NOV 1901
    place: london
    with:
      - person: papadopoulou-maria
        role: informant
    media:
      - file: documents/1901-death-notice.jpg
        type: newspaper
        title: A death notice
        note: Cut from a London daily and kept loose in the album.
media:
  - file: photos/1890-portrait.jpg
    title: Jean Baptiste in his fifties
    date: ABT 1890
    place: london
    depicts: [verdier-jean-baptiste]
    note: A cabinet card, the studio's stamp on the back cut away.
  - file: photos/1875-carte-de-visite.jpg
    type: photo
    title: A carte de visite
    date: ABT 1875
ids:
  familysearch:
    - XXXX-YYY
    - id: XXXX-XXX
      note: >
        A duplicate of him somebody else entered, which FamilySearch merged into
        this one in 2016; links to the old one still lead here.
---

He appears as "J. B. Verdier" in the 1861 census and as "John Verdia" on his marriage
certificate. I treat all three as the same man: the father's name matches, the street
matches, and the 1861 return gives his birthplace as London.

The baptism names him as the son of Louis, in the words his name keeps beside its
parts: the father's name is how the French church told one Jean Baptiste Verdier from
another, and no part of his own name. He was Jean at home and John to the clerks.

His godfather was his grandfather Georg, who came over from Hamburg for it, and
[Georg's own page](../../people/lang-georg/README.md) says so without anything being
written there twice. The godmother, Mary Ann Collins, has no record of her own, so she
is named as [the baptism](../../sources/french-church-london-baptisms/README.md#1834-042)
names her until somebody finds one.

Not to be confused with any of the Ershovs, his mother's second family, who are his
half-brother's line and share none of his blood on the Verdier side.

*Research:* his death is dated from the family's memory and from Maria's informing on
it, and no record of it is cited yet.

- [ ] Find his death in the General Register Office index for the December quarter of 1901.
XX

w people/verdier-louisa/README.md <<'XX'
---
name:
  given: Louisa
  surname: Verdier
sex: F
events:
  - type: birth
    date: 1842
    place: london
    sources:
      - french-church-london-baptisms#1842-007
      - census-1861-england#f214
---

The two sources disagree. The French church gives her baptism in 1842; the 1861
census gives her age as 17, which implies a birth in 1843 or 1844.

I follow the church register: it is contemporary with the event, whereas census ages
were routinely reported by whoever answered the door and are frequently wrong by a
year or two. Both citations stay attached to the event so the conflict is visible
rather than buried, and so a later researcher can disagree with me.
XX

w people/ershov-michael/README.md <<'XX'
---
name:
  given: Michael
  surname: Ershov
sex: M
events:
  - type: birth
    date: 1856
    place: london
  - type: emigration
    date: AFT 1880
    note: To Buenos Aires, where he married in 1884; no departure record found.
    tags: [To do]
  - type: death
    date: NOV (just before the feast)
    sources: [claudia-interview-2019#death-michael]
not:
  - type: naturalisation
    date: FROM 1880 TO 1910
    note: Searched the Argentine naturalisation index for those years.
tags: [Emigrants]
---

Half-brother to Jean Baptiste and Louisa — same mother, different father. The tool
should derive that relationship rather than requiring it to be stated.

His death is the second yearless date here, and a weaker one than the gravestone:
the family remembered that word came in November, and nobody remembered the year.
A month with no year is still worth recording — it will match a record when one
turns up — but it supports no arithmetic at all and must never be widened into a
guess at a year. The interview's own words go with the month, since the feast is the
likeliest thing to put a year to it.

That he was never naturalised in Argentina is a finding of its own, from an index
searched over thirty years, and it is written apart from his events so that nothing
reading them can take it for something that happened.

He is tagged with the others who left, and his emigration with what is still to do,
the words being the researcher's own and meaning nothing to origins.
XX

w people/papadopoulou-maria/README.md <<'XX'
---
name:
  given: Μαρία
  surname: Παπαδοπούλου
  lang: el
also-known-as:
  - given: Maria
    surname: Papadopoulou
    lang: el-Latn
  - Maria Papadopulo
  - given: Maria
    surname: Verdier
    type: married
    from: 1858
sex: F
events:
  - type: birth
    date: FEB 1840
    place: trieste
    sources: [trieste-greek-baptisms#1840-018]
  - type: death
    date: AFT 1901
---

Her surname is her father's in the genitive, Παπαδοπούλου, "of Papadopoulos", as a
Greek woman's is: her uncle Petros is Παπαδόπουλος and she never was.

The Italian clerk of the Greek church at Trieste wrote "Papadopulo", and the London
marriage certificate copies him. That spelling is undated because it is a misspelling,
not a name she ever bore; the married name is dated because it is.

Only the month of her birth is recorded — the baptismal entry says "nel mese di
febbraio" with no day. She was still living when Jean Baptiste died in 1901, since she
is named as the informant on his death record, but I have found nothing after that.
XX

w people/benitez-rosa/README.md <<'XX'
---
name:
  given: Rosa
  surname: Benítez
sex: F
events:
  - type: birth
    date: 1862
    place: buenos-aires
  - type: death
    date: 1931
    place: buenos-aires
---
XX

# ---------------------------------------------------------------- generation 2

w people/verdier-anna/README.md <<'XX'
---
name:
  given: Anna
  surname: Verdier
sex: F
events:
  - type: birth
    date: 9 SEP 1860
    place: london
    sources: [census-1861-england#f088 OSI]
---

She married a cotton merchant of Aleppo at Alexandria, where her father's silk trade
had a correspondent, and stayed there.
XX

w people/al-halabi-mahmud/README.md <<'XX'
---
name:
  prefix: أبو يوسف
  given: محمود
  surname:
    - name: علي
      prefix: بن
      origin: patronymic
    - name: حسن
      prefix: بن
      origin: patronymic
    - name: الحلبي
      origin: nisba
      primary: true
  lang: ar
also-known-as:
  - prefix: Abū Yūsuf
    given: Maḥmūd
    surname:
      - name: ʿAlī
        prefix: ibn
        origin: patronymic
      - name: Ḥasan
        prefix: ibn
        origin: patronymic
      - name: al-Ḥalabī
        origin: nisba
        primary: true
    lang: ar-Latn
sex: M
events:
  - type: birth
    date: ABT 1850
    note: At Aleppo, which his nisba names
  - type: occupation
    value: cotton merchant
    place: alexandria
  - type: death
    date: 1915
    place: alexandria
---

His name is built the Arabic way. Abū Yūsuf, father of Yūsuf, is his kunya, said
before the name as a courtesy; ibn ʿAlī ibn Ḥasan is his nasab, the chain of fathers;
and al-Ḥalabī, the man from Aleppo, is his nisba, which is what Alexandria called him
and what he is filed under.
XX

w people/verdier-pauline/README.md <<'XX'
---
name:
  given: Pauline
  surname: Verdier
sex: F
events:
  - type: birth
    date: 4 JUN 1863
    place: london
    attributes:
      - type: fathers-age
        value: 29
  - type: death
    date: 1941
    place: lisboa
  - type: lds-sealing-to-parents
    date: 14 MAR 1996
    family: papadopoulou-maria--verdier-jean-baptiste
    attributes:
      - type: temple
        value: SLAKE
      - type: status
        value: completed
        date: 20 MAR 1996
---

Twin to Pierre. The register records them as first and second born without times, so
birth order is known but not the interval. It gives the father's age as the register's
own column has it, which is said of the birth and makes nobody her father.

She went out to Lisbon as a governess's companion and married there. She was sealed to
her parents by proxy in 1996, long after her death, and the rite is dated when it was
performed.
XX

w people/verdier-pierre/README.md <<'XX'
---
name:
  given: Pierre
  surname: Verdier
sex: M
events:
  - type: birth
    date: 4 JUN 1863
    place: london
  - type: death
    date: 1863
    place: london
---

Twin to Pauline, the second born. He did not survive the year; the burial entry gives
no date beyond the year.
XX

w people/ershov-lawrence/README.md <<'XX'
---
name:
  given: Lawrence Henry
  surname: Ershov
sex: M
events:
  - type: birth
    date: 17 MAR 1866
    place: london
---

Louisa's son, born in her mother's house and registered under her stepfather's surname
with the father not named. Nothing later has been found: no marriage, and no death,
which for a man born in 1866 is a record still to look for rather than a life still
going on.

An index of the parish's births once gave him as a son of Ivan and Anna Maria, reading
the household's surname for his parentage. The register says otherwise, and the link
is kept in their family as disproven, so that nobody makes the same reading again.

He is first cousin to Anna, Pauline and Pierre, and first cousin once removed to João.
XX

w people/galvao-henrique/README.md <<'XX'
---
name:
  given: Henrique
  surname:
    - name: Cerqueira
      prefix: de
      origin: matrilineal
    - name: Galvão
      origin: patrilineal
      primary: true
sex: M
events:
  - type: birth
    date: 1878
    place: lisboa
    sources: [lisboa-santa-isabel-baptisms#1878-091]
  - type: title
    value: Dom
    date: FROM 1878
  - type: death
    date: 1949
    place: lisboa
---

A Portuguese name puts the mother's surname first and the father's last, the other way
round from a Spanish one, and he is filed under the last, Galvão, which his children
carry on.

The son of a family of minor nobility, he bore its Dom from his birth, as every child
of the family did. It is a fact held from a day and never ended, so it is dated as a
period with one end.
XX

w people/ershov-ivan-1885/README.md <<'XX'
---
name:
  given: Iván
  nickname: Vanya
  surname: Ershov
sex: M
events:
  - type: birth
    date: 1885
    place: buenos-aires
  - type: title
    value: Doctor
  - type: death
    date: 1960
    place: buenos-aires
same-as:
  - id: ershov-ivan
    note: A tree on FamilySearch has the two as one man, living from 1818 to 1960.
---

The slug collision case, and the ordinary cause of one: he was named for his
grandfather, born 1818, who already held ershov-ivan. This one is disambiguated by
birth year — a discriminator that means something, rather than a bare numeric suffix.

Everybody called him el doctor, and so does his gravestone, but no degree has been
found and nothing says when he took it. It is held with no date at all, and shown
among his details rather than placed on a timeline it has no place in.

That he and his grandfather are one man is somebody else's claim, kept so that it is
answered rather than forgotten: two people said to be one, which `check` warns of
until they are merged or the claim is taken away. Here the dates take it away, a man
of 142 being no man at all, and the merge is refused besides, the two names
disagreeing.
XX

w people/chen-yuying/README.md <<'XX'
---
name:
  given: 玉英
  surname: 陳
  lang: zh-Hant
  display: '{surname}{given}'
also-known-as:
  - given: Yùyīng
    surname: Chén
    lang: zh-Latn-pinyin
    display: '{surname} {given}'
  - given: Teresa
    type: religious
sex: F
events:
  - type: birth
    date: 1865
    place: wenzhou
  - type: death
    date: 1932
    place: wenzhou
not:
  - type: marriage
    note: Unmarried on her death record, at sixty-seven.
---

Yongfu's mother, baptised Teresa by the mission at Wenzhou. She never married, and the
father is not named in the mission's register nor identified by any other source, so the
family Yongfu was born to has `unknown` beside her: there was somebody, and nobody
knows who. Inventing a record to stand for him would create a person with no evidence
behind them.

Later she took in 陳玉珍, an orphan, on her own. That family has `none` beside her,
which says there was nobody else rather than somebody unknown, and the girl is a name
in it, nobody having looked for her own records.
XX

w people/chen-yongfu/README.md <<'XX'
---
name:
  given: 永福
  surname: 陳
  lang: zh-Hant
  display: '{surname}{given}'
also-known-as:
  - given: Yǒngfú
    surname: Chén
    lang: zh-Latn-pinyin
    display: '{surname} {given}'
  - given: Louis
    type: religious
  - given: Yung-fu
    surname: Ch'en
    lang: zh-Latn-wadegile
    display: '{SURNAME} {given}'
    note: >-
      As the Chinese Labour Corps roll writes him, surname first and in
      capitals.
sex: M
events:
  - type: birth
    date: 3 AUG 1888
    place: wenzhou
  - type: military-service
    date: FROM 1917 TO 1920
    note: The Chinese Labour Corps, in France
  - type: residence
    date: FROM 1920
    place: paris
  - type: death
    date: 1954
    place: paris
---

Registered under his mother's surname, with the father recorded as unknown, and
called her natural son, a word origins keeps as the register wrote it and reads as a
parent of a kind nobody says. One parent known and one unknown; the validator must
accept both as normal rather than as missing data.

He went to France with the Labour Corps in 1917 and stayed in Paris when it was
disbanded. The Corps' roll files every labourer surname first, in the Wade–Giles
spelling the British used, and the way it writes him is kept as a pattern of his
name's parts rather than as a second spelling of it.

The family's story of his father is in [The boatman of
Wenzhou](../../notes/the-boatman-of-wenzhou/README.md), a note kept private while the
family it names is living.
XX

# ---------------------------------------------------------------- generation 3

w people/galvao-joao/README.md <<'XX'
---
name:
  given: João
  surname:
    - name: Verdier
      origin: matrilineal
    - name: Galvão
      origin: patrilineal
      primary: true
sex: M
events:
  - type: birth
    date: 1902
    place: lisboa
  - type: death
    date: 1975
    place: paris
---

His mother's surname first and his father's last, as Portugal writes them. He went to
Paris as a young engineer and stayed.
XX

w people/al-halabi-zaynab/README.md <<'XX'
---
name:
  given: زينب
  surname:
    - name: محمود
      prefix: بنت
      origin: patronymic
    - name: الحلبي
      origin: nisba
      primary: true
  lang: ar
also-known-as:
  - given: Zaynab
    surname:
      - name: Maḥmūd
        prefix: bint
        origin: patronymic
      - name: al-Ḥalabī
        origin: nisba
        primary: true
    lang: ar-Latn
sex: F
events:
  - type: birth
    date: ISLAMIC 3 SHAWW 1304
    place: alexandria
  - type: death
    date: 1960
    place: alexandria
---

Bint Maḥmūd, Maḥmūd's daughter, as ibn is a son's. Her birth was registered at
Alexandria in the Hijri calendar, and is kept in it.
XX

w people/al-halabi-1889/README.md <<'XX'
---
name:
  surname: الحلبي
  lang: ar
also-known-as:
  - surname: al-Ḥalabī
    lang: ar-Latn
events:
  - type: birth
    date: 12 MAY 1889
    place: alexandria
    attributes:
      - type: classification
        value: Stillborn
  - type: death
    date: 12 MAY 1889
    place: alexandria
---

The register classes the birth as stillborn, and it is still the birth that dates the
child, with the death entered beside it. No given name and no sex was recorded, which
is why the slug is descriptive rather than a name and why there is no sex here at
all. A record that says nothing is written by leaving the field out, which is a
different statement from a record setting the sex down as neither.

Birth and death on the same date is legitimate and must not be flagged as a
contradiction by the validator.
XX

w people/modiano-rachel/README.md <<'XX'
---
name:
  given: Rachel
  surname: Modiano
also-known-as:
  - given: Raquel
    surname: Ershov
    type: married
    from: 1934
    note: >-
      Taken on marrying [Iván](../../people/ershov-ivan-1885/README.md), and
      kept as a widow.
  - given: רחל
    surname: מודיאנו
    lang: he
sex: F
events:
  - type: birth
    date: HEBREW 17 IYR 5661
    place: thessaloniki
    sources: [salonica-community-births#5661-073]
  - type: registration
    date: ISLAMIC 3 SAFAR 1319
    place: thessaloniki
  - type: Μετανάστευση
    date: JULIAN 20 AUG 1917
    place: thessaloniki
  - type: death
    date: 1969
    place: buenos-aires
---

Four records in four reckonings, and not one of them converted. Her birth is
entered in the register of the Jewish community of Salonica, which counted in the
Hebrew calendar, and a fortnight later in the Ottoman population register, which
dated its entries in the Hijri one; she left after the fire of 1917, in the Greek
administration's own word for it and on a date it wrote in the Julian calendar it still
kept; and she died under the Gregorian one. The four sort into that order all the same,
because a date is ordered by derivation and stated as written.

The community's register writes her name in Hebrew letters, and that spelling is kept
as another name, in the language it is written in. She went to Buenos Aires, where
the Ladino-speaking community called her Raquel.
XX

w people/wit-geertruida/README.md <<'XX'
---
name:
  given: Geertruida Maria
  surname-prefix: de
  surname: Wit
sex: F
events:
  - type: birth
    date: 1895
    place: amsterdam
    sources: [amsterdam-population-register#de-wit-1895]
  - type: occupation
    value: governess
    place: lisboa
  - type: Burial
    place: lisboa
    media:
      - file: photos/gravestone.jpg
        type: tombstone
        note: Her name alone, the stone too worn to show any dates.
---

Helena's birth mother. Nothing is known of her beyond the population register's card,
which has her leave Amsterdam for Lisbon as a governess, the name on the adoption papers
and a plot in the English cemetery at Lisbon with no date on it — a stub that exists
because a relationship points at her, which is a perfectly ordinary state for a person
in a tree to be in. No death record has been found, and the burial says she died all
the same.

The `de` is a tussenvoegsel, kept apart from the surname so that she sorts under W, as
the Dutch file her.
XX

w people/galvao-helena/README.md <<'XX'
---
name:
  given: Helena
  surname:
    - name: Verdier
      origin: matrilineal
    - name: Galvão
      origin: patrilineal
      primary: true
sex: F
events:
  - type: birth
    date: 2 FEB 1920
    place: lisboa
  - type: adoption
    date: 1923
    place: lisboa
    sources: [claudia-interview-2019#adoption-helena]
  - type: death
    date: 2001
    place: lisboa
media:
  - file: D:/Fotos/helena-1940.jpg
    type: photo
    title: Helena at twenty
    note: Scanned by a cousin, on whose computer it was, and not yet found.
    tags: [To do]
---

Three parents, three kinds of relationship: born to Geertruida, adopted by Pauline, and
stepdaughter to her husband Henrique, who is not named in the adoption. Her birth father
is not named anywhere, so the family she was born into has `unknown` beside her mother.

The adoption is known only from a recorded family interview — the papers themselves
have not been found. An oral source is weaker than a document, but it is a source,
and recording it as one is better than recording the fact with nothing attached.
XX

w people/chen-jiaming/README.md <<'XX'
---
name:
  given: 家明
  surname: 陳
  lang: zh-Hant
  display: '{surname}{given}'
also-known-as:
  - given: Jiāmíng
    surname: Chén
    lang: zh-Latn-pinyin
    display: '{surname} {given}'
sex: M
events:
  - type: birth
    date: 1912
    place: wenzhou
  - type: death
    date: 1980
    place: wenzhou
---

家, the first character of his given name, is the generation name he shares with his
sister, as every child of the generation did. He stayed at Wenzhou when his father went
to France.
XX

w people/chen-jialan/README.md <<'XX'
---
name:
  given: 家蘭
  surname: 陳
  lang: zh-Hant
  display: '{surname}{given}'
also-known-as:
  - given: Jiālán
    surname: Chén
    lang: zh-Latn-pinyin
    display: '{surname} {given}'
sex: F
events:
  - type: birth
    date: 1914
    place: wenzhou
  - type: residence
    date: FROM 1930
    place: paris
  - type: death
    date: 1990
    place: paris
---

She shares the generation name 家 with her brother, and followed her father to Paris.
XX

# ---------------------------------------------------------------- generation 4

w people/galvao-claudia/README.md <<'XX'
---
name:
  given: Cláudia
  surname:
    - name: Chen
      origin: matrilineal
    - name: Galvão
      origin: patrilineal
      primary: true
also-known-as:
  - private: true
    given: Cláudia
    surname:
      - Galvão
      - name: Marchena
        prefix: de
    type: married
    from: 4 SEP 1971
    until: 16 FEB 1983
sex: F
events:
  - type: birth
    date: 1946
    place: paris
---

Married once, and divorced in 1983; with Kwame Ofori since 1991, and not married to
him. Her mother's surname comes first, as her father's family wrote theirs.

She went by her husband's surname for as long as the marriage lasted, and would rather
it were not what anybody outside the family found her under, so the name is marked
private: a view for anybody else leaves it out and keeps the rest of her.
XX

w people/marchena-pablo/README.md <<'XX'
---
name:
  given: Pablo
  surname:
    - name: Marchena
      origin: patrilineal
    - name: Ruiz
      origin: matrilineal
sex: M
events:
  - type: birth
    date: 1945
    place: madrid
  - private: true
    type: illness
    date: 2009
    place: madrid
    note: a heart operation at La Paz
---

His father's surname and then his mother's, the Spanish order. His heart operation is
kept for the family's medical history and marked private, the event and not the man: a
view for anybody else shows him without it.
XX

w people/ofori-kwame/README.md <<'XX'
---
living: false
name:
  given: Kwame
  surname: Ofori
sex: M
events:
  - type: birth
    date: 7 JAN 1950
    place: kumasi
    sources: [ofori-family-account#kwame]
---

Kwame is the day name of a boy born on a Saturday, which 7 January 1950 was, and the
family's account gives the day as the name does. No register of his birth has been
found.

Cláudia says he died in Ghana some years after they parted, and nobody has found the
record. He is marked as not living, which the rule alone would not believe of a man born
in 1950.
XX

w people/sato-yumiko/README.md <<'XX'
---
private: true
name:
  given: 由美子
  surname: 佐藤
  lang: ja
  display: '{surname}{given}'
also-known-as:
  - given: Yumiko
    surname: Satō
    lang: ja-Latn-hepburn
    display: '{surname} {given}'
  - given: ゆみこ
    surname: さとう
    lang: ja-Hira
    display: '{surname} {given}'
sex: F
events:
  - type: birth
    date: 21 JUN 1955
    place: tono
    sources: [tono-koseki#sato-yumiko OPD]
---

Pablo's second wife, who asked not to be in a tree that leaves the family, and so is
marked private. A view for anybody else shows her as a placeholder in her place, as
Pablo's wife and Lucas's mother, and nothing of her own. Nothing else in the corpus
names Tōno, the prefecture it lies in, or the koseki her birth is cited to, so the
same view leaves those out with her.
XX

w people/marchena-marco/README.md <<'XX'
---
name:
  given: Marco
  surname:
    - name: Marchena
      origin: patrilineal
    - name: Chen
      origin: matrilineal
sex: M
events:
  - type: birth
    date: 14 MAR 1975 09:20
    place: madrid
  - type: residence
    date: FROM 2010
    place: kinkerstraat-41
media:
  - private: true
    file: photos/2019-at-home.jpg
    title: At home, with Stefán and Sasha
    date: 2019
    depicts:
      - marchena-marco
      - solveigarson-stefan
      - marchena-solveigarson-sasha
---

A modern birth certificate records the time, so the date carries it. A living person
has no death event at all — not an empty one, not an open range. Absence is the
representation. His second surname is his mother's first, Chen, as Spain takes it.

Where he lives is kept so that the family can write to him, and the address is a
record marked private: a view for anybody else keeps the residence and clears where.
The photograph taken at home is marked private too, and the same view leaves it out,
with the portrait it would otherwise give the three of them.
XX

w people/jonsson-gunnar/README.md <<'XX'
---
name:
  given: Gunnar
  surname:
    - name: Jónsson
      origin: patronymic
sex: M
events:
  - type: birth
    date: 1925
    place: reykjavik
  - type: education
    date: FROM 1948 TO 1952
    place: edinburgh
  - type: death
    date: 1999
    place: reykjavik
---

Jón's son. He read geology at Edinburgh, and met Parvin there.
XX

w people/rahimi-parvin/README.md <<'XX'
---
name:
  given: پروین
  surname: رحیمی
  lang: fa
also-known-as:
  - given: Parvin
    surname: Rahimi
    lang: fa-Latn
sex: F
events:
  - type: birth
    date: PERSIAN 1308
    place: tehran
  - type: education
    date: FROM PERSIAN 1 MEHR 1326 TO PERSIAN KHORD 1330
    place: tehran
    note: The University of Tehran
  - type: death
    date: 2010
    place: reykjavik
---

She studied at the University of Tehran from the autumn of 1947 until the summer of
1951. Both ends come from her transcript, the first from the day the year opened and
the second from the examination session, which gives the month alone. The transcript
was written in the Solar Hijri calendar, and each end of the period is kept in it.
XX

w people/gunnarsdottir-solveig/README.md <<'XX'
---
name:
  given: Sólveig
  surname:
    - name: Gunnarsdóttir
      origin: patronymic
sex: F
events:
  - type: birth
    date: 1953
    place: reykjavik
---

Gunnar's daughter, sharing no surname with her father: an Icelandic name says whose
child you are, not which family. She studied in Copenhagen, where she met Ramesh.
XX

w people/vitthal-ramesh/README.md <<'XX'
---
name:
  given: रमेश
  surname:
    - name: विठ्ठल
      origin: patronymic
  lang: mr
also-known-as:
  - given: Ramesh
    surname:
      - name: Vitthal
        origin: patronymic
    lang: mr-Latn
  - given: रमेश
    surname:
      - name: विठ्ठल
        origin: patronymic
      - name: कासेगावकर
        origin: locative
        primary: true
    lang: mr
    type: birth
    until: 1975
sex: M
events:
  - type: birth
    date: 1950
    place: kasegaon
---

Vitthal is his father's given name, which a Marathi name carries after his own. His
birth record adds Kasegaonkar, the man from Kasegaon, the village the family came from,
which he dropped when he went abroad to study.
XX

w people/solveigarson-stefan/README.md <<'XX'
---
name:
  given: Stefán
  surname:
    - name: Sólveigarson
      origin: matronymic
  sort: '{given} {patronymic}'
sex: M
events:
  - type: birth
    date: 8 NOV 1978
    place: reykjavik
---

Sólveig's son: his name is made from his mother's, a matronymic, which Iceland allows
as freely as a patronymic. Icelanders sort by the given name, and so does his.
XX

# ---------------------------------------------------------------- generation 5

w people/marchena-solveigarson-sasha/README.md <<'XX'
---
name:
  given: Sasha
  surname: [Marchena, Sólveigarson]
sex:
  value: X
  sources: [amsterdam-births-2018#2018-021733 OPD]
events:
  - type: birth
    date: 3 APR 2018
    place: amsterdam
    sources: [amsterdam-births-2018#2018-021733 OPD]
sources: [amsterdam-births-2018#2018-021733]
---

Marco and Stefán's child, born in Amsterdam. The birth certificate names each of them
as "Ouder", parent, and says no more — not whether by birth, by surrogacy or by
adoption — so their family lists the child with the relation unknown, and neither is
held to the age rules a birth would bring.

The same certificate sets the sex down as X, neither male nor female. That is a
statement the record makes, and it is cited as one: a different thing from a record
saying nothing, as the stillborn child's of 1889 does. The certificate is the whole of
what is known so far, and the record cites it as a whole as well as fact by fact.
XX

w people/marchena-lucas/README.md <<'XX'
---
name:
  given: Lucas
  surname:
    - name: Marchena
      origin: patrilineal
    - name: Sato
      origin: matrilineal
sex: M
events:
  - type: birth
    date: 1988
    place: madrid
media:
  - file: photos/1988-christening.jpg
    title: Lucas's christening, in his mother's arms
    date: 1988
    place: madrid
    depicts: [marchena-lucas, sato-yumiko]
---

Pablo's son by his second marriage, and Marco's half-brother. His mother is private
and he is not, so shown to somebody else his page names a placeholder as one of his
parents. The picture of his christening is not private, but she is in it, and a view
for anybody else leaves it out for her sake.
XX

w people/bak-seoyeon/README.md <<'XX'
---
name:
  given: 서연
  surname: 박
  lang: ko
  display: '{surname}{given}'
also-known-as:
  - given: Seo-yeon
    surname: Bak
    lang: ko-Latn
    display: '{surname} {given}'
sex: F
events:
  - type: birth
    date: 1990
    place: busan
---

Surname first, in Hangul and in the Revised Romanization, which writes 박 as Bak
where older spellings had Park. She met Lucas studying in Madrid.
XX

# ------------------------------------------------------------- not yet placed

w people/doe-john/README.md <<'XX'
---
name:
  given: John
  surname: Doe
sex: M
---

A John Doe of Spitalfields turns up in a rate book beside the Verdiers. Nothing yet says
he is anybody's relative, and nothing should until a record does.
XX

# ---------------------------------------------------------------------- families

w families/andersson-per--unknown/README.md <<'XX'
---
partners: [andersson-per, unknown]
children: [persdotter-karin]
---
XX

w families/persdotter-karin--unknown/README.md <<'XX'
---
partners:
  - persdotter-karin
  - name: Hinrich Lang
children: [lang-georg]
---

Stade's register gives the father as Hinrich Lang and says nothing else of him. He is
named in the family as the register names him, which is somebody known, if only by
name, until somebody finds his own records.
XX

w families/rossi-bartolomeo--sarti-angela/README.md <<'XX'
---
partners: [rossi-bartolomeo, sarti-angela]
type: clandestine
events:
  - type: marriage
    date: 1775
  - type: annulment
    date: 1776
---

They first married clandestinely, before two witnesses and without their parish
priest, which since the Council of Trent was no marriage at all, and the curia
declared it null the next year. A clandestine marriage is none of the types origins
knows, so it is written in its own word rather than forced into one of them.

What came after is a family of its own beside this one: a couple who married twice
and one marriage recorded twice look the same from the names, and which it is is the
researcher's to say.
XX

w families/rossi-bartolomeo--sarti-angela-1778/README.md <<'XX'
---
partners: [rossi-bartolomeo, sarti-angela]
children: [rossi-caterina]
events:
  - type: marriage
    date: BEF 1778
---

The family that went on after the annulment. No record of a second marriage survives,
and the only bound is the parish's list of households for 1778, which has them under
one roof, so the date is recorded as a ceiling and nothing more. Whether they married
again before the priest or simply went on living together nothing says, so the family
has no type.

The couple's first family already has their name, so this one is told apart by the
year of its first dated event, which is what origins offers where a family's name is
taken.
XX

w families/lang-georg--rossi-caterina/README.md <<'XX'
---
partners: [lang-georg, rossi-caterina]
type: married
children: [lang-joseph, lang-anna-maria]
events:
  - type: marriage
    date: 1802
    place: firenze
---
XX

w families/lang-anna-maria--verdier-louis/README.md <<'XX'
---
partners: [lang-anna-maria, verdier-louis]
type: married
children:
  - id: verdier-jean-baptiste
    sources: [french-church-london-baptisms#1834-042 OPD]
  - verdier-louisa
events:
  - type: marriage
    date: ABT 1832
    place: london
---

No marriage entry has been found. The date is inferred from Jean Baptiste's birth in
1834 and from Anna Maria appearing as Mrs Verdier in the dressmaker's order book of
1833, so it is bounded rather than recorded.
XX

w families/ershov-ivan--lang-anna-maria/README.md <<'XX'
---
partners: [ershov-ivan, lang-anna-maria]
type: married
children:
  - ershov-michael
  - id: ershov-lawrence
    status: disproven
    note: An index of births read the household's surname for his parentage.
events:
  - type: marriage
    date: 7 MAY 1853
    place: london
---

Anna Maria's second marriage, after Louis's death rather than after a divorce. The
distinction is not stated anywhere in this file — it follows from Louis's death
event preceding this date, and the tool should be able to work that out rather
than requiring a flag.
XX

w families/papadopoulou-maria--verdier-jean-baptiste/README.md <<'XX'
---
partners: [papadopoulou-maria, verdier-jean-baptiste]
type: married
children:
  - verdier-anna
  - verdier-pauline
  - verdier-pierre
  - id: unknown
    note: >-
      Jean Baptiste's death record counts four children of the marriage, and
      three are known.
events:
  - type: marriage
    date: 21 APR 1858
    place: london
    with:
      - person: verdier-jean-baptiste
        role: groom
        age: 24
      - person: papadopoulou-maria
        role: bride
        age: 18
      - person: lang-anna-maria
        role: witness
      - name:
          given: Πέτρος
          surname: Παπαδόπουλος
          lang: el
        role: witness
    sources: [london-marriages-1858#p112]
media:
  - file: photos/1858-wedding-group.jpg
    title: The wedding party
    date: 21 APR 1858
    place: london
    depicts:
      - person: papadopoulou-maria
        region: [0.41, 0.12, 0.58, 0.47]
        note: Named in pencil on the back, as the bride.
      - person: verdier-jean-baptiste
        region: [0.59, 0.1, 0.76, 0.49]
  - file: ../../people/verdier-jean-baptiste/photos/1875-carte-de-visite.jpg
    note: Kept in the album beside the wedding group.
sources: [census-1861-england#f088 OSI]
---

The certificate gives the groom as "John Verdia" and the bride as "Maria Papadopulo".
The first spelling occurs exactly once; the second is the Trieste clerk's, copied from
her baptismal certificate.

The 1861 census, which lists her in his household as his wife, is cited for the
family as a whole rather than for the marriage, which it does not date.

The certificate gives each partner's age, so the partners are named in the marriage as
well as being its partners: an age belongs to one of them, and has to say which. The
witnesses were the groom's mother and the bride's uncle, the one of them with a record
named by it and the other by his name, in the Greek he signed it in.

The photograph of the wedding party is the only picture of Maria there is. It is
kept here because it belongs to the marriage rather than to either of them, and it
says where each of them stands in it, so that her page can show her from it.
XX

w families/al-halabi-mahmud--verdier-anna/README.md <<'XX'
---
partners: [al-halabi-mahmud, verdier-anna]
type: married
children: [al-halabi-zaynab, al-halabi-1889]
events:
  - type: marriage
    date: ISLAMIC 12 RAJAB 1302
    place: alexandria
    sources: [alexandria-sharia-court-1302#marriage-al-halabi]
---

The marriage was recorded by the sharia court at Alexandria, which dated its registers
in the Hijri calendar; the date is kept in it.
XX

w families/galvao-henrique--verdier-pauline/README.md <<'XX'
---
partners: [galvao-henrique, verdier-pauline]
type: married
children:
  - galvao-joao
  - id: galvao-helena
    relation: {galvao-henrique: step, verdier-pauline: adoptive}
events:
  - type: marriage
    date: 11 JUN 1900
    place: lisboa
---
XX

w families/verdier-louisa--unknown/README.md <<'XX'
---
partners: [verdier-louisa, unknown]
children: [ershov-lawrence]
---
XX

w families/benitez-rosa--ershov-michael/README.md <<'XX'
---
partners: [benitez-rosa, ershov-michael]
type: married
children: [ershov-ivan-1885]
events:
  - type: marriage
    date: 1884
    place: buenos-aires
---
XX

w families/ershov-ivan-1885--modiano-rachel/README.md <<'XX'
---
partners: [ershov-ivan-1885, modiano-rachel]
type: married
events:
  - type: marriage
    date: 14 OCT 1934
    place: buenos-aires
---
XX

w families/chen-yuying--unknown/README.md <<'XX'
---
partners: [chen-yuying, unknown]
children:
  - id: chen-yongfu
    relation: natural
    note: 私生子, a natural son, as the mission's register has it.
---

Yuying and a man nobody has named. The family has no type, since nothing says what
there was between them.
XX

w families/chen-yuying--none/README.md <<'XX'
---
partners: [chen-yuying, none]
children:
  - name: 陳玉珍
    relation: adoptive
---

A woman who adopted alone. There was nobody beside her, which is a different
statement from somebody unknown, and the family has no birth child.
XX

w families/chen-yongfu--unknown/README.md <<'XX'
---
partners: [chen-yongfu, unknown]
children: [chen-jiaming, chen-jialan]
---
XX

w families/wit-geertruida--unknown/README.md <<'XX'
---
partners: [wit-geertruida, unknown]
children: [galvao-helena]
---
XX

w families/chen-jialan--galvao-joao/README.md <<'XX'
---
partners: [chen-jialan, galvao-joao]
children: [galvao-claudia]
events:
  - type: residence
    date: FROM 1940
    place: paris
---
XX

w families/jonsson-gunnar--rahimi-parvin/README.md <<'XX'
---
partners: [jonsson-gunnar, rahimi-parvin]
type: married
children: [gunnarsdottir-solveig]
events:
  - type: marriage
    date: 1952
    place: edinburgh
---
XX

w families/galvao-claudia--marchena-pablo/README.md <<'XX'
---
partners: [galvao-claudia, marchena-pablo]
type: married
children: [marchena-marco]
events:
  - type: marriage
    date: 4 SEP 1971
    place: madrid
  - type: divorce
    date: 16 FEB 1983
    place: madrid
---

A family that ended in divorce rather than death. Both events belong to the couple,
which is exactly why they live here and not duplicated across two person files.
XX

w families/galvao-claudia--ofori-kwame/README.md <<'XX'
---
private: true
partners: [galvao-claudia, ofori-kwame]
type: unmarried
events:
  - type: residence
    date: FROM 1991
    place: madrid
---

Cláudia's partner after her divorce. They lived together in Madrid from 1991 and never
married, which the type says: without it, a couple who never married could not be told
from a marriage nobody has found the record of yet. A family with no children, too — a
perfectly ordinary record that a children-centric model has trouble representing at
all.

She would rather their life together stayed in the family, so the family is marked
private: a view for anybody else keeps the two of them as partners and nothing more.
XX

w families/marchena-pablo--sato-yumiko/README.md <<'XX'
---
partners: [marchena-pablo, sato-yumiko]
type: married
children: [marchena-lucas]
events:
  - type: marriage
    date: 1986
    place: madrid
---

Pablo's second marriage, three years after his divorce from Cláudia. The family is not
private, but Yumiko is, and that is enough for a view for anybody else to keep it with
a placeholder for her and to say nothing else about it.
XX

w families/gunnarsdottir-solveig--vitthal-ramesh/README.md <<'XX'
---
partners: [gunnarsdottir-solveig, vitthal-ramesh]
children: [solveigarson-stefan]
---
XX

w families/marchena-marco--solveigarson-stefan/README.md <<'XX'
---
partners: [marchena-marco, solveigarson-stefan]
type: civil-union
children:
  - id: marchena-solveigarson-sasha
    relation: unknown
    sources: [amsterdam-births-2018#2018-021733 OPD]
events:
  - type: civil-union
    date: 19 MAY 2018
    place: amsterdam
    with:
      - person: sato-yumiko
        role: witness
---

Partners rather than husband and wife: a geregistreerd partnerschap, registered at
Amsterdam. This needs no special case in the model, which is the argument for a
partners list over role-named fields.

Marco's stepmother was a witness. She is private and the family is not, so a view for
anybody else keeps the family and names a placeholder as the witness. The words here
are not read, so they say who she is without saying her name.
XX

w families/bak-seoyeon--marchena-lucas/README.md <<'XX'
---
partners: [bak-seoyeon, marchena-lucas]
type: married
events:
  - type: marriage
    date: 2016
    place: madrid
---
XX

# ---------------------------------------------------------------------- places

w places/granducato-di-toscana/README.md <<'XX'
---
name: Granducato di Toscana
also-known-as: [Grand Duchy of Tuscany]
type: state
until: 1860
---

Annexed to the Kingdom of Sardinia in 1860 and absorbed into the Kingdom of Italy in
1861. Records before that date sit under the grand-ducal administration, which is
also why the earliest generation here uses the Florentine year.
XX

w places/toscana/README.md <<'XX'
---
name: Toscana
also-known-as: [Tuscany]
type: region
from: 1861
---
XX

w places/firenze/README.md <<'XX'
---
name: Firenze
also-known-as:
  - name: Florence
    lang: en
  - name: Florenz
    lang: de
    note: As the German guidebooks of the period have it.
type: city
within:
  - place: granducato-di-toscana
    until: 1860
    note: The register heads each volume Granducato di Toscana.
    sources: [parish-san-lorenzo-baptisms#1779-042]
  - place: toscana
    from: 1861
coordinates: 43.7696, 11.2558
ids:
  geonames: 3176959
  wikidata: Q2044
---
XX

w places/pian-del-mugnone/README.md <<'XX'
---
name: Pian del Mugnone
type: hamlet
within:
  - place: granducato-di-toscana
    until: 1860
  - place: toscana
    from: 1861
coordinates: (43°48'N 11°17'E)
addresses:
  - city: Fiesole
    parish: San Michele a Pian del Mugnone
    note: Where the parish registers put it, and the comune that absorbed it.
until: 1897
---

A hamlet that no longer exists as a distinct settlement — it was absorbed into the
comune of Fiesole in 1897 and the name survives only on older maps. Its coordinates
are one of those maps' degrees and minutes, kept as written until somebody reads
them as two angles.

A place can end as well as begin, and references to it from before that date remain
correct. Nothing should rewrite Bartolomeo's birthplace to the modern name.
XX

w places/thessaloniki/README.md <<'XX'
---
name:
  name: Θεσσαλονίκη
  lang: el
  from: 1912
also-known-as:
  - name: Thessaloniki
    lang: el-Latn
    type: romanized
  - name: Selanik
    lang: ota
    until: 1912
  - name: Salonicco
    lang: it
type: city
coordinates: 40.6401, 22.9444
---

Filed under the name it has gone by since it became Greek in 1912. Selanik is the
Ottoman name before that, and Salonicco the one the Italian consulate's papers write.
XX

w places/stade/README.md <<'XX'
---
name: Stade
type: town
coordinates: 53.5997, 9.4767
---

The seat of the Swedish duchies of Bremen and Verden until 1712, when the Danes took it
and the Swedish garrison was given up. Some of the soldiers stayed.
XX

w places/hamburg/README.md <<'XX'
---
name: Hamburg
type: city
coordinates: 53.5511, 9.9937
---
XX

w places/london/README.md <<'XX'
---
name: London
type: city
coordinates: 51.5074, -0.1278
---
XX

w places/lyon/README.md <<'XX'
---
name: Lyon
type: city
coordinates: 45.764, 4.8357
---
XX

w places/trieste/README.md <<'XX'
---
name: Trieste
also-known-as:
  - name: Triest
    lang: de
  - name: Τεργέστη
    lang: el
type: city
coordinates: 45.6495, 13.7768
---
XX

w places/nizhny-tagil/README.md <<'XX'
---
name:
  name: Нижний Тагил
  lang: ru
also-known-as:
  - name: Nizhny Tagil
    lang: ru-Latn
    type: romanized
type: town
coordinates: 57.9194, 59.965
---
XX

w places/alexandria/README.md <<'XX'
---
name:
  name: الإسكندرية
  lang: ar
also-known-as:
  - name: al-Iskandariyya
    lang: ar-Latn
    type: romanized
  - name: Alexandria
    lang: en
type: city
coordinates: 31.2001, 29.9187
---
XX

w places/lisboa/README.md <<'XX'
---
name: Lisboa
also-known-as:
  - name: Lisbon
    lang: en
type: city
coordinates: 38.7223, -9.1393
---
XX

w places/amsterdam/README.md <<'XX'
---
name: Amsterdam
type: city
coordinates: 52.3676, 4.9041
---
XX

w places/kinkerstraat-41/README.md <<'XX'
---
private: true
name: Kinkerstraat 41
type: address
within: amsterdam
---

Marco's address. A house somebody lives in is not for sharing, so the place is
private, and the residence that names it is shown elsewhere without it.
XX

w places/buenos-aires/README.md <<'XX'
---
name: Buenos Aires
type: city
coordinates: '-34.6037, -58.3816'
---
XX

w places/paris/README.md <<'XX'
---
name: Paris
type: city
coordinates: 48.8566, 2.3522
---
XX

w places/wenzhou/README.md <<'XX'
---
name:
  name: 溫州
  lang: zh-Hant
also-known-as:
  - name: Wēnzhōu
    lang: zh-Latn-pinyin
    type: romanized
  - name: Wenchow
    lang: en
    note: As the Treaty Port's customs returns spell it.
type: city
coordinates: 27.9938, 120.6994
---
XX

w places/madrid/README.md <<'XX'
---
name: Madrid
type: city
coordinates: 40.4168, -3.7038
---
XX

w places/kumasi/README.md <<'XX'
---
name: Kumasi
type: city
coordinates: 6.6885, -1.6244
---
XX

w places/edinburgh/README.md <<'XX'
---
name: Edinburgh
type: city
coordinates: 55.9533, -3.1883
---
XX

w places/reykjavik/README.md <<'XX'
---
name: Reykjavík
type: city
coordinates: 64.1466, -21.9426
---
XX

w places/tehran/README.md <<'XX'
---
name:
  name: تهران
  lang: fa
also-known-as:
  - name: Tehran
    lang: fa-Latn
    type: romanized
type: city
coordinates: 35.6892, 51.389
---
XX

w places/kasegaon/README.md <<'XX'
---
name:
  name: कासेगाव
  lang: mr
also-known-as:
  - name: Kasegaon
    lang: mr-Latn
    type: romanized
type: village
coordinates: 17.1167, 74.2
---
XX

w places/busan/README.md <<'XX'
---
name:
  name: 부산
  lang: ko
also-known-as:
  - name: Busan
    lang: ko-Latn
    type: romanized
  - name: Pusan
    lang: ko-Latn
    note: The McCune–Reischauer spelling, before 2000.
type: city
coordinates: 35.1796, 129.0756
---
XX

w places/iwate/README.md <<'XX'
---
name:
  name: 岩手県
  lang: ja
also-known-as:
  - name: Iwate
    lang: ja-Latn-hepburn
    type: romanized
type: prefecture
---
XX

w places/tono/README.md <<'XX'
---
name:
  name: 遠野市
  lang: ja
also-known-as:
  - name: Tōno
    lang: ja-Latn-hepburn
    type: romanized
type: city
within: iwate
coordinates: 39.3275, 141.5333
---

Named by Yumiko Satō's birth and by nothing else, and the only place that names Iwate.
Neither is private, and with her hidden neither is named by anything left, so a hidden
view drops Tōno and then the prefecture.
XX

# --------------------------------------------------------------------- sources

w sources/parish-san-lorenzo-baptisms/README.md <<'XX'
---
title: Basilica di San Lorenzo, Firenze — Baptismal Register 1770-1790
short-title: S. Lorenzo, Battesimi 12
type: parish-register
covers:
  - events: [baptism]
    date: FROM 1770 TO 1790
    place: firenze
repositories:
  - repository: archivio-diocesano-di-firenze
    call-number: ADF/SL/B/12
  - repository: archivio-diocesano-di-firenze
    call-number: S. Lorenzo, Battesimi XII
    note: The shelf mark before the archive was reordered, still on the spine.
  - repository: archivio-di-stato-di-firenze
    call-number: Film 1903581
    medium: microfilm
citations:
  1779-042:
    event: baptism
    page: c. 88r, n. 112
    date: 5 APR 1779
    text: >
      Caterina, figlia di Bartolomeo Rossi e Angela Sarti, del popolo di San
      Michele a Pian del Mugnone, nata il 3 aprile 1779 e battezzata il 5
      detto.
    with:
      - person: rossi-caterina
        role: child
      - person: rossi-bartolomeo
        role: father
    images:
      - file: scans/1779-042.jpg
        region: [0.06, 0.38, 0.94, 0.61]
      - scans/1779-042-continued.jpg
    note: >
      *Translation, en:* Caterina, daughter of Bartolomeo Rossi and Angela
      Sarti, of the parish of San Michele at Pian del Mugnone, born on 3 April
      1779 and baptised on the 5th.
---

The hand is consistent through 1781 and then changes. The earlier scribe underlines
each child's surname, <u>Rossi</u>, which the transcriptions here leave out.

Children of the parishes outside the walls were often brought into the city to be
baptised where a parent worked, which is why a child of Pian del Mugnone is entered
here and not at San Michele.
XX

w sources/trieste-greek-baptisms/README.md <<'XX'
---
title: San Nicolò dei Greci, Trieste — Baptismal Register 1835-1845
type: parish-register
citations:
  1840-018:
    text: >
      Μαρία, θυγάτηρ του Γεωργίου
      Παπαδόπουλου, εβαπτίσθη τον
      Φεβρουάριον 1840.
---

The Greek Orthodox community's register, written in Greek with an Italian index the
clerk kept for the Austrian authorities. The register gives only the month, which is
why Maria's birth is recorded to month precision rather than estimated to a day — the
imprecision is in the source and belongs in the record.
XX

w sources/london-marriages-1858/README.md <<'XX'
---
title: General Register Office, Marriage Certificate, Whitechapel, 1858
type: civil-registration
repositories: [the-national-archives]
citations:
  p112:
    text: >
      21 April 1858 — John Verdia, 24, bachelor, silk weaver, father Louis
      Verdier; Maria Papadopulo, 18, spinster, father George Papadopulo.
      Witnesses Anna Maria Ershov, Πέτρος Παπαδόπουλος.
    links: [https://www.example.org/gro/marriages/1858/whitechapel/112]
links: [https://www.example.org/gro/marriages/1858/whitechapel]
---

The stated ages agree with the baptismal records for both parties, which is part of
why I am confident the Verdia and Papadopulo spellings are clerical.
XX

w sources/cimitero-san-lorenzo-gravestones/README.md <<'XX'
---
title: Cimitero di San Lorenzo — surviving gravestones, photographed 2018
type: gravestone
repositories: [papers-of-ingrid-de-vries]
citations:
  sarti-angela:
    text: >
      ANGELA SARTI / MORI IL 3 MARZO 1... / DI ANNI [illegible]
    images: [scans/sarti-angela.jpg]
---

The stone is broken across the year and the lower third is lost. The day and month
are cut deeply and legible; the year survives only as a leading 1, and the age at
death is gone entirely.

This is the source behind the yearless gravestone date in the project. Photographing
it again would not help — the stone is not weathered, it is broken.
XX

w sources/census-1861-england/README.md <<'XX'
---
title: Census of England and Wales, 1861 — Christ Church, Spitalfields
type: census
repositories: [the-national-archives]
citations:
  f088:
    text: >
      14 Fournier Street. J. B. Verdier, head, married, 27, silk weaver, born
      London; Maria, wife, 20, born Trieste, Austria; Anna, daughter, 7
      months, born Spitalfields.
    ids:
      gramps:
        - id: C0012
          note: Its ID in the Gramps tree the census was first entered in.
  f214:
    text: >
      Louisa Verdier, daughter, 17, dressmaker, born London.
---

Ages in this census were reported by whoever was present and are unreliable to
within a year or two. Where it conflicts with a parish register, the register wins —
but the conflicting citation stays attached so the disagreement is visible.
XX

w sources/claudia-interview-2019/README.md <<'XX'
---
private: true
title: Recorded interview with Cláudia Chen Galvão, March 2019
type: interview
author: Ingrid de Vries
repositories: [papers-of-ingrid-de-vries]
citations:
  adoption-helena:
    text: >
      "My aunt always knew she was adopted. It was 1923, she was three, and
      the papers went through in Lisbon — her birth mother was a Dutch
      governess."
  death-michael:
    text: >
      "The letter came in November. My grandmother always said November,
      because it was just before the feast. But what year, I could not tell
      you."
---

An oral source, and the weakest kind of evidence in this project. It is recorded as
a source anyway: an assertion with a weak citation is better than the same assertion
with none, and the weakness is visible rather than hidden.

It is a living woman talking about her family, recorded for the family, and so it is
private: a view for anybody else drops it, and the citations of it with it.
XX

w sources/amsterdam-births-2018/README.md <<'XX'
---
title: Gemeente Amsterdam, Akte van geboorte, 2018
type: civil-registration
citations:
  2018-021733:
    text: >
      Sasha Marchena Sólveigarson, geboren 3 april 2018 te Amsterdam.
      Geslacht: X. Ouder: Marco Marchena Chen, geboren te Madrid. Ouder:
      Stefán Sólveigarson, geboren te Reykjavík.
---
XX

w sources/tono-koseki/README.md <<'XX'
---
title: 遠野市 戸籍謄本, 佐藤家
type: civil-registration
citations:
  sato-yumiko:
    text: >
      昭和三十年六月二十一日 岩手県遠野市で出生。
---

An extract of the household register (koseki) of the Satō family at Tōno, which dates
her birth in the Shōwa era, its thirtieth year.
XX

w sources/french-church-london-baptisms/README.md <<'XX'
---
title: French Protestant Church of London, Threadneedle Street — Baptisms 1830-1849
type: parish-register
repositories:
  - repository: the-national-archives
    call-number: RG 4
citations:
  1834-042:
    text: >
      Jean Baptiste, fils de Louis Verdier et d'Anna Maria Lang, né le 12
      mars 1834 et baptisé le 14. Parrain Georg Lang, aïeul maternel;
      marraine Mary Ann Collins.
  1842-007:
    text: >
      Louisa, fille des mêmes, baptisée le 19 mars 1842.
---
XX

w sources/hamburg-st-michaelis-baptisms/README.md <<'XX'
---
title: Hauptkirche St. Michaelis, Hamburg — Taufregister 1805-1815
type: parish-register
repositories: [staatsarchiv-hamburg]
citations:
  1810-003:
    text: >
      Joseph, Sohn des Kaufmanns Georg Lang und der Katharina geb. Rossi,
      geboren den 2. Januar 1810 und am selben Tage nothgetauft.
  1812-017:
    text: >
      Anna Maria, Tochter des Kaufmanns Georg Lang.
---

The register for these years survives only as the copy made after the fire of 1842,
which lists the children of 1811 and 1812 without their days.
XX

w sources/lisboa-santa-isabel-baptisms/README.md <<'XX'
---
title: Paróquia de Santa Isabel, Lisboa — Registo de baptismos 1875-1880
type: parish-register
repositories: [arquivo-nacional-torre-do-tombo]
citations:
  1878-091:
    text: >
      Henrique, filho legítimo de Dom Manuel de Cerqueira Galvão e de Dona
      Maria do Carmo de Cerqueira.
---
XX

w sources/nizhny-tagil-revision-1834/README.md <<'XX'
---
title: Ревизская сказка, Нижнетагильский завод, 1834
type: census
repositories: [gaso]
citations:
  ershov-ivan:
    text: >
      Иван Петров сын Ершов, 16 лет.
---

The eighth revision, the count of male souls owing the poll tax, at the Nizhny Tagil
works. It gives a man's age at the count and his father's name, which is all that
dates his birth before the parish book was found.
XX

w sources/amsterdam-population-register/README.md <<'XX'
---
title: Bevolkingsregister Amsterdam, 1893-1939
type: census
repositories: [stadsarchief-amsterdam]
citations:
  de-wit-1895:
    text: >
      Wit, Geertruida Maria de, geb. Amsterdam 1895; vertrokken naar Lissabon,
      gouvernante.
links: [https://www.example.org/stadsarchief/bevolkingsregister]
---
XX

w sources/salonica-community-births/README.md <<'XX'
---
title: Jewish Community of Salonica — Register of Births 5655-5665
type: parish-register
repositories: [cahjp]
citations:
  5661-073:
    text: >
      רחל בת אברהם מודיאנו, י"ז אייר תרס"א.
---
XX

w sources/alexandria-sharia-court-1302/README.md <<'XX'
---
title: محكمة الإسكندرية الشرعية — سجل عقود الزواج، ١٣٠٢
type: court-record
repositories: [dar-al-wathaiq]
citations:
  marriage-al-halabi:
    text: >
      محمود بن علي بن حسن الحلبي و آنا بنت جان
      بابتيست فردييه.
---

The Alexandria sharia court's register of marriage contracts for the Hijri year 1302,
now at the Egyptian National Archives in Cairo.
XX

w sources/ofori-family-account/README.md <<'XX'
---
title: The Ofori family's account, as told at Kumasi, 2012
type: interview
author: Ingrid de Vries
citations:
  kwame:
    text: >
      "He was born on the Saturday after the New Year, in 1950, and so he is
      Kwame."
---

An oral source, and in Ghana the usual one for a birth before the registration of
births was made general: the day of the week a child was born on is in the name, and
the family keeps the rest.
XX

w sources/upplands-regemente-rolls/README.md <<'XX'
---
title: Upplands regemente, generalmönsterrullor 1700-1712
type: military
repositories: [krigsarkivet]
citations:
  1704-rapp:
    text: >
      Per Rapp, från Väsby, 22 år.
links: [https://www.example.org/krigsarkivet/rullor/uppland]
---

A soldier took the name the company kept for his place in it, and kept it while he
served.
XX

# ---------------------------------------------------------------- repositories

w repositories/archivio-diocesano-di-firenze/README.md <<'XX'
---
name: Archivio Diocesano di Firenze
type: archive
addresses:
  - lines: |
      Piazza San Giovanni 3
      50129 Firenze
    street: Piazza San Giovanni 3
    city: Firenze
    postal: 50129
    country: Italia
    links:
      - tel:055 213456
      - url: tel:055 213457
        type: fax
  - lines: |
      Via dei Servi 15
      50122 Firenze
    date: TO 1998
links:
  - https://www.example.org/archivio-diocesano
  - mailto:archivio@example.org
---

Open Tuesday and Thursday mornings. Registers are ordered a day ahead, and
photographs without flash are allowed. Letters written before the archive moved in
1998 give the old address in Via dei Servi.
XX

w repositories/archivio-di-stato-di-firenze/README.md <<'XX'
---
name: Archivio di Stato di Firenze
type: archive
addresses:
  - |
    Viale Giovine Italia 6
    50122 Firenze
links: [https://www.example.org/archivio-di-stato]
---

The parish registers filmed for the diocese are here on microfilm, so a visit is
needed only for what the film cuts off at the margin.
XX

w repositories/papers-of-ingrid-de-vries/README.md <<'XX'
---
private: true
name: Ingrid de Vries's papers
type: private-collection
addresses:
  - |
    Lindengracht 112
    1015 KM Amsterdam
---

The photographs and the recordings Ingrid de Vries made for this research, kept at her
home. A home is not for sharing, so the record is private, and a view for anybody
else leaves the sources it keeps with no repository.
XX

w repositories/the-national-archives/README.md <<'XX'
---
name: The National Archives
type: archive
addresses:
  - city: Kew
    country: United Kingdom
---
XX

w repositories/staatsarchiv-hamburg/README.md <<'XX'
---
name: Staatsarchiv Hamburg
type: archive
addresses:
  - city: Hamburg
    country: Deutschland
---
XX

w repositories/stadsarchief-amsterdam/README.md <<'XX'
---
name: Stadsarchief Amsterdam
type: archive
addresses:
  - city: Amsterdam
    country: Nederland
---
XX

w repositories/arquivo-nacional-torre-do-tombo/README.md <<'XX'
---
name: Arquivo Nacional Torre do Tombo
type: archive
addresses:
  - city: Lisboa
    country: Portugal
---
XX

w repositories/gaso/README.md <<'XX'
---
name: Государственный архив Свердловской области
type: archive
addresses:
  - city: Екатеринбург
    country: Россия
---

The State Archive of Sverdlovsk Oblast, which keeps the revision lists of the Urals
works.
XX

w repositories/cahjp/README.md <<'XX'
---
name: Central Archives for the History of the Jewish People
type: archive
addresses:
  - city: Jerusalem
---
XX

w repositories/dar-al-wathaiq/README.md <<'XX'
---
name: دار الوثائق القومية
type: archive
addresses:
  - city: القاهرة
    country: مصر
---

The Egyptian National Archives, which keep the registers of the sharia courts.
XX

w repositories/krigsarkivet/README.md <<'XX'
---
name: Krigsarkivet
type: archive
addresses:
  - city: Stockholm
    country: Sverige
---
XX

# ----------------------------------------------------------------------- notes

w notes/a-london-wedding-of-1859/README.md <<'XX'
---
title: A London wedding of 1859
events:
  - type: marriage
    date: 3 OCT 1859
    place: london
    with:
      - person: lang-anna-maria
        role: witness
      - name:
          given: Jan
          surname: Kowalski
        role: groom
      - name:
          given: Bridget
          surname: Murphy
        role: bride
---

*Research:* Anna Maria Ershov witnessed this wedding, and nobody else in it is anybody
the tree has yet. The bride was one of the girls who sewed for her, which is the only
reason to keep it.

- [ ] Ask St Leonard's, Shoreditch, for the banns of September 1859.
XX

w notes/the-boatman-of-wenzhou/README.md <<'XX'
---
private: true
title: The boatman of Wenzhou
---

What Yongfu's granddaughter remembers being told: that his father kept a boat on the
Ou river, was married, paid for Yongfu's schooling at the mission and was never named
in the house. Nobody has looked for him, and the family the story names still lives in
Wenzhou, so the note is private.
XX

# ------------------------------------------------------------------ hypotheses

w hypotheses/louisa-born-1844/README.md <<'XX'
---
status: rejected
base: 8f78553
confidence: possible
supports: [census-1861-england#f214 OSI]
contradicts: [french-church-london-baptisms#1842-007]
edits:
  - op: set
    entity: person/verdier-louisa
    path: events[0].date
    value: 1844
---

# Was Louisa born in 1844 rather than 1842?

The French church has her baptism on 19 March 1842. The 1861 census gives her age
as 17, which would put her birth in 1843 or 1844. Both cannot be right.

I rejected this. The register is contemporary with the event and written by the
minister who performed the baptism; the census age was reported by whoever answered
the door and is wrong by a year or two throughout this return — Maria's own age is
given as 20 when she was 21.

The hypothesis is kept rather than deleted because the conflict is real and a future
researcher will hit the same census page and wonder. This file is the answer to
"yes, I saw that, and here is why I went the other way".

**What would reopen it:** a death record giving her age, or a marriage record naming
her age. Either would be independent of both existing sources.
XX

w hypotheses/unrecorded-child-1845/README.md <<'XX'
---
status: open
base: 8f78553
confidence: possible
supports: [french-church-london-baptisms#1842-007]
edits:
  - op: create
    entity: person/verdier-unknown-1845
    value:
      events:
        - date: ABT 1845
          place: london
          type: birth
      name:
        surname: Verdier
  - op: add
    entity: family/lang-anna-maria--verdier-louis
    path: children
    value: verdier-unknown-1845
---

# Was there a further child between 1842 and Louis's death?

Louis and Anna Maria had Jean Baptiste in 1834 and Louisa in 1842. Louis did not die
until somewhere between 1845 and 1851, and Anna Maria was still in her early thirties,
so a birth around 1845 or 1846 is likely.

The French church's baptismal volume for 1844 to 1849 is missing, and so is the burial
volume of the parish for the same years, so a child born and lost in that window would
leave no trace in either.

This is an argument from a gap, which is the weakest kind there is — it proposes a
person on the strength of a pattern and nothing else. It stays open and unapplied. It
exists to record that the gap was noticed and to say where to look, not to assert
that the child existed.

**What would settle it:** the second page of the household's 1851 census return,
which would list any living child. The first gives Anna Maria as a widow, and the
second is lost from the film.
XX

# The attachments are real JPEGs rather than empty files, so that the pages showing
# them can actually be looked at. Two placeholders, carried here as base64 because a
# fixture script must not need a drawing program to run: a head and shoulders stands
# for a photograph of a person, a ruled page for a picture of paper.

PORTRAIT_JPEG='/9j/2wCEABcQERQRDhcUEhQaGBcbIjklIh8fIkYyNSk5UkhXVVFIUE5bZoNvW2F8Yk5QcptzfIeL
kpSSWG2grJ+OqoOPko0BGBoaIh4iQyUlQ41eUF6NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2N
jY2NjY2NjY2NjY2NjY2NjY2NjY2Njf/AABEIAHgAWgMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAA
AAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGh
CCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hp
anN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV
1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQAC
AQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXx
FxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqS
k5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T1
9vf4+fr/2gAMAwEAAhEDEQA/AN2iiipGFFFFABRRRQAUUUUAFFFFABRRRQAUUUUAFFFFABRUUs6x
nHVvSoftT+i0AW6KrJdc4cY9xVgHIyKAFooooAKKKKACiiigAqOZ9kZI69BUlV7vO1fTPNAFUnJy
aKKKQwqxayHdsJ4PSq9Phz5y7euaAL9FFFMQUUUUAFFFFABUcyeZGR36ipKKAM0jBwaKuyQrJyeG
9RUJtWzwwI96QyCrFrHzvI4HSnJaqOXOfapwMDAoAWiiimIKKKKACiiigAooooAKKKKACiiigAoo
ooAKKKKACiiigAooqCS5VeE+Y/pQBPRVFp5G/iwPam+Y/wDfb86QzQoqgJZAeHP481Kl0R98Z9xQ
BaopqsrjKnIp1MQUUUUAFFFV7qTACA8nr9KAI5py5wpwv86hoopDCiiigAooooAcjshyp/8Ar1dj
kEibh+NUKkhk8twc8HrQBeooopiCqErbpWPv2q/WbSGFFFFABRRRQAUUUUAFFFFAF6Bt0K+3FSVB
a/6o/Wp6Yj//2Q==
'

DOCUMENT_JPEG='/9j/2wCEABcQERQRDhcUEhQaGBcbIjklIh8fIkYyNSk5UkhXVVFIUE5bZoNvW2F8Yk5QcptzfIeL
kpSSWG2grJ+OqoOPko0BGBoaIh4iQyUlQ41eUF6NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2NjY2N
jY2NjY2NjY2NjY2NjY2NjY2NjY2Njf/AABEIAHwAYAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAA
AAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGh
CCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hp
anN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV
1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQAC
AQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXx
FxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqS
k5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T1
9vf4+fr/2gAMAwEAAhEDEQA/ALKI0jhEGWPQVN9iuP8Ann/48KLH/j7T8f5VrVBRk/Yrj/nn/wCP
Cj7Fcf8APP8A8eFa1FAjJ+xXH/PP/wAeFH2K4/55/wDjwrWooAyfsVx/zz/8eFH2K4/55/8AjwrW
ooAyfsVx/wA8/wDx4UfYrj/nn/48K1qKAMn7Fcf88/8Ax4VC6NG5Rxhh1FblZN9/x9v+H8qBhY/8
fafj/KtR3VFLMcAVl2P/AB9p+P8AKtUgMMMAR6GgRF9qh/v/AKGj7VD/AH/0NP8AKj/55p/3yKPK
j/55p/3yKAGfaof7/wCho+1Q/wB/9DT/ACo/+eaf98ijyo/+eaf98igBn2qH+/8AoaPtUP8Af/Q0
/wAqP/nmn/fIo8qP/nmn/fIoAZ9qh/v/AKGj7VD/AH/0NP8AKj/55p/3yKPKj/55p/3yKAES4idg
qtkn2NZt9/x9v+H8q1BGinKooPqBWXff8fb/AIfyoGFj/wAfafj/ACrUkcRoXbOB6Vl2P/H2n4/y
rWoEVvtsf91/yFH22P8Auv8AkKs0UAVvtsf91/yFH22P+6/5CrNFAFb7bH/df8hR9tj/ALr/AJCr
NFAFb7bH/df8hR9tj/uv+QqzRQBBHdJI4RQ2T61Qvv8Aj7f8P5VrVk33/H2/4fyoGFj/AMfafj/K
tSRiiFlUsR2FZdj/AMfafj/KtagRW+0yf8+7/r/hR9pk/wCfd/1/wqzRQBW+0yf8+7/r/hR9pk/5
93/X/CrNFAFb7TJ/z7v+v+FOjnd3CtCyg9zU9FABRRRQAVk33/H2/wCH8q1qyb7/AI+3/D+VABY/
8fafj/KtSQuEJjALdgay7H/j7T8f5VrUAVt93/zyT8//AK9G+7/55J+f/wBerNFAFbfd/wDPJPz/
APr0b7v/AJ5J+f8A9erNFAFbfd/88k/P/wCvRvu/+eSfn/8AXqzRQBW33f8AzyT8/wD69G+7/wCe
Sfn/APXqzRQBBG1wXAkRQvcj/wDXVC+/4+3/AA/lWtWTff8AH2/4fyoGFj/x9p+P8q1JA5QiMgN2
JrLsf+PtPx/lWtQIrbLv/nqn5f8A1qNl3/z1T8v/AK1WaKAK2y7/AOeqfl/9ajZd/wDPVPy/+tVm
igCtsu/+eqfl/wDWo2Xf/PVPy/8ArVZooArbLv8A56p+X/1qNl3/AM9U/L/61WaKAII1uA4MjqV7
gf8A6qoX3/H2/wCH8q1qyb7/AI+3/D+VAwsf+PtPx/lWpIpdCqsVJ7isux/4+0/H+Va1Ait9mk/5
+H/X/Gj7NJ/z8P8Ar/jVmigCt9mk/wCfh/1/xo+zSf8APw/6/wCNWaKAK32aT/n4f9f8adHA6OGa
ZmA7Gp6KACiiigArJvv+Pt/w/lWtWTff8fb/AIfyoAbausdwrucKM5P4Vofbbf8A56f+OmsmigZr
fbbf/np/46aPttv/AM9P/HTWTRQBrfbbf/np/wCOmj7bb/8APT/x01k0UAa3223/AOen/jpo+22/
/PT/AMdNZNFAGt9tt/8Anp/46aPttv8A89P/AB01k0UAa3223/56f+Oms+6dZLhnQ5U4wfwqGigD
/9k=
'

img() { mkdir -p "$(dirname "$2")"; printf '%s' "$1" | base64 -d > "$2"; }

img "$PORTRAIT_JPEG" people/verdier-jean-baptiste/photos/1875-carte-de-visite.jpg
img "$PORTRAIT_JPEG" people/verdier-jean-baptiste/photos/1890-portrait.jpg
img "$PORTRAIT_JPEG" families/papadopoulou-maria--verdier-jean-baptiste/photos/1858-wedding-group.jpg
img "$PORTRAIT_JPEG" people/marchena-marco/photos/2019-at-home.jpg
img "$PORTRAIT_JPEG" people/marchena-lucas/photos/1988-christening.jpg
img "$DOCUMENT_JPEG" people/verdier-jean-baptiste/documents/1901-death-certificate.jpg
img "$DOCUMENT_JPEG" people/verdier-jean-baptiste/documents/1901-death-notice.jpg
img "$PORTRAIT_JPEG" people/wit-geertruida/photos/gravestone.jpg
img "$DOCUMENT_JPEG" sources/parish-san-lorenzo-baptisms/scans/1779-042.jpg
img "$DOCUMENT_JPEG" sources/parish-san-lorenzo-baptisms/scans/1779-042-continued.jpg
img "$DOCUMENT_JPEG" sources/cimitero-san-lorenzo-gravestones/scans/sarti-angela.jpg
echo "done"
