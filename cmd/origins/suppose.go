package main

import (
	"errors"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// several is a --with written more than once. Two hypotheses at a time is the question
// of whether both can be true, and there is no reason to stop at two.
type several []model.Slug

func (n *several) String() string {
	shown := make([]string, len(*n))
	for i, slug := range *n {
		shown[i] = string(slug)
	}
	return strings.Join(shown, ", ")
}

func (n *several) Set(value string) error {
	if value == "" {
		return errors.New("--with takes the name of a hypothesis")
	}
	*n = append(*n, model.Slug(value))
	return nil
}
