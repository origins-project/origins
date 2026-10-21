package model

import (
	"fmt"
	"strings"

	"golang.org/x/text/language"
)

// Language is the language a name is in, as a BCP 47 tag: en, de, la, grc. It is read in
// the tag's canonical spelling, so EN and en are one corpus.
type Language string

// ParseLanguage reads a language tag in its canonical spelling, nothing being no
// language.
func ParseLanguage(s string) (Language, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	tag, err := language.Parse(s)
	if err != nil {
		return "", fmt.Errorf("language %q: expected a language tag such as en, de or la", s)
	}
	return Language(tag.String()), nil
}
