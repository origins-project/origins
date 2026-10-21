package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Qualifier records how certain a date is, or what kind of bound it places.
type Qualifier int

// The qualifiers, Exact being none.
const (
	Exact Qualifier = iota
	About
	Before
	After
	Between
)

var qualifierKeyword = [...]string{
	Exact:   "",
	About:   "ABT",
	Before:  "BEF",
	After:   "AFT",
	Between: "BET",
}

// Reached records how a date was arrived at where the record does not state it: a
// guess from other evidence, or arithmetic on a stated age. It is a claim apart from
// the qualifier, since an estimate can be a range as well as a year.
type Reached int

// The ways a date is reached, Stated being as the record states it.
const (
	Stated Reached = iota
	Estimated
	Calculated
)

var reachedKeyword = [...]string{
	Stated:     "",
	Estimated:  "EST",
	Calculated: "CAL",
}

// newYearKeyword introduces the day a date's year was counted from.
const newYearKeyword = "NY"

// unknownMonth stands where a day's month is not known, since a bare number is a year.
const unknownMonth = "?"

// NewYear is the day and month a year was counted from, where it was not 1 January.
type NewYear struct{ Month, Day int }

// epochKeyword names the era a year is counted back from. There is one, because only
// the calendars that share the common era have a before at all.
const epochKeyword = "BCE"

// periodStart and periodEnd open the two ends of a period.
const (
	periodStart = "FROM"
	periodEnd   = "TO"
)

// TimeOfDay is a wall-clock time. Second is nil when the record gives only minutes.
type TimeOfDay struct {
	Hour, Minute int
	Second       *int
}

// Date is a point or a span in time as a record states it. Every component is
// independently optional: a date may carry a day and a month with no year, and 1834
// must never be promoted to 1 JAN 1834.
//
// Dual is the 1750/51 notation, which says which day a year number changed on and not
// how time is reckoned — so it is not a calendar. BCE counts the year backwards from
// the start of the common era, and only the two calendars that share that era admit it.
// NewYear is the day a year written once was counted from: the year is the one named
// for the 1 January it holds, so a day on or after the new year falls in the one before.
//
// A day may stand without its month, which the record did not give.
//
// End is set only when Qualifier is Between, where the receiver holds the start.
//
// From and To are the ends of a period, a stretch of time as against a moment somewhere
// in one, and either may be missing. Each is a date of its own, qualifier and calendar
// included, and a period sets no other field but Wording.
//
// Wording is what the record itself said, kept beside the date or in place of one. It
// belongs to the whole value, and a date that is only wording is floating.
type Date struct {
	Reached   Reached
	Qualifier Qualifier
	Calendar  Calendar
	Dual      bool
	BCE       bool
	Year      *int
	Month     *int
	Day       *int
	Time      *TimeOfDay
	NewYear   *NewYear
	End       *Date
	From, To  *Date
	Wording   string
}

// IsZero reports whether the date carries no information at all.
func (d Date) IsZero() bool {
	return d.Year == nil && d.Month == nil && d.Day == nil && d.Time == nil &&
		!d.Period() && d.Wording == ""
}

// Period reports whether the date is a stretch of time rather than a moment in one.
func (d Date) Period() bool { return d.From != nil || d.To != nil }

// Start is the date a timeline places this one at: a period at its beginning, or at its
// end where the beginning is not recorded, and any other date as it stands.
func (d Date) Start() Date {
	switch {
	case d.From != nil:
		return *d.From
	case d.To != nil:
		return *d.To
	}
	return d
}

// SortKey is where a list puts the date: the Gregorian day it begins on, written
// YYYY-MM-DD with the year counted astronomically, so that keys of four-digit years sort
// as text. It is the first day the date's own components admit, at the start of a
// period and before its qualifier is applied, so that BEF 1900 goes with 1900 rather
// than before every other date. ok is false for a floating date. A key puts dates in
// order and says nothing of when anything happened, which Compare asks, and it is
// derived and never written to a corpus.
func (d Date) SortKey() (string, bool) {
	lo, _, ok := d.Start().bounds()
	if !ok {
		return "", false
	}
	year, month, day := civilFromDays(int(floorDiv(lo, secondsInDay)))
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day), true
}

// InGregorian is the date restated in the Gregorian calendar, for a format that has no
// word for the one it was written in: the days its own components admit, as the day,
// month or year they make up there, or as a range from the first to the last where
// they make up none. Its qualifier and how it was reached stay as they were, save that
// an approximate date making up no whole Gregorian day, month or year is the range
// alone, and so do the ends of a period. The record's wording and the new year it
// counted from are not carried, the restated date having no use for either. ok is
// false for a floating date, which has no days to restate.
//
// Like a sort key it is derived and never written to a corpus, which keeps the date in
// the calendar the record wrote it in.
func (d Date) InGregorian() (Date, bool) {
	if d.Period() {
		var out Date
		for _, end := range []struct{ from, to **Date }{{&d.From, &out.From}, {&d.To, &out.To}} {
			if *end.from == nil {
				continue
			}
			restated, ok := (*end.from).InGregorian()
			if !ok {
				return Date{}, false
			}
			*end.to = &restated
		}
		return out, true
	}

	lo, hi, ok := d.bounds()
	if !ok {
		return Date{}, false
	}
	first, last := int(floorDiv(lo, secondsInDay)), int(floorDiv(hi, secondsInDay))
	out := gregorianSpan(first, last)
	switch d.Qualifier {
	case Before:
		if out.Qualifier == Between {
			out = gregorianSpan(first, first)
		}
	case After:
		if out.Qualifier == Between {
			out = gregorianSpan(last, last)
		}
	case Between:
		if d.End == nil {
			return Date{}, false
		}
		end, ok := d.End.InGregorian()
		if !ok {
			return Date{}, false
		}
		if out.Qualifier == Between {
			out = gregorianSpan(first, first)
		}
		if end.Qualifier == Between {
			end = *end.End
		}
		out.End = &end
	}
	if out.Qualifier != Between {
		out.Qualifier = d.Qualifier
	}
	out.Reached = d.Reached
	if first == last && d.Time != nil {
		clock := *d.Time
		out.Time = &clock
	}
	return out, true
}

// gregorianSpan is the Gregorian date covering the days first to last: one day, the
// whole of a month, the whole of a year, or a range from the one day to the other.
func gregorianSpan(first, last int) Date {
	if first == last {
		return gregorianDay(first)
	}
	y1, m1, d1 := civilFromDays(first)
	y2, m2, d2 := civilFromDays(last)
	switch {
	case y1 == y2 && m1 == m2 && d1 == 1 && d2 == gregorianDaysIn(m2, y2):
		out := gregorianDay(first)
		out.Day = nil
		return out
	case y1 == y2 && m1 == 1 && d1 == 1 && m2 == 12 && d2 == 31:
		out := gregorianDay(first)
		out.Month, out.Day = nil, nil
		return out
	}
	out, end := gregorianDay(first), gregorianDay(last)
	out.Qualifier, out.End = Between, &end
	return out
}

// gregorianDay is the Gregorian date of a day, its year written as the era counts it.
func gregorianDay(days int) Date {
	year, month, day := civilFromDays(days)
	out := Date{Month: &month, Day: &day}
	if year <= 0 {
		year, out.BCE = 1-year, true
	}
	out.Year = &year
	return out
}

// Anchored reports whether the date has a place on the calendar, which takes a year at
// every end it states. Anchored dates take part in ordering and age arithmetic; floating
// ones are recorded, displayed and matched, but never computed with. A worn gravestone
// giving "3 marzo" and no year is evidence, not an incomplete record.
func (d Date) Anchored() bool {
	_, _, ok := d.span()
	return ok
}

func (d Date) String() string {
	stated := d.stated()
	switch {
	case d.Wording == "":
		return stated
	case stated == "":
		return "(" + d.Wording + ")"
	}
	return stated + " (" + d.Wording + ")"
}

// stated writes the date without the record's wording.
func (d Date) stated() string {
	switch {
	case d.From != nil && d.To != nil:
		return periodStart + " " + d.From.stated() + " " + periodEnd + " " + d.To.stated()
	case d.From != nil:
		return periodStart + " " + d.From.stated()
	case d.To != nil:
		return periodEnd + " " + d.To.stated()
	}

	var b strings.Builder
	for _, k := range []string{reachedKeyword[d.Reached], qualifierKeyword[d.Qualifier]} {
		if k != "" {
			b.WriteString(k)
			b.WriteByte(' ')
		}
	}
	b.WriteString(d.value())
	if d.Qualifier == Between && d.End != nil {
		b.WriteString(" AND ")
		b.WriteString(d.End.value())
	}
	return b.String()
}

// value writes one date of the grammar: the calendar first and the epoch last, which
// is GEDCOM 7.0's own order, and the new year after the date it binds to. A calendar
// binds to the date it precedes and not to the whole value, so each end of a range
// writes its own.
func (d Date) value() string {
	r := d.reckoning()
	var parts []string
	if d.Calendar != Gregorian {
		parts = append(parts, r.keyword)
	}
	if d.Day != nil {
		parts = append(parts, strconv.Itoa(*d.Day))
	}
	if d.Month != nil {
		parts = append(parts, r.months[*d.Month-1])
	} else if d.Day != nil {
		parts = append(parts, unknownMonth)
	}
	if d.Year != nil {
		if d.Dual {
			parts = append(parts, fmt.Sprintf("%d/%02d", *d.Year, (*d.Year+1)%100))
		} else {
			parts = append(parts, strconv.Itoa(*d.Year))
		}
	}
	if d.BCE {
		parts = append(parts, epochKeyword)
	}
	if d.Time != nil {
		parts = append(parts, d.Time.String())
	}
	if ny := d.NewYear; ny != nil {
		parts = append(parts, newYearKeyword, strconv.Itoa(ny.Day), r.months[ny.Month-1])
	}
	return strings.Join(parts, " ")
}

func (t TimeOfDay) String() string {
	if t.Second != nil {
		return fmt.Sprintf("%02d:%02d:%02d", t.Hour, t.Minute, *t.Second)
	}
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

func (d Date) MarshalText() ([]byte, error) {
	if d.IsZero() {
		return nil, nil
	}
	return []byte(d.String()), nil
}

func (d *Date) UnmarshalText(b []byte) error {
	if len(strings.TrimSpace(string(b))) == 0 {
		*d = Date{}
		return nil
	}
	parsed, err := ParseDate(string(b))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// ParseDate reads a date in the corpus grammar. Keywords, calendar names and month
// names are matched without regard to case and surplus spacing is ignored, in the
// wording as in the date; String always writes the canonical form.
func ParseDate(s string) (Date, error) {
	stated, wording, err := cutWording(s)
	if err != nil {
		return Date{}, fmt.Errorf("parse date %q: %w", s, err)
	}

	var d Date
	switch fields := strings.Fields(stated); {
	case len(fields) > 0:
		if d, err = parseStated(fields); err != nil {
			return Date{}, fmt.Errorf("parse date %q: %w", s, err)
		}
	case wording == "":
		return Date{}, fmt.Errorf("parse date %q: empty", s)
	}
	d.Wording = wording
	return d, nil
}

// cutWording separates the record's wording, in parentheses at the end of the value,
// from the date in front of it. The words run from the first opening parenthesis to the
// closing one the value ends with, so they may hold parentheses of their own.
func cutWording(s string) (stated, wording string, err error) {
	open := strings.IndexByte(s, '(')
	if open < 0 {
		return s, "", nil
	}
	rest := strings.TrimSpace(s[open:])
	if !strings.HasSuffix(rest, ")") {
		return "", "", errors.New("the wording does not end the date")
	}
	wording = strings.Join(strings.Fields(rest[1:len(rest)-1]), " ")
	if wording == "" {
		return "", "", errors.New("empty wording")
	}
	return s[:open], wording, nil
}

func parseStated(fields []string) (Date, error) {
	if periodic(fields[0]) {
		return parsePeriod(fields)
	}
	return parseQualified(fields)
}

func periodic(field string) bool {
	return strings.EqualFold(field, periodStart) || strings.EqualFold(field, periodEnd)
}

// parsePeriod reads FROM x, TO y or FROM x TO y, each end a date with its own qualifier.
func parsePeriod(fields []string) (Date, error) {
	var d Date
	end := fields
	if strings.EqualFold(fields[0], periodStart) {
		start := fields[1:]
		end = nil
		for i, f := range start {
			if strings.EqualFold(f, periodEnd) {
				start, end = start[:i], start[i:]
				break
			}
		}
		from, err := parseQualified(start)
		if err != nil {
			return Date{}, err
		}
		d.From = &from
	}
	if end != nil {
		to, err := parseQualified(end[1:])
		if err != nil {
			return Date{}, err
		}
		d.To = &to
	}

	if d.From != nil && d.To != nil {
		if lo, _, ok := d.From.span(); ok {
			if _, hi, ok := d.To.span(); ok && hi < lo {
				return Date{}, errors.New("period ends before it starts")
			}
		}
	}
	return d, nil
}

// parseQualified reads one date that is not a period: how it was reached and a
// qualifier, each if there is one, and the date or range they qualify.
func parseQualified(fields []string) (Date, error) {
	reached := Stated
	if len(fields) > 0 {
		for i, k := range reachedKeyword {
			if k != "" && strings.EqualFold(fields[0], k) {
				reached, fields = Reached(i), fields[1:]
				break
			}
		}
	}
	if len(fields) == 0 {
		return Date{}, errors.New("missing date")
	}
	if periodic(fields[0]) {
		return Date{}, errors.New("each end of a period says how it was reached, and the period does not")
	}
	q := Exact
	for i, k := range qualifierKeyword {
		if k != "" && strings.EqualFold(fields[0], k) {
			q, fields = Qualifier(i), fields[1:]
			break
		}
	}
	var d Date
	var err error
	if q == Between {
		d, err = parseBetween(fields)
	} else {
		d, err = parseValue(fields)
		d.Qualifier = q
	}
	d.Reached = reached
	return d, err
}

func parseBetween(fields []string) (Date, error) {
	sep := -1
	for i, f := range fields {
		if strings.EqualFold(f, "AND") {
			sep = i
			break
		}
	}
	if sep < 0 {
		return Date{}, errors.New("BET without AND")
	}
	start, err := parseValue(fields[:sep])
	if err != nil {
		return Date{}, err
	}
	end, err := parseValue(fields[sep+1:])
	if err != nil {
		return Date{}, err
	}
	start.Qualifier, start.End = Between, &end

	if slo, _, ok := start.bounds(); ok {
		if _, ehi, ok := end.bounds(); ok && ehi < slo {
			return Date{}, errors.New("range ends before it starts")
		}
	}
	return start, nil
}

func parseValue(fields []string) (Date, error) {
	var d Date
	if len(fields) == 0 {
		return d, errors.New("missing date")
	}

	if c, ok := calendarNamed(fields[0]); ok {
		d.Calendar, fields = c, fields[1:]
	}
	var newYear []string
	for i, f := range fields {
		if strings.EqualFold(f, newYearKeyword) {
			fields, newYear = fields[:i], fields[i+1:]
			break
		}
	}
	if last := len(fields) - 1; last >= 0 && strings.Contains(fields[last], ":") {
		t, err := parseTime(fields[last])
		if err != nil {
			return d, err
		}
		d.Time, fields = &t, fields[:last]
	}
	if last := len(fields) - 1; last >= 0 && strings.EqualFold(fields[last], epochKeyword) {
		if !d.reckoning().epoch {
			return d, fmt.Errorf("%s counts from its own beginning and has no %s",
				d.reckoning().keyword, epochKeyword)
		}
		d.BCE, fields = true, fields[:last]
	}

	months := d.reckoning().months

	// A bare number is a year, so a day is only recognisable beside its month.
	switch len(fields) {
	case 0:
		if d.Time != nil {
			return d, errors.New("time with no date")
		}
		return d, errors.New("missing date")
	case 1:
		if m, ok := monthNumber(months, fields[0]); ok {
			d.Month = &m
			break
		}
		if err := d.setYear(fields[0]); err != nil {
			return d, err
		}
	case 2:
		if fields[1] == unknownMonth {
			day, err := parseDay(fields[0])
			if err != nil {
				return d, err
			}
			d.Day = &day
			break
		}
		if m, ok := monthNumber(months, fields[1]); ok {
			day, err := parseDay(fields[0])
			if err != nil {
				return d, err
			}
			d.Day, d.Month = &day, &m
			break
		}
		m, ok := monthNumber(months, fields[0])
		if !ok {
			return d, fmt.Errorf("no month in %q", strings.Join(fields, " "))
		}
		d.Month = &m
		if err := d.setYear(fields[1]); err != nil {
			return d, err
		}
	case 3:
		day, err := parseDay(fields[0])
		if err != nil {
			return d, err
		}
		d.Day = &day
		if fields[1] != unknownMonth {
			m, ok := monthNumber(months, fields[1])
			if !ok {
				return d, fmt.Errorf("unknown month %q", fields[1])
			}
			d.Month = &m
		}
		if err := d.setYear(fields[2]); err != nil {
			return d, err
		}
	default:
		return d, errors.New("too many parts")
	}

	if d.Month != nil {
		if settle := d.reckoning().settle; settle != nil {
			month := settle(*d.Month, d.referenceYear())
			d.Month = &month
		}
		length := d.reckoning().daysIn(*d.Month, d.referenceYear())
		if d.Day != nil && *d.Day > length {
			return d, fmt.Errorf("%s has no day %d", months[*d.Month-1], *d.Day)
		}
	} else if d.Day != nil && !d.someMonthHas(*d.Day) {
		return d, fmt.Errorf("no month has a day %d", *d.Day)
	}
	if newYear != nil {
		if err := d.setNewYear(newYear); err != nil {
			return d, err
		}
	}
	return d, nil
}

// someMonthHas reports whether a day whose month is not known is a day of any month
// the year has.
func (d Date) someMonthHas(day int) bool {
	r := d.reckoning()
	for m := range r.months {
		if r.daysIn(m+1, d.referenceYear()) >= day {
			return true
		}
	}
	return false
}

// setNewYear reads the day and month after NY. Only a calendar counting the common era
// names a year for the 1 January it holds, a dual year already says where its own year
// turned, and a year counted from 1 January is every other year and says nothing.
func (d *Date) setNewYear(fields []string) error {
	r := d.reckoning()
	switch {
	case d.Year == nil:
		return fmt.Errorf("%s says how a year was counted, and the date has none", newYearKeyword)
	case !r.epoch:
		return fmt.Errorf("%s does not count a year from another day", r.keyword)
	case d.Dual:
		return errors.New("a dual year already says which day its year began")
	case len(fields) != 2:
		return fmt.Errorf("%s takes a day and a month", newYearKeyword)
	}
	day, err := parseDay(fields[0])
	if err != nil {
		return err
	}
	month, ok := monthNumber(r.months, fields[1])
	switch {
	case !ok:
		return fmt.Errorf("unknown month %q", fields[1])
	case day > r.daysIn(month, r.open):
		return fmt.Errorf("%s has no day %d", r.months[month-1], day)
	case day == 1 && month == 1:
		return fmt.Errorf("a year counted from 1 %s needs no %s", r.months[0], newYearKeyword)
	}
	d.NewYear = &NewYear{Month: month, Day: day}
	return nil
}

func (d *Date) setYear(s string) error {
	r := d.reckoning()
	digits, dual := s, false
	if before, after, found := strings.Cut(s, "/"); found {
		if !r.epoch {
			return fmt.Errorf("%s does not write a year as two", r.keyword)
		}
		if d.BCE {
			return errors.New("a year before the common era is not dual-dated")
		}
		digits, dual = before, true
		y, err := strconv.Atoi(before)
		if err != nil {
			return fmt.Errorf("bad year %q", s)
		}
		if len(after) != 2 || after != fmt.Sprintf("%02d", (y+1)%100) {
			return fmt.Errorf("dual year %q does not continue %d", s, y)
		}
	}
	y, err := strconv.Atoi(digits)
	if err != nil {
		return fmt.Errorf("bad year %q", s)
	}
	if y < r.first || y > r.last {
		return fmt.Errorf("year %d is outside the %d to %d this calendar reckons over",
			y, r.first, r.last)
	}
	d.Year, d.Dual = &y, dual
	return nil
}

func parseDay(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > 31 {
		return 0, fmt.Errorf("bad day %q", s)
	}
	return n, nil
}

func parseTime(s string) (TimeOfDay, error) {
	var t TimeOfDay
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return t, fmt.Errorf("bad time %q", s)
	}
	limit := []int{23, 59, 59}
	out := make([]int, len(parts))
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || n > limit[i] {
			return t, fmt.Errorf("bad time %q", s)
		}
		out[i] = n
	}
	t.Hour, t.Minute = out[0], out[1]
	if len(out) == 3 {
		t.Second = &out[2]
	}
	return t, nil
}

func monthNumber(months []string, s string) (int, bool) {
	for i, m := range months {
		if strings.EqualFold(s, m) {
			return i + 1, true
		}
	}
	return 0, false
}

// referenceYear is the year a month is measured against. A floating date could fall in
// any year, so it is given the benefit of the doubt and 29 FEB stays valid.
func (d Date) referenceYear() int {
	if d.Year == nil {
		return d.reckoning().open
	}
	return d.reckoningYear()
}

// reckoningYear is the year the date falls in, counted the way the arithmetic needs it
// rather than the way the record writes it. A dual-dated year names the older one
// first and the date falls in the year after. A year before the common era is counted
// back through a zero the written form does not have — 1 BCE is followed by 1 — so
// 1 BCE is year 0, and this is the one place that conversion happens.
func (d Date) reckoningYear() int {
	year := *d.Year
	if d.Dual {
		year++
	}
	if d.BCE {
		year = 1 - year
	}
	return year
}

// Bounds for a date that is open at one end. BEF and AFT reach back or forward without
// limit, and giving that end a number keeps the arithmetic below free of special cases.
const (
	beginningOfDays = -1 << 40
	endOfDays       = 1 << 40
)

const secondsInDay = 24 * 60 * 60

// daysInYear is the mean length of a Gregorian year, which is the right conversion when
// the question is how many years apart two dates are rather than which year each is in.
const daysInYear = 365.2425

// moment is a second on the one line every date reaches, counted from the start of
// 1970-01-01. A date states its day in whichever calendar it was written in, and the
// day number is what makes two dates in two calendars comparable at all.
func moment(day, hour, minute, second int) int64 {
	return int64(day)*secondsInDay + int64(hour)*3600 + int64(minute)*60 + int64(second)
}

// bounds is the earliest and latest moment the date's own components admit, ignoring
// its qualifier. ok is false for a floating date, which has no place on the line.
func (d Date) bounds() (lo, hi int64, ok bool) {
	stretches := d.days()
	if len(stretches) == 0 {
		return 0, 0, false
	}
	r, first, last := d.reckoning(), stretches[0], stretches[len(stretches)-1]
	hourLo, hourHi := 0, 23
	minuteLo, minuteHi := 0, 59
	secondLo, secondHi := 0, 59
	if t := d.Time; t != nil {
		hourLo, hourHi = t.Hour, t.Hour
		minuteLo, minuteHi = t.Minute, t.Minute
		if t.Second != nil {
			secondLo, secondHi = *t.Second, *t.Second
		}
	}
	return moment(r.dayNumber(first.year, first.month, first.from), hourLo, minuteLo, secondLo),
		moment(r.dayNumber(last.year, last.month, last.to), hourHi, minuteHi, secondHi),
		true
}

// stretch is days from and to of one month, in the year it is reckoned in.
type stretch struct{ year, month, from, to int }

// days is every stretch of a month the date's own components admit, earliest first,
// and nothing for a floating date. A year counted from a new-year day begins in the
// reckoning year before the one it is named for, so its days from the new year on are
// taken there.
func (d Date) days() []stretch {
	if d.Year == nil {
		return nil
	}
	r, named := d.reckoning(), d.reckoningYear()
	years := []int{named}
	if d.NewYear != nil {
		years = []int{named - 1, named}
	}
	var out []stretch
	for _, year := range years {
		for month := 1; month <= len(r.months); month++ {
			if d.Month != nil && month != *d.Month {
				continue
			}
			from, to := 1, r.daysIn(month, year)
			if ny := d.NewYear; ny != nil {
				before := year == named
				switch {
				case before && month > ny.Month, !before && month < ny.Month:
					continue
				case before && month == ny.Month:
					to = min(to, ny.Day-1)
				case month == ny.Month:
					from = ny.Day
				}
			}
			if d.Day != nil {
				from, to = max(from, *d.Day), min(to, *d.Day)
			}
			if from <= to {
				out = append(out, stretch{year, month, from, to})
			}
		}
	}
	return out
}

// span is the stretch of time the date denotes once its qualifier is applied. ABT, EST
// and CAL qualify confidence rather than extent, so they span what they state. A period
// spans the whole of its extent, an end it leaves open running without limit.
func (d Date) span() (lo, hi int64, ok bool) {
	if d.Period() {
		lo, hi = moment(beginningOfDays, 0, 0, 0), moment(endOfDays, 0, 0, 0)
		if d.From != nil {
			if lo, _, ok = d.From.span(); !ok {
				return 0, 0, false
			}
		}
		if d.To != nil {
			if _, hi, ok = d.To.span(); !ok {
				return 0, 0, false
			}
		}
		return lo, hi, true
	}

	lo, hi, ok = d.bounds()
	if !ok {
		return 0, 0, false
	}
	switch d.Qualifier {
	case Before:
		return moment(beginningOfDays, 0, 0, 0), lo - 1, true
	case After:
		return hi + 1, moment(endOfDays, 0, 0, 0), true
	case Between:
		if d.End == nil {
			return 0, 0, false
		}
		_, endHi, endOK := d.End.bounds()
		if !endOK {
			return 0, 0, false
		}
		return lo, endHi, true
	}
	return lo, hi, true
}

// dayBounds is the earliest and latest day the date could fall on, which is its span
// read at the resolution the age rules work in. ok is false for a floating date.
func (d Date) dayBounds() (lo, hi int, ok bool) {
	loSecond, hiSecond, ok := d.span()
	if !ok {
		return 0, 0, false
	}
	return int(floorDiv(loSecond, secondsInDay)), int(floorDiv(hiSecond, secondsInDay)), true
}

// Compare orders two dates by the stretch of time each denotes, returning the usual
// -1, 0 or 1. ok is false when the order is not determined — when either date is
// floating, or when their spans overlap without coinciding. 1834 and MAR 1834 are such
// a pair: neither is before the other, and saying otherwise would have check report
// contradictions that the records do not contain.
//
// Two dates in different calendars order as readily as two in the same one, because
// both reach the same coordinate before they are compared.
func Compare(a, b Date) (int, bool) {
	aLo, aHi, aOK := a.span()
	bLo, bHi, bOK := b.span()
	if !aOK || !bOK {
		return 0, false
	}
	switch {
	case aLo == bLo && aHi == bHi:
		return 0, true
	case aHi < bLo:
		return -1, true
	case bHi < aLo:
		return 1, true
	}
	return 0, false
}

// Within says whether one date falls inside the stretch of time another denotes:
// certainly where the whole of the one lies inside the other, and possibly where the two
// overlap or either has no place on the line, since a floating date may be anywhere.
func Within(d, span Date) (certainly, possibly bool) {
	dLo, dHi, dOK := d.span()
	sLo, sHi, sOK := span.span()
	if !dOK || !sOK {
		return false, true
	}
	return sLo <= dLo && dHi <= sHi, dLo <= sHi && sLo <= dHi
}

// YearsBetween is the range of elapsed years that may separate two dates, negative when
// b falls before a. ok is false if either is floating.
//
// Widening to a range rather than picking a number is what lets a caller speak only when
// the records leave no room for doubt: a parent was too young only if they were too
// young on every reading of the dates.
func YearsBetween(a, b Date) (least, most float64, ok bool) {
	aLo, aHi, aOK := a.dayBounds()
	bLo, bHi, bOK := b.dayBounds()
	if !aOK || !bOK {
		return 0, 0, false
	}
	return float64(bLo-aHi) / daysInYear, float64(bHi-aLo) / daysInYear, true
}
