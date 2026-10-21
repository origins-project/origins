package model

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Slug is an entity's identity and its file or directory name at once. There is no
// separate id field anywhere in the corpus, so there is nothing to desynchronise.
//
// Words are joined by a single hyphen. A double hyphen joins the partners in a family's
// slug, which is unambiguous precisely because single hyphens never appear alone
// between them.
type Slug string

// familySeparator joins the partner slugs of a family.
const familySeparator = "--"

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-{1,2}[a-z0-9]+)*$`)

// ParseSlug checks that s is a well-formed slug and returns it.
func ParseSlug(s string) (Slug, error) {
	if !slugPattern.MatchString(s) {
		return "", fmt.Errorf("%q is not a slug: expected lowercase words joined by hyphens", s)
	}
	return Slug(s), nil
}

func (s Slug) String() string { return string(s) }

func (s Slug) MarshalText() ([]byte, error) { return []byte(s), nil }

func (s *Slug) UnmarshalText(b []byte) error {
	parsed, err := ParseSlug(string(b))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// FamilySlug names a family after its partners: the people's slugs in order, so that
// the name does not depend on which partner was recorded first, and then the words for
// those nobody knows or who were nobody, so that a person's families begin with their
// name. It is bounded as any slug is, a family of long names being cut short and left to
// a discriminator like any other collision.
func FamilySlug(partners []Partner) (Slug, error) {
	var people, words []string
	for _, partner := range partners {
		if who := partner.Who(); who.Reserved() {
			words = append(words, string(who))
		} else {
			people = append(people, string(who))
		}
	}
	slices.Sort(people)
	slices.Sort(words)
	return ParseSlug(shorten(strings.Join(append(people, words...), familySeparator), maxSlug))
}

// expansions cover what accent-folding cannot: letters that are letters in their own
// right rather than an accented form of another, and the apostrophes inside names like
// D'Angelo, which bind the name together rather than separating two words, as the ayin
// and the hamza a romanization writes in ʿAlī and ʿĀʾisha do.
var expansions = strings.NewReplacer(
	"'", "",
	"’", "",
	"ʿ", "",
	"ʾ", "",
	"ß", "ss",
	"æ", "ae",
	"œ", "oe",
	"ø", "o",
	"đ", "d",
	"ħ", "h",
	"ł", "l",
	"ŋ", "n",
	"ŧ", "t",
	"ð", "d",
	"þ", "th",
	"ı", "i",
)

// stripAccents is made afresh for every name, because a chain of transformers keeps
// the state of the text passing through it and two goroutines sharing one read each
// other's.
func stripAccents() transform.Transformer {
	return transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
}

var notSlugRune = regexp.MustCompile(`[^a-z0-9]+`)

// maxSlug bounds a slug derived from a name, a family's among them. The slug is the file
// name, so an unbounded one is a name no filesystem will take: an archduchess recorded
// under eighteen given names produces a family nobody can write to disk. Sixty-four
// characters is well past the point where more of a name still tells you whose it is.
const maxSlug = 64

// Slugify derives a slug from a name. Accents are folded rather than dropped, so
// Cambiasù becomes cambiasu and not cambias.
func Slugify(name string) (Slug, error) {
	folded, _, err := transform.String(stripAccents(), expansions.Replace(strings.ToLower(name)))
	if err != nil {
		return "", fmt.Errorf("slugify %q: %w", name, err)
	}
	s := strings.Trim(notSlugRune.ReplaceAllString(folded, "-"), "-")
	if s == "" {
		return "", fmt.Errorf("slugify %q: nothing left to name it by", name)
	}
	return Slug(shorten(s, maxSlug)), nil
}

var notWordRune = regexp.MustCompile(`[^\p{L}\p{M}\p{N}]+`)

// SameWord reports whether two open words are one word: an event's type, a role, a
// relation, the type of a name, a family, a place, a source or an attribute, each kept
// as the record wrote it, so that Birth, birth and BIRTH are one birth. A word is
// compared by its slug, folded as Slugify folds a name but never cut short and keeping
// the letters of a script that does not fold to ASCII, or by its words where it has no
// letter or digit to make a slug of.
func SameWord(a, b string) bool { return wordKey(a) == wordKey(b) }

func wordKey(word string) string {
	if key, ok := asciiKey(word); ok {
		return key
	}
	return foldedKey(word)
}

// asciiKey is the key of a word in ASCII alone, which has nothing to fold but its case
// and its apostrophes, made without the transformers that would otherwise be most of
// what a check spends.
func asciiKey(word string) (string, bool) {
	var key strings.Builder
	key.Grow(len(word))
	gap := false
	for i := 0; i < len(word); i++ {
		c := word[i]
		switch {
		case c >= utf8.RuneSelf:
			return "", false
		case c == '\'':
		case 'a' <= c && c <= 'z', '0' <= c && c <= '9', 'A' <= c && c <= 'Z':
			if gap && key.Len() > 0 {
				key.WriteByte('-')
			}
			gap = false
			if 'A' <= c && c <= 'Z' {
				c += 'a' - 'A'
			}
			key.WriteByte(c)
		default:
			gap = true
		}
	}
	if key.Len() == 0 {
		return strings.Join(strings.Fields(word), " "), true
	}
	return key.String(), true
}

func foldedKey(word string) string {
	folded, _, err := transform.String(stripAccents(), expansions.Replace(strings.ToLower(word)))
	if err == nil {
		if key := strings.Trim(notWordRune.ReplaceAllString(folded, "-"), "-"); key != "" {
			return key
		}
	}
	return strings.Join(strings.Fields(word), " ")
}

// shorten cuts a slug to at most n characters at a word boundary, so what survives is
// still words. A single word longer than the whole allowance is cut mid-word, because
// something has to give and there is no boundary to prefer. Any hyphen left exposed by
// the cut goes with it, including the doubled one joining the halves of a family.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := s[:n]
	if i := strings.LastIndexByte(s[:n+1], '-'); i > 0 {
		cut = s[:i]
	}
	return strings.TrimRight(cut, "-")
}

// Discriminate proposes slugs that tell an entity apart from one already holding base.
// Each hint is a word that should still be worth reading in three years — a birth year,
// a place, "senior" — and they are offered in the order given. A numeric suffix comes
// last and only once, because it distinguishes without saying anything.
//
// taken reports whether a slug is already in use; the proposals exclude those.
func Discriminate(base Slug, hints []string, taken func(Slug) bool) ([]Slug, error) {
	if _, err := ParseSlug(string(base)); err != nil {
		return nil, err
	}
	if taken == nil {
		return nil, errors.New("discriminate: no way to tell which slugs are free")
	}

	var proposals []Slug
	seen := map[Slug]bool{}
	offer := func(suffix string) bool {
		s := discriminated(base, suffix)
		if s == "" || seen[s] || taken(s) {
			return false
		}
		seen[s] = true
		proposals = append(proposals, s)
		return true
	}

	for _, hint := range hints {
		suffix, err := Slugify(hint)
		if err != nil {
			continue
		}
		offer(string(suffix))
	}
	for n := 2; ; n++ {
		if offer(strconv.Itoa(n)) {
			break
		}
	}
	return proposals, nil
}

// discriminated is base told apart by a suffix. The base gives way when the two together
// are too long, because the suffix is the whole reason this slug differs from the one
// already holding the base, and there is nothing to make where even one character of
// the base will not fit.
func discriminated(base Slug, suffix string) Slug {
	room := maxSlug - len(suffix) - 1
	if room < 1 {
		return ""
	}
	return Slug(shorten(string(base), room) + "-" + suffix)
}

// Discriminator is what tells s apart from base, where s is base or base told apart by
// Discriminate, and ok is false where it is neither. It is empty where s is base itself.
// The shortest that will do is taken, a discriminator being a word or two.
func (s Slug) Discriminator(base Slug) (suffix string, ok bool) {
	if s == Slug(shorten(string(base), maxSlug)) {
		return "", true
	}
	for i := len(s) - 2; i > 0; i-- {
		if s[i] == '-' && s[i-1] != '-' && s[i+1] != '-' && discriminated(base, string(s[i+1:])) == s {
			return string(s[i+1:]), true
		}
	}
	return "", false
}

// Rebase is s with the base now in place of the base was, keeping the discriminator s
// had, or now itself where s had none or was not made from was at all.
func (s Slug) Rebase(was, now Slug) Slug {
	suffix, ok := s.Discriminator(was)
	if !ok || suffix == "" {
		return now
	}
	if rebased := discriminated(now, suffix); rebased != "" {
		return rebased
	}
	return now
}
