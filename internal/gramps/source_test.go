package gramps

import (
	"reflect"
	"slices"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A Gramps citation is one entry cited from many places, so it is one anchor in its
// source, named after its page, and whatever cites it points at the anchor with the
// weight its confidence gives the evidence; a source is kept in every repository it
// names, and its notes, a citation's transcription among them, are note records.
func TestSourcesCitationsAndRepositoriesComeIn(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Birth</type>
      <dateval val="1834"/>
      <citationref hlink="_c1"/>
    </event>
  </events>
  <people>
    <person handle="_i1" change="1" id="I0001">
      <name>
        <first>Giovanni</first><surname>Rossi</surname>
        <citationref hlink="_c2"/>
      </name>
      <eventref hlink="_e1" role="Primary">
        <citationref hlink="_c3"/>
      </eventref>
      <citationref hlink="_c4"/>
      <citationref hlink="_c5"/>
    </person>
  </people>
  <citations>
    <citation handle="_c1" change="1" id="C0001">
      <dateval val="1834-03-14"/>
      <page>n. 112</page>
      <confidence>2</confidence>
      <noteref hlink="_n1"/>
      <sourceref hlink="_s1"/>
    </citation>
    <citation handle="_c2" change="1" id="C0002" priv="1">
      <page>n. 112</page>
      <confidence>3</confidence>
      <srcattribute type="Transcriber" value="AC"/>
      <sourceref hlink="_s1"/>
    </citation>
    <citation handle="_c3" change="1" id="C0003">
      <confidence>0</confidence>
      <sourceref hlink="_s1"/>
    </citation>
    <citation handle="_c4" change="1" id="C0004">
      <page>c. 3r</page>
      <confidence>4</confidence>
      <sourceref hlink="_s2"/>
    </citation>
    <citation handle="_c5" change="1" id="C0005">
      <confidence>2</confidence>
      <sourceref hlink="_gone"/>
    </citation>
  </citations>
  <sources>
    <source handle="_s1" change="1" id="S0001">
      <stitle>Parrocchia di San Lorenzo, registro dei battesimi 1820-1850</stitle>
      <sauthor>Parroco di San Lorenzo</sauthor>
      <spubinfo>manuscript</spubinfo>
      <sabbrev>SL bapt.</sabbrev>
      <noteref hlink="_n2"/>
      <srcattribute type="Form" value="register"/>
      <reporef hlink="_r1" callno="Reg. 4" medium="Book"/>
      <reporef hlink="_r2" medium="Microfilm"/>
    </source>
    <source handle="_s2" change="1" id="S0002" priv="1">
      <stitle>Stato delle anime</stitle>
      <reporef hlink="_r1" medium="Manuscript"/>
    </source>
  </sources>
  <repositories>
    <repository handle="_r1" change="1" id="R0001">
      <rname>Archivio Arcivescovile di Firenze</rname>
      <type>Archive</type>
      <address>
        <street>Piazza San Giovanni 3</street>
        <city>Firenze</city>
        <phone>055 123</phone>
      </address>
      <address><city>Prato</city><citationref hlink="_c4"/></address>
      <url href="https://example.org/archivio" type="Web Home"/>
      <url href="https://example.org/catalogo" type="Web Search"/>
      <noteref hlink="_n3"/>
    </repository>
    <repository handle="_r2" change="1" id="R0002">
      <rname>Family History Library</rname>
      <type>Library</type>
      <address priv="1"><city>Salt Lake City</city></address>
    </repository>
  </repositories>
  <notes>
    <note handle="_n1" change="1" id="N0001" type="Transcript"><text>Ioannes filius Petri</text></note>
    <note handle="_n2" change="1" id="N0002" type="Source Note"><text>Indexed by year.</text></note>
    <note handle="_n3" change="1" id="N0003" type="Repository Note"><text>Open on Tuesdays.</text></note>
    <note handle="_n4" change="1" id="N0004" type="General"><text>Nobody points here.</text></note>
  </notes>`)

	register := corpus.Sources["parrocchia-di-san-lorenzo-registro"]
	if register == nil {
		t.Fatalf("no register among %v", corpus.SourceSlugs())
	}
	if register.Author != "Parroco di San Lorenzo" || register.Publication != "manuscript" || register.ShortTitle != "SL bapt." ||
		register.Body != "[Indexed by year.](../../notes/indexed-by-year/README.md)" || idAt(register.IDs, "gramps") != "S0001" ||
		!reflect.DeepEqual(register.Attributes, []model.Attribute{{Type: "Form", Value: "register"}}) {
		t.Errorf("source = %+v", register)
	}
	// a private citation makes private the source it cites, and a book is what Gramps
	// says a source is kept as when nobody said anything
	if !register.Private {
		t.Error("the register cited privately is not private")
	}
	if want := []model.Holding{
		{Repository: "archivio-arcivescovile-di-firenze", CallNumber: "Reg. 4"},
		{Repository: "family-history-library", Medium: "Microfilm"},
	}; !slices.Equal(register.Repositories, want) {
		t.Errorf("kept at %+v", register.Repositories)
	}
	// two citations of one page are two anchors, and one with no page is named after its ID
	entry := register.Citations["n-112"]
	if entry.Page != "n. 112" || entry.Date.String() != "14 MAR 1834" || idAt(entry.IDs, "gramps") != "C0001" ||
		entry.Text != "[Ioannes filius Petri](../../notes/ioannes-filius-petri/README.md)" || entry.Note != "" {
		t.Errorf("citation = %+v", entry)
	}
	if second, ok := register.Citations["n-112-2"]; !ok {
		t.Errorf("anchors = %v", register.Citations)
	} else if !reflect.DeepEqual(second.Attributes, []model.Attribute{{Type: "Transcriber", Value: "AC"}}) {
		t.Errorf("the second citation = %+v", second)
	}
	if _, ok := register.Citations["c0003"]; !ok {
		t.Errorf("anchors = %v", register.Citations)
	}
	census := corpus.Sources["stato-delle-anime"]
	if census == nil || !census.Private || len(census.Repositories) != 1 || census.Repositories[0].Medium != "manuscript" {
		t.Fatalf("the private source = %+v", census)
	}
	for slug, body := range map[model.Slug]string{
		"ioannes-filius-petri": "Ioannes filius Petri", "indexed-by-year": "Indexed by year.",
		"open-on-tuesdays": "Open on Tuesdays.", "nobody-points-here": "Nobody points here.",
	} {
		if note := corpus.Notes[slug]; note == nil || note.Body != body {
			t.Errorf("the note %s is %+v among %v", slug, note, corpus.NoteSlugs())
		}
	}

	archive := corpus.Repositories["archivio-arcivescovile-di-firenze"]
	if archive == nil {
		t.Fatalf("no archive among %v", corpus.RepositorySlugs())
	}
	if want := []model.Address{
		{Street: "Piazza San Giovanni 3", City: "Firenze", Links: []model.Link{{URL: "tel:055 123"}}},
		{City: "Prato", Sources: []model.CitationRef{{Source: "stato-delle-anime", Anchor: "c-3r", Grade: model.Grade{Weight: '3'}}}},
	}; !reflect.DeepEqual(archive.Addresses, want) {
		t.Errorf("addresses = %+v", archive.Addresses)
	}
	if archive.Type != "archive" || len(archive.Links) != 2 || archive.Links[1].URL != "https://example.org/catalogo" ||
		archive.Body != "[Open on Tuesdays.](../../notes/open-on-tuesdays/README.md)" {
		t.Errorf("repository = %+v", archive)
	}

	if library := corpus.Repositories["family-history-library"]; !library.Private || len(library.Addresses) != 1 {
		t.Errorf("the library with a private address = %+v", library)
	}

	giovanni := person(t, corpus, "rossi-giovanni")
	refs := func(sources []model.CitationRef) []string {
		var out []string
		for _, s := range sources {
			out = append(out, s.String())
		}
		return out
	}
	if got := refs(giovanni.Name.Sources); !slices.Equal(got, []string{"parrocchia-di-san-lorenzo-registro#n-112-2 ---2"}) {
		t.Errorf("the name cites %v", got)
	}
	if got := refs(giovanni.Events[0].Sources); !slices.Equal(got, []string{
		"parrocchia-di-san-lorenzo-registro#n-112", "parrocchia-di-san-lorenzo-registro#c0003 ---0"}) {
		t.Errorf("the birth cites %v", got)
	}
	if got := refs(giovanni.Sources); !slices.Equal(got, []string{"stato-delle-anime#c-3r ---3"}) {
		t.Errorf("the person cites %v", got)
	}

	// a citation of a source that is not there is kept as written, with what cites it
	if want := []string{
		"<citation> was kept as written",
		"<citationref> was kept as written",
		"a citation of a source that is not in the file was not imported",
		"a private address made private the record it is the address of",
		"a private citation made private the source it cites",
	}; !slices.Equal(notes, want) {
		t.Errorf("reported %q", notes)
	}
	if note := corpus.Notes["citation-c0005"]; note == nil {
		t.Errorf("the citation of a source not there is not among %v", corpus.NoteSlugs())
	}
}

// A source and an archive named in a script no slug spells are filed under their Gramps
// IDs, the name kept as written.
func TestASourceNamedInAnotherScriptIsFiledUnderItsID(t *testing.T) {
	corpus, _ := convert(t, `
  <sources>
    <source handle="_s1" change="1" id="S0001">
      <stitle>سجل عقود الزواج، حلب</stitle>
      <reporef hlink="_r1" medium="Book"/>
    </source>
  </sources>
  <repositories>
    <repository handle="_r1" change="1" id="R0001">
      <rname>دار الوثائق القومية</rname>
      <type>Archive</type>
    </repository>
  </repositories>`)
	source := corpus.Sources["source-s0001"]
	if source == nil || source.Title != "سجل عقود الزواج، حلب" {
		t.Fatalf("the source is %+v among %v", source, corpus.SlugsOf(model.KindSource))
	}
	if archive := corpus.Repositories["repository-r0001"]; archive == nil || archive.Name != "دار الوثائق القومية" {
		t.Errorf("the archive is %+v among %v", archive, corpus.SlugsOf(model.KindRepository))
	}
	if len(source.Repositories) != 1 || source.Repositories[0].Repository != "repository-r0001" {
		t.Errorf("the source is kept in %+v", source.Repositories)
	}
}
