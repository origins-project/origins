package model_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

var benchToday = model.DateOf(time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC))

// large is the corpus tools/mkcorpora.go writes, 5,000 people over twelve generations.
func large(b *testing.B) *model.Corpus {
	b.Helper()
	root := filepath.Join("..", "..", "corpora", "large")
	if _, err := os.Stat(filepath.Join(root, store.ProjectFile)); err != nil {
		b.Skip("corpora/large is not built")
	}
	corpus, err := store.Load(root)
	if err != nil {
		b.Fatal(err)
	}
	return corpus.Corpus
}

func BenchmarkHide(b *testing.B) {
	corpus := large(b)
	for b.Loop() {
		model.Hide(corpus, model.Hiding{Private: true, Living: true}, benchToday)
	}
}

func BenchmarkLiving(b *testing.B) {
	corpus := large(b)
	for b.Loop() {
		lives := corpus.Lives(benchToday)
		for _, p := range corpus.People {
			lives.Living(p)
		}
	}
}

func BenchmarkParentage(b *testing.B) {
	corpus := large(b)
	for b.Loop() {
		corpus.Parentage()
	}
}

// Every person's ancestors and descendants, as the lists and charts of the whole tree
// ask for them.
func BenchmarkAncestorsAndDescendants(b *testing.B) {
	corpus := large(b)
	people := corpus.PeopleSlugs()
	for b.Loop() {
		for _, p := range people {
			corpus.Ancestors(p)
			corpus.Descendants(p)
		}
	}
}

// The relation between the first person and each of a hundred others spread across the
// tree.
func BenchmarkKinship(b *testing.B) {
	corpus := large(b)
	people := corpus.PeopleSlugs()
	step := max(1, len(people)/100)
	for b.Loop() {
		for i := step; i < len(people); i += step {
			corpus.Kinship(people[0], people[i])
		}
	}
}
