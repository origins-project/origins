package gedcom

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// readDate reads a GEDCOM DATE value into a date, with the words the record gave beside
// it as the date's wording. read is false where the value was meant as a date the
// grammar cannot state, which survives as wording alone and is the caller's to report.
//
// The two grammars overlap almost exactly: ABT, EST, CAL, BEF, AFT, BET..AND and
// FROM..TO all mean here what they mean there, 1750/51 is dual-dated in both, and the
// calendars and the epoch are the ones the corpus writes. What GEDCOM has and the model
// does not is somewhere to put words a date cannot be made of, and those become the
// wording of a date that is only words, which is what a record saying them means.
func readDate(value string) (date model.Date, read bool) {
	text := strings.TrimSpace(value)
	if text == "" {
		return model.Date{}, true
	}

	// A phrase is GEDCOM's escape hatch for a date its grammar cannot state, and a
	// yearless date is exactly what it gets used for. Recovering one into a real date is
	// the whole reason the model's dates are not GEDCOM's.
	if phrase, ok := parenthesised(text); ok {
		return phrased(phrase), true
	}

	// INT gives an interpreted date and the words it was interpreted from. The date is
	// usable and the words are the evidence for it, so both are kept.
	if rest, found := cutWord(text, "INT"); found {
		stated, phrase := splitPhrase(rest)
		date, read := readDate(stated)
		return worded(date, phrase), read
	}

	// A calendar's name with no date after it is a word somebody wrote, the DATE UNKNOWN
	// of an exporter that means the date is.
	if escape, rest, ok := calendar(text); ok && rest != "" {
		if _, reckoned := corpusCalendar[escape]; !reckoned {
			// A calendar origins does not reckon in could only be stated by converting
			// it, and converting would put a day in the record that the record never
			// gave. The words are kept with the calendar named, which is what a reader
			// needs to convert it by hand later.
			return worded(model.Date{}, rest+" ("+calendarName(escape)+")"), false
		}
	}

	if d, err := model.ParseDate(tolerate(text)); err == nil {
		return d, true
	}
	return worded(model.Date{}, text), false
}

// readDateNode reads a DATE line together with what version 7.0 puts on lines beneath
// it. That version moved the words a date was read from out of the value, where 5.5.1
// wrote them as INT <date> (<phrase>), and into a PHRASE line; they mean the same thing,
// become the same wording, and get the same chance to turn out to be a date the GEDCOM
// grammar could not state.
func readDateNode(node *Node) (date model.Date, read bool) {
	value := node.Text()
	if d, ok := restated(value, node.First("TIME").Text(), node.First("PHRASE").Text()); ok {
		node.Take("TIME")
		node.Take("PHRASE")
		return d, true
	}
	date, read = readDate(value)

	// A TIME belongs to the date on the line above it, and the model carries one.
	if clock := strings.TrimSpace(node.First("TIME").Text()); clock != "" {
		if withClock, ok := asDate(value + " " + clock); ok {
			withClock.Wording = date.Wording
			date = withClock
			node.Take("TIME")
		}
	}

	written := collapse(node.Take("PHRASE").Text())
	switch {
	case written == "" || written == date.Wording:
		return date, read
	case date.IsZero():
		return phrased(written), read
	}
	return worded(date, written), read
}

// restated is the date a phrase is where it is the date as origins writes it and the
// value and the time beside it are what origins would state of it: a date 7.0 could only
// say less of, a value restated in another calendar or a date with no value at all,
// which another program's phrase saying the same words would not be.
func restated(value, clock, phrase string) (model.Date, bool) {
	phrase = collapse(phrase)
	if phrase == "" {
		return model.Date{}, false
	}
	d, err := model.ParseDate(phrase)
	if err != nil || d.String() != phrase {
		return model.Date{}, false
	}
	v, c, p := statement(d)
	return d, v == collapse(value) && c == collapse(clock) && p == phrase
}

// phrased is the date some words are, or the words alone where they are not one.
func phrased(words string) model.Date {
	if d, ok := asDate(words); ok {
		return d
	}
	return worded(model.Date{}, words)
}

// worded adds a record's words to a date's wording.
func worded(date model.Date, words string) model.Date {
	switch words = collapse(words); {
	case words == "":
	case date.Wording == "":
		date.Wording = words
	default:
		date.Wording += "; " + words
	}
	return date
}

// asDate reads words as a date if they are one, and not merely more words.
func asDate(words string) (model.Date, bool) {
	d, err := model.ParseDate(tolerate(words))
	return d, err == nil && d.Wording == ""
}

func parenthesised(s string) (string, bool) {
	if len(s) >= 2 && strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return strings.TrimSpace(s[1 : len(s)-1]), true
	}
	return "", false
}

// splitPhrase separates a value from the parenthesised words trailing it.
func splitPhrase(s string) (value, phrase string) {
	s = strings.TrimSpace(s)
	open := strings.Index(s, "(")
	if open < 0 || !strings.HasSuffix(s, ")") {
		return s, ""
	}
	return strings.TrimSpace(s[:open]), strings.TrimSpace(s[open+1 : len(s)-1])
}

// cutWord removes a leading keyword, matched as a whole word and without regard to case.
func cutWord(s, word string) (rest string, found bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 || !strings.EqualFold(fields[0], word) {
		return s, false
	}
	return strings.Join(fields[1:], " "), true
}

// calendars are the reckonings the two versions name. Version 5.5.1 wraps the name in
// an escape — @#DJULIAN@ — and 7.0 writes it as the first word of the value, where its
// grammar reads date = [calendar] [[day] month] year [epoch]. FRENCH R in the one is
// FRENCH_R in the other.
var calendars = map[string]string{
	"GREGORIAN": "Gregorian",
	"JULIAN":    "Julian",
	"HEBREW":    "Hebrew",
	"FRENCH":    "French Republican",
	"FRENCH_R":  "French Republican",
	"ROMAN":     "Roman",
	"UNKNOWN":   "an unnamed calendar",
}

// calendar splits off the reckoning a date is stated in, written either way.
func calendar(s string) (name, rest string, ok bool) {
	if body, escaped := strings.CutPrefix(s, "@#D"); escaped {
		inside, after, closed := strings.Cut(body, "@")
		if !closed {
			return "", s, false
		}
		// FRENCH R is the only escape written as two words, and the first names it.
		inside, _, _ = strings.Cut(strings.ToUpper(strings.TrimSpace(inside)), " ")
		return inside, strings.TrimSpace(after), true
	}

	word, after, _ := strings.Cut(strings.TrimSpace(s), " ")
	if word = strings.ToUpper(word); calendars[word] == "" {
		return "", s, false
	}
	return word, strings.TrimSpace(after), true
}

// calendarName spells a calendar the way a reader would write it, since the note it
// lands in is prose and not a tag.
func calendarName(name string) string {
	if spelled := calendars[name]; spelled != "" {
		return spelled
	}
	return name
}

// corpusCalendar maps the calendars GEDCOM names onto the words the corpus writes.
// Roman and the unnamed one are absent because origins does not reckon in them, and a
// date stated in one is kept as words instead.
var corpusCalendar = map[string]string{
	"GREGORIAN": "",
	"JULIAN":    "JULIAN",
	"HEBREW":    "HEBREW",
	"FRENCH":    "FRENCH_R",
	"FRENCH_R":  "FRENCH_R",
}

// calendarEscape matches 5.5.1's way of naming a calendar. FRENCH R is the one name
// written as two words.
var calendarEscape = regexp.MustCompile(`@#D([A-Za-z_]+(?: [A-Za-z_]+)*)@ *`)

// calendarWords rewrites the escape into the word 7.0 and the corpus both use. A range
// names its reckoning on each end, so this runs over the whole value and not only over
// its first word; an escape naming a calendar origins cannot reckon in is left standing,
// which is what makes the value fail to parse and survive as words.
func calendarWords(s string) string {
	return calendarEscape.ReplaceAllStringFunc(s, func(match string) string {
		inside := calendarEscape.FindStringSubmatch(match)[1]
		name, _, _ := strings.Cut(strings.ToUpper(strings.TrimSpace(inside)), " ")
		keyword, reckoned := corpusCalendar[name]
		if !reckoned {
			return match
		}
		if keyword == "" {
			return ""
		}
		return keyword + " "
	})
}

// epochWords rewrites the several spellings of the era a year is counted back from into
// the one the corpus writes. Both versions put it after the year, and 7.0 standardises
// this one alone — its other calendars count from their own beginnings and have no
// before.
func epochWords(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		switch strings.ToUpper(f) {
		case "BC", "BCE", "B.C.", "B.C.E.":
			fields[i] = "BCE"
		}
	}
	return strings.Join(fields, " ")
}

// englishMonth maps what exporters actually write to the three letters the model reads.
// The spelled-out forms are not GEDCOM, but plenty of files carry them.
var englishMonth = map[string]string{
	"JANUARY": "JAN", "FEBRUARY": "FEB", "MARCH": "MAR", "APRIL": "APR",
	"MAY": "MAY", "JUNE": "JUN", "JULY": "JUL", "AUGUST": "AUG",
	"SEPTEMBER": "SEP", "OCTOBER": "OCT", "NOVEMBER": "NOV", "DECEMBER": "DEC",
	"SEPT": "SEP",
}

// tolerate rewrites what exporters write into what the model reads. It is the one place
// where the file's spelling of a date is bent towards the format's, which keeps the
// tolerance at the boundary rather than loosening the grammar origins writes.
func tolerate(s string) string { return dualYears(months(epochWords(calendarWords(s)))) }

// dualYear matches a year written as two, the form used either side of the Julian to
// Gregorian change. The model writes 1750/51; exporters also write 1750/1751 and
// 1750/1, and a leading day or month keeps its own field, so the whole field must be
// the year for this to apply.
var dualYear = regexp.MustCompile(`^([0-9]{3,4})/([0-9]{1,4})$`)

// dualYears rewrites a dual year into the two digits the model reads. The second part
// has to name the year after the first, which is what dual dating means — so 1546/7 and
// 1510/1511 are both rewritten, and 1056/1060 is left alone to be kept as a phrase,
// because two years four apart are a guess at a range and not one year written twice.
func dualYears(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		match := dualYear.FindStringSubmatch(f)
		if match == nil {
			continue
		}
		year, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		next := fmt.Sprintf("%04d", year+1)
		if written := match[2]; written != next[len(next)-len(written):] {
			continue
		}
		fields[i] = match[1] + "/" + next[2:]
	}
	return strings.Join(fields, " ")
}

// months rewrites spelled-out month names to the abbreviations the model's grammar
// uses.
func months(s string) string {
	fields := strings.Fields(s)
	for i, f := range fields {
		if month, ok := englishMonth[strings.ToUpper(strings.Trim(f, "."))]; ok {
			fields[i] = month
		}
	}
	return strings.Join(fields, " ")
}
