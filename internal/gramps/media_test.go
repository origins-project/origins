package gramps

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// A media object is kept by the first record to name it, a family's before a person's,
// and every other person naming a photograph is somebody in it, in the part of it their
// reference marks; a scan is its source's, and a citation points at the part of it the
// entry is in.
func TestAPictureIsKeptOnceAndShowsWhoIsInIt(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"group.jpg", "portrait.jpg", "will.pdf", "register.jpg", "entry.jpg", "grave.jpg", "private.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	corpus, attachments, notes := convertFinding(t, `
  <events>
    <event handle="_death" change="1" id="E0001">
      <type>Death</type>
      <objref hlink="_grave"><region corner1_x="0" corner1_y="0" corner2_x="50" corner2_y="50"/></objref>
    </event>
  </events>
  <people>
    <person handle="_carlo" change="1" id="I0001">
      <gender>M</gender>
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <eventref hlink="_death" role="Primary"/>
      <objref hlink="_portrait"><region corner1_x="25" corner1_y="10" corner2_x="75" corner2_y="60"/></objref>
      <objref hlink="_group"><region corner1_x="40" corner1_y="90" corner2_x="10" corner2_y="20"/></objref>
      <objref hlink="_will"/>
      <objref hlink="_lost"/>
    </person>
    <person handle="_anna" change="1" id="I0002">
      <gender>F</gender>
      <name><first>Anna</first><surname>Bianchi</surname></name>
      <objref hlink="_group"><region corner1_x="50" corner1_y="20" corner2_x="80" corner2_y="90"/></objref>
      <objref hlink="_portrait">
        <citationref hlink="_c1"/>
      </objref>
      <objref hlink="_will"/>
      <objref hlink="_lost"/>
      <objref hlink="_private" priv="1"/>
    </person>
  </people>
  <families>
    <family handle="_f1" change="1" id="F0001">
      <father hlink="_carlo"/>
      <mother hlink="_anna"/>
      <objref hlink="_group"/>
    </family>
  </families>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <page>n. 112</page>
      <confidence>2</confidence>
      <objref hlink="_register"><region corner1_x="10" corner1_y="50" corner2_x="90" corner2_y="70"/></objref>
      <sourceref hlink="_s1"/>
    </citation>
    <citation handle="_c2" change="1" id="C0002">
      <page>n. 113</page>
      <confidence>2</confidence>
      <objref hlink="_entry"/>
      <sourceref hlink="_s1"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001">
      <stitle>Battesimi</stitle>
      <objref hlink="_register"/>
    </source>
  </sources>
  <objects>
    <object handle="_group" change="1" id="O0001">
      <file src="group.jpg" mime="image/jpeg" description="The wedding"/>
      <dateval val="1870-05-01"/>
      <citationref hlink="_c2"/>
    </object>
    <object handle="_portrait" change="1" id="O0002">
      <file src="portrait.jpg" mime="image/jpeg" description="Carlo"/>
    </object>
    <object handle="_will" change="1" id="O0003">
      <file src="will.pdf" mime="application/pdf" description="His will"/>
      <noteref hlink="_n1"/>
    </object>
    <object handle="_register" change="1" id="O0004">
      <file src="register.jpg" mime="image/jpeg" description="Page 12"/>
    </object>
    <object handle="_entry" change="1" id="O0005">
      <file src="entry.jpg" mime="image/jpeg" description="Entry 113"/>
    </object>
    <object handle="_grave" change="1" id="O0006">
      <file src="grave.jpg" mime="image/jpeg" description="The grave"/>
    </object>
    <object handle="_lost" change="1" id="O0007">
      <file src="lost.jpg" mime="image/jpeg" description="Lost"/>
    </object>
    <object handle="_private" change="1" id="O0008">
      <file src="private.jpg" mime="image/jpeg" description="Private"/>
    </object>
    <object handle="_nobody" change="1" id="O0009">
      <file src="group.jpg" mime="image/jpeg" description="Nobody's"/>
    </object>
  </objects>
  <notes>
    <note handle="_n1" change="1" id="N0001" type="Media Note"><text>Witnessed.</text></note>
  </notes>`, importing.Beside(dir))

	type file struct {
		record model.EntityRef
		file   string
	}
	var got []file
	for _, a := range attachments {
		got = append(got, file{a.Record, a.File})
		if filepath.Base(a.From) != filepath.Base(a.File) {
			t.Errorf("%s was found at %s", a.File, a.From)
		}
	}
	ref := func(kind model.Kind, slug model.Slug) model.EntityRef { return model.EntityRef{Kind: kind, Slug: slug} }
	family := ref(model.KindFamily, "bianchi-anna--rossi-carlo")
	carloRef, annaRef := ref(model.KindPerson, "rossi-carlo"), ref(model.KindPerson, "bianchi-anna")
	want := []file{
		{ref(model.KindSource, "battesimi"), "scans/register.jpg"},
		{ref(model.KindSource, "battesimi"), "scans/entry.jpg"},
		{family, "photos/group.jpg"},
		{carloRef, "photos/portrait.jpg"},
		{carloRef, "documents/will.pdf"},
		{carloRef, "photos/grave.jpg"},
		{annaRef, "photos/private.jpg"},
		{ref(model.KindNote, "nobodys"), "photos/group.jpg"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("attachments = %v\nwant %v", got, want)
	}

	// the family's photograph is its own, and each partner is somebody in it
	group := corpus.Families["bianchi-anna--rossi-carlo"].Media
	if len(group) != 1 || group[0].Title != "The wedding" || group[0].Date.String() != "1 MAY 1870" ||
		len(group[0].Sources) != 1 || group[0].Sources[0].String() != "battesimi#n-113" {
		t.Fatalf("the family's media = %+v", group)
	}
	if len(group[0].Depicts) != 2 || group[0].Depicts[0].Person != "rossi-carlo" || group[0].Depicts[1].Person != "bianchi-anna" ||
		*group[0].Depicts[0].Region != (model.Region{Left: .1, Top: .2, Right: .4, Bottom: .9}) ||
		*group[0].Depicts[1].Region != (model.Region{Left: .5, Top: .2, Right: .8, Bottom: .9}) {
		t.Errorf("the group shows %+v", group[0].Depicts)
	}

	carlo := person(t, corpus, "rossi-carlo").Media
	if len(carlo) != 3 {
		t.Fatalf("Carlo's media = %+v", carlo)
	}
	// a person keeping a photograph of themselves is in it, before anybody else naming it
	portrait := carlo[0]
	if len(portrait.Depicts) != 2 || portrait.Depicts[0].Person != "rossi-carlo" ||
		*portrait.Depicts[0].Region != (model.Region{Left: .25, Top: .1, Right: .75, Bottom: .6}) ||
		portrait.Depicts[1].Person != "bianchi-anna" || portrait.Depicts[1].Region != nil {
		t.Errorf("the portrait shows %+v", portrait.Depicts)
	}
	if will := carlo[1]; will.File != "documents/will.pdf" || will.Note != "[Witnessed.](../../notes/witnessed/README.md)" {
		t.Errorf("the will = %+v", will)
	}
	// a picture whose file is not to hand is an entry all the same, by the path it had
	if lost := carlo[2]; lost.File != "lost.jpg" || lost.Title != "Lost" || len(lost.Depicts) != 2 {
		t.Errorf("the lost picture = %+v", lost)
	}
	// a picture named through an event is the event's, kept by whoever holds it
	death := person(t, corpus, "rossi-carlo").Events[0].Media
	if len(death) != 1 || death[0].File != "photos/grave.jpg" || len(death[0].Depicts) != 0 ||
		*death[0].Region != (model.Region{Left: 0, Top: 0, Right: .5, Bottom: .5}) {
		t.Errorf("the death's media = %+v", death)
	}
	// a document is referred to by its path into the record keeping it
	if anna := person(t, corpus, "bianchi-anna").Media; len(anna) != 2 ||
		anna[0].File != "../../people/rossi-carlo/documents/will.pdf" || anna[0].Title != "" || !anna[1].Private {
		t.Errorf("Anna's media = %+v", anna)
	}
	// a picture nothing refers to is kept by a note of its own
	if nobody := corpus.Notes["nobodys"]; nobody == nil || len(nobody.Media) != 1 || nobody.Media[0].File != "photos/group.jpg" {
		t.Errorf("the picture nothing refers to is kept by %+v among %v", nobody, corpus.NoteSlugs())
	}

	register := corpus.Sources["battesimi"]
	if len(register.Media) != 2 || register.Media[0].File != "scans/register.jpg" || register.Media[1].File != "scans/entry.jpg" {
		t.Errorf("the source's media = %+v", register.Media)
	}
	entry := register.Citations["n-112"]
	if len(entry.Images) != 1 || entry.Images[0].File != "scans/register.jpg" ||
		*entry.Images[0].Region != (model.Region{Left: .1, Top: .5, Right: .9, Bottom: .7}) {
		t.Errorf("the citation's images = %+v", entry.Images)
	}
	if images := register.Citations["n-113"].Images; len(images) != 1 || images[0].File != "scans/entry.jpg" || images[0].Region != nil {
		t.Errorf("the second citation's images = %+v", images)
	}

	// a citation on somebody in a picture is not something the picture can say of them
	for _, want := range []string{
		"<citationref> was kept as written",
		"a picture the export names was not found where Gramps would look for it, and is kept as the path it gave",
	} {
		if !reported(notes, want) {
			t.Errorf("did not report %q in %q", want, notes)
		}
	}
	if len(notes) != 2 {
		t.Errorf("reported %q", notes)
	}
}

// A document another record keeps is referred to by its path into that record, and a
// picture of an event nobody holds is kept by the note holding the event.
func TestAPictureElsewhereIsReferredToByItsPath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"letter.pdf", "ship.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	corpus, attachments, notes := convertFinding(t, `
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Emigration</type>
      <dateval val="1901"/>
      <objref hlink="_ship"/>
    </event>
  </events>
  <people>
    <person handle="_carlo" change="1" id="I0001">
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <objref hlink="_letter"/>
    </person>
  </people>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <page>p. 2</page>
      <objref hlink="_letter"><region corner1_x="0" corner1_y="50" corner2_x="100" corner2_y="100"/></objref>
      <sourceref hlink="_s1"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001"><stitle>Lettere</stitle></source>
  </sources>
  <objects>
    <object handle="_letter" change="1" id="O0001">
      <file src="letter.pdf" mime="application/pdf" description="A letter"/>
    </object>
    <object handle="_ship" change="1" id="O0002">
      <file src="ship.jpg" mime="image/jpeg" description="The ship"/>
    </object>
  </objects>`, importing.Beside(dir))

	// the source reads its citations before anybody else names a picture, so the scan
	// is the source's
	images := corpus.Sources["lettere"].Citations["p-2"].Images
	if len(images) != 1 || images[0].File != "scans/letter.pdf" ||
		*images[0].Region != (model.Region{Left: 0, Top: .5, Right: 1, Bottom: 1}) {
		t.Errorf("the citation's images = %+v", images)
	}
	if media := person(t, corpus, "rossi-carlo").Media; len(media) != 1 || media[0].File != "../../sources/lettere/scans/letter.pdf" {
		t.Errorf("Carlo's media = %+v", media)
	}
	var unheld *model.Note
	for _, note := range corpus.Notes {
		unheld = note
	}
	if len(corpus.Notes) != 1 || len(unheld.Events) != 1 || len(unheld.Events[0].Media) != 1 ||
		unheld.Events[0].Media[0].Title != "The ship" {
		t.Fatalf("the event nobody holds is in %+v", corpus.Notes)
	}
	if len(attachments) != 2 || attachments[1].Record != model.RefOf(unheld) {
		t.Errorf("attachments = %+v", attachments)
	}
	if len(notes) != 0 {
		t.Errorf("reported %q", notes)
	}
}
