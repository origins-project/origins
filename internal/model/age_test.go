package model

import "testing"

func TestAnAgeIsReadAsGEDCOMWritesOne(t *testing.T) {
	for in, want := range map[string]Age{
		"":            "",
		"24":          "24",
		" 67y ":       "67y",
		"3m 12d":      "3m 12d",
		"<1y":         "< 1y",
		">  80Y":      "> 80y",
		"1y 2m 3w 4d": "1y 2m 3w 4d",
		"0d":          "0d",
		"2w":          "2w",
	} {
		got, err := ParseAge(in)
		if err != nil || got != want {
			t.Errorf("ParseAge(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
}

func TestWhatIsNotAnAgeIsRefused(t *testing.T) {
	for _, in := range []string{
		"twenty", "24 years", "12d 3m", "1y 1y", "<", "> ", "-3y", "y", "1.5y", "about 24",
	} {
		if got, err := ParseAge(in); err == nil {
			t.Errorf("ParseAge(%q) = %q, want an error", in, got)
		}
	}
}
