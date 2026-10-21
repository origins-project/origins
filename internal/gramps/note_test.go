package gramps

import (
	"html"
	"testing"

	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
)

// A note is markdown: bold, italic and links around the characters Gramps' ranges say,
// hugging the words rather than the spaces, closed at the end of every line, and nested
// where the ranges overlap.
func TestANoteIsWrittenAsMarkdown(t *testing.T) {
	for _, tc := range []struct {
		name, text, styles, want string
	}{
		{"plain", "Born at home.", ``, "Born at home."},
		{"bold", "Born at home.", `<style name="bold"><range start="0" end="4"/></style>`, "**Born** at home."},
		{"spaces outside the marks", "Born at home.", `<style name="italic"><range start="4" end="8"/></style>`,
			"Born *at* home."},
		{"two ranges of one style", "one two three", `<style name="bold"><range start="0" end="3"/><range start="8" end="13"/></style>`,
			"**one** two **three**"},
		{"nested", "one two three", `<style name="bold"><range start="0" end="13"/></style>` +
			`<style name="italic"><range start="4" end="7"/></style>`,
			"**one *two* three**"},
		{"across a line", "one\ntwo", `<style name="bold"><range start="0" end="7"/></style>`, "**one**\n**two**"},
		{"link", "see the register", `<style name="link" value="https://example.org/r"><range start="4" end="16"/></style>`,
			"see [the register](https://example.org/r)"},
		{"link with brackets", "Whig", `<style name="link" value="https://en.wikipedia.org/wiki/Whigs_(British)"><range start="0" end="4"/></style>`,
			"[Whig](<https://en.wikipedia.org/wiki/Whigs_(British)>)"},
		{"bold link", "the register", `<style name="link" value="https://example.org/r"><range start="0" end="12"/></style>` +
			`<style name="bold"><range start="4" end="12"/></style>`,
			"[the **register**](https://example.org/r)"},
		{"a link into the tree is its text", "his father", `<style name="link" value="gramps://Person/handle/abc"><range start="4" end="10"/></style>`,
			"his father"},
		{"underlined", "his father", `<style name="underline"><range start="4" end="10"/></style>`, "his <u>father</u>"},
		{"struck through", "his father", `<style name="strikethrough"><range start="4" end="10"/></style>`, "his ~~father~~"},
		{"superscript and subscript", "1st H2O", `<style name="superscript"><range start="1" end="3"/></style>` +
			`<style name="subscript"><range start="5" end="6"/></style>`, "1<sup>st</sup> H<sub>2</sub>O"},
		{"highlighted", "his father", `<style name="highlight" value="#ffff00"><range start="4" end="10"/></style>`,
			`his <span style="background-color: #ffff00">father</span>`},
		{"coloured", "his father", `<style name="fontcolor" value="#ff0000"><range start="4" end="10"/></style>`,
			`his <span style="color: #ff0000">father</span>`},
		{"in a face and a size", "his father", `<style name="fontface" value="Serif"><range start="4" end="10"/></style>` +
			`<style name="fontsize" value="14"><range start="4" end="10"/></style>`,
			`his <span style="font-family: Serif"><span style="font-size: 14pt">father</span></span>`},
		{"a size that is not one", "his father", `<style name="fontsize" value="large"><range start="4" end="10"/></style>`, "his father"},
		{"a range past the end", "abc", `<style name="bold"><range start="1" end="99"/></style>`, "a**bc**"},
		{"characters, not bytes", "Né à Paris", `<style name="bold"><range start="5" end="10"/></style>`, "Né à **Paris**"},
	} {
		c := &converter{notes: importing.NewNotes()}
		note := element(t, `<note handle="_n" change="1" type="General"><text>`+tc.text+`</text>`+tc.styles+`</note>`)
		if got := c.markdown(note); got != tc.want {
			t.Errorf("%s: markdown = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A preformatted note is kept as it was written, in a block markdown leaves alone, or
// where it is styled in a <pre> block, its styles as HTML.
func TestAPreformattedNoteIsKeptAsItWasWritten(t *testing.T) {
	c := &converter{notes: importing.NewNotes()}
	text := "<text>  name | age\n  Anna &lt; 30\n```</text>"
	plain := element(t, `<note handle="_n" change="1" type="General" format="1">`+text+`</note>`)
	if got, want := c.markdown(plain), "````\n  name | age\n  Anna < 30\n```\n````"; got != want {
		t.Errorf("markdown = %q, want %q", got, want)
	}
	styled := element(t, `<note handle="_n" change="1" type="General" format="1">`+text+
		`<style name="bold"><range start="0" end="4"/></style></note>`)
	if got, want := c.markdown(styled), "<pre><b>  na</b>me | age\n  Anna &lt; 30\n```</pre>"; got != want {
		t.Errorf("markdown = %q, want %q", got, want)
	}
}

// A note is written as markdown in the words its type and where it was made call for:
// labelled with its type where that says something the place it is in does not, a task
// to do as one, and HTML as a block of it.
func TestANoteSaysWhatKindOfNoteItIs(t *testing.T) {
	for _, tc := range []struct {
		name, kind, at, text, want string
	}{
		{"general", "General", "event", "Of a fever.", "Of a fever."},
		{"where it was made", "Event Note", "event", "Of a fever.", "Of a fever."},
		{"away from where it was made", "Person Note", "event", "Of a fever.", "*Person Note:* Of a fever."},
		{"of a kind of its own", "Hearsay", "person", "Of a fever.", "*Hearsay:* Of a fever."},
		{"research", "Research", "person", "Ask the parish.", "*Research:* Ask the parish."},
		{"to do", "To Do", "person", "Ask the parish.\nThen the comune.", "- [ ] Ask the parish.\n  Then the comune."},
		{"html", "Html code", "person", "<p>Hi</p>", "```html\n<p>Hi</p>\n```"},
		{"a label before a block", "Research", "person", "", "*Research:*\n\n````\n  a\n```\n````"},
	} {
		c := &converter{notes: importing.NewNotes(), noteAt: map[string][]string{"_n": {tc.at}}}
		format, text := "", tc.text
		if text == "" {
			format, text = ` format="1"`, "  a\n```"
		}
		note := element(t, `<note handle="_n" change="1" type="`+tc.kind+`"`+format+`><text>`+html.EscapeString(text)+`</text></note>`)
		if got := c.noteBody(note); got != tc.want {
			t.Errorf("%s: body = %q, want %q", tc.name, got, tc.want)
		}
	}
}

// A link Gramps makes into the tree leads to the file of the record it names in the
// corpus, a citation's to its anchor in its source, an event's to whoever holds it and a
// picture's to whoever keeps it.
func TestALinkIntoTheTreeLeadsToTheRecord(t *testing.T) {
	corpus, _ := convert(t, `
  <events>
    <event handle="_e1" change="1" id="E0001"><type>Death</type></event>
  </events>
  <people>
    <person handle="_i1" change="1" id="I0001">
      <name><first>Anna</first><surname>Rossi</surname></name>
      <eventref hlink="_e1" role="Primary"/>
      <objref hlink="_o1"/>
      <noteref hlink="_n1"/>
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
  <places>
    <placeobj handle="_p1" change="1" id="P0001" type="City"><pname value="Firenze"/></placeobj>
  </places>
  <objects>
    <object handle="_o1" change="1" id="O0001"><file src="a.jpg" mime="image/jpeg" description="A"/></object>
  </objects>
  <repositories>
    <repository handle="_r1" change="1" id="R0001"><rname>Archivio</rname><type>Archive</type></repository>
  </repositories>
  <notes>
    <note handle="_n1" change="1" id="N0001" type="Person Note"><text>Anna, Firenze, Morti, n. 4, Archivio, her death, a photo, this</text>`+
		`<style name="link" value="gramps://Person/handle/i1"><range start="0" end="4"/></style>`+
		`<style name="link" value="gramps://Place/handle/_p1"><range start="6" end="13"/></style>`+
		`<style name="link" value="gramps://Source/handle/_s1"><range start="15" end="20"/></style>`+
		`<style name="link" value="gramps://Citation/handle/_c1"><range start="22" end="26"/></style>`+
		`<style name="link" value="gramps://Repository/handle/_r1"><range start="28" end="36"/></style>`+
		`<style name="link" value="gramps://Event/handle/_e1"><range start="38" end="47"/></style>`+
		`<style name="link" value="gramps://Media/handle/_o1"><range start="49" end="56"/></style>`+
		`<style name="link" value="gramps://Note/handle/_n1"><range start="58" end="62"/></style>`+
		`</note>
  </notes>`)
	want := "[Anna](../../people/rossi-anna/README.md), [Firenze](../../places/firenze/README.md), " +
		"[Morti](../../sources/morti/README.md), [n. 4](../../sources/morti/README.md#n-4), " +
		"[Archivio](../../repositories/archivio/README.md), [her death](../../people/rossi-anna/README.md), " +
		"[a photo](../../people/rossi-anna/README.md), [this](../../notes/anna-firenze-morti-n-4-archivio/README.md)"
	for slug, note := range corpus.Notes {
		if note.Body != want {
			t.Errorf("the note %s is %q,\nwant %q", slug, note.Body, want)
		}
	}
	if len(corpus.Notes) != 1 {
		t.Errorf("notes are %v", corpus.NoteSlugs())
	}
}

// A note is a note record, linked from the body of a record, the note of an event, a
// name and an identifier, by the title its first line gives it.
func TestANoteGoesWhereTheModelHasText(t *testing.T) {
	corpus, notes := convert(t, `
  <events>
    <event handle="_e1" change="1" id="E0001">
      <type>Death</type>
      <description>at home</description>
      <noteref hlink="_n3"/>
    </event>
  </events>
  <people>
    <person handle="_i1" change="1" id="I0001">
      <name>
        <first>Anna</first><surname>Rossi</surname>
        <noteref hlink="_n4"/>
      </name>
      <eventref hlink="_e1" role="Primary">
        <noteref hlink="_n5"/>
      </eventref>
      <attribute type="Occupation" value="Sarta">
        <noteref hlink="_n6"/>
      </attribute>
      <attribute type="REFN" value="17">
        <noteref hlink="_n6"/>
      </attribute>
      <noteref hlink="_n1"/>
      <noteref hlink="_n2"/>
    </person>
    <person handle="_i2" change="1" id="I0002">
      <name><first>Carlo</first><surname>Rossi</surname></name>
      <noteref hlink="_private"/>
    </person>
  </people>
  <places>
    <placeobj handle="_p1" change="1" id="P0001" type="City">
      <pname value="Firenze"/>
      <noteref hlink="_n1"/>
    </placeobj>
  </places>
  <notes>
    <note handle="_n1" change="1" id="N0001" type="Person Note"><text>She sewed.</text></note>
    <note handle="_n2" change="1" id="N0002" type="Person Note"><text>Second paragraph.</text></note>
    <note handle="_n3" change="1" id="N0003" type="Event Note"><text>Of a fever.</text></note>
    <note handle="_n4" change="1" id="N0004" type="Name Note"><text>Spelt Ana once.</text></note>
    <note handle="_n5" change="1" id="N0005" type="Event Reference Note"><text>She was 80.</text></note>
    <note handle="_n6" change="1" id="N0006" type="Attribute Note"><text>For the gentry.</text></note>
    <note handle="_private" change="1" id="N0007" type="Person Note" priv="1"><text>Not for sharing.</text></note>
    <note handle="_orphan" change="1" id="N0008" type="General"><text>Nobody's.</text></note>
  </notes>`)

	link := func(title string, slug model.Slug) string {
		return "[" + title + "](../../notes/" + string(slug) + "/README.md)"
	}
	anna := person(t, corpus, "rossi-anna")
	if want := link("She sewed.", "she-sewed") + "\n\n" + link("Second paragraph.", "second-paragraph"); anna.Body != want || anna.Private {
		t.Errorf("Anna's body = %q, private %v", anna.Body, anna.Private)
	}
	if want := link("Of a fever.", "of-a-fever") + "\n\n" + link("She was 80.", "she-was-80"); anna.Events[0].Note != want || anna.Events[0].Value != "at home" {
		t.Errorf("the death's note = %q, its value %q", anna.Events[0].Note, anna.Events[0].Value)
	}
	if anna.Events[1].Note != link("For the gentry.", "for-the-gentry") {
		t.Errorf("the occupation's note = %q", anna.Events[1].Note)
	}
	// a private note is a private note record, and what links to it says nothing private
	carlo := person(t, corpus, "rossi-carlo")
	if carlo.Private || carlo.Body != link("Not for sharing.", "not-for-sharing") || !corpus.Notes["not-for-sharing"].Private {
		t.Errorf("Carlo = %+v", carlo)
	}
	if body := place(t, corpus, "firenze").Body; body != link("She sewed.", "she-sewed") {
		t.Errorf("the place's body = %q", body)
	}
	if anna.Name.Note != link("Spelt Ana once.", "spelt-ana-once") {
		t.Errorf("the name's note = %q", anna.Name.Note)
	}
	if refn := anna.IDs["refn"]; len(refn) != 1 || refn[0].Note != link("For the gentry.", "for-the-gentry") {
		t.Errorf("ids = %v", anna.IDs)
	}

	// every note is a record, with its text, its identifier and the type its place says
	for slug, body := range map[model.Slug]string{
		"she-sewed": "She sewed.", "of-a-fever": "Of a fever.", "she-was-80": "She was 80.",
		"spelt-ana-once": "Spelt Ana once.", "for-the-gentry": "For the gentry.", "nobodys": "Nobody's.",
	} {
		if note := corpus.Notes[slug]; note == nil || note.Body != body || len(note.IDs["gramps"]) != 1 {
			t.Errorf("the note %s is %+v among %v", slug, note, corpus.NoteSlugs())
		}
	}
	if len(notes) != 0 {
		t.Errorf("reported %q", notes)
	}
}
