package gramps

import (
	"reflect"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// What the export says of the tree is the manifest's: the owner of the database is a
// researcher, the home person is whom the tree starts from, a bookmark is a record kept
// to hand, the record holding it for an event, a citation and a picture, and a tag is
// shown in its colour and priority.
func TestTheFileIsTheManifest(t *testing.T) {
	root := element(t, `<?xml version="1.0" encoding="UTF-8"?>
<database xmlns="http://gramps-project.org/xml/1.7.2/">
  <header>
    <created date="2026-10-05" version="6.0.0"/>
    <researcher>
      <resname>Anna Rossi</resname>
      <resaddr>Via Esempio 1</resaddr>
      <rescity>Firenze</rescity>
      <rescountry>Italia</rescountry>
      <resphone>000 0000</resphone>
      <resemail>anna@example.org</resemail>
    </researcher>
  </header>
  <tags>
    <tag handle="_t1" change="1" name="emigrant" color="#ff0000000000" priority="0"/>
    <tag handle="_t2" change="1" name="to check" color="#000000000000" priority="1"/>
  </tags>
  <events>
    <event handle="_e1" change="1" id="E0001"><type>Death</type></event>
  </events>
  <people home="_i2">
    <person handle="_i1" change="1" id="I0001">
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <eventref hlink="_e1" role="Primary"/>
      <tagref hlink="_t1"/>
    </person>
    <person handle="_i2" change="1" id="I0002">
      <name><first>Anna</first><surname>Rossi</surname></name>
    </person>
  </people>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <page>n. 4</page>
      <sourceref hlink="_s1"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001"><stitle>Morti</stitle></source>
  </sources>
  <bookmarks>
    <bookmark target="person" hlink="_i1"/>
    <bookmark target="event" hlink="_e1"/>
    <bookmark target="citation" hlink="_c1"/>
    <bookmark target="source" hlink="_s1"/>
  </bookmarks>
</database>`)
	corpus, _, notes := Convert(&Export{Root: root, Version: "1.7.2"}, nil, nil)
	project := corpus.Project

	want := model.Researcher{
		Name:      "Anna Rossi",
		Addresses: []model.Address{{Street: "Via Esempio 1", City: "Firenze", Country: "Italia"}},
		Links:     []model.Link{{URL: "tel:000 0000"}, {URL: "mailto:anna@example.org"}},
	}
	if len(project.Researchers) != 1 || !reflect.DeepEqual(project.Researchers[0], want) {
		t.Errorf("researchers = %+v", project.Researchers)
	}
	if project.MainPerson != "rossi-anna" {
		t.Errorf("the tree starts from %q", project.MainPerson)
	}
	carlo := model.EntityRef{Kind: model.KindPerson, Slug: "rossi-carlo"}
	if want := []model.EntityRef{carlo, {Kind: model.KindSource, Slug: "morti"}}; !slices.Equal(project.Bookmarks, want) {
		t.Errorf("bookmarks = %v", project.Bookmarks)
	}
	if want := map[string]model.TagStyle{
		"emigrant": {Color: "#ff0000000000", Priority: "0"}, "to check": {Color: "#000000000000", Priority: "1"},
	}; !reflect.DeepEqual(project.Tags, want) {
		t.Errorf("tags = %v", project.Tags)
	}
	if tags := person(t, corpus, "rossi-carlo").Tags; !slices.Equal(tags, []string{"emigrant"}) {
		t.Errorf("Carlo's tags = %v", tags)
	}
	if len(notes) != 0 {
		t.Errorf("reported %q", notes)
	}
}

// A researcher with nothing in it says nothing, and leaves nothing to keep; one with no
// name is no researcher of the manifest's, and what it does say is kept.
func TestAnEmptyResearcherSaysNothing(t *testing.T) {
	for _, c := range []struct{ researcher, kept string }{
		{"<researcher>\n    </researcher>", ""},
		{"<researcher><resname> </resname><resemail/></researcher>", ""},
		{"<researcher><rescity>Firenze</rescity></researcher>", "<rescity>Firenze</rescity>"},
	} {
		root := element(t, `<?xml version="1.0" encoding="UTF-8"?>
<database xmlns="http://gramps-project.org/xml/1.7.2/">
  <header>
    <created date="2026-10-05" version="6.0.0"/>
    `+c.researcher+`
  </header>
</database>`)
		corpus, _, notes := Convert(&Export{Root: root, Version: "1.7.2"}, nil, nil)
		if len(corpus.Project.Researchers) != 0 {
			t.Errorf("%s: researchers = %+v", c.researcher, corpus.Project.Researchers)
		}
		if c.kept == "" {
			if len(corpus.Notes) != 0 || len(notes) != 0 {
				t.Errorf("%s: notes %v, reported %q", c.researcher, corpus.NoteSlugs(), notes)
			}
			continue
		}
		if len(corpus.Notes) != 1 {
			t.Fatalf("%s: notes %v", c.researcher, corpus.NoteSlugs())
		}
		for _, note := range corpus.Notes {
			keeps(t, c.researcher, note.Body, c.kept)
		}
	}
}
