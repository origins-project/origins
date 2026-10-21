package importing

import (
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Chooser settles a slug collision. what describes the thing being named — a person's
// full name, a place's full comma-separated path — and proposals are the discriminated
// slugs origins offers, best first. Returning a slug not among them is allowed: a human
// who knows the family can name it better than any rule.
//
// A nil Chooser takes the first proposal, which is what makes importing five thousand
// people possible without five thousand questions.
type Chooser func(what string, proposals []model.Slug) model.Slug

// Namer hands out slugs and remembers which it has handed out.
//
// Reserving on assignment rather than on insertion matters because a place is named
// before its containers are built and a family before its own entity exists. A namer
// that only consulted the corpus would hand the same slug out twice in between.
type Namer struct {
	corpus *model.Corpus
	choose Chooser
	given  map[model.Slug]bool
}

// NewNamer hands out slugs the corpus does not hold, choose naming a collision where it
// can.
func NewNamer(corpus *model.Corpus, choose Chooser) *Namer {
	return &Namer{corpus: corpus, choose: choose, given: map[model.Slug]bool{}}
}

func (n *Namer) taken(s model.Slug) bool { return n.given[s] || n.corpus.Taken(s) }

// Assign returns a free slug for base, discriminating it when base is spoken for.
//
// hints are offered in order and should be facts still worth reading in three years — a
// birth year, a place, the containing region. A numeric suffix is the last resort,
// because it distinguishes without saying anything.
func (n *Namer) Assign(base model.Slug, hints []string, what string) model.Slug {
	if _, err := model.ParseSlug(string(base)); err != nil {
		return ""
	}
	if !n.taken(base) {
		n.given[base] = true
		return base
	}

	proposals, err := model.Discriminate(base, hints, n.taken)
	if err != nil || len(proposals) == 0 {
		return ""
	}
	chosen := proposals[0]
	if n.choose != nil {
		if picked := n.choose(what, proposals); picked != "" && !n.taken(picked) {
			chosen = picked
		}
	}
	n.given[chosen] = true
	return chosen
}

// Reserve records a slug that was not handed out by Assign, so that nothing else is
// given it later.
func (n *Namer) Reserve(s model.Slug) { n.given[s] = true }

// slugWords caps how much of a title becomes a slug. Titles run to whole sentences —
// "Parish of San Lorenzo — Baptismal Register 1820–1850" — and a slug made of all of it
// is a directory name nobody can read or type. Six words is enough to tell two registers
// apart, and a collision is discriminated like any other.
const slugWords = 6

// joining words carry no meaning on their own, so a slug that stops on one has spent a
// word saying nothing and reads as though it were cut off mid-thought — which it was.
var joining = map[string]bool{
	"a": true, "al": true, "an": true, "and": true, "da": true, "de": true,
	"degli": true, "dei": true, "del": true, "della": true, "di": true, "e": true,
	"en": true, "et": true, "for": true, "in": true, "la": true, "le": true,
	"of": true, "on": true, "or": true, "the": true, "to": true, "und": true, "y": true,
}

// Shorten cuts a slug made of a title to slugWords words, and drops a joining word it
// would otherwise end on.
func Shorten(slug model.Slug) model.Slug {
	words := strings.Split(string(slug), "-")
	if len(words) > slugWords {
		words = words[:slugWords]
	}
	for len(words) > 1 && joining[words[len(words)-1]] {
		words = words[:len(words)-1]
	}
	return model.Slug(strings.Join(words, "-"))
}
