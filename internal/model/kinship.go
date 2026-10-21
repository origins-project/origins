package model

import (
	"fmt"
	"strings"
)

// Name is what the first person of a kinship is to the second, in the words English
// uses for it — father, great-aunt, second cousin once removed — given the first
// person's sex. An unrecorded sex takes the word that does not say, or both where
// English has no such word.
//
// A kinship through one shared parent is named as though it ran through two. Which
// it was is a question about the records, and the path answers it.
func (k Kinship) Name(sex Sex) string {
	up, down := k.Up, k.Down
	switch {
	case up == 0 && down == 0:
		return "the same person"
	case up == 0:
		return lineal(down, parent, grandparent, sex)
	case down == 0:
		return lineal(up, child, grandchild, sex)
	case up == 1 && down == 1:
		return sibling.of(sex, "")
	case up == 1:
		return auncle.of(sex, greats(down-2))
	case down == 1:
		return nibling.of(sex, greats(up-2))
	}
	degree, removed := min(up, down)-1, max(up, down)-min(up, down)
	name := ordinal(degree) + " cousin"
	if removed > 0 {
		name += " " + times(removed) + " removed"
	}
	return name
}

// Generation is what a generation of ancestors or descendants is called, the nearest
// being the first: parents, grandparents, great-grandparents going up, and children,
// grandchildren going down.
func Generation(n int, upward bool) string {
	if upward {
		return lineal(n, parents, grandparents, SexUnknown)
	}
	return lineal(n, children, grandchildren, SexUnknown)
}

// word is one relation in each of the ways English says it. A word with no neutral
// form is said both ways where the sex is not recorded.
type word struct{ male, female, neutral string }

var (
	parent        = word{"father", "mother", "parent"}
	grandparent   = word{"grandfather", "grandmother", "grandparent"}
	child         = word{"son", "daughter", "child"}
	grandchild    = word{"grandson", "granddaughter", "grandchild"}
	sibling       = word{"brother", "sister", "sibling"}
	auncle        = word{"uncle", "aunt", ""}
	nibling       = word{"nephew", "niece", ""}
	parents       = word{neutral: "parents"}
	grandparents  = word{neutral: "grandparents"}
	children      = word{neutral: "children"}
	grandchildren = word{neutral: "grandchildren"}
)

func (w word) of(sex Sex, prefix string) string {
	switch {
	case sex == Male:
		return prefix + w.male
	case sex == Female:
		return prefix + w.female
	case w.neutral != "":
		return prefix + w.neutral
	}
	return prefix + w.female + " or " + prefix + w.male
}

// lineal names a relative in a straight line n generations away.
func lineal(n int, first, second word, sex Sex) string {
	if n == 1 {
		return first.of(sex, "")
	}
	return second.of(sex, greats(n-2))
}

func greats(n int) string { return strings.Repeat("great-", n) }

func ordinal(n int) string {
	words := []string{"", "first", "second", "third", "fourth", "fifth", "sixth",
		"seventh", "eighth", "ninth", "tenth"}
	if n < len(words) {
		return words[n]
	}
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

func times(n int) string {
	words := []string{"", "once", "twice", "three times", "four times", "five times",
		"six times", "seven times", "eight times", "nine times", "ten times"}
	if n < len(words) {
		return words[n]
	}
	return fmt.Sprintf("%d times", n)
}
