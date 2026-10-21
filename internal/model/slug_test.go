package model

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

// the family slugs of the fixture corpus, which pin the sorted double-hyphen join
var fixtureFamilies = []string{
	"bianchi-maria--rossi-giovanni",
	"ferri-stefano--rossi-marco",
	"galli-ines--rossi-pietro",
	"landi-rosa--rossi-francesco",
	"marchi-paolo--rossi-claudia",
	"neri-caterina--rossi-antonio",
	"rossi-antonio--verdi-lucia",
	"rossi-bartolomeo--sarti-angela",
	"rossi-claudia--sanna-davide",
}

func TestParseSlugAccepts(t *testing.T) {
	accepted := []string{
		"rossi-giovanni",
		"rossi-giovanni-1902",
		"rossi-infante-1866",
		"borgo-san-lorenzo",
		"firenze",
		"censimento-1861-firenze",
		"1861",
	}
	for _, s := range append(accepted, fixtureFamilies...) {
		if _, err := ParseSlug(s); err != nil {
			t.Errorf("ParseSlug(%q): %v", s, err)
		}
	}
}

func TestParseSlugRejects(t *testing.T) {
	for _, s := range []string{
		"",
		"Rossi-Giovanni",
		"rossi giovanni",
		"rossi_giovanni",
		"rossì-giovanni",
		"-rossi",
		"rossi-",
		"--rossi",
		"rossi--",
		"rossi---giovanni",
		"rossi/giovanni",
		"rossi.giovanni",
	} {
		if got, err := ParseSlug(s); err == nil {
			t.Errorf("ParseSlug(%q) = %q, want an error", s, got)
		}
	}
}

func TestSlugify(t *testing.T) {
	for in, want := range map[string]Slug{
		"Rossi Giovanni":            "rossi-giovanni",
		"Borgo San Lorenzo":         "borgo-san-lorenzo",
		"  Rossi   Giovanni  ":      "rossi-giovanni",
		"Niccolò Macchiavelli":      "niccolo-macchiavelli",
		"Müller":                    "muller",
		"Straße":                    "strasse",
		"Ærø":                       "aero",
		"D'Angelo":                  "dangelo",
		"Dell’Acqua":                "dellacqua",
		"ʿĀʾisha al-Ḥalabī":         "aisha-al-halabi",
		"Censimento 1861 (Firenze)": "censimento-1861-firenze",
		"Saint-Étienne":             "saint-etienne",
	} {
		got, err := Slugify(in)
		if err != nil {
			t.Errorf("Slugify(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
		if _, err := ParseSlug(string(got)); err != nil {
			t.Errorf("Slugify(%q) produced %q, which is not a valid slug: %v", in, got, err)
		}
	}
}

func TestSlugifyRejectsWhatLeavesNothing(t *testing.T) {
	for _, in := range []string{"", "   ", "!!!", "¿?"} {
		if got, err := Slugify(in); err == nil {
			t.Errorf("Slugify(%q) = %q, want an error", in, got)
		}
	}
}

func TestAnOpenWordIsOneWordHoweverItIsWritten(t *testing.T) {
	long := "Served in the Royal Navy aboard HMS Victory as a powder monkey, and later as a gunner"
	same := [][2]string{
		{"Birth", "birth"},
		{"BIRTH", "birth"},
		{"Father's Age", "fathers-age"},
		{"Bürgerrecht", "burgerrecht"},
		{"marriage  licence", "Marriage-Licence"},
		{"兵役", "兵役"},
		{long, strings.ToUpper(long)},
		{"???", " ??? "},
	}
	for _, pair := range same {
		if !SameWord(pair[0], pair[1]) {
			t.Errorf("%q and %q are two words", pair[0], pair[1])
		}
	}
	different := [][2]string{
		{"兵役 1", "服役 1"},
		{long, long[:64]},
		{"???", "!!!"},
		{"birth", "births"},
		{"", "birth"},
	}
	for _, pair := range different {
		if SameWord(pair[0], pair[1]) {
			t.Errorf("%q and %q are one word", pair[0], pair[1])
		}
	}
}

func TestAWordInASCIIHasTheKeyFoldingGivesIt(t *testing.T) {
	alphabet := []byte("aZ9 '-_.\t")
	words := []string{"", "Father's Age", "  marriage  licence ", "D'Angelo", "???", "'", "a' b"}
	var grow func(word []byte)
	grow = func(word []byte) {
		words = append(words, string(word))
		if len(word) < 4 {
			for _, c := range alphabet {
				grow(append(word, c))
			}
		}
	}
	grow(nil)
	for _, word := range words {
		if fast, _ := asciiKey(word); fast != foldedKey(word) {
			t.Errorf("%q has the key %q, folded %q", word, fast, foldedKey(word))
		}
	}
	if _, ok := asciiKey("Bürgerrecht"); ok {
		t.Error("a word beyond ASCII was given a key without folding")
	}
}

func TestARelationIsKnownHoweverItIsWritten(t *testing.T) {
	for written, kin := range map[Relation]Relation{
		"Adoptive": Adoptive, "STEP": Step, "": Birth, "Sponsored": RelationUnknown,
	} {
		if got := written.Kin(); got != kin {
			t.Errorf("%q is %q, want %q", written, got, kin)
		}
	}
}

func TestAPersonIsFiledUnderTheSurnameFirst(t *testing.T) {
	for _, c := range []struct {
		name PersonName
		want Slug
	}{
		{PersonName{Given: "Giovanni", Surname: SurnameOf("Rossi")}, "rossi-giovanni"},
		{PersonName{Given: "Giovanni Battista", Call: "Giovanni", Surname: SurnameOf("Rossi")}, "rossi-giovanni-battista"},
		{PersonName{Prefix: "Don", Given: "Pietro", SurnamePrefix: "de'", Surname: SurnameOf("Medici"), Suffix: "sen."}, "medici-pietro"},
		{PersonName{Surname: SurnameOf("Rossi")}, "rossi"},
		{PersonName{Nickname: "Gianni"}, "gianni"},
		{PersonName{Text: "Gio. Rossi"}, "gio-rossi"},
	} {
		if got, err := c.name.Slug(); err != nil || got != c.want {
			t.Errorf("%+v is filed under %q, %v, want %q", c.name, got, err, c.want)
		}
	}
}

func TestFamilySlugDoesNotDependOnOrder(t *testing.T) {
	forward, err := FamilySlug(PartnersOf("rossi-giovanni", "bianchi-maria"))
	if err != nil {
		t.Fatal(err)
	}
	back, err := FamilySlug(PartnersOf("bianchi-maria", "rossi-giovanni"))
	if err != nil {
		t.Fatal(err)
	}
	if forward != back || forward != "bianchi-maria--rossi-giovanni" {
		t.Errorf("got %q and %q", forward, back)
	}
}

// A person's families begin with their name, so the words come after the people, and a
// partner known only by name is one nobody knows as far as the name goes.
func TestAFamilySlugPutsThePeopleFirst(t *testing.T) {
	stockdale := Partner{Name: PersonName{Text: "Mr Stockdale"}}
	for _, c := range []struct {
		partners []Partner
		want     Slug
	}{
		{PartnersOf(Unknown, "rossi-giovanni"), "rossi-giovanni--unknown"},
		{PartnersOf(None, "moretti-giulia"), "moretti-giulia--none"},
		{PartnersOf(Unknown, Unknown), "unknown--unknown"},
		{[]Partner{stockdale, {Person: "landi-rosa"}}, "landi-rosa--unknown"},
		{PartnersOf("rossi-giovanni", "bianchi-maria", "verdi-lucia"),
			"bianchi-maria--rossi-giovanni--verdi-lucia"},
	} {
		if got, err := FamilySlug(c.partners); err != nil || got != c.want {
			t.Errorf("%+v names %q, %v, want %q", c.partners, got, err, c.want)
		}
	}
	if _, err := FamilySlug(nil); err == nil {
		t.Error("a family of nobody was given a name")
	}
}

func TestTheFixtureFamiliesAreNamedByTheirPartners(t *testing.T) {
	for _, u := range fixtureFamilies {
		a, b, _ := strings.Cut(u, "--")
		if got, err := FamilySlug(PartnersOf(Slug(a), Slug(b))); err != nil || string(got) != u {
			t.Errorf("%q and %q name %q, %v", a, b, got, err)
		}
	}
}

// A discriminator is read back off a slug exactly as Discriminate put it on, the base
// shortened to make room or not, so that a family renamed keeps the words telling it
// apart from its partners' other family.
func TestADiscriminatorIsReadBackAndCarried(t *testing.T) {
	long := Slug(strings.Repeat("giovanni-", 6) + "rossi")
	for _, base := range []Slug{"bianchi-maria--rossi-giovanni", long} {
		proposals, err := Discriminate(base, []string{"1858", "Borgo San Lorenzo"}, func(s Slug) bool { return s == base })
		if err != nil {
			t.Fatal(err)
		}
		for _, proposal := range proposals {
			suffix, ok := proposal.Discriminator(base)
			if !ok || discriminated(base, suffix) != proposal {
				t.Errorf("%q read back as %q off %q, %v", proposal, suffix, base, ok)
			}
		}
		if suffix, ok := base.Discriminator(base); !ok || suffix != "" {
			t.Errorf("%q is its own base, read as %q, %v", base, suffix, ok)
		}
	}
	for _, other := range []Slug{"bianchi-maria--rossi-giovann", "bianchi-maria--rossi", "landi-rosa"} {
		if suffix, ok := other.Discriminator("bianchi-maria--rossi-giovanni"); ok {
			t.Errorf("%q was read as discriminated by %q", other, suffix)
		}
	}

	for _, c := range []struct{ slug, want Slug }{
		{"rossi-giovanni--unknown-1858", "bianchi-maria--rossi-giovanni-1858"},
		{"rossi-giovanni--unknown", "bianchi-maria--rossi-giovanni"},
		{"somebody-else", "bianchi-maria--rossi-giovanni"},
	} {
		if got := c.slug.Rebase("rossi-giovanni--unknown", "bianchi-maria--rossi-giovanni"); got != c.want {
			t.Errorf("%q rebased is %q, want %q", c.slug, got, c.want)
		}
	}
}

func TestDiscriminate(t *testing.T) {
	taken := func(used ...Slug) func(Slug) bool {
		return func(s Slug) bool { return slices.Contains(used, s) }
	}

	got, err := Discriminate("rossi-giovanni",
		[]string{"1902", "Firenze", "Senior"},
		taken("rossi-giovanni", "rossi-giovanni-1902"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Slug{"rossi-giovanni-firenze", "rossi-giovanni-senior", "rossi-giovanni-2"}
	if !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}

	got, err = Discriminate("rossi-giovanni", nil, taken("rossi-giovanni"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []Slug{"rossi-giovanni-2"}) {
		t.Errorf("with no hints, got %q", got)
	}

	got, err = Discriminate("rossi-giovanni", nil,
		taken("rossi-giovanni", "rossi-giovanni-2", "rossi-giovanni-3"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, []Slug{"rossi-giovanni-4"}) {
		t.Errorf("the numeric fallback should skip what is taken, got %q", got)
	}

	if _, err := Discriminate("Rossi Giovanni", nil, taken()); err == nil {
		t.Error("Discriminate should reject a base that is not a slug")
	}
}

func TestDiscriminateProposalsAreValidSlugs(t *testing.T) {
	proposals, err := Discriminate("bianchi-maria--rossi-giovanni",
		[]string{"1858", "Borgo San Lorenzo"},
		func(Slug) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range proposals {
		if _, err := ParseSlug(string(p)); err != nil {
			t.Errorf("proposed %q: %v", p, err)
		}
	}
}

// An archduchess recorded under eighteen given names is a real record, and the slug it
// produces has to be a file name — twice over, since two of them join to name a family.
func TestASlugIsShortEnoughToBeAFileName(t *testing.T) {
	long := "Carolina Maria Immakulata Josepha Ferdinanda Therese Leopoldine " +
		"Antoinette Franziska Isabella Luise Januaria Christine Benedikta"
	slug, err := Slugify("Österreich-Toscana " + long)
	if err != nil {
		t.Fatal(err)
	}
	if len(slug) > maxSlug {
		t.Errorf("Slugify returned %d characters: %q", len(slug), slug)
	}
	if _, err := ParseSlug(string(slug)); err != nil {
		t.Errorf("%q: %v", slug, err)
	}
	if !strings.HasPrefix(string(slug), "osterreich-toscana-carolina") {
		t.Errorf("the front of the name should survive, got %q", slug)
	}

	family, err := FamilySlug(PartnersOf(slug, slug+"a"))
	if err != nil {
		t.Fatal(err)
	}
	if len(family) > maxSlug {
		t.Errorf("a family of two names is %d characters: %q", len(family), family)
	}
}

// The discriminator is the whole reason two slugs differ, so it is the base that gives
// way when the two together are too long.
func TestDiscriminatingWithinTheLengthASlugMayHave(t *testing.T) {
	base := Slug(strings.Repeat("giovanni-", 7) + "rossi")
	proposals, err := Discriminate(base, []string{"1834", "Borgo San Lorenzo"},
		func(Slug) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range proposals {
		if len(p) > maxSlug {
			t.Errorf("proposed %d characters: %q", len(p), p)
		}
		if _, err := ParseSlug(string(p)); err != nil {
			t.Errorf("proposed %q: %v", p, err)
		}
	}
	if !strings.HasSuffix(string(proposals[0]), "-1834") ||
		!strings.HasSuffix(string(proposals[1]), "-borgo-san-lorenzo") {
		t.Errorf("the discriminators did not survive: %q", proposals)
	}
}

// Shortening a family slug must not leave the double hyphen that joins its halves
// hanging off the end, which would be a slug no parser accepts.
func TestShorteningNeverEndsOnAHyphen(t *testing.T) {
	for n := 1; n <= 12; n++ {
		got := shorten("bianchi--rossi", n)
		if _, err := ParseSlug(got); err != nil {
			t.Errorf("shorten(%d) = %q: %v", n, got, err)
		}
	}
}

// A slug is asked for wherever a name is compared, often from goroutines side by side,
// so two names being slugified at once must each come out as themselves.
func TestSlugifyingSideBySideGivesEachNameItsOwnSlug(t *testing.T) {
	names := map[string]Slug{
		"Cambiasù Nicolò":      "cambiasu-nicolo",
		"Łukasz Żółć":          "lukasz-zolc",
		"Ōtomo no Yakamochi":   "otomo-no-yakamochi",
		"Élisabeth d'Autriche": "elisabeth-dautriche",
	}
	var wrong sync.Map
	var group sync.WaitGroup
	for name, want := range names {
		for range 8 {
			group.Go(func() {
				for range 500 {
					if got, err := Slugify(name); err != nil || got != want {
						wrong.Store(name, got)
					}
				}
			})
		}
	}
	group.Wait()
	wrong.Range(func(name, got any) bool {
		t.Errorf("%s slugified as %q, want %q", name, got, names[name.(string)])
		return true
	})
}
