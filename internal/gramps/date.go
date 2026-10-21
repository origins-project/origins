package gramps

import (
	"strconv"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// calendars are Gramps' calendars by the names its XML writes them under. They are the
// seven the model reckons in, so a date is never converted on the way in.
var calendars = map[string]model.Calendar{
	"Gregorian":         model.Gregorian,
	"Julian":            model.Julian,
	"Hebrew":            model.Hebrew,
	"French Republican": model.FrenchRepublican,
	"Persian":           model.Persian,
	"Islamic":           model.Islamic,
	"Swedish":           model.Swedish,
}

// dateOf reads the date an element holds, in whichever of Gramps' four forms it is
// written: a dateval, a daterange, a datespan or a datestr. Anything the date could
// not carry comes back as lines for the report.
func dateOf(e *Element) (model.Date, []string) {
	if e == nil {
		return model.Date{}, nil
	}
	for _, child := range e.Children {
		switch child.Name {
		case "dateval", "daterange", "datespan", "datestr":
			return readDate(child.Mark())
		}
	}
	return model.Date{}, nil
}

// readDate reads one of Gramps' date elements.
//
// A dateval is a date, about, before and after qualifying it, and from and to making it
// a period open at one end; a daterange is a range and a datespan a period; a datestr
// is a date that is only words. Estimated and calculated are EST and CAL in front of
// whatever else the date says, each end of a period saying it for itself, and a new
// year other than the first of January is the NY of every date in it. The date is built
// as the model holds one and read back through the model's own grammar, so a date Gramps
// holds and the model cannot state — a thirtieth of February, a range ending before it
// starts — survives as its words rather than as a date the model would refuse to read
// back.
//
// A new year on a date that is dual-dated, whose dual year already says where its year
// turned, or on one the grammar cannot count from another day, is left unread, for the
// date to be kept as written beside what was made of it.
func readDate(e *Element) (model.Date, []string) {
	if e.Name == "datestr" {
		return model.Date{Wording: collapse(e.Use("val"))}, nil
	}
	calendar := model.Gregorian
	if name := e.Attr("cformat"); name != "" {
		known, ok := calendars[name]
		if !ok {
			return worded(e), []string{"a date in a calendar origins does not reckon in was kept as words"}
		}
		e.Use("cformat")
		calendar = known
	}
	dual := e.Use("dualdated") == "1"
	newYear, counted := newYearOf(e.Attr("newyear"))
	if counted && newYear == nil {
		e.Use("newyear")
	}
	reached := model.Stated
	switch e.Attr("quality") {
	case "estimated":
		reached = model.Estimated
	case "calculated":
		reached = model.Calculated
	}
	if reached != model.Stated {
		e.Use("quality")
	}

	if date, ok := build(e, calendar, dual, reached, newYear); ok {
		if counted && stated(date) {
			e.Use("newyear")
		}
		return date, nil
	}
	if newYear != nil {
		if date, ok := build(e, calendar, dual, reached, nil); ok {
			return date, nil
		}
	}
	return worded(e), []string{"a date could not be read and was kept as words"}
}

// stated reports whether a date, or either end of it, says what day its year was
// counted from.
func stated(d model.Date) bool {
	for _, end := range []*model.Date{&d, d.End, d.From, d.To} {
		if end != nil && end.NewYear != nil {
			return true
		}
	}
	return false
}

// build makes the date one of Gramps' date elements holds, as the model's grammar reads
// it back, reporting whether it could.
func build(e *Element, calendar model.Calendar, dual bool, reached model.Reached, newYear *model.NewYear) (model.Date, bool) {
	value := func(attr string) (model.Date, bool) {
		d, ok := ymd(e.Attr(attr), calendar, dual)
		d.Reached = reached
		if d.Year != nil && !dual {
			d.NewYear = newYear
		}
		return d, ok
	}
	var date model.Date
	switch e.Name {
	case "dateval":
		d, ok := value("val")
		if !ok {
			return date, false
		}
		switch e.Attr("type") {
		case "about":
			d.Qualifier = model.About
		case "before":
			d.Qualifier = model.Before
		case "after":
			d.Qualifier = model.After
		}
		switch e.Attr("type") {
		case "from":
			date = model.Date{From: &d}
		case "to":
			date = model.Date{To: &d}
		default:
			date = d
		}
	case "daterange", "datespan":
		start, ok := value("start")
		stop, stopped := value("stop")
		if !ok || !stopped {
			return date, false
		}
		if e.Name == "daterange" {
			stop.Reached = model.Stated
			start.Qualifier, start.End = model.Between, &stop
			date = start
		} else {
			date = model.Date{From: &start, To: &stop}
		}
	default:
		return date, false
	}
	if date.IsZero() {
		return date, true
	}
	read, err := model.ParseDate(date.String())
	if err != nil {
		return date, false
	}
	switch e.Name {
	case "dateval":
		e.Use("val")
		e.Use("type")
	default:
		e.Use("start")
		e.Use("stop")
	}
	return read, true
}

// newYearOf reads the day Gramps says a year was counted from: one of the three it
// names, or a month and a day of somebody's own. The first of January is every other
// year, and is no new year to state; one Gramps could not have written is not counted.
func newYearOf(text string) (*model.NewYear, bool) {
	switch text = strings.ToLower(strings.TrimSpace(text)); text {
	case "", "0", "jan1":
		return nil, true
	case "1", "mar1":
		return &model.NewYear{Month: 3, Day: 1}, true
	case "2", "mar25":
		return &model.NewYear{Month: 3, Day: 25}, true
	case "3", "sep1":
		return &model.NewYear{Month: 9, Day: 1}, true
	}
	month, day, ok := strings.Cut(text, "-")
	m, err1 := strconv.Atoi(month)
	d, err2 := strconv.Atoi(day)
	if !ok || err1 != nil || err2 != nil || m < 1 || m > 12 || d < 1 || d > 31 {
		return nil, false
	}
	if m == 1 && d == 1 {
		return nil, true
	}
	return &model.NewYear{Month: m, Day: d}, true
}

// ymd reads a Gramps date value, year-month-day with as much of it as the date gives.
// A year or a month of nought, or of question marks, is one the record did not give, a
// day without its month among them, and a negative year is
// before the common era. A dual-dated year is written by Gramps as the later of its
// two, 1751 for 1750/51, and the model writes the earlier.
func ymd(text string, calendar model.Calendar, dual bool) (date model.Date, ok bool) {
	date.Calendar = calendar
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "-") {
		date.BCE, text = true, text[1:]
	}
	parts := strings.Split(text, "-")
	if len(parts) > 3 {
		return date, false
	}
	numbers := make([]int, 3)
	for i, part := range parts {
		// Gramps writes a part nobody gave as question marks as well as noughts
		if strings.Trim(part, "?") == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || i == 1 && n > calendar.Months() {
			return date, false
		}
		numbers[i] = n
	}
	year, month, day := numbers[0], numbers[1], numbers[2]
	if year != 0 {
		if dual {
			year--
			date.Dual = true
		}
		date.Year = &year
	} else {
		date.BCE = false
	}
	if month != 0 {
		date.Month = &month
	}
	if day != 0 {
		date.Day = &day
	}
	return date, true
}

// worded is a date Gramps holds and the model cannot state, as the words of the element
// it was written in.
func worded(e *Element) model.Date {
	var words string
	switch e.Name {
	case "daterange":
		words = "between " + e.Use("start") + " and " + e.Use("stop")
	case "datespan":
		words = "from " + e.Use("start") + " to " + e.Use("stop")
	default:
		words = e.Use("type") + " " + e.Use("val")
	}
	if calendar := e.Use("cformat"); calendar != "" {
		words += " in the " + calendar + " calendar"
	}
	return model.Date{Wording: collapse(words)}
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
