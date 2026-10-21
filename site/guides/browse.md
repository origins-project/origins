# Working in the browser

`origins browse` serves the corpus to your browser, where you can read the tree, look
at the pictures and the charts, and change any record. Each change you make there is a
commit, the same commit that the command for that change makes.

```console
$ origins browse rossi
http://127.0.0.1:8100/ — serving rossi
```

Origins opens the address in your default browser. It serves only to your own
computer, and stops when you press Ctrl+C in the terminal. Give `--no-open` if you want
to open the address yourself, and `--listen` to serve on another address, as
[Running on a server](./docker) describes.

## The pages

The top bar has the lists of each kind of record, the search, the findings, and a menu
with the hidden views and Settings.

- **Home** shows the project's name and description, the main person, the latest
  changes, the surnames, your bookmarks, the count of findings and the open
  hypotheses. You edit the project's fields here, as you edit a record.
- **The lists** show every record of a kind. You can filter a list by its text, sort
  it by any column, and create a record of that kind.
- **A record's page** has three tabs. *Overview* shows everything the record says, and
  what Origins works out from the other records, such as a person's parents, siblings
  and children, and the events of their relatives on their timeline. *Charts* draws
  the record's charts, and *History* lists the commits that changed it.
- **Findings** lists every error and warning of `origins check` and every item of
  `origins todo`, each with a link to the record it is about. A record's page shows its
  own findings too, beside the field they are about.
- **Settings** has the project's fields, the researchers, the bookmarks and the tags'
  colours, and the git section, where you fetch, pull and push, as
  [Sharing a corpus](./sharing#in-the-browser) describes.

## Editing

Every value on a record's page is a control. Click a value to change it, and click an
empty field, such as *Living* or *Tags* in a person's details, to fill it. An event
opens in a panel with every field of the event, including the ones that the
[format](../format/events) describes and that you seldom need, such as the sort date,
the address and the attributes. A date has a *Build date* button that writes the date
from its parts.

Each panel has *Cite this change* at the bottom. It takes a citation, and Origins
records it in the commit as the source of the change, as `--source` does on the command
line.

The *Actions* menu of a record has *Rename*, *Merge into*, *Split* and *Delete*.
Before a delete, a merge, a split or an undo, the page shows what would change, such
as the files that would be moved and removed, and asks you to confirm.

You make a family from a person's page, by adding a partner, a parent or a child.

If the record changed since the page drew it, for example because you edited the file
in a text editor, Origins refuses the change and writes nothing, so that a change made
on an old page never overwrites a newer one. The page shows the refusal and keeps what
you typed. Each page updates within two seconds of a change to the corpus, whoever
made it.

## Supposing

*Suppose*, in the top bar, turns the tab to a draft. While the tab supposes, each
change you make goes into the draft instead of into the corpus, and the pages show the
tree as if the draft were true. A bar under the top bar counts the changes and lets you
take one back, discard the draft, or keep it as a [hypothesis](./hypotheses), with its
question, its argument and the evidence for and against it.

Origins keeps the draft in the git directory until you keep or discard it, so it
survives a restart of `origins browse`.

A hypothesis can be supposed from its own page too, with *Suppose*. A bar under the
top bar then lists it, as [Hypotheses](./hypotheses#testing-a-hypothesis) describes.

## Hidden views

The menu has *Hide private* and *Hide living*. Each switches the tab to a
[hidden view](./privacy#the-hidden-view), which shows the tree as somebody else would
see it and takes no change. A tick beside each shows which part of the tree the tab
hides.

If you start `origins browse` with `--hide-private`, `--hide-living` or `--public`, it
serves only that view, read-only. The menu shows the view ticked and you can't
switch it. With `--with`, the bar under the top bar lists the hypotheses, and you
can't stop supposing them.
