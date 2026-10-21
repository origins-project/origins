package gedcom

import "testing"

// The two grammars overlap over most of their range, and the table is the record of
// exactly where. Words marks a value meant as a date that the grammar cannot state, which
// survives as the wording of a date that is only words.
func TestReadingDates(t *testing.T) {
	for _, c := range []struct {
		gedcom string
		want   string
		words  bool
	}{
		{gedcom: "1834", want: "1834"},
		{gedcom: "MAR 1834", want: "MAR 1834"},
		{gedcom: "12 MAR 1834", want: "12 MAR 1834"},
		{gedcom: "ABT 1834", want: "ABT 1834"},
		{gedcom: "EST 1834", want: "EST 1834"},
		{gedcom: "CAL 1834", want: "CAL 1834"},
		{gedcom: "BEF 1900", want: "BEF 1900"},
		{gedcom: "AFT 1850", want: "AFT 1850"},
		{gedcom: "BET 1830 AND 1835", want: "BET 1830 AND 1835"},
		{gedcom: "1750/51", want: "1750/51"},

		// A dual-dated year is written three ways in the wild and means one thing. What
		// makes it dual is that the second year follows the first, so two years further
		// apart than that are a guess at a range and stay words.
		{gedcom: "1 Jan 1510/1511", want: "1 JAN 1510/11"},
		{gedcom: "FROM 7 Feb 1476/7", want: "FROM 7 FEB 1476/77"},
		{gedcom: "1799/1800", want: "1799/00"},
		{gedcom: "1056/1060", want: "(1056/1060)", words: true},
		{gedcom: "0029 / 1829 / 1929", want: "(0029 / 1829 / 1929)", words: true},
		{gedcom: "@#DGREGORIAN@ 12 MAR 1834", want: "12 MAR 1834"},

		// A period says something lasted and a range says it happened somewhere inside
		// one, and both are carried as what they are.
		{gedcom: "FROM 1850 TO 1880", want: "FROM 1850 TO 1880"},
		{gedcom: "FROM 1850", want: "FROM 1850"},
		{gedcom: "TO 1880", want: "TO 1880"},
		{gedcom: "FROM 1880 TO 1850", want: "(FROM 1880 TO 1850)", words: true},

		// Spelled-out months are not GEDCOM and plenty of files carry them.
		{gedcom: "12 March 1834", want: "12 MAR 1834"},
		{gedcom: "Sept 1834", want: "SEP 1834"},

		// An interpreted date is usable and the words behind it are the evidence for it,
		// so both are kept, the words as the date's wording.
		{gedcom: "INT 12 MAR 1834 (twelfth of March)", want: "12 MAR 1834 (twelfth of March)"},
		{gedcom: "INT ABT 1834 (the winter of the cholera)", want: "ABT 1834 (the winter of the cholera)"},

		// A calendar the corpus writes travels as itself, in the escape 5.5.1 wraps it
		// in or the word 7.0 puts in front, and is never converted on the way.
		{gedcom: "@#DJULIAN@ 14 FEB 1799", want: "JULIAN 14 FEB 1799"},
		{gedcom: "JULIAN 14 FEB 1799", want: "JULIAN 14 FEB 1799"},
		{gedcom: "@#DHEBREW@ 5 NSN 5594", want: "HEBREW 5 NSN 5594"},
		{gedcom: "@#DFRENCH R@ 18 BRUM 8", want: "FRENCH_R 18 BRUM 8"},
		{gedcom: "FRENCH_R 18 BRUM 8", want: "FRENCH_R 18 BRUM 8"},

		// A range or a period states its reckoning on each end.
		{gedcom: "BET @#DJULIAN@ 1750 AND @#DJULIAN@ 1755", want: "BET JULIAN 1750 AND JULIAN 1755"},
		{gedcom: "FROM @#DJULIAN@ 1750 TO 1755", want: "FROM JULIAN 1750 TO 1755"},

		// The epoch is the last word, spelled several ways and meaning one.
		{gedcom: "44 BCE", want: "44 BCE"},
		{gedcom: "44 B.C.", want: "44 BCE"},
		{gedcom: "0022 B.C.", want: "22 BCE"},
		{gedcom: "1 JAN 44 BC", want: "1 JAN 44 BCE"},
		{gedcom: "BET 50 BC AND 44 BC", want: "BET 50 BCE AND 44 BCE"},

		// A calendar origins does not reckon in could only be stated by converting it.
		{gedcom: "@#DROMAN@ 14 FEB 1799", want: "(14 FEB 1799 (Roman))", words: true},
		{gedcom: "@#DUNKNOWN@ 14 FEB 1799", want: "(14 FEB 1799 (an unnamed calendar))", words: true},

		{gedcom: "UNKNOWN", want: "(UNKNOWN)", words: true},
		{gedcom: "", want: ""},
		{gedcom: "sometime before the war", want: "(sometime before the war)", words: true},
		{gedcom: "1834?", want: "(1834?)", words: true},

		// Words a record gives in place of a date are what it says, and are carried.
		{gedcom: "(the register is illegible here)", want: "(the register is illegible here)"},
		{gedcom: "(from 1830 to 1860)", want: "FROM 1830 TO 1860"},
	} {
		date, read := readDate(c.gedcom)
		if got := date.String(); got != c.want {
			t.Errorf("readDate(%q) = %q, want %q", c.gedcom, got, c.want)
		}
		if read == c.words {
			t.Errorf("readDate(%q) read it = %v, want %v", c.gedcom, read, !c.words)
		}
	}
}

// The reason the model's dates are not GEDCOM's. GEDCOM makes the year mandatory at
// every level, so a worn gravestone reading "3 marzo" can only travel as free text —
// and free text is opaque to everything downstream. Recovering it into a real date is
// what makes the import worth more than the file it read.
func TestAYearlessDateIsRecoveredFromAPhrase(t *testing.T) {
	for _, c := range []struct{ phrase, want string }{
		{phrase: "(3 MAR)", want: "3 MAR"},
		{phrase: "(NOV)", want: "NOV"},
		{phrase: "(3 March)", want: "3 MAR"},
	} {
		date, _ := readDate(c.phrase)
		if got := date.String(); got != c.want {
			t.Errorf("readDate(%q) = %q, want the date %q", c.phrase, got, c.want)
		}
		if date.Anchored() {
			t.Errorf("readDate(%q) invented a year", c.phrase)
		}
	}
}

// A phrase that is not a date is still evidence, and nothing a record stated is dropped:
// it is a date that is only words, which floats.
func TestAPhraseThatIsNotADateSurvivesAsWords(t *testing.T) {
	date, read := readDate("(the register is illegible here)")
	if !read || date.Wording != "the register is illegible here" || date.Year != nil {
		t.Errorf("readDate = %+v, read %v", date, read)
	}
	if date.Anchored() {
		t.Errorf("words alone were placed on the calendar: %q", date)
	}
}
