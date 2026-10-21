package ops

import (
	"fmt"
	"slices"

	"github.com/origins-project/origins/internal/model"
)

// retag gives a tag a new word: every list carrying it carries the new one in its
// place, and its style moves with it.
//
// Renaming to a word already in use is how two tags become one, so a list that comes
// to carry the word twice keeps the first, and the two styles are one style or the
// rename is refused, which colour the researcher meant being theirs to say. Words are
// compared as written, To do and to do being two words as the researcher wrote two.
func retag(corpus *model.Corpus, e model.Rename) error {
	from, to := string(e.Entity.Slug), string(e.To)
	if _, err := model.ParseTag(to); err != nil {
		return fmt.Errorf("%s: %w", e.Entity, err)
	}
	if to == from {
		return fmt.Errorf("%s: that is the word it is", e.Entity)
	}
	carriers := tagged(corpus, from)
	style, styled := corpus.Project.Tags[from]
	if len(carriers) == 0 && !styled {
		return fmt.Errorf("%s: no record carries it and no style names it", e.Entity)
	}
	if styled {
		if other, ok := corpus.Project.Tags[to]; ok && other != style {
			return fmt.Errorf("%s: it is styled %s and %s is styled %s, and which of the two is meant is yours to say: unset one of them first",
				e.Entity, styling(style), to, styling(other))
		}
		delete(corpus.Project.Tags, from)
		corpus.Project.Tags[to] = style
	}
	for _, slug := range corpus.AllSlugs() {
		model.Tags(corpus.Entity(slug), func(_ string, tags *[]string) {
			if !slices.Contains(*tags, from) {
				return
			}
			var kept []string
			for _, tag := range *tags {
				if tag == from {
					tag = to
				}
				if tag != to || !slices.Contains(kept, to) {
					kept = append(kept, tag)
				}
			}
			*tags = kept
		})
	}
	return nil
}

// tagged is every record carrying the word, in slug order.
func tagged(corpus *model.Corpus, word string) []model.EntityRef {
	var refs []model.EntityRef
	for _, slug := range corpus.AllSlugs() {
		carries := false
		model.Tags(corpus.Entity(slug), func(_ string, tags *[]string) {
			carries = carries || slices.Contains(*tags, word)
		})
		if carries {
			refs = append(refs, model.RefOf(corpus.Entity(slug)))
		}
	}
	return refs
}

// retagging is what a tag's rename changes besides the two words: every record carrying
// the word and, where the style moves, the manifest. It is read before the rename is
// carried out, while they still carry the word it is losing.
func retagging(corpus *model.Corpus, e model.Rename) []model.EntityRef {
	refs := tagged(corpus, string(e.Entity.Slug))
	if _, styled := corpus.Project.Tags[string(e.Entity.Slug)]; styled {
		refs = append(refs, model.TheProject)
	}
	return refs
}

func styling(style model.TagStyle) string {
	switch {
	case style.Color != "" && style.Priority != "":
		return fmt.Sprintf("%s with priority %s", style.Color, style.Priority)
	case style.Priority != "":
		return "with priority " + style.Priority
	}
	return style.Color
}
