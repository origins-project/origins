package model

import (
	"slices"
	"testing"
)

func TestALinkLeadsIntoTheRecordItNames(t *testing.T) {
	from := EntityRef{KindPerson, "rossi-maria"}
	text := "Her father, [Giovanni](../../people/rossi-giovanni/README.md), signed\n" +
		"[the entry](../../sources/parish/README.md#1834-042 \"the register\") and\n" +
		"![her](photos/maria.jpg) is from [Vicchio](<../../places/vicchio/README.md>).\n" +
		"See [above](#analysis), the [parish site](https://example.org/a/b) and\n" +
		"[the notes](../../../notes.md), and [nothing](../../people/).\n" +
		"[archive]: ../../repositories/asfi/README.md\n" +
		"[^1]: a footnote, not a link\n"
	type want struct {
		dest   string
		record EntityRef
		file   string
		anchor string
	}
	wanted := []want{
		{"../../people/rossi-giovanni/README.md", EntityRef{KindPerson, "rossi-giovanni"}, "", ""},
		{"../../sources/parish/README.md#1834-042", EntityRef{KindSource, "parish"}, "", "1834-042"},
		{"photos/maria.jpg", from, "photos/maria.jpg", ""},
		{"../../places/vicchio/README.md", EntityRef{KindPlace, "vicchio"}, "", ""},
		{"#analysis", from, "", "analysis"},
		{"../../../notes.md", EntityRef{}, "", ""},
		{"../../people/", EntityRef{}, "", ""},
		{"../../repositories/asfi/README.md", EntityRef{KindRepository, "asfi"}, "", ""},
	}
	links := Links(from, text)
	if len(links) != len(wanted) {
		t.Fatalf("found %d links, want %d: %+v", len(links), len(wanted), links)
	}
	for i, link := range links {
		w := wanted[i]
		if got := text[link.Start:link.End]; got != w.dest || link.Record != w.record ||
			link.File != w.file || link.Anchor != w.anchor {
			t.Errorf("link %d = %q %v %q #%q, want %q %v %q #%q", i, got, link.Record, link.File,
				link.Anchor, w.dest, w.record, w.file, w.anchor)
		}
	}
}

func TestALinkInCodeIsNotALink(t *testing.T) {
	text := "Written `[Giovanni](../../people/rossi-giovanni/README.md)` in a span,\n" +
		"``a [b](../../people/b/README.md) c`` in a longer one, and\n\n" +
		"```markdown\n[Giovanni](../../people/rossi-giovanni/README.md)\n```\n\n" +
		"~~~\n- [ ] not a task\n~~~\n" +
		"\\[escaped](../../people/escaped/README.md)\n"
	if links := Links(EntityRef{KindPerson, "a"}, text); len(links) != 0 {
		t.Errorf("found links in code: %+v", links)
	}
	if tasks := Tasks(text); len(tasks) != 0 {
		t.Errorf("found tasks in code: %+v", tasks)
	}
}

func TestRelinkRewritesTheSlugAndNothingElse(t *testing.T) {
	person := &Person{Slug: "rossi-maria", Body: "Daughter of [Giovanni](../../people/rossi-giovanni/README.md#birth),\n" +
		"![portrait](photos/maria.jpg), and see [Giovanni's photograph](../rossi-giovanni/photos/g.jpg)\n" +
		"and [her own page](../../people/rossi-maria/README.md).\n",
		Events: []Event{{Type: "birth", Note: "[Giovanni](../../people/./rossi-giovanni/README.md) registered it."}},
	}
	Relink(person, func(slug *Slug) {
		switch *slug {
		case "rossi-giovanni":
			*slug = "rossi-giovanni-1834"
		case "rossi-maria":
			*slug = "rossi-maria-1860"
		}
	})
	body := "Daughter of [Giovanni](../../people/rossi-giovanni-1834/README.md#birth),\n" +
		"![portrait](photos/maria.jpg), and see [Giovanni's photograph](../rossi-giovanni-1834/photos/g.jpg)\n" +
		"and [her own page](../../people/rossi-maria-1860/README.md).\n"
	if person.Body != body {
		t.Errorf("body =\n%s\nwant\n%s", person.Body, body)
	}
	if note := "[Giovanni](../../people/./rossi-giovanni-1834/README.md) registered it."; person.Events[0].Note != note {
		t.Errorf("note = %q, want %q", person.Events[0].Note, note)
	}
}

func TestProseIsEveryNoteAndBody(t *testing.T) {
	source := &Source{Slug: "parish", Body: "b", Text: "s",
		Covers:       []Coverage{{Events: []string{"baptism"}, Note: "c"}},
		Repositories: []Holding{{Repository: "archive", Note: "r"}},
		Citations: map[string]Citation{
			"b": {Text: "t2", Images: []Media{{File: "scan.jpg", Note: "i"}}},
			"a": {Text: "t1", Note: "n1", With: []Named{{Person: "p", Role: "child", Note: "w", Name: PersonName{Given: "G", Note: "g"}}},
				IDs: IDs{"gramps": {{ID: "C0001", Note: "i1"}}}},
		}, Media: []Media{{File: "x.jpg", Note: "m", Depicts: []Depicted{{Person: "p", Note: "d"}}}}}
	var paths []string
	Prose(source, func(path string, text *string) {
		paths = append(paths, path)
		*text += "!"
	})
	want := []string{"citations.a.with[0].name.note", "covers[0].note", "repositories[0].note", "text",
		"citations.a.text", "citations.a.with[0].note", "citations.a.note",
		"citations.a.ids.gramps[0].note", "citations.b.text", "citations.b.note", "citations.b.images[0].note",
		"media[0].depicts[0].note", "media[0].note", "body"}
	if !slices.Equal(paths, want) {
		t.Errorf("paths = %v, want %v", paths, want)
	}
	a := source.Citations["a"]
	if a.Note != "n1!" || source.Body != "b!" || source.Media[0].Depicts[0].Note != "d!" || source.Text != "s!" ||
		a.With[0].Note != "w!" || a.With[0].Name.Note != "g!" || source.Citations["b"].Images[0].Note != "i!" ||
		a.IDs["gramps"][0].Note != "i1!" {
		t.Errorf("Prose did not hand out the prose itself: %+v", source)
	}
}

func TestUnlinkTakesOutALinkWithItsText(t *testing.T) {
	from := EntityRef{KindPerson, "conti-luigi"}
	text := "His father [was a carter](../../notes/the-carter/README.md \"a note\") by trade, and\n" +
		"![the cart](../../notes/the-carter/cart.jpg) and [the census][c] and [c][] and [c] say so,\n" +
		"as does [the register](../../sources/parish/README.md#f088) and [another note][other].\n" +
		"`[in code](../../notes/the-carter/README.md)` stays.\n" +
		"[C]: ../../notes/the-carter/README.md\n" +
		"[other]: <../../notes/another/README.md>\n"
	gone := func(r EntityRef) bool { return r == EntityRef{KindNote, "the-carter"} }
	want := "His father by trade, and\n" +
		" and and and say so,\n" +
		"as does [the register](../../sources/parish/README.md#f088) and [another note][other].\n" +
		"`[in code](../../notes/the-carter/README.md)` stays.\n" +
		"[other]: <../../notes/another/README.md>\n"
	if got := unlink(from, text, gone); got != want {
		t.Errorf("unlink =\n%s\nwant\n%s", got, want)
	}
	if got := unlink(from, want, gone); got != want {
		t.Errorf("unlink took more out of what has no link to anything gone:\n%s", got)
	}
}

func TestTasksAreTheBoxesLeftUnticked(t *testing.T) {
	text := "*Research:* still to do.\n\n" +
		"- [ ] Look for the banns\n  in Vicchio, January 1861\n" +
		"- [x] Ask the parish\n" +
		"* [ ] Write to the archive\n" +
		"  - [ ] about the 1841 census\n" +
		"1. [ ] Read the will\n" +
		"- [] not a box\n"
	var got []string
	for _, task := range Tasks(text) {
		got = append(got, task.Text)
	}
	want := []string{"Look for the banns in Vicchio, January 1861", "Write to the archive",
		"about the 1841 census", "Read the will"}
	if !slices.Equal(got, want) {
		t.Errorf("Tasks = %q, want %q", got, want)
	}
}
