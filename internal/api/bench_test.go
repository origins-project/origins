package api

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

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

func BenchmarkIndex(b *testing.B) {
	corpus := large(b)
	today := model.DateOf(time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC))
	for b.Loop() {
		indexOf(corpus, today)
	}
}
