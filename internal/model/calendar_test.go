package model

import (
	"slices"
	"testing"
)

// gregorian is the day number of a proleptic Gregorian date, which is the coordinate
// every other calendar in the table is checked against.
func gregorian(year, month, day int) int { return daysFromCivil(year, month, day) }

func TestJulianAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
		what             string
	}{
		// The day after Thursday 4 October 1582 in the old reckoning was Friday
		// 15 October in the new one, so the two labels below name the same Friday.
		{1582, 10, 5, 1582, 10, 15, "the day the reform skipped to"},
		{1582, 10, 4, 1582, 10, 14, "the last day the old reckoning was kept in Rome"},
		{1, 1, 1, 0, 12, 30, "the proleptic start, two days ahead of the Gregorian one"},
		{1700, 3, 1, 1700, 3, 12, "after a century the Gregorian calendar did not leap"},
		{1900, 3, 1, 1900, 3, 14, "and after another two"},
		{1799, 2, 14, 1799, 2, 25, "an ordinary parish date"},
		{-3760, 10, 7, -3760, 9, 7, "1 Tishrei 1, which the specification dates in both"},
	} {
		got := julianDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("%d-%02d-%02d Julian is day %d, want %d (%s)",
				c.year, c.month, c.day, got, want, c.what)
		}
	}
}

func TestHebrewAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
	}{
		{5784, 1, 1, 2023, 9, 16}, // 1 Tishrei 5784
		{5785, 1, 1, 2024, 10, 3}, // 1 Tishrei 5785
		{5784, 8, 1, 2024, 4, 9},  // 1 Nisan 5784
		{5760, 1, 1, 1999, 9, 11}, // 1 Tishrei 5760
		// the first day of the calendar, which the specification places against both
		// of the other two
		{1, 1, 1, -3760, 9, 7},
	} {
		got := hebrewDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("%d-%02d-%02d Hebrew is day %d, want %d (%d-%02d-%02d)",
				c.year, c.month, c.day, got, want, c.gy, c.gm, c.gd)
		}
	}
}

// TestHebrewYearsAreOneOfSixLengths is the check that the postponements are right: get
// one of them wrong and a year comes out a length the calendar does not have.
func TestHebrewYearsAreOneOfSixLengths(t *testing.T) {
	for year := 1; year <= 6000; year++ {
		length := hebrewYearLength(year)
		switch length {
		case 353, 354, 355, 383, 384, 385:
		default:
			t.Fatalf("year %d is %d days long", year, length)
		}
		if leap := hebrewLeap(year); leap != (length > 360) {
			t.Fatalf("year %d is %d days long and leap reports %v", year, length, leap)
		}
	}
}

// TestHebrewMonthsFillTheirYear holds the month lengths to the year lengths. Each month
// must start the day after the one before ends, and the last must end the day before
// the next new year.
func TestHebrewMonthsFillTheirYear(t *testing.T) {
	for year := 5500; year <= 5900; year++ {
		day := hebrewNewYear(year)
		for month := 1; month <= 13; month++ {
			length := hebrewDaysIn(month, year)
			if length == 0 {
				if month != 6 || hebrewLeap(year) {
					t.Fatalf("year %d month %d has no days", year, month)
				}
				continue
			}
			if got := hebrewDayNumber(year, month, 1); got != day {
				t.Fatalf("year %d month %d starts on day %d, want %d", year, month, got, day)
			}
			day += length
		}
		if day != hebrewNewYear(year+1) {
			t.Fatalf("year %d ends on day %d and the next starts on %d",
				year, day, hebrewNewYear(year+1))
		}
	}
}

func TestRepublicanAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
		what             string
	}{
		{1, 1, 1, 1792, 9, 22, "the day the Republic was proclaimed"},
		{1, 5, 2, 1793, 1, 21, "the day Louis XVI was guillotined"},
		{2, 1, 25, 1793, 10, 16, "and the day Marie Antoinette was"},
		{2, 11, 9, 1794, 7, 27, "9 Thermidor"},
		{3, 1, 1, 1794, 9, 22, "the third year still begins on the equinox"},
		{4, 1, 1, 1795, 9, 23, "and the fourth a day later, the third having been sextile"},
		{8, 2, 18, 1799, 11, 9, "18 Brumaire"},
		{14, 4, 10, 1805, 12, 31, "the last day the calendar was in use"},
	} {
		got := republicanDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("year %d month %d day %d is day %d, want %d (%s)",
				c.year, c.month, c.day, got, want, c.what)
		}
	}
}

func TestPersianAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
		what             string
	}{
		{1, 1, 1, 622, 3, 22, "the first day of the calendar"},
		{1300, 1, 1, 1921, 3, 21, "Nowruz 1300"},
		{1304, 1, 1, 1925, 3, 21, "the year Iran adopted the calendar"},
		{1354, 1, 1, 1975, 3, 21, "Nowruz 1354"},
		{1399, 12, 30, 2021, 3, 20, "the leap day of 1399"},
		{1403, 1, 1, 2024, 3, 20, "Nowruz 1403, on the twentieth"},
		{1403, 10, 1, 2024, 12, 21, "1 Dey, the morning after Yalda"},
		{1403, 12, 30, 2025, 3, 20, "the leap day of 1403"},
		{1404, 1, 1, 2025, 3, 21, "and Nowruz 1404, the day after it"},
	} {
		got := persianDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("%d-%02d-%02d Persian is day %d, want %d (%s)",
				c.year, c.month, c.day, got, want, c.what)
		}
	}
}

func TestPersianLeapYears(t *testing.T) {
	var got []int
	for year := 1370; year <= 1410; year++ {
		if _, leap := persianYear(year); leap {
			got = append(got, year)
		}
	}
	want := []int{1370, 1375, 1379, 1383, 1387, 1391, 1395, 1399, 1403, 1408}
	if !slices.Equal(got, want) {
		t.Errorf("leap years 1370 to 1410 are %v, want %v", got, want)
	}
}

func TestIslamicAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
		what             string
	}{
		{1, 1, 1, 622, 7, 19, "the first day of the calendar, 16 July in the Julian one"},
		{1250, 9, 1, 1835, 1, 1, "1 Ramadan 1250"},
		{1319, 2, 3, 1901, 5, 22, "3 Safar 1319"},
		{1400, 1, 1, 1979, 11, 21, "the first day of the fifteenth century"},
		{1421, 1, 1, 2000, 4, 6, "1 Muharram 1421"},
		{1445, 9, 1, 2024, 3, 11, "1 Ramadan 1445"},
		{1445, 12, 30, 2024, 7, 7, "the leap day of 1445"},
		{1446, 1, 1, 2024, 7, 8, "and the new year the day after it"},
	} {
		got := islamicDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("%d-%02d-%02d Islamic is day %d, want %d (%s)",
				c.year, c.month, c.day, got, want, c.what)
		}
	}
}

func TestIslamicLeapYearsOfTheCycle(t *testing.T) {
	var got []int
	for year := 1; year <= 30; year++ {
		if islamicLeap(year) {
			got = append(got, year)
		}
	}
	want := []int{2, 5, 7, 10, 13, 16, 18, 21, 24, 26, 29}
	if !slices.Equal(got, want) {
		t.Errorf("leap years of the cycle are %v, want %v", got, want)
	}
}

func TestSwedishAgainstKnownDays(t *testing.T) {
	for _, c := range []struct {
		year, month, day int
		gy, gm, gd       int
		what             string
	}{
		{1700, 2, 28, 1700, 3, 10, "the last day the Swedish and Julian calendars agreed"},
		{1700, 3, 1, 1700, 3, 11, "the day after, Julian 29 February having been left out"},
		{1700, 11, 20, 1700, 11, 30, "Narva, 19 November in the Julian calendar"},
		{1709, 6, 28, 1709, 7, 8, "Poltava, 27 June in the Julian calendar"},
		{1712, 2, 29, 1712, 3, 10, "29 February 1712"},
		{1712, 2, 30, 1712, 3, 11, "30 February 1712, the day no other calendar has"},
		{1712, 3, 1, 1712, 3, 12, "and 1 March, the Julian calendar again"},
	} {
		got := swedishDayNumber(c.year, c.month, c.day)
		want := gregorian(c.gy, c.gm, c.gd)
		if got != want {
			t.Errorf("%d-%02d-%02d Swedish is day %d, want %d (%s)",
				c.year, c.month, c.day, got, want, c.what)
		}
	}
}

// TestEveryCalendarCountsWithoutGaps walks each reckoning day by day over the years it
// covers. A month length that disagrees with the conversion shows up here as a day
// number that repeats or skips, which is the failure every hand-written calendar has.
func TestEveryCalendarCountsWithoutGaps(t *testing.T) {
	for _, c := range []struct {
		calendar    Calendar
		first, last int
	}{
		{Gregorian, 1580, 1620},
		{Gregorian, 1890, 1910},
		{Julian, 1580, 1620},
		{Julian, 1, 20},
		{Hebrew, 5560, 5600},
		{Hebrew, 5700, 5800},
		{FrenchRepublican, 1, 14},
		{Persian, 1, 60},
		{Persian, 1170, 1650},
		{Islamic, 1, 60},
		{Islamic, 1300, 1500},
		{Swedish, 1700, 1712},
	} {
		r := reckonings[c.calendar]
		previous := r.dayNumber(c.first, 1, 1) - 1
		for year := c.first; year <= c.last; year++ {
			for month := 1; month <= len(r.months); month++ {
				for day := 1; day <= r.daysIn(month, year); day++ {
					got := r.dayNumber(year, month, day)
					if got != previous+1 {
						t.Fatalf("%s year %d month %d day %d jumped from %d to %d",
							r.keyword, year, month, day, previous, got)
					}
					previous = got
				}
			}
		}
	}
}

// TestTheOpenYearAdmitsEveryDay holds the year a yearless date is measured against to
// the reason it is there: no real year of the calendar may have a longer month than it.
func TestTheOpenYearAdmitsEveryDay(t *testing.T) {
	for _, c := range []struct {
		calendar    Calendar
		first, last int
	}{
		{Gregorian, 1, 2400},
		{Julian, 1, 2400},
		{Hebrew, 5000, 6000},
		{FrenchRepublican, 1, 14},
		{Persian, 1, 3177},
		{Islamic, 1, 3000},
		{Swedish, 1700, 1712},
	} {
		r := reckonings[c.calendar]
		for month := 1; month <= len(r.months); month++ {
			open := r.daysIn(month, r.open)
			for year := c.first; year <= c.last; year++ {
				if got := r.daysIn(month, year); got > open {
					t.Errorf("%s month %d has %d days in year %d and %d in the open year %d",
						r.keyword, month, got, year, open, r.open)
					break
				}
			}
		}
	}
}
