package chart_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/chart"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// large is the index of the corpus tools/mkcorpora.go writes, 5,000 people over twelve
// generations, as the pages are handed it, and somebody in the middle of its tree: a
// child of one family and a partner in another, with the most of both.
func large(b *testing.B) (chart.Index, string) {
	b.Helper()
	root := filepath.Join("..", "..", "corpora", "large")
	if _, err := os.Stat(filepath.Join(root, store.ProjectFile)); err != nil {
		b.Skip("corpora/large is not built")
	}
	corpus, err := store.Load(root)
	if err != nil {
		b.Fatal(err)
	}
	indexed := api.IndexOfCorpus(corpus.Corpus, model.DateOf(time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)))
	data, err := json.Marshal(indexed)
	if err != nil {
		b.Fatal(err)
	}
	var index chart.Index
	if err := json.Unmarshal(data, &index); err != nil {
		b.Fatal(err)
	}
	subject, most := "", 0
	for _, p := range indexed.People {
		if ties := len(p.ChildIn) + len(p.PartnerIn); len(p.ChildIn) > 0 && len(p.PartnerIn) > 0 && ties > most {
			subject, most = strings.TrimPrefix(p.Ref, "person/"), ties
		}
	}
	return index, subject
}

func BenchmarkKin(b *testing.B) {
	index, _ := large(b)
	for b.Loop() {
		chart.KinOf(index)
	}
}

// Each chart of the subject at the distance the pages open it at and at the farthest
// they offer, the farthest stopping at 400 people.
func BenchmarkDraw(b *testing.B) {
	index, subject := large(b)
	kin := chart.KinOf(index)
	to := strings.TrimPrefix(index.People[len(index.People)-1].Ref, "person/")
	for _, distance := range []int{5, 12} {
		for _, shape := range chart.Shapes {
			ask := chart.Ask{Subject: subject, Shape: shape, Distance: distance, To: to, Year: 2026}
			b.Run(fmt.Sprintf("%s/%d", shape, distance), func(b *testing.B) {
				for b.Loop() {
					if _, ok := chart.Draw(kin, ask); !ok {
						b.Fatalf("no chart %s", shape)
					}
				}
			})
		}
	}
}
