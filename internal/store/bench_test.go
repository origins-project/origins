package store

import (
	"os"
	"path/filepath"
	"testing"
)

// large is the corpus tools/mkcorpora.go writes, 5,000 people over twelve generations.
func large(b *testing.B) string {
	b.Helper()
	root := filepath.Join("..", "..", "corpora", "large")
	if _, err := os.Stat(filepath.Join(root, ProjectFile)); err != nil {
		b.Skip("corpora/large is not built")
	}
	return root
}

func BenchmarkLoad(b *testing.B) {
	root := large(b)
	for b.Loop() {
		if _, err := Load(root); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEmit(b *testing.B) {
	corpus, err := Load(large(b))
	if err != nil {
		b.Fatal(err)
	}
	slugs := corpus.AllSlugs()
	for b.Loop() {
		for _, slug := range slugs {
			if _, err := Emit(corpus.Entity(slug)); err != nil {
				b.Fatal(err)
			}
		}
	}
}
