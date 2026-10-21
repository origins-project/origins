# Private records

Two fields say what a hidden view leaves out: `private` and `living`. Neither changes
anything while you work. `origins check`, the other commands and the pages read the
whole corpus. [Privacy](../guides/privacy) explains how you ask for a hidden view.

```yaml
private: true
living: false
name:
  given: Ottavio
  surname: Landi
events:
  - private: true
    type: residence
    date: FROM 1990
    place: prato
```

## private

`private: true` marks a record, or an entry in one, as not for sharing. These take it:

- a person, a family, a place, a source, a repository and a note
- an event
- a name in a person's `also-known-as`
- a media entry

A name written as a plain string becomes a mapping to take it, with the string as its
`text`. The main `name` doesn't take it: to hide a person's name, make the person
private.

A citation, a child and a sex don't take `private`. Somebody named in an event is
hidden with the event, or with their own record.

## living

`living` is on a person only. A hidden view can leave out the people who may be living,
and a rule decides who they are. `living` overrules the rule for one person: `false`
for somebody who has died without a record of the death, and `true` for somebody who is
alive and whom the rule takes as dead. Leave it out to let the rule decide.

## Values

`private` is `true` or left out. `living` is `true`, `false` or left out. Origins
refuses `yes`, `on` and `1`, and writes each field first in its mapping, `private`
before `living`.

You set them with the same commands as any other field:

```bash
origins set person/landi-ottavio private true
origins unset person/landi-ottavio living
origins set person/landi-ottavio 'events[1].private' true
```
