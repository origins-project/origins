package ops

import (
	"reflect"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func tag(word string) model.EntityRef {
	return model.EntityRef{Kind: model.KindTag, Slug: model.Slug(word)}
}

// A tag is followed onto an event, a citation and a picture as onto a record, and its
// style goes with it.
func TestATagIsRenamedWhereverItIsCarried(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus.Clone()
	census := corpus.Sources["census-1861-england"]
	citation := census.Citations["f214"]
	citation.Tags = []string{"To do"}
	census.Citations["f214"] = citation
	style := corpus.Project.Tags["To do"]

	apply(t, corpus, model.Rename{Entity: tag("To do"), To: "Da fare"})

	for where, got := range map[string][]string{
		"Domenico's emigration": corpus.People["ershov-michael"].Events[1].Tags,
		"Elena's photograph":    corpus.People["galvao-helena"].Media[0].Tags,
		"the census entry":      census.Citations["f214"].Tags,
	} {
		if !reflect.DeepEqual(got, []string{"Da fare"}) {
			t.Errorf("%s is tagged %q", where, got)
		}
	}
	if _, kept := corpus.Project.Tags["To do"]; kept || corpus.Project.Tags["Da fare"] != style {
		t.Errorf("the styles are %v", corpus.Project.Tags)
	}
	if got := corpus.People["ershov-michael"].Tags; !reflect.DeepEqual(got, []string{"Emigrants"}) {
		t.Errorf("another tag was changed: %q", got)
	}
}

// Renaming to a word in use makes the two one: a list carrying both keeps the first of
// them, where it stood.
func TestATagRenamedToOneInUseIsOneTag(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus.Clone()
	domenico := corpus.People["ershov-michael"]
	domenico.Tags = []string{"Italia", "Emigrants", "Emigrati"}
	delete(corpus.Project.Tags, "Emigrants")

	apply(t, corpus, model.Rename{Entity: tag("Emigrants"), To: "Emigrati"})

	if got := domenico.Tags; !reflect.DeepEqual(got, []string{"Italia", "Emigrati"}) {
		t.Errorf("tagged %q", got)
	}
}

func TestATagRenamedIsRefused(t *testing.T) {
	t.Parallel()
	for name, one := range map[string]struct {
		edit model.Edit
		says string
	}{
		"onto a word styled otherwise": {model.Rename{Entity: tag("To do"), To: "Emigrants"}, "#d33 with priority 1"},
		"from a word nobody uses":      {model.Rename{Entity: tag("to do"), To: "Done"}, "no record carries it"},
		"to the word it is":            {model.Rename{Entity: tag("To do"), To: "To do"}, "the word it is"},
		"to nothing":                   {model.Rename{Entity: tag("To do"), To: " "}, "cannot be empty"},
		"by anything but a rename":     {model.Set{Entity: tag("To do"), Path: "color", Value: "#000"}, "only renamed"},
		"by merging it":                {model.Merge{From: tag("To do"), Into: tag("Emigrants")}, "only renamed"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			corpus := fixture(t).Corpus.Clone()
			_, err := applied(corpus, one.edit)
			if err == nil || !strings.Contains(err.Error(), one.says) {
				t.Errorf("got %v, want it refused saying %q", err, one.says)
			}
		})
	}
}

// The commit names the two words, every record whose tags it rewrote, and the manifest
// where the style moved.
func TestATagRenamedNamesWhatItRewrote(t *testing.T) {
	t.Parallel()
	corpus := fixture(t).Corpus.Clone()
	entry, _, err := performed(corpus, model.Rename{Entity: tag("To do"), To: "Da fare"})
	if err != nil {
		t.Fatal(err)
	}
	want := []model.EntityRef{tag("To do"), tag("Da fare"),
		ref(model.KindPerson, "ershov-michael"), ref(model.KindPerson, "galvao-helena"), model.TheProject}
	if entry.Op != "tags.rename" || !reflect.DeepEqual(entry.Entities, want) {
		t.Errorf("filed under %s about %v", entry.Op, entry.Entities)
	}
}
