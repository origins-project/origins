# Sharing a corpus

A corpus is a git repository, so you share it through a git remote: a repository on
GitHub, GitLab, a server of your own, or a directory on a shared drive. Origins has
its own commands for remotes. They work like git's, and they also check that every
commit they send or take in is one that Origins wrote, so that the history stays one
that Origins can read.

Git handles the authentication, with an SSH key or a credential helper. Origins stores
nothing of it.

## Sending the corpus to a remote

Create an empty repository on the host, add it as a remote and push:

```console
$ origins remote add origin git@example.org:rossi.git
added origin  git@example.org:rossi.git
$ origins push
pushed master to origin at 811af9a740
master now follows origin/master
```

`origins push` sends the current branch and every snapshot on it. If the remote has
commits that you don't have, `push` refuses, and you run `origins pull` first.

## Copying a corpus

Somebody you share the corpus with copies it with `origins clone`:

```console
$ origins clone git@example.org:rossi.git rossi
Cloning into 'rossi'...
done.
cloned git@example.org:rossi.git into rossi
```

`clone` checks out every file exactly as it was committed, whatever git is configured
to do with line endings. Then it checks the copy: whether it holds a corpus, what
`origins check` finds in it, and whether Origins can account for its history. If one of
these checks fails, `clone` keeps the copy, says what it found, and exits 1.

## Taking in other changes

`origins fetch` brings in what the remote holds and says how far apart you are, without
changing anything. `origins pull` takes the changes in:

```console
$ origins fetch
From example.org:rossi
   811af9a..72cf571  master     -> origin/master
master is 1 commit behind and 0 commits ahead of origin/master
$ origins pull
took in 1 commit from origin/master, at 72cf571713
```

If both of you made changes, git merges them, and Origins records the merge as one
commit. `pull` refuses the merge if you both changed the same lines, or if the merged
corpus contradicts itself, and then it leaves the corpus as it was. `pull` doesn't
start while you have changes that aren't recorded.

## When you both changed the same lines

Here Marco recorded Giovanni's birth as 12 March and pushed, and you recorded it as 13
March. `pull` refuses, and names the files:

```console
$ origins pull
From example.org:rossi
   2fa0ddb..4af9dea  master     -> origin/master
origins: this clone and origin/master changed the same lines, and nothing has been taken in:
  people/rossi-giovanni/README.md

git merge origin/master, settling them, and origins recover after it take them in by hand
```

Run the merge with git. Git marks the lines that differ in each file:

```console
$ git merge origin/master
Auto-merging people/rossi-giovanni/README.md
CONFLICT (content): Merge conflict in people/rossi-giovanni/README.md
Automatic merge failed; fix conflicts and then commit the result.
$ cat people/rossi-giovanni/README.md
---
name:
  given: Giovanni
  surname: Rossi
events:
  - type: birth
<<<<<<< HEAD
    date: 13 MAR 1842
=======
    date: 12 MAR 1842
>>>>>>> origin/master
---
```

Edit each file to say what you have decided, without the marker lines, and commit the
merge with git:

```bash
git add people/rossi-giovanni/README.md
git commit --no-edit
```

Origins didn't write that commit, so the next command that reads the history stops
until you run `origins recover`. `--adopt` keeps the merge and records that you have
read it:

```console
$ origins recover --adopt
983fcd6331  Adopt d9bbe5f689, a commit origins did not write
  d9bbe5f689  Merge remote-tracking branch 'origin/master'  (Rosa Bianchi)
$ origins push
pushed master to origin at 983fcd6331
```

Run `origins check` before you push, since git doesn't know whether the merged files
contradict each other.

## Commits from outside Origins

If the remote has a commit that Origins didn't write, `pull` refuses before it takes
anything in. Whoever made that commit can run `origins recover --adopt` in their clone
and push again.

## In the browser

`origins browse` shows how far the corpus is from its remote, and you can fetch, pull
and push from the pages. The pages run git where nobody can type a password, so a
remote that asks for one is refused with git's message. Use an SSH key, or a credential
helper that git can read without asking.
