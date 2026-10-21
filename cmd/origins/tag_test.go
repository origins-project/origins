package main

import (
	"strings"
	"testing"
)

// Two styled tags made one: refused while their styles differ, carried out once one of
// them is unset, and taken back by undo, every file as it was before.
func TestATagIsRenamedIntoAnother(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	head := git(t, root, "rev-parse", "HEAD")

	code, _, stderr := invoke(t, "rename", "-C", root, "tag/To do", "Emigrants")
	if code == exitOK || !strings.Contains(stderr, "#d33 with priority 1") || !strings.Contains(stderr, "#2a7") {
		t.Fatalf("renaming onto a word styled otherwise: exit %d\n%s", code, stderr)
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != head {
		t.Fatal("a refused rename was committed")
	}

	if code, stdout, stderr := invoke(t, "unset", "-C", root, "project", "tags.Emigrants"); code != exitOK {
		t.Fatalf("unsetting a style: exit %d\n%s%s", code, stdout, stderr)
	}
	files := []string{"origins.yaml", "people/ershov-michael/README.md", "people/galvao-helena/README.md"}
	before := map[string]string{}
	for _, file := range files {
		before[file] = read(t, root, file)
	}
	if code, stdout, stderr := invoke(t, "rename", "-C", root, "tag/To do", "Emigrants"); code != exitOK {
		t.Fatalf("rename: exit %d\n%s%s", code, stdout, stderr)
	}
	if op := trailer(t, root, "Origins-Op"); op != "tags.rename" {
		t.Errorf("the rename is filed under %s", op)
	}
	subjects := strings.Fields(strings.ReplaceAll(trailer(t, root, "Origins-Subject"), "\n", " | "))
	if got := strings.Join(subjects, " "); got !=
		"tag/To do | tag/Emigrants | person/ershov-michael | person/galvao-helena | project" {
		t.Errorf("the rename is about %s", got)
	}
	michael := read(t, root, "people/ershov-michael/README.md")
	if strings.Contains(michael, "To do") || strings.Count(michael, "tags: [Emigrants]") != 2 {
		t.Errorf("Michael reads:\n%s", michael)
	}
	if helena := read(t, root, "people/galvao-helena/README.md"); !strings.Contains(helena, "tags: [Emigrants]") {
		t.Errorf("Helena's photograph is not retagged:\n%s", helena)
	}
	manifest := read(t, root, "origins.yaml")
	if strings.Contains(manifest, "To do") || !strings.Contains(manifest, "    Emigrants:\n      color: '#d33'\n      priority: 1\n") {
		t.Errorf("the style did not move:\n%s", manifest)
	}
	if code, stdout, stderr := invoke(t, "check", root); code != exitOK {
		t.Errorf("check: exit %d\n%s%s", code, stdout, stderr)
	}
	if logged := operations(reading(t, "log", "-C", root, "tag/Emigrants")); len(logged) != 1 || logged[0] != "tags.rename" {
		t.Errorf("the tag's log reads %v", logged)
	}

	if code, _, stderr := invoke(t, "undo", "-C", root); code != exitOK {
		t.Fatalf("undo: exit %d: %s", code, stderr)
	}
	for _, file := range files {
		if got := read(t, root, file); got != before[file] {
			t.Errorf("%s after the undo:\n%s\nwant:\n%s", file, got, before[file])
		}
	}
}

// A hypothesis can say a tag is to be renamed as it says anything else, and is carried
// where it was written and accepted as any other.
func TestAHypothesisRenamesATag(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	base := git(t, root, "rev-parse", "HEAD")
	asking(t, root, "it-is-done", "---\nstatus: open\nbase: "+base+"\nedits:\n"+
		"  - op: rename\n    entity: tag/To do\n    to: Done\n---\n")

	if code, stdout, stderr := invoke(t, "check", "--with", "it-is-done", root); code != exitOK {
		t.Errorf("check --with: exit %d\n%s%s", code, stdout, stderr)
	}
	if code, stdout, stderr := invoke(t, "hypothesis", "accept", "-C", root, "it-is-done"); code != exitOK {
		t.Fatalf("accepting: exit %d\n%s%s", code, stdout, stderr)
	}
	for _, file := range []string{"origins.yaml", "people/ershov-michael/README.md", "people/galvao-helena/README.md"} {
		if written := read(t, root, file); strings.Contains(written, "To do") || !strings.Contains(written, "Done") {
			t.Errorf("%s after accepting:\n%s", file, written)
		}
	}
	code, stdout, stderr := invoke(t, "hypothesis", "status", "-C", root, "it-is-done")
	if code != exitOK || !strings.Contains(stdout, "accepted, and the corpus still holds it") {
		t.Errorf("status exited %d\n%s%s", code, stdout, stderr)
	}
}
