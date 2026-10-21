package check

import (
	"slices"
	"strings"
	"testing"

	"github.com/origins-project/origins/internal/model"
)

func TestDanglingReferences(t *testing.T) {
	f := fires(t, "dangling-child", build(&model.Person{Slug: "rossi-antonio"}, childOf("rossi-antonio", "rossi-giovanni")))
	if f.Path != "children[0]" || !strings.Contains(f.Message, "rossi-giovanni") {
		t.Errorf("got %+v", f)
	}

	fires(t, "dangling-partner", build(&model.Family{
		Slug:     "bianchi-maria--rossi-giovanni",
		Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
	}))

	f = fires(t, "dangling-place", build(&model.Person{
		Slug:   "rossi-giovanni",
		Events: []model.Event{{Type: "birth", Place: "borgo-san-lorenzo"}},
	}))
	if f.Path != "events[0].place" {
		t.Errorf("path = %q", f.Path)
	}

	fires(t, "dangling-place", build(&model.Place{
		Slug:   "firenze",
		Within: []model.PlaceRef{{Place: "toscana"}},
	}))

	f = fires(t, "dangling-source", build(&model.Person{
		Slug: "rossi-giovanni",
		Events: []model.Event{{
			Type:    "birth",
			Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}},
		}},
	}))
	if f.Path != "events[0].sources[0]" {
		t.Errorf("path = %q", f.Path)
	}

	f = fires(t, "dangling-named", build(&model.Person{
		Slug: "rossi-giovanni",
		Events: []model.Event{{Type: "baptism", With: []model.Named{
			{Name: model.PersonName{Text: "Maria Neri"}, Role: "godmother"},
			{Person: "verdi-giacomo", Role: "godfather"},
		}}},
	}))
	if f.Path != "events[0].with[1].person" || !strings.Contains(f.Message, "verdi-giacomo") {
		t.Errorf("got %+v", f)
	}

	// somebody the record says was there and does not name is nobody's reference
	silent(t, "dangling-named", build(&model.Person{
		Slug:   "rossi-giovanni",
		Events: []model.Event{{Type: "baptism", With: []model.Named{{Person: model.Unknown, Role: "witness"}}}},
	}))

	f = fires(t, "dangling-family", build(&model.Person{
		Slug:   "rossi-giovanni",
		Events: []model.Event{{Type: "lds-sealing-to-parents", Family: "bianchi-maria--rossi-antonio"}},
	}))
	if f.Path != "events[0].family" || !strings.Contains(f.Message, "bianchi-maria--rossi-antonio") {
		t.Errorf("got %+v", f)
	}
	sealed := build(
		&model.Person{Slug: "rossi-antonio"},
		&model.Person{Slug: "bianchi-maria"},
		&model.Person{Slug: "rossi-giovanni",
			Events: []model.Event{{Type: "lds-sealing-to-parents", Family: "bianchi-maria--rossi-antonio"}}},
		&model.Family{
			Slug:     "bianchi-maria--rossi-antonio",
			Partners: model.PartnersOf("bianchi-maria", "rossi-antonio"),
			Events:   []model.Event{{Type: "lds-sealing-to-spouse"}},
		},
	)
	silent(t, "dangling-family", sealed)
	silent(t, "family-event-family", sealed)
	sealed.Families["bianchi-maria--rossi-antonio"].Events[0].Family = "bianchi-maria--rossi-antonio"
	if f := fires(t, "family-event-family", sealed); f.Path != "events[0].family" {
		t.Errorf("got %+v", f)
	}

	fires(t, "dangling-citation", build(
		&model.Person{
			Slug: "rossi-giovanni",
			Events: []model.Event{{
				Type:    "birth",
				Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-999"}},
			}},
		},
		&model.Source{
			Slug:      "parish-san-lorenzo-baptisms",
			Citations: map[string]model.Citation{"1834-042": {Text: "..."}},
		},
	))
}

// A claim that two records are one is research until it is settled, and one that no
// merge could settle is wrong.
func TestARecordSaidToBeAnother(t *testing.T) {
	corpus := build(
		&model.Person{Slug: "rossi-giovanni", SameAs: []model.SameAs{
			{ID: "rossi-giovanni-1834"},
			{ID: "rossi-giovanni-1835", Note: "The diocesan copy."},
			{Name: "Giovanni Rossi of Boston"},
		}},
		&model.Person{Slug: "rossi-giovanni-1834"},
		&model.Person{Slug: "rossi-giovanni-1835"},
	)
	var got []string
	for _, f := range Run(corpus) {
		if strings.Contains(f.Rule, "same-as") {
			got = append(got, string(f.Severity)+" "+f.Rule+" "+f.Path)
		}
	}
	want := []string{
		"warning same-as same-as[0]",
		"warning same-as same-as[1].id",
		"warning same-as same-as[2].name",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	f := fires(t, "dangling-same-as", build(&model.Place{Slug: "vicchio", SameAs: []model.SameAs{{ID: "vicchio-di-mugello"}}}))
	if f.Severity != Error || f.Path != "same-as[0]" || !strings.Contains(f.Message, "place") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "same-as-another-kind", build(
		&model.Place{Slug: "vicchio", SameAs: []model.SameAs{{ID: "vicchio-1861", Note: "n"}}},
		&model.Note{Slug: "vicchio-1861"}))
	if f.Severity != Error || f.Path != "same-as[0].id" {
		t.Errorf("got %+v", f)
	}
	if f = fires(t, "same-as-itself", build(&model.Source{Slug: "register", SameAs: []model.SameAs{{ID: "register"}}})); f.Severity != Error {
		t.Errorf("got %+v", f)
	}
}

// A link in prose is followed into the record it names, and on to the file beside it or
// the citation in it; a link leading anywhere else is nobody's reference.
func TestALinkInProseLeadsSomewhere(t *testing.T) {
	maria := func(body string, note string) *model.Person {
		return &model.Person{Slug: "bianchi-maria", Body: body,
			Events: []model.Event{{Type: "birth", Note: note, Sources: []model.CitationRef{{Source: "parish", Anchor: "1"}}}}}
	}
	parish := &model.Source{Slug: "parish", Files: []string{"scans/1.jpg"},
		Citations: map[string]model.Citation{"1": {Text: "…"}}}
	giovanni := &model.Person{Slug: "rossi-giovanni", Files: []string{"photos/1858.jpg"}}

	for _, c := range []struct {
		body, note string
		path, says string
	}{
		{"Wife of [Giovanni](../../people/rossi-giovanni-1834/README.md).", "",
			"body", `no person "rossi-giovanni-1834"`},
		{"", "Entered as [Giovanni](../../places/rossi-giovanni/README.md).",
			"events[0].note", `no place "rossi-giovanni"`},
		{"![him](../rossi-giovanni/photos/1859.jpg)", "",
			"body", `no file "photos/1859.jpg" beside person/rossi-giovanni`},
		{"![me](photos/1858.jpg)", "",
			"body", `no file "photos/1858.jpg" beside person/bianchi-maria`},
		{"", "[The entry](../../sources/parish/README.md#2)",
			"events[0].note", `source "parish" has no citation "2"`},
	} {
		f := fires(t, "dangling-link", build(maria(c.body, c.note), parish, giovanni))
		if f.Entity != "person/bianchi-maria" || f.Path != c.path || !strings.Contains(f.Message, c.says) {
			t.Errorf("%q%q: got %+v", c.body, c.note, f)
		}
	}

	silent(t, "dangling-link", build(maria("Wife of [Giovanni](../../people/rossi-giovanni/README.md#birth),\n"+
		"![him](../rossi-giovanni/photos/1858.jpg), [see](#below), [the notes](../../../notes.md),\n"+
		"[the site](https://example.org/people/nobody/README.md) and `[code](../../people/nobody/README.md)`.\n",
		"[The entry](../../sources/parish/README.md#1) and [its scan](../../sources/parish/scans/1.jpg)."),
		parish, giovanni))
}

func TestDanglingMainPerson(t *testing.T) {
	c := build()
	c.Project.MainPerson = "rossi-giovanni"
	fires(t, "dangling-main-person", c)

	c.People["rossi-giovanni"] = &model.Person{Slug: "rossi-giovanni"}
	silent(t, "dangling-main-person", c)
}

// A bookmark leads to a record of the kind its directory says.
func TestDanglingBookmark(t *testing.T) {
	c := build(&model.Person{Slug: "rossi-giovanni"}, &model.Place{Slug: "firenze"})
	c.Project.Bookmarks = []model.EntityRef{
		{Kind: model.KindPerson, Slug: "rossi-giovanni"},
		{Kind: model.KindPerson, Slug: "firenze"},
		{Kind: model.KindSource, Slug: "census-1871"},
	}
	var got []string
	for _, f := range Run(c) {
		if f.Rule == "dangling-bookmark" {
			got = append(got, f.Path)
		}
	}
	if want := []string{"project.bookmarks[1]", "project.bookmarks[2]"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestFamilyShape(t *testing.T) {
	partners := []any{
		&model.Person{Slug: "bianchi-maria"},
		&model.Person{Slug: "rossi-giovanni"},
	}

	fires(t, "family-needs-partners", build(append(partners,
		&model.Family{Slug: "rossi-giovanni", Partners: model.PartnersOf("rossi-giovanni")},
	)...))

	f := fires(t, "family-slug-mismatch", build(append(partners,
		&model.Family{
			Slug:     "rossi-giovanni--bianchi-maria",
			Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
		},
	)...))
	if !strings.Contains(f.Message, "bianchi-maria--rossi-giovanni") {
		t.Errorf("the message should name the slug the partners imply: %q", f.Message)
	}

	for _, named := range []*model.Family{
		{Slug: "bianchi-maria--rossi-giovanni", Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni")},
		{Slug: "bianchi-maria--rossi-giovanni-1858", Partners: model.PartnersOf("rossi-giovanni", "bianchi-maria")},
		{Slug: "rossi-giovanni--unknown", Partners: model.PartnersOf(model.Unknown, "rossi-giovanni")},
		{Slug: "rossi-giovanni--unknown", Partners: []model.Partner{
			{Person: "rossi-giovanni"}, {Name: model.PersonName{Text: "Mr Stockdale"}}}},
		{Slug: "unknown--unknown-landi", Partners: model.PartnersOf(model.Unknown, model.Unknown)},
		{Slug: "bianchi-maria--none", Partners: model.PartnersOf(model.None, "bianchi-maria")},
	} {
		c := build(append(partners, named)...)
		silent(t, "family-slug-mismatch", c)
		silent(t, "family-none", c)
		silent(t, "dangling-partner", c)
	}

	for _, alone := range []*model.Family{
		{Slug: "none--unknown", Partners: model.PartnersOf(model.None, model.Unknown)},
		{Slug: "bianchi-maria--rossi-giovanni--none",
			Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni", model.None)},
		{Slug: "none--none", Partners: model.PartnersOf(model.None, model.None)},
	} {
		fires(t, "family-none", build(append(partners, alone)...))
	}
}

// Neither word for a partner without a record is ever a record's name, or a partner
// written as one would be two things at once.
func TestAReservedWordNamesNoRecord(t *testing.T) {
	fires(t, "reserved-slug", build(&model.Person{Slug: model.Unknown}))
	fires(t, "reserved-slug", build(&model.Place{Slug: model.None}))
	silent(t, "reserved-slug", build(&model.Person{Slug: "unknown-1858"}))
}

func TestSourceAttachments(t *testing.T) {
	scans := func(files ...string) map[string]model.Citation {
		var images []model.Media
		for _, file := range files {
			images = append(images, model.Media{File: file})
		}
		return map[string]model.Citation{"1834-042": {Images: images}}
	}
	f := fires(t, "missing-image", build(&model.Source{
		Slug:      "parish-san-lorenzo-baptisms",
		Citations: scans("scans/1834-042.jpg", "scans/1834-043.jpg"),
		Files:     []string{"scans/1834-042.jpg"},
	}))
	if f.Path != "citations.1834-042.images[1]" || !strings.Contains(f.Message, "1834-043.jpg") {
		t.Errorf("got %+v", f)
	}

	silent(t, "missing-image", build(&model.Source{
		Slug:      "parish-san-lorenzo-baptisms",
		Citations: scans("scans/1834-042.jpg"),
		Files:     []string{"scans/1834-042.jpg"},
	}))

	// a scan still to be found is research, as any picture is
	abroad := build(&model.Source{
		Slug:      "parish-san-lorenzo-baptisms",
		Citations: scans(`C:\scans\1834-042.jpg`),
	})
	silent(t, "missing-image", abroad)
	if f := fires(t, "missing-media", abroad); f.Severity != Warning || f.Path != "citations.1834-042.images[0]" {
		t.Errorf("got %+v", f)
	}

	fires(t, "untranscribed-scan", build(&model.Source{
		Slug:      "parish-san-lorenzo-baptisms",
		Citations: scans("scans/1834-042.jpg"),
		Files:     []string{"scans/1834-042.jpg", "scans/1835-001.jpg"},
	}))
}

// Coordinates a record gave as words are kept, and are a place no map can show yet.
func TestCoordinatesAsWordsAreAPlaceNoMapShows(t *testing.T) {
	f := fires(t, "coordinates-as-words", build(&model.Place{
		Slug: "firenze", Coordinates: &model.Coordinates{Wording: "N43.77.5, E11.25.3"},
	}))
	if f.Severity != Warning || f.Path != "coordinates" {
		t.Errorf("got %+v", f)
	}
	silent(t, "coordinates-as-words", build(&model.Place{
		Slug: "firenze", Coordinates: &model.Coordinates{Latitude: 43.7696, Longitude: 11.2558},
	}))
}

// A region is four fractions of the picture it marks, so one reaching past an edge of
// it, or with its edges the wrong way round, marks nothing that is there.
func TestARegionLiesWithinItsPicture(t *testing.T) {
	marked := func(region model.Region) *model.Corpus {
		return build(&model.Source{
			Slug:  "parish-san-lorenzo-baptisms",
			Files: []string{"scans/1834-042.jpg"},
			Citations: map[string]model.Citation{"1834-042": {Images: []model.Media{
				{File: "scans/1834-042.jpg", Region: &region},
			}}},
		})
	}
	for _, region := range []model.Region{
		{Left: -0.1, Top: 0.2, Right: 0.5, Bottom: 0.6},
		{Left: 0.1, Top: 0.2, Right: 1.5, Bottom: 0.6},
		{Left: 0.5, Top: 0.2, Right: 0.1, Bottom: 0.6},
		{Left: 0.1, Top: 0.6, Right: 0.5, Bottom: 0.2},
		{Left: 0.1, Top: 0.2, Right: 0.1, Bottom: 0.6},
	} {
		f := fires(t, "impossible-region", marked(region))
		if f.Path != "citations.1834-042.images[0].region" {
			t.Errorf("%+v: reported at %q", region, f.Path)
		}
	}
	silent(t, "impossible-region", marked(model.Region{Left: 0, Top: 0, Right: 1, Bottom: 1}))
	silent(t, "impossible-region", marked(model.Region{Left: 0.08, Top: 0.41, Right: 0.93, Bottom: 0.62}))
}

func TestASourceIsKeptInARepositoryThatIsThere(t *testing.T) {
	kept := &model.Source{Slug: "parish-san-lorenzo-baptisms",
		Repositories: []model.Holding{{Repository: "archivio-diocesano-di-firenze"}}}
	f := fires(t, "dangling-repository", build(kept))
	if f.Entity != "source/parish-san-lorenzo-baptisms" || f.Path != "repositories[0]" {
		t.Errorf("got %+v", f)
	}
	silent(t, "dangling-repository", build(kept, &model.Repository{Slug: "archivio-diocesano-di-firenze"}))

	filmed := &model.Source{Slug: "parish-san-lorenzo-baptisms", Repositories: []model.Holding{
		{CallNumber: "B 4/1834"},
		{Repository: "familysearch", CallNumber: "Film 1903581", Medium: "microfilm"},
	}}
	f = fires(t, "dangling-repository", build(filmed))
	if f.Path != "repositories[1].repository" {
		t.Errorf("got %+v", f)
	}
	silent(t, "dangling-repository", build(filmed, &model.Repository{Slug: "familysearch"}))
}

// What a source covers and who a citation names are references like any other.
func TestASourceCoversAndNamesWhatIsThere(t *testing.T) {
	source := &model.Source{Slug: "parish-san-lorenzo-baptisms",
		Covers: []model.Coverage{{Events: []string{"baptism"}}, {Place: "borgo-san-lorenzo"}},
		Citations: map[string]model.Citation{"1834-042": {With: []model.Named{
			{Person: model.Unknown, Role: "godmother"},
			{Person: "rossi-giovanni", Role: "child"},
		}, Images: []model.Media{{File: "scans/1834-042.jpg", Place: "firenze",
			Region: &model.Region{Left: 0.5, Top: 0, Right: 0.2, Bottom: 1}}}}},
		Files: []string{"scans/1834-042.jpg"},
	}
	f := fires(t, "dangling-named", build(source))
	if f.Entity != "source/parish-san-lorenzo-baptisms" || f.Path != "citations.1834-042.with[1].person" {
		t.Errorf("got %+v", f)
	}
	var places []string
	for _, f := range Run(build(source)) {
		if f.Rule == "dangling-place" {
			places = append(places, f.Path)
		}
	}
	if want := []string{"citations.1834-042.images[0].place", "covers[1].place"}; !slices.Equal(places, want) {
		t.Errorf("places reported at %q", places)
	}
	if f := fires(t, "impossible-region", build(source)); f.Path != "citations.1834-042.images[0].region" {
		t.Errorf("got %+v", f)
	}
	there := build(source, &model.Person{Slug: "rossi-giovanni"},
		&model.Place{Slug: "borgo-san-lorenzo"}, &model.Place{Slug: "firenze"})
	silent(t, "dangling-named", there)
	silent(t, "dangling-place", there)
}

func TestPersonAttachments(t *testing.T) {
	f := fires(t, "missing-portrait", build(&model.Person{
		Slug:     "rossi-giovanni",
		Portrait: model.Image{File: "photos/1890-portrait.jpg"},
		Files:    []string{"photos/1888-portrait.jpg"},
	}))
	if f.Path != "portrait" || !strings.Contains(f.Message, "1890-portrait.jpg") {
		t.Errorf("got %+v", f)
	}

	silent(t, "missing-portrait", build(&model.Person{
		Slug:     "rossi-giovanni",
		Portrait: model.Image{File: "photos/1890-portrait.jpg"},
		Files:    []string{"photos/1890-portrait.jpg"},
	}))

	// the field is optional, and a person who declares nothing declares nothing wrong
	silent(t, "missing-portrait", build(&model.Person{
		Slug:  "rossi-giovanni",
		Files: []string{"photos/1890-portrait.jpg"},
	}))

	f = fires(t, "impossible-region", build(&model.Person{
		Slug:     "rossi-giovanni",
		Portrait: model.Image{File: "photos/group.jpg", Region: &model.Region{Left: 0.6, Top: 0.1, Right: 0.4, Bottom: 0.5}},
		Files:    []string{"photos/group.jpg"},
	}))
	if f.Path != "portrait.region" {
		t.Errorf("got %+v", f)
	}
}

// What a record says its files are has to be about files it has, once each, and the
// people and places it names have to be there.
func TestMediaIsAboutTheRecordsOwnFiles(t *testing.T) {
	family := func(media ...model.Media) *model.Family {
		return &model.Family{
			Slug:     "bianchi-maria--rossi-giovanni",
			Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
			Media:    media,
			Files:    []string{"photos/1858-wedding.jpg"},
		}
	}
	partners := []any{&model.Person{Slug: "bianchi-maria"}, &model.Person{Slug: "rossi-giovanni"}}
	with := func(entities ...any) *model.Corpus { return build(append(entities, partners...)...) }

	f := fires(t, "missing-media", with(family(model.Media{File: "photos/1858-wedding.jpg"}, model.Media{File: "photos/1859.jpg"})))
	if f.Entity != "family/bianchi-maria--rossi-giovanni" || f.Path != "media[1].file" || !strings.Contains(f.Message, "1859.jpg") ||
		f.Severity != Warning {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "duplicate-media", with(family(model.Media{File: "photos/1858-wedding.jpg"},
		model.Media{File: "photos/1858-wedding.jpg", Title: "The wedding"})))
	if f.Path != "media[1].file" || !strings.Contains(f.Message, "media[0]") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "missing-media", build(&model.Repository{Slug: "archivio", Media: []model.Media{{File: "facade.jpg"}}}))
	if f.Entity != "repository/archivio" {
		t.Errorf("got %+v", f)
	}

	wedding := model.Media{File: "photos/1858-wedding.jpg", Place: "firenze", Depicts: []model.Depicted{
		{Person: "rossi-giovanni"}, {Person: "bianchi-mario", Region: &model.Region{Left: 0.1, Top: 0.2, Right: 0.3, Bottom: 0.4}},
	}}
	f = fires(t, "dangling-depicted", with(family(wedding)))
	if f.Path != "media[0].depicts[1]" || !strings.Contains(f.Message, "bianchi-mario") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "dangling-place", with(family(wedding)))
	if f.Path != "media[0].place" {
		t.Errorf("got %+v", f)
	}
	wedding.Sources = []model.CitationRef{{Source: "album", Anchor: "p3"}}
	f = fires(t, "dangling-source", with(family(wedding)))
	if f.Path != "media[0].sources[0]" {
		t.Errorf("got %+v", f)
	}

	wedding.Depicts[1].Region = &model.Region{Left: 0.1, Top: 0.2, Right: 1.3, Bottom: 0.4}
	f = fires(t, "impossible-region", with(family(wedding)))
	if f.Path != "media[0].depicts[1].region" {
		t.Errorf("got %+v", f)
	}

	wedding = model.Media{File: "photos/1858-wedding.jpg", Depicts: []model.Depicted{{Person: "rossi-giovanni"}}}
	for _, rule := range []string{"missing-media", "duplicate-media", "dangling-depicted", "impossible-region"} {
		silent(t, rule, with(family(wedding)))
	}
}

// A picture is kept by one record and referred to by its path from any other, and one
// whose file is not to hand is an entry still.
func TestAPictureIsReferredToByItsPath(t *testing.T) {
	giovanni := func(media ...model.Media) *model.Person {
		return &model.Person{Slug: "rossi-giovanni", Media: media, Files: []string{"photos/1875.jpg"}}
	}
	family := func(file string) *model.Family {
		return &model.Family{
			Slug:     "bianchi-maria--rossi-giovanni",
			Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"),
			Media:    []model.Media{{File: file, Note: "In the album."}},
		}
	}
	maria := &model.Person{Slug: "bianchi-maria"}
	kept := giovanni(model.Media{File: "photos/1875.jpg", Type: "photo"})
	referring := "../../people/rossi-giovanni/photos/1875.jpg"
	for _, rule := range []string{"missing-media", "unlisted-media", "dangling-media", "media-outside", "duplicate-media"} {
		silent(t, rule, build(kept, maria, family(referring)))
	}

	f := fires(t, "unlisted-media", build(giovanni(), maria, family(referring)))
	if f.Entity != "family/bianchi-maria--rossi-giovanni" || f.Path != "media[0].file" || f.Severity != Error ||
		!strings.Contains(f.Message, "person/rossi-giovanni") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "dangling-media", build(kept, maria, family("../../people/rossi-gianni/photos/1875.jpg")))
	if !strings.Contains(f.Message, "rossi-gianni") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "media-outside", build(kept, maria, family("../../album.jpg")))
	if f.Path != "media[0].file" || f.Severity != Error {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "missing-media", build(kept, maria, family("C:/Foto/1875.jpg")))
	if f.Severity != Warning || !strings.Contains(f.Message, "still to be found") {
		t.Errorf("got %+v", f)
	}

	// an event's pictures are a list of their own, kept beside the record holding it
	burial := func(media ...model.Media) *model.Person {
		p := giovanni()
		p.Events = []model.Event{{Type: "burial", Media: media}}
		p.Media = []model.Media{{File: "photos/1875.jpg"}}
		return p
	}
	stone := model.Media{File: "photos/1875.jpg", Type: "tombstone"}
	silent(t, "duplicate-media", build(burial(stone)))
	f = fires(t, "duplicate-media", build(burial(stone, model.Media{File: "photos/./1875.jpg", Note: "again"})))
	if f.Path != "events[0].media[1].file" || !strings.Contains(f.Message, "events[0].media[0]") {
		t.Errorf("got %+v", f)
	}
	f = fires(t, "missing-media", build(burial(model.Media{File: "photos/stone.jpg"})))
	if f.Path != "events[0].media[0].file" {
		t.Errorf("got %+v", f)
	}
	stone.Place = "prato"
	f = fires(t, "dangling-place", build(burial(stone)))
	if f.Path != "events[0].media[0].place" {
		t.Errorf("got %+v", f)
	}
}

func TestSurnameSlug(t *testing.T) {
	fires(t, "surname-slug-mismatch", build(&model.Person{
		Slug: "giovanni-rossi", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")},
	}))
	for _, slug := range []model.Slug{"rossi-giovanni", "rossi-giovanni-1902", "rossi"} {
		silent(t, "surname-slug-mismatch", build(&model.Person{
			Slug: slug, Name: model.PersonName{Surname: model.SurnameOf("Rossi")},
		}))
	}
	// a surname that only differs by accent should still group
	silent(t, "surname-slug-mismatch", build(&model.Person{
		Slug: "cambiasu-mario", Name: model.PersonName{Surname: model.SurnameOf("Cambiasù")},
	}))
}

// A name's surnames say one thing: a list takes no surname-prefix beside it, each
// surname having its own, and one surname at most is the one it is filed under.
func TestSurnameLists(t *testing.T) {
	double := model.Surnames{{Name: "Galli"}, {Name: "Cerreto", Prefix: "da"}}
	f := fires(t, "surname-prefix-beside-list", build(&model.Person{
		Slug: "galli-ines", AlsoKnownAs: []model.PersonName{{Given: "Ines", SurnamePrefix: "de", Surname: double}},
	}))
	if f.Severity != Error || f.Path != "also-known-as[0].surname-prefix" {
		t.Errorf("got %+v", f)
	}
	fires(t, "surname-prefix-beside-list", build(&model.Family{
		Slug: "galli-ines", Partners: []model.Partner{{Person: model.Unknown}, {Name: model.PersonName{
			SurnamePrefix: "de", Surname: model.Surnames{{Name: "Galli", Origin: "patrilineal"}},
		}}},
	}))
	silent(t, "surname-prefix-beside-list", build(&model.Person{
		Slug: "medici-pietro", Name: model.PersonName{SurnamePrefix: "de'", Surname: model.SurnameOf("Medici")},
	}))

	both := model.Surnames{{Name: "Galli", Primary: true}, {Name: "Cerreto", Primary: true}}
	f = fires(t, "two-primary-surnames", build(&model.Person{
		Slug: "galli-ines", Events: []model.Event{{Type: "baptism", With: []model.Named{{
			Name: model.PersonName{Surname: both}, Role: "godmother",
		}}}},
	}))
	if f.Severity != Error || f.Path != "events[0].with[0].name.surname[1].primary" {
		t.Errorf("got %+v", f)
	}
	both[1].Primary = false
	silent(t, "two-primary-surnames", build(&model.Person{Slug: "galli-ines", Name: model.PersonName{Surname: both}}))
}

func TestUnsourcedEvent(t *testing.T) {
	fires(t, "unsourced-event", build(&model.Person{
		Slug: "rossi-giovanni", Events: []model.Event{born("1834")},
	}))
	silent(t, "unsourced-event", build(
		&model.Person{Slug: "rossi-giovanni", Events: []model.Event{{
			Type:    "birth",
			Sources: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1834-042"}},
		}}},
		&model.Source{
			Slug:      "parish-san-lorenzo-baptisms",
			Citations: map[string]model.Citation{"1834-042": {Text: "..."}},
		},
	))
}

func TestCircularAncestry(t *testing.T) {
	a, b, c, d := &model.Person{Slug: "a"}, &model.Person{Slug: "b"}, &model.Person{Slug: "c"}, &model.Person{Slug: "d"}
	f := fires(t, "circular-ancestry", build(a, b, childOf("b", "a"), childOf("a", "b")))
	if f.Entity != "family/a--unknown" || f.Path != "children[0]" {
		t.Errorf("got %+v", f)
	}
	fires(t, "circular-ancestry", build(a, childOf("a", "a")))
	fires(t, "circular-ancestry", build(a, b, c, childOf("b", "a"), childOf("c", "b"), childOf("a", "c")))
	silent(t, "circular-ancestry", build(a, b, c, childOf("b", "a"), childOf("c", "b")))
	// a disproven link is no parentage, and so no cycle
	disproven := childOf("a", "b")
	disproven.Children[0].Status = model.Disproven
	silent(t, "circular-ancestry", build(a, b, childOf("b", "a"), disproven))

	// Two lines of descent that meet again further up. Cousins marrying is ordinary in
	// any tree that goes back far enough, and it is the case a cycle test most easily
	// mistakes for a cycle.
	silent(t, "circular-ancestry", build(a, b, c, d,
		&model.Family{Slug: "b--c", Partners: model.PartnersOf("b", "c"), Children: model.ChildrenOf("a")},
		childOf("d", "b", "c"),
	))

	// A cycle two people share, reached from someone standing outside it.
	fires(t, "circular-ancestry", build(a, b, c, childOf("b", "a"), childOf("c", "b"), childOf("b", "c")))
}

func TestFamilyChildren(t *testing.T) {
	family := func(partners []model.Partner, children ...model.Child) *model.Corpus {
		return build(&model.Person{Slug: "conti-teresa"}, &model.Person{Slug: "rossi-pietro"}, &model.Person{Slug: "rossi-elena"},
			&model.Family{Slug: "x", Partners: partners, Children: children})
	}
	couple := model.PartnersOf("conti-teresa", "rossi-pietro")
	alone := model.PartnersOf("conti-teresa", model.None)
	word := func(r model.Relation) model.Relationship { return model.Relationship{Word: r} }
	keyed := func(pairs ...model.PartnerRelation) model.Relationship { return model.Relationship{Partners: pairs} }

	fires(t, "relation-not-a-partner", family(couple,
		model.Child{ID: "rossi-elena", Relation: keyed(model.PartnerRelation{Partner: "galli-ines", Relation: model.Step})}))
	silent(t, "relation-not-a-partner", family(couple,
		model.Child{ID: "rossi-elena", Relation: keyed(model.PartnerRelation{Partner: "rossi-pietro", Relation: model.Step})}))

	fires(t, "child-listed-twice", family(couple, model.ChildrenOf("rossi-elena", "rossi-elena")...))
	silent(t, "child-listed-twice", family(couple, model.ChildrenOf(model.Unknown, model.Unknown)...))
	silent(t, "child-listed-twice", family(couple,
		model.Child{Name: model.PersonName{Text: "Gemma"}}, model.Child{Name: model.PersonName{Text: "Gemma"}}))

	fires(t, "child-is-partner", family(couple, model.ChildrenOf("rossi-pietro")...))

	fires(t, "child-of-nobody", family(couple, model.Child{ID: "rossi-elena", Relation: word(model.NoRelation)}))
	fires(t, "child-of-nobody", family(alone, model.Child{ID: "rossi-elena", Relation: word(model.NoRelation)}))
	// the wife's son from before the marriage, never adopted
	silent(t, "child-of-nobody", family(couple, model.Child{ID: "rossi-elena",
		Relation: keyed(model.PartnerRelation{Partner: "rossi-pietro", Relation: model.NoRelation})}))

	fires(t, "none-with-birth-child", family(alone, model.ChildrenOf("rossi-elena")...))
	fires(t, "none-with-birth-child", family(alone, model.Child{ID: model.Unknown}))
	silent(t, "none-with-birth-child", family(alone, model.Child{ID: "rossi-elena", Relation: word(model.Adoptive)}))
	silent(t, "none-with-birth-child", family(alone, model.Child{ID: "rossi-elena", Relation: word("godchild")}))
	silent(t, "none-with-birth-child", family(couple, model.ChildrenOf("rossi-elena")...))

	f := fires(t, "disproven-link", family(couple, model.Child{ID: "conti-teresa", Relation: word(model.Step)},
		model.Child{ID: "rossi-elena", Status: model.Disproven}))
	if f.Severity != Warning || f.Path != "children[1].status" {
		t.Errorf("got %+v", f)
	}
	silent(t, "disproven-link", family(couple, model.Child{ID: "rossi-elena", Status: model.Challenged}))
}

// Somebody unknown in two families counts twice, since nothing says they were the same
// person; the mother listed in both counts once.
func TestBirthParents(t *testing.T) {
	people := []any{&model.Person{Slug: "bianchi-maria"}, &model.Person{Slug: "rossi-giovanni"}, &model.Person{Slug: "rossi-anna"}}
	both := &model.Family{Slug: "bianchi-maria--rossi-giovanni",
		Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"), Children: model.ChildrenOf("rossi-anna")}
	with := func(families ...any) *model.Corpus { return build(append(slices.Clone(people), families...)...) }

	f := fires(t, "too-many-birth-parents", with(both, childOf("bianchi-maria", "rossi-anna")))
	if f.Entity != "person/rossi-anna" {
		t.Errorf("got %+v", f)
	}
	silent(t, "too-many-birth-parents", with(both,
		&model.Family{Slug: "bianchi-maria--rossi-giovanni-1860",
			Partners: model.PartnersOf("bianchi-maria", "rossi-giovanni"), Children: model.ChildrenOf("rossi-anna")}))
	fires(t, "too-many-birth-parents", with(childOf("bianchi-maria", "rossi-anna"), childOf("rossi-giovanni", "rossi-anna")))

	adopted := childOf("rossi-giovanni")
	adopted.Children = []model.Child{{ID: "rossi-anna", Relation: model.Relationship{Word: model.Adoptive}}}
	silent(t, "too-many-birth-parents", with(childOf("bianchi-maria", "rossi-anna"), adopted))
	disproven := childOf("rossi-giovanni")
	disproven.Children = []model.Child{{ID: "rossi-anna", Status: model.Disproven}}
	silent(t, "too-many-birth-parents", with(both, disproven))
}

func TestDuplicateCandidates(t *testing.T) {
	// no date tells them apart
	fires(t, "duplicate-person", build(
		&model.Person{Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}},
		&model.Person{Slug: "rossi-giovanni-2", Name: model.PersonName{Given: "Giovanni Battista", Surname: model.SurnameOf("Rossi")}},
	))

	// the fixture's grandfather and grandson, named for each other in the usual way
	silent(t, "duplicate-person", build(
		&model.Person{
			Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni Battista", Surname: model.SurnameOf("Rossi")},
			Events: []model.Event{born("12 MAR 1834")},
		},
		&model.Person{
			Slug: "rossi-giovanni-1902", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")},
			Events: []model.Event{born("3 NOV 1902")},
		},
	))

	silent(t, "duplicate-person", build(
		&model.Person{Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}},
		&model.Person{Slug: "rossi-giuseppe", Name: model.PersonName{Given: "Giuseppe", Surname: model.SurnameOf("Rossi")}},
	))
}

func TestDeathBeforeBirth(t *testing.T) {
	fires(t, "death-before-birth", build(&model.Person{
		Slug: "rossi-giovanni", Events: []model.Event{born("1834"), died("1830")},
	}))
	// a birth and a death in the same year cannot be ordered, and nor should they be
	silent(t, "death-before-birth", build(&model.Person{
		Slug: "rossi-giovanni", Events: []model.Event{born("12 MAR 1834"), died("1834")},
	}))
	silent(t, "death-before-birth", build(&model.Person{
		Slug: "rossi-giovanni", Events: []model.Event{born("12 MAY 1866"), died("12 MAY 1866")},
	}))
	// a type is one word however it is written
	written := born("1834")
	written.Type = "Birth"
	fires(t, "death-before-birth", build(&model.Person{
		Slug: "rossi-giovanni", Events: []model.Event{written, died("1830")},
	}))
}

func TestParentAges(t *testing.T) {
	family := func(parentEvents, childEvents []model.Event, sex model.Sex) *model.Corpus {
		return build(
			&model.Person{Slug: "rossi-anna", Events: childEvents},
			&model.Person{Slug: "rossi-giovanni", Sex: model.StatedSex{Value: sex}, Events: parentEvents},
			childOf("rossi-giovanni", "rossi-anna"),
		)
	}

	fires(t, "parent-too-young", family(
		[]model.Event{born("1850")}, []model.Event{born("1858")}, model.Female))
	silent(t, "parent-too-young", family(
		[]model.Event{born("1850")}, []model.Event{born("1875")}, model.Female))

	// a range wide enough to allow an ordinary age says nothing
	silent(t, "parent-too-young", family(
		[]model.Event{born("BET 1830 AND 1860")}, []model.Event{born("1858")}, model.Female))

	fires(t, "parent-too-old", family(
		[]model.Event{born("1800")}, []model.Event{born("1860")}, model.Female))
	silent(t, "parent-too-old", family(
		[]model.Event{born("1800")}, []model.Event{born("1860")}, model.Male))
	// an unrecorded sex takes the looser bound rather than guessing
	silent(t, "parent-too-old", family(
		[]model.Event{born("1800")}, []model.Event{born("1860")}, model.SexUnknown))

	fires(t, "child-before-parent", family(
		[]model.Event{born("1860")}, []model.Event{born("1834")}, model.Male))
}

func TestPosthumousChild(t *testing.T) {
	family := func(parentDeath, childBirth string) *model.Corpus {
		return build(
			&model.Person{Slug: "rossi-anna", Events: []model.Event{born(childBirth)}},
			childOf("rossi-giovanni", "rossi-anna"),
			&model.Person{
				Slug:   "rossi-giovanni",
				Events: []model.Event{born("1834"), died(parentDeath)},
			},
		)
	}
	fires(t, "posthumous-child", family("1860", "1863"))

	// a child born months after its father died is ordinary
	silent(t, "posthumous-child", family("12 MAR 1860", "3 NOV 1860"))
	silent(t, "posthumous-child", family("1860", "1861"))
}

func TestFamilyEventOrder(t *testing.T) {
	event := func(kind, date string) model.Event { return model.Event{Type: kind, Date: on(date)} }
	family := func(events ...model.Event) *model.Corpus {
		return build(
			&model.Person{Slug: "marchi-paolo", Events: []model.Event{born("1945")}},
			&model.Person{Slug: "rossi-claudia", Events: []model.Event{born("1948")}},
			&model.Family{
				Slug:     "marchi-paolo--rossi-claudia",
				Partners: model.PartnersOf("marchi-paolo", "rossi-claudia"),
				Events:   events,
			},
		)
	}
	married := event("marriage", "4 SEP 1971")

	for _, before := range []string{"engagement", "marriage-banns", "marriage-licence"} {
		f := fires(t, "family-event-order", family(married, event(before, "1972")))
		if f.Path != "events[1].date" || !strings.Contains(f.Message, before) {
			t.Errorf("%s: got %+v", before, f)
		}
		silent(t, "family-event-order", family(married, event(before, "AUG 1971")))
	}
	for _, after := range []string{"separation", "divorce-filed", "divorce", "annulment"} {
		f := fires(t, "family-event-order", family(event(after, "1970"), married))
		if f.Path != "events[0].date" || !strings.Contains(f.Message, after) {
			t.Errorf("%s: got %+v", after, f)
		}
		silent(t, "family-event-order", family(married, event(after, "1983")))
	}
	fires(t, "family-event-order", family(married, event("divorce-filed", "1983"), event("divorce", "1982")))
	silent(t, "family-event-order", family(married, event("divorce-filed", "1982"), event("divorce", "1983")))

	// a contract or a settlement may be made on either side of a wedding
	silent(t, "family-event-order", family(event("marriage-contract", "1975"), married))
	silent(t, "family-event-order", family(event("marriage-settlement", "1965"), married))

	// two dates are out of order only where Compare can say so
	silent(t, "family-event-order", family(event("engagement", "1971"), married))
	silent(t, "family-event-order", family(event("divorce", "FEB"), married))
	silent(t, "family-event-order", family(model.Event{Type: "divorce"}, married))
	silent(t, "family-event-order", family(event("separation", "BET 1968 AND 1975"), married))
	silent(t, "family-event-order", family(married, event("residence", "1950")))

	// the fixture's divorced family
	silent(t, "family-event-order", family(
		event("marriage", "4 SEP 1971"), event("divorce", "16 FEB 1983")))
}

func TestFamilyEndedAfterDeath(t *testing.T) {
	family := func(kind, date, death string) *model.Corpus {
		return build(
			&model.Person{Slug: "marchi-paolo", Events: []model.Event{born("1945"), died(death)}},
			&model.Person{Slug: "rossi-claudia", Events: []model.Event{born("1948")}},
			&model.Family{
				Slug:     "marchi-paolo--rossi-claudia",
				Partners: model.PartnersOf("marchi-paolo", "rossi-claudia"),
				Events: []model.Event{
					{Type: "marriage", Date: on("4 SEP 1971")},
					{Type: kind, Date: on(date)},
				},
			},
		)
	}

	for _, kind := range []string{"separation", "divorce"} {
		f := fires(t, "family-ended-after-death", family(kind, "1983", "1980"))
		if f.Path != "events[1].date" || !strings.Contains(f.Message, "marchi-paolo") {
			t.Errorf("%s: got %+v", kind, f)
		}
		silent(t, "family-ended-after-death", family(kind, "1983", "1990"))
		silent(t, "family-ended-after-death", family(kind, "1983", "1983"))
	}

	// an annulment may be sought of a marriage that has already ended
	silent(t, "family-ended-after-death", family("annulment", "1983", "1980"))
}

// What did not happen is held against what did: an event certainly within its date, or
// at any time where it has none, contradicts it, and one that may be within it is to be
// looked at.
func TestWhatDidNotHappen(t *testing.T) {
	woman := func(not model.NonEvent, events ...model.Event) *model.Corpus {
		return build(&model.Person{Slug: "neri-teresa", Events: events, Not: []model.NonEvent{not}})
	}
	never := model.NonEvent{Type: "marriage"}
	abroad := model.NonEvent{Type: "naturalisation", Date: on("FROM 1880 TO 1902")}

	f := fires(t, "ruled-out", woman(never, model.Event{Type: "Marriage", Date: on("1870")}))
	if f.Entity != "person/neri-teresa" || f.Path != "not[0]" || !strings.Contains(f.Message, "events[0]") {
		t.Errorf("got %+v", f)
	}
	fires(t, "ruled-out", woman(never, model.Event{Type: "marriage"}))
	fires(t, "ruled-out", woman(abroad, model.Event{Type: "naturalisation", Date: on("12 MAY 1891")}))
	f = fires(t, "possibly-ruled-out", woman(abroad, model.Event{Type: "naturalisation", Date: on("AFT 1895")}))
	if f.Severity != Warning {
		t.Errorf("got %+v", f)
	}
	fires(t, "possibly-ruled-out", woman(abroad, model.Event{Type: "naturalisation"}))
	quiet := woman(abroad, model.Event{Type: "naturalisation", Date: on("1905")}, model.Event{Type: "marriage", Date: on("1885")})
	silent(t, "ruled-out", quiet)
	silent(t, "possibly-ruled-out", quiet)

	// a marriage is the family's, and a person's `not` is held against it
	f = fires(t, "ruled-out", build(
		&model.Person{Slug: "neri-teresa", Not: []model.NonEvent{never}},
		&model.Person{Slug: "rossi-pietro"},
		&model.Family{
			Slug:     "neri-teresa--rossi-pietro",
			Partners: model.PartnersOf("neri-teresa", "rossi-pietro"),
			Events:   []model.Event{{Type: "marriage", Date: on("1880")}},
		},
	))
	if f.Entity != "person/neri-teresa" || !strings.Contains(f.Message, "family/neri-teresa--rossi-pietro events[0]") {
		t.Errorf("got %+v", f)
	}
	fires(t, "ruled-out", build(
		&model.Person{Slug: "neri-teresa"},
		&model.Person{Slug: "rossi-pietro"},
		&model.Family{
			Slug:     "neri-teresa--rossi-pietro",
			Partners: model.PartnersOf("neri-teresa", "rossi-pietro"),
			Events:   []model.Event{{Type: "divorce", Date: on("1890")}},
			Not:      []model.NonEvent{{Type: "divorce"}},
		},
	))
}

func TestHypothesisEvidence(t *testing.T) {
	edits := []model.Edit{model.Delete{
		Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-giovanni"},
	}}

	fires(t, "hypothesis-changes-nothing", build(&model.Hypothesis{
		Slug: "a-thought", Status: model.Open,
	}))

	fires(t, "unevidenced-hypothesis", build(&model.Hypothesis{
		Slug: "a-hunch", Status: model.Open, Edits: edits,
	}))

	// an unevidenced machine proposal is the failure mode this domain punishes hardest
	f := fires(t, "unevidenced-proposal", build(&model.Hypothesis{
		Slug: "a-machine-hunch", Status: model.Open, Edits: edits, ProposedBy: "some-agent",
	}))
	if !strings.Contains(f.Message, "some-agent") {
		t.Errorf("the message should name the proposer: %q", f.Message)
	}

	silent(t, "unevidenced-hypothesis", build(
		&model.Hypothesis{
			Slug: "a-case", Status: model.Open, Edits: edits,
			Contradicts: []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1842-007"}},
		},
		&model.Source{
			Slug:      "parish-san-lorenzo-baptisms",
			Citations: map[string]model.Citation{"1842-007": {Text: "..."}},
		},
	))
}

func TestFindingsAreOrderedTheSameWayEveryRun(t *testing.T) {
	c := build(
		&model.Person{Slug: "rossi-anna", Events: []model.Event{born("1858")}},
		&model.Person{Slug: "rossi-giovanni", Events: []model.Event{born("1834")}},
		childOf("nobody", "rossi-anna", "rossi-giovanni"),
		&model.Place{Slug: "firenze", Within: []model.PlaceRef{{Place: "toscana"}}},
	)
	first := Run(c)
	if len(first) == 0 {
		t.Fatal("expected findings to order")
	}
	for range 20 {
		again := Run(c)
		if len(again) != len(first) {
			t.Fatalf("got %d findings then %d", len(first), len(again))
		}
		for i := range first {
			if again[i] != first[i] {
				t.Fatalf("finding %d moved: %+v then %+v", i, first[i], again[i])
			}
		}
	}
}

func TestErrorsCount(t *testing.T) {
	findings := []Finding{
		{Severity: Error}, {Severity: Warning}, {Severity: Error},
	}
	if got := Errors(findings); got != 2 {
		t.Errorf("Errors = %d", got)
	}
	if got := Errors(nil); got != 0 {
		t.Errorf("Errors(nil) = %d", got)
	}
}

// A citation is checked wherever in a record it is written, not only on an event.
func TestEveryCitationIsChecked(t *testing.T) {
	cited := []model.CitationRef{{Source: "a-lost-register", Anchor: "1"}}
	corpus := build(
		&model.Person{
			Slug:        "rossi-giovanni",
			Name:        model.PersonName{Surname: model.SurnameOf("Rossi"), Sources: cited},
			AlsoKnownAs: []model.PersonName{{Surname: model.SurnameOf("Rosi"), Sources: cited}},
			Sex:         model.StatedSex{Value: model.Male, Sources: cited},
			Events: []model.Event{{Type: "baptism", With: []model.Named{
				{Name: model.PersonName{Given: "Maria", Surname: model.SurnameOf("Neri"), Sources: cited}, Role: "godmother",
					Attributes: []model.Attribute{{Type: "occupation", Sources: cited}}, Sources: cited},
			}, Address: model.Address{Lines: "Via de' Benci 8", Sources: cited},
				Age:        model.StatedAge{Value: "0", Sources: cited},
				Cause:      model.StatedCause{Value: "cholera", Sources: cited},
				Attributes: []model.Attribute{{Type: "classification", Sources: cited}}}},
			Not: []model.NonEvent{{Type: "marriage", Sources: cited}},
			Media: []model.Media{{File: "portrait.jpg", Attributes: []model.Attribute{{Type: "medium", Sources: cited}},
				Depicts: []model.Depicted{{Person: "rossi-giovanni", Attributes: []model.Attribute{{Type: "pose", Sources: cited}}}}}},
			Files:   []string{"portrait.jpg"},
			Sources: cited,
			IDs:     model.IDs{"familysearch": {{ID: "XXXX-YYY"}, {ID: "XXXX-XXX", Sources: cited}}},
		},
		&model.Person{Slug: "rossi-antonio"},
		&model.Family{
			Slug:     "rossi-antonio--rossi-giovanni",
			Partners: model.PartnersOf("rossi-antonio", "rossi-giovanni"),
			Children: []model.Child{{Name: model.PersonName{Given: "Gemma", Sources: cited}, Sources: cited}},
			Not:      []model.NonEvent{{Type: "divorce", Sources: cited}},
			Sources:  cited,
		},
		&model.Place{
			Slug:        "firenze",
			Name:        model.PlaceName{Text: "Firenze", Sources: cited},
			AlsoKnownAs: []model.PlaceName{{Text: "Florence", Lang: "en", Sources: cited}},
			Within:      []model.PlaceRef{{Place: "toscana", Sources: cited}},
			Addresses:   []model.Address{{Lines: "Palazzo Vecchio", Sources: cited}},
			Sources:     cited,
		},
		&model.Place{Slug: "toscana"},
		&model.Source{Slug: "register", IDs: model.IDs{"wikidata": {{ID: "Q1", Sources: cited}}},
			Attributes: []model.Attribute{{Type: "volumes", Value: "3", Sources: cited}},
			Citations:  map[string]model.Citation{"1": {Attributes: []model.Attribute{{Type: "folio", Sources: cited}}}}},
		&model.Repository{Slug: "archivio", Addresses: []model.Address{{City: "Firenze", Sources: cited}},
			SameAs: []model.SameAs{{Name: "Archivio di Stato", Sources: cited}}},
		&model.Note{Slug: "vicchio-1861", Events: []model.Event{{Type: "marriage", Sources: cited}},
			Sources: cited, IDs: model.IDs{"gramps": {{ID: "N0001", Sources: cited}}}},
	)
	var got []string
	for _, f := range Run(corpus) {
		if f.Rule == "dangling-source" {
			got = append(got, f.Entity+" "+f.Path)
		}
	}
	slices.Sort(got)
	want := []string{
		"family/rossi-antonio--rossi-giovanni children[0].name.sources[0]",
		"family/rossi-antonio--rossi-giovanni children[0].sources[0]",
		"family/rossi-antonio--rossi-giovanni not[0].sources[0]",
		"family/rossi-antonio--rossi-giovanni sources[0]",
		"note/vicchio-1861 events[0].sources[0]",
		"note/vicchio-1861 ids.gramps[0].sources[0]",
		"note/vicchio-1861 sources[0]",
		"person/rossi-giovanni also-known-as[0].sources[0]",
		"person/rossi-giovanni events[0].address.sources[0]",
		"person/rossi-giovanni events[0].age.sources[0]",
		"person/rossi-giovanni events[0].attributes[0].sources[0]",
		"person/rossi-giovanni events[0].cause.sources[0]",
		"person/rossi-giovanni events[0].with[0].attributes[0].sources[0]",
		"person/rossi-giovanni events[0].with[0].name.sources[0]",
		"person/rossi-giovanni events[0].with[0].sources[0]",
		"person/rossi-giovanni ids.familysearch[1].sources[0]",
		"person/rossi-giovanni media[0].attributes[0].sources[0]",
		"person/rossi-giovanni media[0].depicts[0].attributes[0].sources[0]",
		"person/rossi-giovanni name.sources[0]",
		"person/rossi-giovanni not[0].sources[0]",
		"person/rossi-giovanni sex.sources[0]",
		"person/rossi-giovanni sources[0]",
		"place/firenze addresses[0].sources[0]",
		"place/firenze also-known-as[0].sources[0]",
		"place/firenze name.sources[0]",
		"place/firenze sources[0]",
		"place/firenze within[0].sources[0]",
		"repository/archivio addresses[0].sources[0]",
		"repository/archivio same-as[0].sources[0]",
		"source/register attributes[0].sources[0]",
		"source/register citations.1.attributes[0].sources[0]",
		"source/register ids.wikidata[0].sources[0]",
	}
	if !slices.Equal(got, want) {
		t.Errorf("dangling citations found at\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}
