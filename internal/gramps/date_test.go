package gramps

import (
	"slices"
	"strings"
	"testing"
)

func element(t *testing.T, xml string) *Element {
	t.Helper()
	e, err := parse(strings.NewReader(xml))
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Every form a Gramps date takes, as the date the model writes, what of it is reported
// and which attribute of it is left unread.
func TestADateIsReadAsTheModelStatesOne(t *testing.T) {
	for _, tc := range []struct {
		xml, want, note string
		loose           []string
	}{
		{`<dateval val="1834-03-12"/>`, "12 MAR 1834", "", nil},
		{`<dateval val="1834-03"/>`, "MAR 1834", "", nil},
		{`<dateval val="1834"/>`, "1834", "", nil},
		// a year or a month of nought is one the record did not give
		{`<dateval val="0000-03-12"/>`, "12 MAR", "", nil},
		{`<dateval val="????-06-24"/>`, "24 JUN", "", nil},
		// and a day may be given without its month
		{`<dateval val="1834-00-12"/>`, "12 ? 1834", "", nil},
		{`<dateval val="1834" type="about"/>`, "ABT 1834", "", nil},
		{`<dateval val="1834" type="before"/>`, "BEF 1834", "", nil},
		{`<dateval val="1834" type="after"/>`, "AFT 1834", "", nil},
		{`<dateval val="1834" type="from"/>`, "FROM 1834", "", nil},
		{`<dateval val="1834" type="to"/>`, "TO 1834", "", nil},
		{`<daterange start="1834" stop="1836"/>`, "BET 1834 AND 1836", "", nil},
		{`<datespan start="1834-03" stop="1861"/>`, "FROM MAR 1834 TO 1861", "", nil},
		{`<dateval val="1834" quality="estimated"/>`, "EST 1834", "", nil},
		{`<dateval val="1834" quality="calculated"/>`, "CAL 1834", "", nil},
		{`<datespan start="1800" stop="1805" quality="estimated"/>`, "FROM EST 1800 TO EST 1805", "", nil},
		// how a date was reached stands in front of whatever else it says
		{`<dateval val="1850" type="about" quality="estimated"/>`, "EST ABT 1850", "", nil},
		{`<dateval val="1850" type="from" quality="estimated"/>`, "FROM EST 1850", "", nil},
		{`<daterange start="1186" stop="1188" quality="calculated"/>`, "CAL BET 1186 AND 1188", "", nil},
		// Gramps holds a dual year as the later of the two
		{`<dateval val="1751-03-10" cformat="Julian" dualdated="1" newyear="Mar25"/>`, "JULIAN 10 MAR 1750/51", "", []string{"newyear on <dateval>"}},
		// a year counted from another day than the first of January says which
		{`<dateval val="1916-02-05" newyear="Mar25"/>`, "5 FEB 1916 NY 25 MAR", "", nil},
		{`<dateval val="1700-02-15" newyear="Sep1"/>`, "15 FEB 1700 NY 1 SEP", "", nil},
		{`<dateval val="1700-02-15" newyear="3-1"/>`, "15 FEB 1700 NY 1 MAR", "", nil},
		{`<daterange start="1700" stop="1702" newyear="2"/>`, "BET 1700 NY 25 MAR AND 1702 NY 25 MAR", "", nil},
		{`<dateval val="1700-02-15" newyear="Jan1"/>`, "15 FEB 1700", "", nil},
		{`<dateval val="0000-02-15" newyear="Mar25"/>`, "15 FEB", "", []string{"newyear on <dateval>"}},
		{`<dateval val="1700-02-15" newyear="whenever"/>`, "15 FEB 1700", "", []string{"newyear on <dateval>"}},
		{`<dateval val="-044-03-15"/>`, "15 MAR 44 BCE", "", nil},
		{`<dateval val="0241-03-12" cformat="Islamic"/>`, "ISLAMIC 12 RABIA 241", "", nil},
		{`<dateval val="5600-13-01" cformat="Hebrew"/>`, "HEBREW 1 ELL 5600", "", nil},
		{`<datestr val="the Tuesday after Easter"/>`, "(the Tuesday after Easter)", "", nil},
		{`<datestr val=""/>`, "", "", nil},
		// what Gramps holds and the model cannot state is kept as its words
		{`<dateval val="1834-02-30"/>`, "(1834-02-30)", "could not be read", nil},
		{`<daterange start="1699" stop="1600"/>`, "(between 1699 and 1600)", "could not be read", nil},
		{`<dateval val="1834-13-01"/>`, "(1834-13-01)", "could not be read", nil},
		{`<dateval val="0-0100"/>`, "(0-0100)", "could not be read", nil},
		{`<dateval val="5600-14-01" cformat="Hebrew"/>`, "(5600-14-01 in the Hebrew calendar)", "could not be read", nil},
		{`<dateval val="1834-02-30" type="about" cformat="Julian"/>`, "(about 1834-02-30 in the Julian calendar)", "could not be read", nil},
	} {
		e := element(t, tc.xml)
		date, notes := readDate(e)
		if got := date.String(); got != tc.want {
			t.Errorf("%s = %q, want %q", tc.xml, got, tc.want)
		}
		switch {
		case tc.note == "" && len(notes) > 0:
			t.Errorf("%s reported %q", tc.xml, notes)
		case tc.note != "" && !slices.ContainsFunc(notes, func(n string) bool { return strings.Contains(n, tc.note) }):
			t.Errorf("%s reported %q, want a line saying %q", tc.xml, notes, tc.note)
		}
		if tc.note == "" && !slices.Equal(e.Loose(), tc.loose) {
			t.Errorf("%s left %q unread, want %q", tc.xml, e.Loose(), tc.loose)
		}
	}
}
