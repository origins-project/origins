package model

import (
	"fmt"
	"strconv"
	"strings"
)

// Age is the age a record gives somebody, as GEDCOM 7 writes one: years, months, weeks
// and days, each a number and its letter and in that order, optionally bounded below or
// above — `67y`, `3m 12d`, `< 1y`, `> 80y`. A bare number is years, which is how most
// records give an age and how most people would type one.
//
// It is what the record says and is carried as the record said it, normalised only in
// its spacing and the case of its letters; nothing is computed from it, so a stated age
// and a birth date that do not agree are both kept.
type Age string

// ageUnits are the letters of an age's parts, in the order they are written.
const ageUnits = "ymwd"

// ParseAge reads an age, writing its bound and its parts apart by one space each.
func ParseAge(s string) (Age, error) {
	text := strings.TrimSpace(s)
	if text == "" {
		return "", nil
	}
	if _, err := strconv.ParseUint(text, 10, 32); err == nil {
		return Age(text), nil
	}

	var words []string
	if bound := text[0]; bound == '<' || bound == '>' {
		words = append(words, string(bound))
		text = strings.TrimSpace(text[1:])
	}
	next := 0
	for _, part := range strings.Fields(text) {
		number, unit := part[:len(part)-1], strings.ToLower(part[len(part)-1:])
		at := strings.Index(ageUnits[next:], unit)
		if _, err := strconv.ParseUint(number, 10, 32); err != nil || at < 0 {
			return "", fmt.Errorf("age %q: expected years, months, weeks and days in that order, "+
				"as in 67y or 3m 12d, after an optional < or >", s)
		}
		next += at + 1
		words = append(words, number+unit)
	}
	if len(words) == 0 || words[len(words)-1] == "<" || words[len(words)-1] == ">" {
		return "", fmt.Errorf("age %q: a bound is followed by an age", s)
	}
	return Age(strings.Join(words, " ")), nil
}

func (a Age) String() string { return string(a) }
