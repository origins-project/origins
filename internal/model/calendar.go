package model

import "strings"

// Calendar is the reckoning a date was written under. A reckoning names its own months,
// decides how long each of them is, and places a complete date on the day number every
// date is ordered by; nothing else about a date depends on which one it is stated in.
//
// A date is never converted out of the calendar it was recorded in. The day number is
// derived when two dates have to be compared and is never written back to a corpus.
type Calendar int

// The calendars a date can be written in, Gregorian being the default.
const (
	Gregorian Calendar = iota
	Julian
	Hebrew
	FrenchRepublican
	Persian
	Islamic
	Swedish
)

// reckoning is everything the rest of the package needs to know about a calendar. The
// calendars are a table rather than an interface because they differ only in data — a
// month list, a month length, a conversion and a handful of limits — and another would
// be a row in it.
type reckoning struct {
	// keyword names the calendar in a date value. It is written for every calendar but
	// Gregorian, which is the default and says nothing.
	keyword string
	months  []string
	// epoch reports whether years may be counted backwards from the start of the common
	// era. Only the two calendars that share that era admit one.
	epoch bool
	// first and last are the years the reckoning is defined over. A calendar that was
	// abandoned stops where it was abandoned rather than continuing under a rule nobody
	// ever agreed on.
	first, last int
	// open is the year a date with no year has its day measured against, chosen so that
	// every day the calendar ever has stands: 29 FEB and 6 COMP are both real days, and
	// a record that gives no year cannot rule either of them out.
	open int
	// settle maps a month onto the one the year it falls in actually has, where a
	// calendar names one month two ways. Only the Hebrew calendar does, and nil here
	// means a month is read as written.
	settle    func(month, year int) int
	daysIn    func(month, year int) int
	dayNumber func(year, month, day int) int
}

var gregorianMonths = []string{
	"JAN", "FEB", "MAR", "APR", "MAY", "JUN",
	"JUL", "AUG", "SEP", "OCT", "NOV", "DEC",
}

var reckonings = [...]reckoning{
	Gregorian: {
		keyword:   "GREGORIAN",
		months:    gregorianMonths,
		epoch:     true,
		first:     1,
		last:      9999,
		open:      2000,
		daysIn:    gregorianDaysIn,
		dayNumber: daysFromCivil,
	},
	Julian: {
		keyword:   "JULIAN",
		months:    gregorianMonths,
		epoch:     true,
		first:     1,
		last:      9999,
		open:      2000,
		daysIn:    julianDaysIn,
		dayNumber: julianDayNumber,
	},
	Hebrew: {
		keyword: "HEBREW",
		months: []string{
			"TSH", "CSH", "KSL", "TVT", "SHV", "ADR", "ADS",
			"NSN", "IYR", "SVN", "TMZ", "AAV", "ELL",
		},
		first:     1,
		last:      9999,
		open:      5787, // leap and complete, so every month is at its longest
		settle:    hebrewSettle,
		daysIn:    hebrewDaysIn,
		dayNumber: hebrewDayNumber,
	},
	FrenchRepublican: {
		keyword: "FRENCH_R",
		months: []string{
			"VEND", "BRUM", "FRIM", "NIVO", "PLUV", "VENT", "GERM",
			"FLOR", "PRAI", "MESS", "THER", "FRUC", "COMP",
		},
		first:     1,
		last:      14,
		open:      3, // sextile, so the sixth complementary day stands
		daysIn:    republicanDaysIn,
		dayNumber: republicanDayNumber,
	},
	Persian: {
		keyword: "PERSIAN",
		months: []string{
			"FARVA", "ORDIB", "KHORD", "TIR", "MORDA", "SHAHR",
			"MEHR", "ABAN", "AZAR", "DEY", "BAHMA", "ESFAN",
		},
		first:     1,
		last:      persianBreaks[len(persianBreaks)-1] - 1,
		open:      1403, // leap, so 30 Esfand stands
		daysIn:    persianDaysIn,
		dayNumber: persianDayNumber,
	},
	Islamic: {
		keyword: "ISLAMIC",
		months: []string{
			"MUHAR", "SAFAR", "RABIA", "RABIT", "JUMAA", "JUMAT",
			"RAJAB", "SHAAB", "RAMAD", "SHAWW", "DHUAQ", "DHUAH",
		},
		first:     1,
		last:      9999,
		open:      1445, // leap, so 30 Dhu al-Hijjah stands
		daysIn:    islamicDaysIn,
		dayNumber: islamicDayNumber,
	},
	Swedish: {
		keyword:   "SWEDISH",
		months:    gregorianMonths,
		first:     1700,
		last:      1712,
		open:      1712, // the year February had thirty days
		daysIn:    swedishDaysIn,
		dayNumber: swedishDayNumber,
	},
}

// Months is how many months a year of the calendar has, numbered from 1.
func (c Calendar) Months() int { return len(Date{Calendar: c}.reckoning().months) }

func (d Date) reckoning() reckoning {
	if int(d.Calendar) < 0 || int(d.Calendar) >= len(reckonings) {
		return reckonings[Gregorian]
	}
	return reckonings[d.Calendar]
}

// calendarNamed reads a leading calendar word. Gregorian has one so that a file stating
// the default is read; String never writes it back.
func calendarNamed(word string) (Calendar, bool) {
	for i, r := range reckonings {
		if strings.EqualFold(word, r.keyword) {
			return Calendar(i), true
		}
	}
	return Gregorian, false
}

// floorDiv divides rounding towards negative infinity, which is what calendar
// arithmetic needs and what Go division does not do. The year before 1 is 0 and the one
// before that is -1, so truncating towards zero would give two different years the same
// quotient and put the leap day in the wrong one.
func floorDiv[T int | int64](a, b T) T {
	q := a / b
	if a%b != 0 && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// daysFromCivil is the number of days from 1970-01-01 to a proleptic Gregorian date,
// negative before it. Doing the arithmetic in days rather than in year numbers is what
// keeps a death in 1860 and a birth in 1861 from reading as a year apart when they may
// be one day apart.
func daysFromCivil(year, month, day int) int {
	if month <= 2 {
		year--
	}
	era := year
	if era < 0 {
		era -= 399
	}
	era /= 400
	yearOfEra := year - era*400
	shifted := month + 9
	if month > 2 {
		shifted = month - 3
	}
	dayOfYear := (153*shifted+2)/5 + day - 1
	dayOfEra := yearOfEra*365 + yearOfEra/4 - yearOfEra/100 + dayOfYear
	return era*146097 + dayOfEra - 719468
}

// civilFromDays is the proleptic Gregorian date a number of days from 1970-01-01 falls
// on, which undoes daysFromCivil.
func civilFromDays(days int) (year, month, day int) {
	days += 719468
	era := floorDiv(days, 146097)
	dayOfEra := days - era*146097
	yearOfEra := (dayOfEra - dayOfEra/1460 + dayOfEra/36524 - dayOfEra/146096) / 365
	dayOfYear := dayOfEra - (365*yearOfEra + yearOfEra/4 - yearOfEra/100)
	shifted := (5*dayOfYear + 2) / 153
	day = dayOfYear - (153*shifted+2)/5 + 1
	month = shifted + 3
	if shifted >= 10 {
		month = shifted - 9
	}
	year = yearOfEra + era*400
	if month <= 2 {
		year++
	}
	return year, month, day
}

func gregorianDaysIn(month, year int) int {
	if month == 2 && year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		return 29
	}
	return monthLength(month)
}

// julianDaysIn differs from the Gregorian in the leap rule alone, which is the whole of
// the difference between the two calendars: every fourth year, without the century
// exceptions that were the reason for the reform.
func julianDaysIn(month, year int) int {
	if month == 2 && year%4 == 0 {
		return 29
	}
	return monthLength(month)
}

func monthLength(month int) int {
	switch month {
	case 2:
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

// julianDayNumber places a proleptic Julian date on the same line as a Gregorian one.
// The arithmetic runs over a year that starts in March, so the leap day is the last day
// of the last month and no month after it has to be stepped over.
func julianDayNumber(year, month, day int) int {
	if month <= 2 {
		year, month = year-1, month+12
	}
	julianDay := floorDiv(1461*(year+4716), 4) + (153*(month-3)+2)/5 + day - 1402
	return julianDay - 2440588 // the Julian day of 1970-01-01
}

// hebrewLeap reports whether a year carries a second Adar. Seven of every nineteen do,
// at the places the cycle has put them since the calendar was fixed by arithmetic.
func hebrewLeap(year int) bool { return (7*year+1)%19 < 7 }

// hebrewNewYear is the day number of 1 Tishrei, which is where the whole calendar is
// decided. The mean conjunction that opens the year is computed in parts of an hour and
// then postponed by up to two days: once if it falls at or after noon, once more for
// either of two cases that would otherwise leave a neighbouring year an impossible
// length, and once again to keep the day off Sunday, Wednesday and Friday. Everything
// else — the length of the year, the length of Cheshvan and of Kislev — follows from
// the distance between two of these.
func hebrewNewYear(year int) int {
	const epoch = -2092591 // the day number of 1 Tishrei of year 1, less one elapsed day

	months := 235*((year-1)/19) + 12*((year-1)%19) + (7*((year-1)%19)+1)/19
	parts := 204 + 793*(months%1080)
	hours := 5 + 12*months + 793*(months/1080) + parts/1080
	day := 1 + 29*months + hours/24
	remainder := (hours%24)*1080 + parts%1080

	switch {
	case remainder >= 19440,
		day%7 == 2 && remainder >= 9924 && !hebrewLeap(year),
		day%7 == 1 && remainder >= 16789 && hebrewLeap(year-1):
		day++
	}
	if weekday := day % 7; weekday == 0 || weekday == 3 || weekday == 5 {
		day++
	}
	return day + epoch
}

func hebrewYearLength(year int) int { return hebrewNewYear(year+1) - hebrewNewYear(year) }

// hebrewDaysIn gives the length of a month. Only three of the thirteen vary: Cheshvan
// and Kislev absorb the day a year has to gain or lose, and Adar I has no length at all
// in a common year, which has no Adar I — settle is what keeps a date out of it.
func hebrewDaysIn(month, year int) int {
	switch month {
	case 2: // Cheshvan, long only in a complete year
		if hebrewYearLength(year)%10 == 5 {
			return 30
		}
		return 29
	case 3: // Kislev, short only in a deficient year
		if hebrewYearLength(year)%10 == 3 {
			return 29
		}
		return 30
	case 6: // Adar I, which only a leap year has
		if hebrewLeap(year) {
			return 30
		}
		return 0
	case 7: // Adar II in a leap year, and the single Adar of a common one
		return 29
	case 4, 9, 11, 13: // Tevet, Iyar, Tammuz, Elul
		return 29
	}
	return 30
}

// hebrewSettle reads Adar I in a year that has none as the Adar it must mean. ADR is
// the tag for the first of two, ADS for the second and for the only one of a common
// year — but a writer recording a common year writes Adar, and the specification asks
// a reader that knows which years have two to read it as the one that exists.
func hebrewSettle(month, year int) int {
	if month == 6 && !hebrewLeap(year) {
		return 7
	}
	return month
}

func hebrewDayNumber(year, month, day int) int {
	total := hebrewNewYear(year)
	for m := 1; m < month; m++ {
		total += hebrewDaysIn(m, year)
	}
	return total + day - 1
}

// republicanSextile reports whether a year was given a sixth complementary day. The
// calendar was abolished before the rule for placing them was settled — the bill that
// would have fixed it was never voted — so these are the three that were observed,
// rather than a formula continued past the evidence for it.
func republicanSextile(year int) bool { return year == 3 || year == 7 || year == 11 }

// republicanDaysIn: twelve months of thirty days, and then the five or six days left
// over at the end of the year, which the calendar counts as a short thirteenth.
func republicanDaysIn(month, year int) int {
	if month < 13 {
		return 30
	}
	if republicanSextile(year) {
		return 6
	}
	return 5
}

func republicanDayNumber(year, month, day int) int {
	start := daysFromCivil(1792, 9, 22) // 1 Vendémiaire of year 1, the day the Republic began
	for y := 1; y < year; y++ {
		start += 365
		if republicanSextile(y) {
			start++
		}
	}
	return start + (month-1)*30 + day - 1
}

// persianBreaks are the years at which the pattern of leap years in the Solar Hijri
// calendar shifts. The calendar Iran keeps begins each year on the day of the March
// equinox as observed at Tehran, which no fixed cycle follows for long; within each
// stretch between two of these years a 33-year cycle reproduces it, and the breaks
// are where one cycle hands over to the next. This is Kazimierz Borkowski's arithmetic,
// and the calendar is defined over the years it covers.
var persianBreaks = [...]int{
	-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210,
	1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178,
}

// persianYear is the day of March, in the Gregorian year the Persian one begins in, on
// which 1 Farvardin falls, and whether the year has a thirtieth of Esfand. It counts the
// leap years since the calendar began, and the Gregorian leap years over the same
// stretch, and the difference between the two moves the new year along March.
func persianYear(year int) (march int, leap bool) {
	leaps, previous, jump := -14, persianBreaks[0], 0
	for _, next := range persianBreaks[1:] {
		jump = next - previous
		if year < next {
			break
		}
		leaps += jump/33*8 + jump%33/4
		previous = next
	}
	n := year - previous
	leaps += n/33*8 + (n%33+3)/4
	if jump%33 == 4 && jump-n == 4 {
		leaps++
	}
	gregorian := year + 621
	gregorianLeaps := gregorian/4 - (gregorian/100+1)*3/4 - 150
	march = 20 + leaps - gregorianLeaps

	if jump-n < 6 {
		n = n - jump + (jump+4)/33*33
	}
	return march, ((n+1)%33-1)%4 == 0
}

// persianDaysIn: six months of thirty-one days, five of thirty, and Esfand, which takes
// the day a leap year gains.
func persianDaysIn(month, year int) int {
	switch {
	case month <= 6:
		return 31
	case month <= 11:
		return 30
	}
	if _, leap := persianYear(year); leap {
		return 30
	}
	return 29
}

func persianDayNumber(year, month, day int) int {
	march, _ := persianYear(year)
	start := daysFromCivil(year+621, 3, march)
	if month <= 7 {
		return start + (month-1)*31 + day - 1
	}
	return start + 6*31 + (month-7)*30 + day - 1
}

// islamicLeap reports whether a year of the tabular Islamic calendar has a thirtieth
// of Dhu al-Hijjah. Eleven years in every thirty do — the 2nd, 5th, 7th, 10th, 13th,
// 16th, 18th, 21st, 24th, 26th and 29th — which keeps twelve months of a mean lunation
// in step with the moon to within a day over centuries.
func islamicLeap(year int) bool { return (14+11*year)%30 < 11 }

// islamicDaysIn alternates thirty and twenty-nine days from Muharram, with the last
// month long in a leap year. A record's month was begun by a sighting of the new moon,
// which the table can miss by a day; since the day number only puts dates in order,
// that is the whole of the error.
func islamicDaysIn(month, year int) int {
	if month == 12 && islamicLeap(year) {
		return 30
	}
	return 30 - (month+1)%2
}

// islamicDayNumber counts from 1 Muharram of year 1 where the civil reckoning places it,
// on Friday 16 July 622 in the Julian calendar; the astronomical reckoning puts it a day
// earlier.
func islamicDayNumber(year, month, day int) int {
	epoch := julianDayNumber(622, 7, 16)
	leapsBefore := (3 + 11*year) / 30
	return epoch + (year-1)*354 + leapsBefore + (month-1)*29 + month/2 + day - 1
}

// swedishDaysIn follows the Julian calendar but for the two Februaries Sweden changed:
// 1700 left out its leap day, the first of the eleven leap days to 1740 by whose leaving
// out Sweden meant to reach the Gregorian calendar, and 1712 abandoned the move by
// adding back the one day that had been dropped, as a 30 February.
func swedishDaysIn(month, year int) int {
	switch {
	case month != 2:
		return monthLength(month)
	case year == 1700:
		return 28
	case year == 1712:
		return 30
	}
	return julianDaysIn(month, year)
}

// swedishDayNumber: from 1 March 1700 to 30 February 1712, a Swedish date is one day
// ahead of the Julian one with the same name, and before and after that it is the
// Julian date.
func swedishDayNumber(year, month, day int) int {
	julian := julianDayNumber(year, month, day)
	if (year > 1700 || month > 2) && (year < 1712 || month <= 2) {
		return julian - 1
	}
	return julian
}
