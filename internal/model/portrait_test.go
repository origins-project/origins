package model

import (
	"reflect"
	"testing"
)

// The portrait is chosen once, here, so that every page, list and chart shows the same
// one: the record's own, else its first photograph, else a picture it is in.
func TestThePortraitIsChosenByOneRule(t *testing.T) {
	c := rossi(t)
	giovanni, maria := c.People["rossi-giovanni"], c.People["bianchi-maria"]
	region := &Region{Left: 0.4, Top: 0.1, Right: 0.6, Bottom: 0.5}
	family := c.Families["bianchi-maria--rossi-giovanni"]
	family.Files = []string{"photos/wedding.jpg"}
	family.Media = []Media{{File: "photos/wedding.jpg", Depicts: []Depicted{
		{Person: "rossi-giovanni"}, {Person: "bianchi-maria", Region: region},
	}}}
	self := EntityRef{Kind: KindPerson, Slug: "rossi-giovanni"}
	kept := EntityRef{Kind: KindFamily, Slug: "bianchi-maria--rossi-giovanni"}

	cases := []struct {
		name     string
		files    []string
		portrait Image
		want     Portrait
		found    bool
	}{
		{"nothing of their own takes a picture they are in", nil, Image{},
			Portrait{Record: kept, File: "photos/wedding.jpg"}, true},
		{"the first image under photos/", []string{"documents/a.jpg", "photos/b.txt", "photos/c.png", "photos/d.jpg"}, Image{},
			Portrait{Record: self, File: "photos/c.png"}, true},
		{"the portrait named, cropped", []string{"photos/a.jpg", "photos/b.jpg"},
			Image{File: "photos/b.jpg", Region: region},
			Portrait{Record: self, File: "photos/b.jpg", Region: region}, true},
		{"a portrait with no file beside it is passed over", []string{"photos/a.jpg"}, Image{File: "photos/gone.jpg"},
			Portrait{Record: self, File: "photos/a.jpg"}, true},
	}
	for _, tc := range cases {
		giovanni.Files, giovanni.Portrait = tc.files, tc.portrait
		got, found := c.PortraitOf("rossi-giovanni")
		if found != tc.found || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%s: PortraitOf = %+v, %v, want %+v", tc.name, got, found, tc.want)
		}
		everybody(t, c)
	}

	if got, _ := c.PortraitOf("bianchi-maria"); !reflect.DeepEqual(got, Portrait{Record: kept, File: "photos/wedding.jpg", Region: region}) {
		t.Errorf("a picture somebody is in is cropped to them: %+v", got)
	}
	maria.Files = []string{"photos/alone.jpg"}
	if got, _ := c.PortraitOf("bianchi-maria"); got.Record.Slug != "bianchi-maria" {
		t.Errorf("a photograph of one's own comes before one of a group: %+v", got)
	}
	everybody(t, c)
	family.Files = nil
	if got, found := c.PortraitOf("rossi-anna"); found {
		t.Errorf("somebody in no picture has a portrait: %+v", got)
	}
	giovanni.Files, giovanni.Portrait = nil, Image{}
	if got, found := c.PortraitOf("rossi-giovanni"); found {
		t.Errorf("a picture not beside its record is shown: %+v", got)
	}
}

// Portraits answers for everybody at once what PortraitOf answers for each.
func everybody(t *testing.T, c *Corpus) {
	t.Helper()
	all := c.Portraits()
	for slug := range c.People {
		got, found := all[slug]
		want, ok := c.PortraitOf(slug)
		if found != ok || !reflect.DeepEqual(got, want) {
			t.Errorf("Portraits()[%s] = %+v, %v, want %+v, %v", slug, got, found, want, ok)
		}
	}
}
