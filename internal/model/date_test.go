package model

import "testing"

// every canonical form the grammar admits, alongside every date literal the fixture
// corpus uses
var canonical = []string{
	"1834",
	"MAR 1834",
	"12 MAR 1834",
	"12 MAR 1834 09:20",
	"3 MAR",
	"NOV",
	"ABT 1834",
	"EST 1834",
	"CAL 1834",
	"EST ABT 1834",
	"CAL BEF 1850",
	"EST BET 1840 AND 1850",
	"3 ? 1850",
	"3 ?",
	"3 ? 1850 09:20",
	"15 FEB 1700 NY 25 MAR",
	"1700 NY 25 MAR",
	"12 MAR 1834 09:20 NY 1 SEP",
	"JULIAN 15 FEB 1700 NY 25 MAR",
	"BET 1700 NY 25 MAR AND 1701",
	"BEF 1900",
	"AFT 1850",
	"BET 1830 AND 1835",
	"BET 1830 AND 1860",
	"1750/51",
	"12 FEB 1750/51",
	"11 JUN 1900",
	"14 MAR 1975 09:20",
	"19 MAY 2018",
	"2 FEB 1920",
	"3 AUG 1888",
	"9 SEP 1860",
	"FEB 1840",
	"BET 1845 AND 1851",
	"AFT 1901",
	"CAL 1782",
	"EST 1755",
	"ABT 1805",
	"BEF 1778",
	"1 JAN 1000 00:00:00",
	"JULIAN 14 FEB 1799",
	"JULIAN 29 FEB 1900",
	"JULIAN 1750/51",
	"HEBREW 5 NSN 5594",
	"HEBREW 1 ADR 5784",
	"HEBREW 1 ADS 5783",
	"HEBREW NSN",
	"HEBREW ADR",
	"FRENCH_R 18 BRUM 8",
	"FRENCH_R 6 COMP 11",
	"44 BCE",
	"1 JAN 1 BCE",
	"BET 50 BCE AND 44 BCE",
	"BET JULIAN 1 JAN 1700 AND 12 JAN 1700",
	"BEF JULIAN 1750",
	"ABT HEBREW 5594",
	"AFT 44 BCE",
	"PERSIAN 12 FARVA 1300",
	"PERSIAN 30 ESFAN 1403",
	"ISLAMIC 1 RAMAD 1250",
	"ISLAMIC 3 SAFAR 1319",
	"SWEDISH 30 FEB 1712",
	"SWEDISH 28 JUN 1709",
	"FROM 1830 TO 1860",
	"FROM 12 MAR 1834",
	"TO 1860",
	"FROM ABT 1830 TO JULIAN 1860",
	"FROM BET 1828 AND 1830 TO BEF 1860",
	"FROM SWEDISH 1704 TO SWEDISH 30 FEB 1712",
	"FROM EST 1830 TO CAL BEF 1860",
	"FROM PERSIAN 1 MEHR 1352 TO PERSIAN DEY 1357",
	"ABT 1854 (the winter of the cholera)",
	"(before the war)",
	"NOV (l'anno della grande nevicata)",
	"FROM 1830 TO 1860 (while the lease ran)",
	"1834 (in the year (so the priest wrote) of the flood)",
}

func TestRoundTrip(t *testing.T) {
	for _, in := range canonical {
		d, err := ParseDate(in)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", in, err)
			continue
		}
		if got := d.String(); got != in {
			t.Errorf("ParseDate(%q).String() = %q", in, got)
			continue
		}
		again, err := ParseDate(d.String())
		if err != nil {
			t.Errorf("reparse %q: %v", in, err)
			continue
		}
		if got := again.String(); got != in {
			t.Errorf("formatting %q is not idempotent: %q", in, got)
		}
	}
}

func TestParsedComponents(t *testing.T) {
	d, err := ParseDate("12 MAR 1834 09:20")
	if err != nil {
		t.Fatal(err)
	}
	if *d.Day != 12 || *d.Month != 3 || *d.Year != 1834 {
		t.Errorf("got day %d month %d year %d", *d.Day, *d.Month, *d.Year)
	}
	if d.Time.Hour != 9 || d.Time.Minute != 20 || d.Time.Second != nil {
		t.Errorf("got time %+v", *d.Time)
	}
	if d.Qualifier != Exact || d.Calendar != Gregorian {
		t.Errorf("got qualifier %d calendar %d", d.Qualifier, d.Calendar)
	}

	dual, err := ParseDate("12 FEB 1750/51")
	if err != nil {
		t.Fatal(err)
	}
	if !dual.Dual || dual.Calendar != Gregorian || *dual.Year != 1750 || dual.reckoningYear() != 1751 {
		t.Errorf("got dual %v calendar %d year %d reckoning %d",
			dual.Dual, dual.Calendar, *dual.Year, dual.reckoningYear())
	}

	span, err := ParseDate("BET 1830 AND 1860")
	if err != nil {
		t.Fatal(err)
	}
	if span.Qualifier != Between || *span.Year != 1830 || span.End == nil || *span.End.Year != 1860 {
		t.Errorf("got %+v", span)
	}

	reached, err := ParseDate("EST BET 1840 AND 1850")
	if err != nil {
		t.Fatal(err)
	}
	if reached.Reached != Estimated || reached.Qualifier != Between || *reached.End.Year != 1850 {
		t.Errorf("got %+v", reached)
	}

	monthless, err := ParseDate("3 ? 1850")
	if err != nil {
		t.Fatal(err)
	}
	if *monthless.Day != 3 || monthless.Month != nil || *monthless.Year != 1850 {
		t.Errorf("got %+v", monthless)
	}

	pisan, err := ParseDate("15 FEB 1700 NY 25 MAR")
	if err != nil {
		t.Fatal(err)
	}
	if *pisan.Year != 1700 || pisan.NewYear == nil || *pisan.NewYear != (NewYear{Month: 3, Day: 25}) {
		t.Errorf("got %+v", pisan)
	}

	floating, err := ParseDate("NOV")
	if err != nil {
		t.Fatal(err)
	}
	if *floating.Month != 11 || floating.Year != nil || floating.Day != nil {
		t.Errorf("got %+v", floating)
	}
}

func TestNormalisation(t *testing.T) {
	for in, want := range map[string]string{
		"abt 1834":              "ABT 1834",
		"12 mar 1834":           "12 MAR 1834",
		"  12   MAR   1834 ":    "12 MAR 1834",
		"14 MAR 1975 9:20":      "14 MAR 1975 09:20",
		"bet 1830 and 1835":     "BET 1830 AND 1835",
		"12 feb 1750/51":        "12 FEB 1750/51",
		"julian 14 feb 1799":    "JULIAN 14 FEB 1799",
		"gregorian 12 MAR 1834": "12 MAR 1834",
		"44 bce":                "44 BCE",
		"est abt 1834":          "EST ABT 1834",
		"15 feb 1700 ny 25 mar": "15 FEB 1700 NY 25 MAR",

		// a writer recording a common year writes Adar, and ADS is the tag for the
		// one Adar such a year has
		"HEBREW 1 ADR 5783": "HEBREW 1 ADS 5783",
		"HEBREW 1 ADR 5784": "HEBREW 1 ADR 5784",

		"from 1830 to 1860":                          "FROM 1830 TO 1860",
		"persian 1 mehr 1352":                        "PERSIAN 1 MEHR 1352",
		"ABT 1854(the winter of the cholera)":        "ABT 1854 (the winter of the cholera)",
		"ABT 1854 (  the winter\n  of the cholera )": "ABT 1854 (the winter of the cholera)",
		"  ( before the war )  ":                     "(before the war)",
	} {
		d, err := ParseDate(in)
		if err != nil {
			t.Errorf("ParseDate(%q): %v", in, err)
			continue
		}
		if got := d.String(); got != want {
			t.Errorf("ParseDate(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestParseRejects(t *testing.T) {
	for _, in := range []string{
		"",
		"   ",
		"12 1834",       // no month
		"32 MAR 1834",   // no such day
		"30 FEB 1834",   // no such day in February
		"29 FEB 1834",   // 1834 is not a leap year
		"12 MAR 1834 X", // trailing rubbish
		"09:20",         // a time with no date
		"14 MAR 1975 25:00",
		"14 MAR 1975 09:20:60",
		"1750/52", // a dual year must name the year after
		"1750/1751",
		"0",
		"BET 1830",          // no AND
		"BET 1860 AND 1830", // ends before it starts
		"BEF",
		"MARCH 1834",
		"HEBREW 44 BCE",     // a calendar counting from its own beginning has no before
		"FRENCH_R 5794 BCE", // likewise
		"HEBREW 5594/95",    // nor does either write a year as two
		"44 BCE BCE",
		"1750/51 BCE",        // a dual year names two years of one era
		"FRENCH_R 1 VEND 20", // the calendar was abandoned in year 14
		"FRENCH_R 1 VEND 0",
		"HEBREW 30 ADS 5784", // Adar II is 29 days; Adar I is the long one
		"HEBREW 1 ADR 5784 BCE",
		"HEBREW 30 ELL 5784", // Elul is always 29 days
		"FRENCH_R 6 COMP 8",  // year 8 was not sextile and has five spare days
		"FRENCH_R 31 BRUM 8", // a republican month is thirty days
		"JULIAN 5 NSN 1799",  // the Julian calendar names the Gregorian months
		"HEBREW 5 MAR 5594",

		"PERSIAN 30 ESFAN 1402", // 1402 is not leap
		"PERSIAN 32 FARVA 1403", // the first six months are thirty-one days
		"PERSIAN 31 MEHR 1403",  // and the next five thirty
		"PERSIAN 3178",          // past the years the arithmetic covers
		"PERSIAN 1300 BCE",
		"PERSIAN 1300/01",
		"PERSIAN 12 MAR 1300",
		"ISLAMIC 30 DHUAH 1446", // 1446 is not leap
		"ISLAMIC 30 SAFAR 1446", // an even month is twenty-nine days
		"ISLAMIC 1250 BCE",
		"ISLAMIC 1250/51",
		"SWEDISH 1699", // Sweden kept the calendar from 1700
		"SWEDISH 1713", // and gave it up in 1712
		"SWEDISH 29 FEB 1700",
		"SWEDISH 30 FEB 1708",
		"SWEDISH 1704/05",

		"FROM",
		"TO",
		"FROM 1830 TO",
		"FROM TO 1860",
		"FROM 1860 TO 1830", // ends before it starts
		"FROM FROM 1830",
		"FROM 1830 TO TO 1860",
		"FROM 1830 AND 1860",
		"()",
		"(   )",
		"1834 (",
		"1834 )",
		"1834 (the flood) and more",
		"(the flood) 1834",
		"FROM 1860 TO 1830 (a lease)",

		"EST",
		"EST EST 1834",
		"ABT EST 1834", // how a date was reached comes first
		"EST FROM 1830 TO 1860",
		"CAL TO 1860",
		"32 ? 1850",
		"30 ? 1850 MAR",
		"? 1850",
		"?",
		"HEBREW 31 ? 5594", // no Hebrew month has thirty-one days
		"15 FEB 1700 NY",
		"15 FEB 1700 NY 25",
		"15 FEB 1700 NY 25 MAR 1699",
		"15 FEB 1700 NY 32 MAR",
		"15 FEB 1700 NY 1 JAN", // every other year is counted from 1 January
		"15 FEB 1700/01 NY 25 MAR",
		"HEBREW 5 NSN 5594 NY 1 TSH",
		"FRENCH_R 18 BRUM 8 NY 1 VEND",
		"3 MAR NY 25 MAR", // no year to count
		"NY 25 MAR",
	} {
		if d, err := ParseDate(in); err == nil {
			t.Errorf("ParseDate(%q) = %q, want an error", in, d)
		}
	}
}

// TestADateIsNeverConverted is the invariant the calendars rest on: what a record
// wrote is what the corpus keeps, whatever arithmetic is done with it afterwards.
func TestADateIsNeverConverted(t *testing.T) {
	d := mustParse(t, "JULIAN 14 FEB 1799")
	if d.Calendar != Julian || *d.Day != 14 || *d.Month != 2 || *d.Year != 1799 {
		t.Fatalf("got %+v", d)
	}
	if got := d.String(); got != "JULIAN 14 FEB 1799" {
		t.Errorf("wrote %q", got)
	}
	// and it still lands on the day the Gregorian calendar calls 25 February
	if lo, hi, ok := d.dayBounds(); !ok || lo != hi || lo != daysFromCivil(1799, 2, 25) {
		t.Errorf("day bounds %d %d %v", lo, hi, ok)
	}
}

// TestTheEraHasNoYearZero: the written years run 1 BCE, 1, so subtracting one from the
// other has to go through a zero that neither of them spells.
func TestTheEraHasNoYearZero(t *testing.T) {
	if got := mustParse(t, "1 BCE").reckoningYear(); got != 0 {
		t.Errorf("1 BCE reckons as year %d", got)
	}
	if got := mustParse(t, "2 BCE").reckoningYear(); got != -1 {
		t.Errorf("2 BCE reckons as year %d", got)
	}
	last, first := mustParse(t, "31 DEC 1 BCE"), mustParse(t, "1 JAN 1")
	if got, ok := Compare(last, first); got != -1 || !ok {
		t.Errorf("the last day before the era does not precede the first: %d %v", got, ok)
	}
	least, most, ok := YearsBetween(last, first)
	if !ok || least < 0 || most > 0.01 {
		t.Errorf("they are one day apart, not a year: %v %v %v", least, most, ok)
	}
}

// TestCalendarBindsToTheDateItPrecedes: the two ends of a range may be stated in
// different reckonings, and each keeps its own.
func TestCalendarBindsToTheDateItPrecedes(t *testing.T) {
	d := mustParse(t, "BET JULIAN 1 JAN 1700 AND 12 JAN 1700")
	if d.Calendar != Julian || d.End == nil || d.End.Calendar != Gregorian {
		t.Fatalf("got %+v", d)
	}
	// 1 January 1700 in the old reckoning is 11 January in the new one, so the range
	// covers two days and not none
	lo, hi, ok := d.dayBounds()
	if !ok || hi-lo != 1 {
		t.Errorf("day bounds %d %d %v", lo, hi, ok)
	}
}

func TestBareNumberIsAYear(t *testing.T) {
	d, err := ParseDate("12")
	if err != nil {
		t.Fatalf("a bare number is a year, however small: %v", err)
	}
	if d.Day != nil || d.Month != nil || *d.Year != 12 {
		t.Errorf("got %+v", d)
	}
}

func TestFloatingDayIsGivenTheBenefitOfTheDoubt(t *testing.T) {
	if _, err := ParseDate("29 FEB"); err != nil {
		t.Errorf("29 FEB should parse without a year to rule it out: %v", err)
	}
}

// TestAPeriodIsNotARange: the two are written differently and held differently, though
// they cover the same years.
func TestAPeriodIsNotARange(t *testing.T) {
	period, rangeOf := mustParse(t, "FROM 1830 TO 1860"), mustParse(t, "BET 1830 AND 1860")
	if !period.Period() || rangeOf.Period() {
		t.Fatalf("Period() is %v for the period and %v for the range", period.Period(), rangeOf.Period())
	}
	if period.Year != nil || *period.From.Year != 1830 || *period.To.Year != 1860 {
		t.Errorf("got %+v", period)
	}

	ends := mustParse(t, "FROM ABT 1830 TO JULIAN 1860")
	if ends.From.Qualifier != About || ends.To.Calendar != Julian || ends.From.Calendar != Gregorian {
		t.Errorf("each end keeps its own qualifier and calendar: %+v %+v", *ends.From, *ends.To)
	}

	open := mustParse(t, "FROM 12 MAR 1834")
	if open.From == nil || open.To != nil {
		t.Errorf("got %+v", open)
	}
	if open := mustParse(t, "TO 1860"); open.From != nil || open.To == nil {
		t.Errorf("got %+v", open)
	}
}

func TestWording(t *testing.T) {
	d := mustParse(t, "ABT 1854 (the winter of the cholera)")
	if d.Wording != "the winter of the cholera" || d.Qualifier != About || *d.Year != 1854 {
		t.Errorf("got %+v", d)
	}
	alone := mustParse(t, "(before the war)")
	if alone.Wording != "before the war" || alone.Year != nil || alone.IsZero() {
		t.Errorf("got %+v", alone)
	}
	period := mustParse(t, "FROM 1830 TO 1860 (while the lease ran)")
	if period.Wording != "while the lease ran" || period.From.Wording != "" || period.To.Wording != "" {
		t.Errorf("the wording belongs to the whole value: %+v", period)
	}
}

func TestStart(t *testing.T) {
	for in, want := range map[string]string{
		"FROM 1830 TO 1860":   "1830",
		"FROM ABT 1830":       "ABT 1830",
		"TO 1860":             "1860",
		"BET 1830 AND 1860":   "BET 1830 AND 1860",
		"12 MAR 1834 (Lent)":  "12 MAR 1834 (Lent)",
		"FROM 1830 (a lease)": "1830",
	} {
		if got := mustParse(t, in).Start().String(); got != want {
			t.Errorf("ParseDate(%q).Start() = %q, want %q", in, got, want)
		}
	}
}

func TestAnchored(t *testing.T) {
	for in, want := range map[string]bool{
		"1834":                   true,
		"12 MAR 1834":            true,
		"BET 1830 AND 1835":      true,
		"3 MAR":                  false,
		"NOV":                    false,
		"FROM 1830":              true,
		"TO 1860":                true,
		"FROM 3 MAR TO 1860":     false,
		"ABT 1854 (the cholera)": true,
		"(before the war)":       false,
		"NOV (the great snow)":   false,
		"SWEDISH 30 FEB 1712":    true,
	} {
		d, err := ParseDate(in)
		if err != nil {
			t.Fatal(err)
		}
		if got := d.Anchored(); got != want {
			t.Errorf("ParseDate(%q).Anchored() = %v", in, got)
		}
	}
}

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
		ok   bool
	}{
		{"1834", "1835", -1, true},
		{"1835", "1834", 1, true},
		{"1834", "1834", 0, true},
		{"12 MAR 1834", "14 MAR 1834", -1, true},
		{"12 MAR 1834", "12 MAR 1834", 0, true},
		{"14 MAR 1975 09:20", "14 MAR 1975 10:00", -1, true},
		{"MAR 1834", "APR 1834", -1, true},
		{"ABT 1834", "1834", 0, true},

		// overlapping spans of different extent are not ordered
		{"1834", "MAR 1834", 0, false},
		{"MAR 1834", "12 MAR 1834", 0, false},
		{"BEF 1900", "1899", 0, false},
		{"BET 1830 AND 1860", "1845", 0, false},

		// floating dates are not ordered at all
		{"3 MAR", "1834", 0, false},
		{"NOV", "MAR", 0, false},
		{"3 MAR", "3 MAR", 0, false},

		{"BEF 1900", "1901", -1, true},
		{"AFT 1880", "1875", 1, true},
		{"BET 1830 AND 1860", "1870", -1, true},
		{"BET 1830 AND 1860", "1820", 1, true},

		// 1750/51 is February 1751 under Gregorian reckoning, after all of 1750
		{"12 FEB 1750/51", "1750", 1, true},
		{"12 FEB 1750/51", "1752", -1, true},

		// a period is the whole of its extent, and an open end runs without limit
		{"FROM 1830 TO 1860", "1870", -1, true},
		{"FROM 1830 TO 1860", "1820", 1, true},
		{"FROM 1830 TO 1860", "1845", 0, false},
		{"FROM 1830 TO 1860", "BET 1830 AND 1860", 0, true},
		{"FROM 1830", "1870", 0, false},
		{"FROM 1830", "1820", 1, true},
		{"TO 1860", "1870", -1, true},
		{"TO 1860", "1820", 0, false},
		{"FROM 1830 TO 1840", "FROM 1850 TO 1860", -1, true},
		{"FROM 1830 TO 1850", "FROM 1840 TO 1860", 0, false},

		// how a date was reached bounds nothing
		{"EST 1834", "1834", 0, true},
		{"EST BET 1840 AND 1850", "1860", -1, true},
		{"FROM EST 1830 TO CAL 1860", "FROM 1830 TO 1860", 0, true},

		// a day without its month is a day of one of the twelve
		{"3 ? 1850", "1850", 0, false},
		{"3 ? 1850", "1851", -1, true},
		{"3 ? 1850", "2 JAN 1850", 1, true},
		{"3 ? 1850", "4 DEC 1850", -1, true},
		{"3 ? 1850", "3 JAN 1850", 0, false},
		{"3 ?", "3 ?", 0, false},

		// a year counted from 25 March is named for the 1 January inside it, so a day
		// on or after the new year falls in the year before
		{"15 FEB 1700 NY 25 MAR", "15 FEB 1700", 0, true},
		{"1 APR 1700 NY 25 MAR", "1 APR 1699", 0, true},
		{"25 MAR 1700 NY 25 MAR", "25 MAR 1699", 0, true},
		{"24 MAR 1700 NY 25 MAR", "24 MAR 1700", 0, true},
		{"1 APR 1700 NY 25 MAR", "15 FEB 1700 NY 25 MAR", -1, true},
		{"1700 NY 25 MAR", "BET 25 MAR 1699 AND 24 MAR 1700", 0, true},
		{"1700 NY 25 MAR", "24 MAR 1699", 1, true},
		{"1700 NY 25 MAR", "25 MAR 1700", -1, true},
		{"MAR 1700 NY 25 MAR", "1 MAR 1700", 0, false},
		{"MAR 1700 NY 25 MAR", "1 MAR 1699", 1, true},
		{"MAR 1700 NY 25 MAR", "1 APR 1700", -1, true},
		{"MAR 1700 NY 25 MAR", "1 APR 1699", 0, false},
		{"12 MAR 1834 09:20 NY 1 SEP", "12 MAR 1834 10:00", -1, true},
		{"12 SEP 1834 09:20 NY 1 SEP", "12 SEP 1833 09:20", 0, true},

		// wording says nothing the arithmetic can use
		{"ABT 1854 (the winter of the cholera)", "1860", -1, true},
		{"ABT 1854 (the winter of the cholera)", "ABT 1854", 0, true},
		{"(before the war)", "1834", 0, false},
		{"(before the war)", "(before the war)", 0, false},
	}
	for _, c := range cases {
		a, err := ParseDate(c.a)
		if err != nil {
			t.Fatalf("%q: %v", c.a, err)
		}
		b, err := ParseDate(c.b)
		if err != nil {
			t.Fatalf("%q: %v", c.b, err)
		}
		got, ok := Compare(a, b)
		if got != c.want || ok != c.ok {
			t.Errorf("Compare(%q, %q) = (%d, %v), want (%d, %v)", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

// TestCompareAcrossCalendars: two dates order as readily in two reckonings as in one,
// because both reach the same coordinate before they are compared.
func TestCompareAcrossCalendars(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		// the same Friday, written either way
		{"JULIAN 5 OCT 1582", "15 OCT 1582", 0, true},
		{"JULIAN 1 MAR 1700", "1 MAR 1700", 1, true},
		{"JULIAN 1 MAR 1700", "12 MAR 1700", 0, true},
		{"HEBREW 1 TSH 5784", "16 SEP 2023", 0, true},
		{"FRENCH_R 18 BRUM 8", "9 NOV 1799", 0, true},
		{"FRENCH_R 18 BRUM 8", "1800", -1, true},
		{"44 BCE", "43 BCE", -1, true},
		{"44 BCE", "1834", -1, true},
		{"1 BCE", "1", -1, true},

		// a Hebrew year straddles two Gregorian ones, so neither contains the other
		{"HEBREW 5784", "2023", 0, false},

		{"PERSIAN 1 FARVA 1403", "20 MAR 2024", 0, true},
		{"PERSIAN 30 ESFAN 1403", "PERSIAN 1 FARVA 1404", -1, true},
		{"ISLAMIC 1 MUHAR 1400", "21 NOV 1979", 0, true},
		{"ISLAMIC 3 SAFAR 1319", "HEBREW 17 IYR 5661", 1, true},
		{"SWEDISH 28 JUN 1709", "JULIAN 27 JUN 1709", 0, true},
		{"SWEDISH 28 JUN 1709", "8 JUL 1709", 0, true},
		{"SWEDISH 30 FEB 1712", "JULIAN 29 FEB 1712", 0, true},
		{"SWEDISH 30 FEB 1712", "SWEDISH 1 MAR 1712", -1, true},
		{"SWEDISH 1712", "JULIAN 1712", 0, false},

		// the two ends of a period may be written in two calendars
		// and Julian 20 August 1917 is 2 September in the Gregorian calendar
		{"FROM JULIAN 20 AUG 1917 TO 1920", "3 SEP 1917", 0, false},
		{"FROM JULIAN 20 AUG 1917 TO 1920", "1 SEP 1917", 1, true},
	} {
		got, ok := Compare(mustParse(t, c.a), mustParse(t, c.b))
		if got != c.want || ok != c.ok {
			t.Errorf("Compare(%q, %q) = (%d, %v), want (%d, %v)", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}

func TestCompareIsAntisymmetric(t *testing.T) {
	for _, x := range canonical {
		for _, y := range canonical {
			a, err := ParseDate(x)
			if err != nil {
				t.Fatal(err)
			}
			b, err := ParseDate(y)
			if err != nil {
				t.Fatal(err)
			}
			forward, okForward := Compare(a, b)
			back, okBack := Compare(b, a)
			if okForward != okBack || forward != -back {
				t.Errorf("Compare(%q, %q) = (%d, %v) but Compare(%q, %q) = (%d, %v)",
					x, y, forward, okForward, y, x, back, okBack)
			}
		}
	}
}

func TestText(t *testing.T) {
	for _, in := range canonical {
		var d Date
		if err := d.UnmarshalText([]byte(in)); err != nil {
			t.Errorf("UnmarshalText(%q): %v", in, err)
			continue
		}
		b, err := d.MarshalText()
		if err != nil {
			t.Errorf("MarshalText(%q): %v", in, err)
			continue
		}
		if string(b) != in {
			t.Errorf("round trip through text gave %q, want %q", b, in)
		}
	}

	var zero Date
	if !zero.IsZero() {
		t.Error("the zero Date should report itself zero")
	}
	if b, err := zero.MarshalText(); err != nil || len(b) != 0 {
		t.Errorf("zero Date marshalled to %q, %v", b, err)
	}
	var back Date
	if err := back.UnmarshalText(nil); err != nil || !back.IsZero() {
		t.Errorf("empty text should unmarshal to the zero Date, got %+v, %v", back, err)
	}
}

func TestADateWithinAnother(t *testing.T) {
	cases := []struct {
		d, span             string
		certainly, possibly bool
	}{
		{"1885", "FROM 1880 TO 1902", true, true},
		{"1880", "FROM 1880 TO 1902", true, true},
		{"1875", "FROM 1880 TO 1902", false, false},
		{"BET 1878 AND 1882", "FROM 1880 TO 1902", false, true},
		{"AFT 1890", "FROM 1880 TO 1902", false, true},
		{"1850", "BEF 1900", true, true},
		{"1903", "FROM 1880", true, true},
		{"MAR 1834", "1834", true, true},
		{"1834", "MAR 1834", false, true},
		{"3 MAR", "FROM 1880 TO 1902", false, true},
		{"1885", "NOV", false, true},
	}
	for _, c := range cases {
		certainly, possibly := Within(mustParse(t, c.d), mustParse(t, c.span))
		if certainly != c.certainly || possibly != c.possibly {
			t.Errorf("Within(%s, %s) = %v, %v, want %v, %v", c.d, c.span, certainly, possibly, c.certainly, c.possibly)
		}
	}
	if certainly, possibly := Within(Date{}, mustParse(t, "1880")); certainly || !possibly {
		t.Error("an undated event may have happened at any time")
	}
}

func TestYearsBetween(t *testing.T) {
	years := func(a, b string) (float64, float64, bool) {
		t.Helper()
		return YearsBetween(mustParse(t, a), mustParse(t, b))
	}
	near := func(got, want float64) bool { return got-want < 0.01 && want-got < 0.01 }

	if least, most, ok := years("12 MAR 1834", "12 MAR 1864"); !ok || !near(least, 30) || !near(most, 30) {
		t.Errorf("two exact dates are an exact span apart: %v %v %v", least, most, ok)
	}

	// a death somewhere in 1860 and a birth somewhere in 1861 may be one day apart
	if least, most, ok := years("1860", "1861"); !ok || least > 0.01 || !near(most, 2) {
		t.Errorf("year-wide spans: least %v most %v ok %v", least, most, ok)
	}

	// the same year gives a span that reaches either way, because neither is placed
	if least, most, ok := years("1834", "1834"); !ok || least > -0.9 || most < 0.9 {
		t.Errorf("one year against itself: least %v most %v ok %v", least, most, ok)
	}

	if _, _, ok := years("NOV", "1834"); ok {
		t.Error("a floating date has no place on the calendar and no span to measure")
	}
	if _, _, ok := years("1834", "3 MAR"); ok {
		t.Error("a floating date has no place on the calendar and no span to measure")
	}

	// an open-ended date measures what it can and leaves the rest unbounded
	if least, most, ok := years("BEF 1900", "1901"); !ok || !near(least, 1.0) || most < 1000 {
		t.Errorf("BEF: least %v most %v ok %v", least, most, ok)
	}
	if least, most, ok := years("AFT 1850", "1855"); !ok || least > -1000 || !near(most, 5) {
		t.Errorf("AFT: least %v most %v ok %v", least, most, ok)
	}
}

func TestDaysFromCivil(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		want             int
	}{
		{1970, 1, 1, 0},
		{1970, 1, 2, 1},
		{1969, 12, 31, -1},
		{1972, 2, 29, 789},
		{1, 1, 1, -719162},
		{2000, 3, 1, 11017},
	} {
		if got := daysFromCivil(c.year, c.month, c.day); got != c.want {
			t.Errorf("daysFromCivil(%d, %d, %d) = %d, want %d", c.year, c.month, c.day, got, c.want)
		}
	}

	// every day of a leap year and the year after it must advance by exactly one
	previous := daysFromCivil(1900, 1, 1)
	for _, year := range []int{1900, 1901, 1904, 2000} {
		for month := 1; month <= 12; month++ {
			for day := 1; day <= gregorianDaysIn(month, year); day++ {
				got := daysFromCivil(year, month, day)
				if year == 1900 && month == 1 && day == 1 {
					continue
				}
				if got != previous+1 {
					t.Fatalf("%d-%02d-%02d jumped from %d to %d", year, month, day, previous, got)
				}
				previous = got
			}
		}
		if year == 1901 {
			previous = daysFromCivil(1904, 1, 1) - 1
		}
		if year == 1904 {
			previous = daysFromCivil(2000, 1, 1) - 1
		}
	}
}

func mustParse(t *testing.T, s string) Date {
	t.Helper()
	d, err := ParseDate(s)
	if err != nil {
		t.Fatalf("ParseDate(%q): %v", s, err)
	}
	return d
}

// A sort date puts an event in order and claims nothing of when it happened, so it is
// only ever asked where the event goes.
func TestASortDatePlacesAnEventAndDatesNothing(t *testing.T) {
	census := mustParse(t, "1861")
	between := mustParse(t, "BET 1855 AND 1858")
	undated := Event{Type: "residence", Sort: mustParse(t, "1857")}
	dated := Event{Type: "census", Date: census}
	placed := Event{Type: "residence", Date: between, Sort: mustParse(t, "1857")}

	if got := undated.Placing(); got.String() != "1857" {
		t.Errorf("an undated event is placed at %q", got)
	}
	if got := dated.Placing(); got.String() != "1861" {
		t.Errorf("an event without a sort date is placed at %q", got)
	}
	if got := placed.Placing(); got.String() != "1857" {
		t.Errorf("an event whose date does not place it is placed at %q", got)
	}
	if !undated.Date.IsZero() || placed.Date.String() != "BET 1855 AND 1858" {
		t.Error("a sort date became a date")
	}
}

// A sort key is the Gregorian day a date begins on, whatever calendar it was written in
// and however it is bounded, so that a list can order dates as text.
func TestASortKeyIsTheDayADateBeginsOn(t *testing.T) {
	for written, want := range map[string]string{
		"1834":                  "1834-01-01",
		"MAR 1834":              "1834-03-01",
		"15 MAR 1834":           "1834-03-15",
		"BEF 1900":              "1900-01-01",
		"AFT 1900":              "1900-01-01",
		"ABT 1845":              "1845-01-01",
		"BET 1855 AND 1858":     "1855-01-01",
		"FROM 1870 TO 1901":     "1870-01-01",
		"TO 1901":               "1901-01-01",
		"JULIAN 4 OCT 1582":     "1582-10-14",
		"12 FEB 1750/51":        "1751-02-12",
		"1700 NY 25 MAR":        "1699-03-25",
		"15 FEB 1700 NY 25 MAR": "1700-02-15",
		"HEBREW 1 TSH 5785":     "2024-10-03",
		"25 DEC 1 BCE":          "0000-12-25",
		"SWEDISH 30 FEB 1712":   "1712-03-11",
	} {
		got, ok := mustParse(t, written).SortKey()
		if !ok || got != want {
			t.Errorf("%s sorts at %q (%v), want %s", written, got, ok, want)
		}
	}
	for _, floating := range []string{"3 MAR", "(before the war)"} {
		if got, ok := mustParse(t, floating).SortKey(); ok {
			t.Errorf("%s has no place on the line and sorts at %s", floating, got)
		}
	}
	for days := -800000; days <= 800000; days += 997 {
		if y, m, d := civilFromDays(days); daysFromCivil(y, m, d) != days {
			t.Fatalf("day %d reads as %d-%02d-%02d and back as %d", days, y, m, d, daysFromCivil(y, m, d))
		}
	}
}

// A date restated in the Gregorian calendar covers the days it covered in its own, as
// the day, month or year they make up there or as a range between the first and the
// last, its qualifier and how it was reached kept.
func TestADateRestatedInTheGregorianCalendar(t *testing.T) {
	for written, want := range map[string]string{
		"1834":                                     "1834",
		"MAR 1834":                                 "MAR 1834",
		"12 MAR 1834 09:20":                        "12 MAR 1834 09:20",
		"ABT 1834 (the winter of the cholera)":     "ABT 1834",
		"SWEDISH 30 FEB 1712":                      "11 MAR 1712",
		"HEBREW 1 TSH 5785":                        "3 OCT 2024",
		"HEBREW TSH 5785":                          "BET 3 OCT 2024 AND 1 NOV 2024",
		"JULIAN 1700":                              "BET 11 JAN 1700 AND 11 JAN 1701",
		"EST JULIAN 1700":                          "EST BET 11 JAN 1700 AND 11 JAN 1701",
		"ABT JULIAN 1700":                          "BET 11 JAN 1700 AND 11 JAN 1701",
		"BEF JULIAN 1700":                          "BEF 11 JAN 1700",
		"AFT JULIAN 1700":                          "AFT 11 JAN 1701",
		"BET JULIAN 1700 AND JULIAN 1710":          "BET 11 JAN 1700 AND 11 JAN 1711",
		"BET 1830 AND 1835":                        "BET 1830 AND 1835",
		"12 FEB 1750/51":                           "12 FEB 1751",
		"ISLAMIC 1 RAMAD 1250":                     "1 JAN 1835",
		"PERSIAN 12 FARVA 1300":                    "1 APR 1921",
		"FROM SWEDISH 1704 TO SWEDISH 30 FEB 1712": "FROM BET 11 JAN 1704 AND 10 JAN 1705 TO 11 MAR 1712",
		"25 DEC 1 BCE":                             "25 DEC 1 BCE",
	} {
		got, ok := mustParse(t, written).InGregorian()
		if !ok || got.String() != want {
			t.Errorf("%s is restated as %q (%v), want %q", written, got, ok, want)
		}
	}
	for _, floating := range []string{"3 MAR", "(before the war)", "FROM 3 MAR TO 1860"} {
		if got, ok := mustParse(t, floating).InGregorian(); ok {
			t.Errorf("%s has no days to restate and is restated as %s", floating, got)
		}
	}
}
