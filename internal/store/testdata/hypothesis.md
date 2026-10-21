---
status: open
base: 8f78553
confidence: possible
proposed-by: agent:1
supports: [censimento-1861-firenze#f214]
contradicts: [parish-san-lorenzo-baptisms#1842-007]
edits:
  - op: set
    entity: person/rossi-annunziata
    path: events[0].date
    value: 1844
  - op: set
    entity: person/rossi-annunziata
    path: body
    value: |
      # Annunziata Rossi

      Born in 1844, if the census is right.
  - op: unset
    entity: person/rossi-annunziata
    path: surname
  - op: add
    entity: family/a--b
    path: partners
    value: c
  - op: remove
    entity: family/a--b
    path: partners
    value: c
  - op: create
    entity: person/rossi-unknown-1845
    value:
      events:
        - date: ABT 1845
          note: |-
            no record of the birth
            only a gap of five years
          type: birth
      name: Unknown Rossi
      sex: U
  - op: delete
    entity: person/a-duplicate
  - op: rename
    entity: person/a-duplicate
    to: rossi-giovanni-1902
  - op: merge
    from: person/a-duplicate
    into: person/rossi-giovanni
  - op: split
    entity: person/two-men
    into: [person/one-man, person/another-man]
---
