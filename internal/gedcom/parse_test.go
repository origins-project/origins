package gedcom

import (
	"strings"
	"testing"
)

func records(t *testing.T, text string) []*Node {
	t.Helper()
	parsed, err := Parse(strings.NewReader(text))
	if err != nil {
		t.Fatalf("parsing:\n%s\n%v", text, err)
	}
	return parsed
}

func TestParsingNests(t *testing.T) {
	found := records(t, `0 @I1@ INDI
1 NAME Giovanni /Rossi/
1 BIRT
2 DATE 12 MAR 1834
2 PLAC Firenze
0 TRLR
`)
	if len(found) != 2 {
		t.Fatalf("got %d records, want 2", len(found))
	}
	person := found[0]
	if person.Xref != "I1" || person.Tag != "INDI" {
		t.Errorf("record = %+v", person)
	}
	if got := person.First("NAME").Text(); got != "Giovanni /Rossi/" {
		t.Errorf("name = %q", got)
	}
	if got := person.First("BIRT").First("DATE").Text(); got != "12 MAR 1834" {
		t.Errorf("birth date = %q", got)
	}
}

// Asking an absent subtree about itself is what lets a caller chain First without a
// check at every step, and most of the tags a converter looks for are absent.
func TestAnAbsentNodeAnswersForItself(t *testing.T) {
	person := records(t, "0 @I1@ INDI\n1 NAME Giovanni\n")[0]
	if got := person.First("BIRT").First("DATE").Text(); got != "" {
		t.Errorf("a missing birth date reported %q", got)
	}
	if got := person.First("BIRT").All("SOUR"); got != nil {
		t.Errorf("a missing birth listed sources: %v", got)
	}
}

// CONC and CONT are how GEDCOM carries a value too long for one line, or one with a
// newline in it. They belong to the line above rather than being children of it.
func TestContinuationsRebuildTheValue(t *testing.T) {
	note := records(t, `0 @N1@ NOTE The handwriting is consistent
1 CONC  through 1841, then changes.
1 CONT The later hand abbreviates heavily.
`)[0]
	want := "The handwriting is consistent through 1841, then changes.\n" +
		"The later hand abbreviates heavily."
	if note.Text() != want {
		t.Errorf("note = %q, want %q", note.Text(), want)
	}
	if len(note.Children) != 0 {
		t.Errorf("continuations became %d children", len(note.Children))
	}
}

// Every one of these is a real exporter's idea of GEDCOM. Repairing structure is safe in
// a way that guessing at text is not: the worst case is a line attached one level from
// where it was meant, and the alternative is refusing the only file anyone has.
func TestParsingToleratesWhatExportersActuallyWrite(t *testing.T) {
	found := records(t, "0 @I1@ INDI\r\n"+
		"  1 NAME Giovanni /Rossi/\r\n"+ // indented
		"\r\n"+ // blank
		"1 BIRT\r\n"+
		"3 DATE 1834\r\n"+ // a level skipped
		"1 EMAIL rossi@@example.com\r\n"+ // the doubled at-sign
		"1 note lowercase tag\r\n")
	person := found[0]
	if got := person.First("BIRT").First("DATE").Text(); got != "1834" {
		t.Errorf("a skipped level lost the date: %q", got)
	}
	if got := person.First("NAME").Text(); got != "Giovanni /Rossi/" {
		t.Errorf("an indented line parsed as %q", got)
	}
	if got := person.First("EMAIL").Text(); got != "rossi@example.com" {
		t.Errorf("email = %q", got)
	}
	if got := person.First("NOTE").Text(); got != "lowercase tag" {
		t.Errorf("a lowercase tag parsed as %q", got)
	}
}

// A value may legitimately begin with spaces — a transcription keeping the layout of the
// page it came off — so exactly one space separates the tag from the value and the rest
// is data.
func TestOnlyTheFirstSpaceAfterATagSeparates(t *testing.T) {
	node := records(t, "0 @N1@ NOTE   indented transcription\n")[0]
	if node.Text() != "  indented transcription" {
		t.Errorf("value = %q", node.Text())
	}
}

// A .ged that is really a saved web page is a common enough mishap that the error
// should name it, rather than pointing at a line and inviting a hunt for a typo.
func TestAFileThatIsNotAGedcomAtAllSaysSo(t *testing.T) {
	_, err := Parse(strings.NewReader("<!DOCTYPE html>\n<html>\n"))
	if err == nil || !strings.Contains(err.Error(), "does not look like a GEDCOM file") {
		t.Errorf("err = %v", err)
	}
}

func TestParsingRejectsWhatIsNotAGedcomLine(t *testing.T) {
	for _, text := range []string{
		"not a gedcom file at all\n",
		"0 @I1@\n",
		"0 @I1 INDI\n",
	} {
		if _, err := Parse(strings.NewReader(text)); err == nil {
			t.Errorf("Parse(%q) found no problem", text)
		}
	}
}

// An exporter may write an accent as a letter and a mark after it, and the corpus writes
// it as the one character, or the files it writes would not be canonical.
func TestTextIsComposed(t *testing.T) {
	found := records(t, "0 @I1@ INDI\n1 NAME René /Pérez/\n1 NOTE Né\n2 CONC é à Paris\n")
	if got, want := found[0].First("NAME").Value, "René /Pérez/"; got != want {
		t.Errorf("NAME = %q, want %q", got, want)
	}
	if got, want := found[0].First("NOTE").Value, "Néé à Paris"; got != want {
		t.Errorf("NOTE = %q, want %q", got, want)
	}
}
