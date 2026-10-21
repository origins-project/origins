package main

import (
	"github.com/origins-project/origins/internal/api"
	"strings"
	"testing"
)

// The manifest is changed as a record is, through the four field operations at the
// paths origins.yaml spells under project, and each change is a commit that log, blame
// and undo answer for.
func TestEveryFieldOfTheManifestIsChangedThroughTheCommands(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)

	for _, change := range []struct {
		args    []string
		op      string
		written string
		gone    string
	}{
		{[]string{"set", "project", "name", "Verdier and Papadopoulou"}, "name.set", "name: Verdier and Papadopoulou", ""},
		{[]string{"set", "project", "description", "Two families of the Mugnone"}, "description.set",
			"description: Two families of the Mugnone", ""},
		{[]string{"set", "project", "lang", "it"}, "lang.set", "lang: it", ""},
		{[]string{"unset", "project", "copyright"}, "copyright.unset", "", "copyright:"},
		{[]string{"add", "project", "researchers", "{name: Carla Verdi}"}, "researchers.add", "- name: Carla Verdi", ""},
		{[]string{"set", "project", "researchers[2].note", "Read the Vaiano registers."}, "researchers.set",
			"note: Read the Vaiano registers.", ""},
		{[]string{"remove", "project", "researchers[0].links", "mailto:ingrid@example.org"}, "researchers.remove",
			"", "mailto:ingrid@example.org"},
		{[]string{"add", "project", "bookmarks", "places/firenze"}, "bookmarks.add", "- places/firenze", ""},
		{[]string{"remove", "project", "bookmarks", "sources/parish-san-lorenzo-baptisms"}, "bookmarks.remove",
			"", "- sources/parish-san-lorenzo-baptisms"},
		{[]string{"set", "project", "tags.Emigrants.priority", "2"}, "tags.set", "priority: 2", ""},
		{[]string{"unset", "project", "tags.To do"}, "tags.unset", "", "To do:"},
		{[]string{"set", "project", "main-person", "rossi-bartolomeo"}, "main-person.set",
			"main-person: rossi-bartolomeo", ""},
	} {
		args := append([]string{change.args[0], "-C", root}, change.args[1:]...)
		if code, stdout, stderr := invoke(t, args...); code != exitOK {
			t.Fatalf("%s: exit %d\nstdout: %s\nstderr: %s", strings.Join(change.args, " "), code, stdout, stderr)
		}
		if op := trailer(t, root, "Origins-Op"); op != change.op {
			t.Errorf("%s is recorded as %s", strings.Join(change.args, " "), op)
		}
		if subject := trailer(t, root, "Origins-Subject"); subject != "project" {
			t.Errorf("%s is recorded about %s", strings.Join(change.args, " "), subject)
		}
		manifest := read(t, root, "origins.yaml")
		if change.written != "" && !strings.Contains(manifest, change.written) {
			t.Errorf("%s did not write %q:\n%s", strings.Join(change.args, " "), change.written, manifest)
		}
		if change.gone != "" && strings.Contains(manifest, change.gone) {
			t.Errorf("%s left %q:\n%s", strings.Join(change.args, " "), change.gone, manifest)
		}
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
	if code, stdout, stderr := invoke(t, "check", root); code != exitOK {
		t.Errorf("check: exit %d\n%s%s", code, stdout, stderr)
	}

	logged := operations(reading(t, "log", "-C", root, "project"))
	if len(logged) < 12 || logged[0] != "main-person.set" || logged[11] != "name.set" {
		t.Errorf("the manifest's log reads %v", logged)
	}

	blamed := fields(answering[api.Provenance](t, "blame", "-C", root, "project", "main-person"))
	if one := blamed["main-person"]; one.Commit == nil || one.Commit.Op != "main-person.set" {
		t.Errorf("main-person is blamed on %+v", one.Commit)
	}
	reasoned := answering[api.Reason](t, "why", "-C", root, "project", "tags.Emigrants.priority")
	if reasoned.Field == nil || reasoned.Field.Commit == nil || reasoned.Field.Commit.Op != "tags.set" {
		t.Errorf("why answers %+v", reasoned.Field)
	}

	if code, _, stderr := invoke(t, "undo", "-C", root); code != exitOK {
		t.Fatalf("undo: exit %d: %s", code, stderr)
	}
	if manifest := read(t, root, "origins.yaml"); !strings.Contains(manifest, "main-person: verdier-jean-baptiste\n") {
		t.Errorf("the undo did not put the main person back:\n%s", manifest)
	}
}

// What would leave the manifest naming nothing is refused as any change introducing an
// error is, and so is anything done to the manifest as a whole, or to the version of
// the form it is written in.
func TestTheManifestIsRefusedWhatItCannotBe(t *testing.T) {
	t.Parallel()
	root := gitFixture(t)
	before := git(t, root, "rev-parse", "HEAD")

	for _, refused := range []struct {
		args []string
		says string
	}{
		{[]string{"set", "project", "main-person", "firenze"}, "main-person"},
		{[]string{"add", "project", "bookmarks", "people/nobody-at-all"}, "nobody-at-all"},
		{[]string{"create", "project", "name=Another"}, "cannot be created"},
		{[]string{"delete", "project"}, "cannot be deleted"},
		{[]string{"rename", "project", "another"}, "cannot be renamed"},
		{[]string{"set", "project", "format-version", "2"}, "origins fmt"},
		{[]string{"set", "project/verdier-jean-baptiste", "name", "Rossi"}, "project"},
	} {
		args := append([]string{refused.args[0], "-C", root}, refused.args[1:]...)
		code, _, stderr := invoke(t, args...)
		if code == exitOK {
			t.Errorf("%s was carried out", strings.Join(refused.args, " "))
			continue
		}
		if !strings.Contains(stderr, refused.says) {
			t.Errorf("%s is refused saying %q", strings.Join(refused.args, " "), stderr)
		}
	}
	if now := git(t, root, "rev-parse", "HEAD"); now != before {
		t.Errorf("a refused change was committed as %s", now)
	}
	if left := git(t, root, "status", "--porcelain"); left != "" {
		t.Errorf("the tree was left dirty:\n%s", left)
	}
}
