package api

import "github.com/origins-project/origins/internal/model"

// DateRead is a date as origins reads it: written as the corpus writes it, whether it
// has a place on the calendar, and where a list puts it.
type DateRead struct {
	Date     string `json:"date"`
	Anchored bool   `json:"anchored"`
	Sort     string `json:"sort,omitempty"`
}

// DateOf reads a date as it was typed, for whoever is writing one to see what it comes
// to before it is written anywhere.
func DateOf(text string) (*DateRead, error) {
	d, err := model.ParseDate(text)
	if err != nil {
		return nil, err
	}
	read := &DateRead{Date: d.String(), Anchored: d.Anchored()}
	read.Sort, _ = d.SortKey()
	return read, nil
}
