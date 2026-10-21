package model

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// Sex records what the sources say, because it is evidence rather than identity. The
// zero value is unrecorded, which is an ordinary state and not a gap to be filled.
// Neither is GEDCOM 7's X, for a person the records set down as neither male nor
// female, which is a different statement from the records saying nothing.
type Sex int

// The sexes a record can state, SexUnknown being none stated.
const (
	SexUnknown Sex = iota
	Male
	Female
	Neither
)

var sexLetter = [...]string{SexUnknown: "U", Male: "M", Female: "F", Neither: "X"}

func (s Sex) String() string {
	if int(s) >= len(sexLetter) {
		return "?"
	}
	return sexLetter[s]
}

func (s Sex) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

func (s *Sex) UnmarshalText(b []byte) error {
	parsed, err := ParseSex(string(b))
	if err != nil {
		return err
	}
	*s = parsed
	return nil
}

// ParseSex reads a sex as written, M, F, X, or U or nothing for none stated.
func ParseSex(s string) (Sex, error) {
	switch strings.TrimSpace(s) {
	case "M":
		return Male, nil
	case "F":
		return Female, nil
	case "X":
		return Neither, nil
	case "U", "":
		return SexUnknown, nil
	}
	return SexUnknown, fmt.Errorf("sex %q: expected M, F, X or U", s)
}

// StatedSex is a person's sex as the sources state it, and the sources stating it.
type StatedSex struct {
	Value   Sex
	Sources []CitationRef
}

// Qualified reports whether the sex needs more than a bare letter to express.
func (s StatedSex) Qualified() bool { return len(s.Sources) > 0 }

// Relation is how a child stands to a partner of a family that lists them. The list is
// open, as event types are: a word the record uses is kept as written, and the six
// below are the ones kinship is worked out from. A word origins does not know counts as
// RelationUnknown, and None says the child is no child of that partner.
type Relation string

// The relations kinship is worked out from.
const (
	Birth           Relation = "birth"
	Adoptive        Relation = "adoptive"
	Step            Relation = "step"
	Foster          Relation = "foster"
	Guardian        Relation = "guardian"
	RelationUnknown Relation = "unknown"
	NoRelation      Relation = "none"
)

// Kin is the relation kinship reads: one of the six, or none, however it is written, the
// empty word being birth and any other word unknown.
func (r Relation) Kin() Relation {
	switch known := Relation(wordKey(string(r))); known {
	case "":
		return Birth
	case Birth, Adoptive, Step, Foster, Guardian, RelationUnknown, NoRelation:
		return known
	}
	return RelationUnknown
}

// Biological reports whether the relation is one that age and interval rules apply to.
// An adoptive mother may be younger than her daughter, and a parent of unknown relation
// is not known to have given birth to anybody.
func (r Relation) Biological() bool { return r.Kin() == Birth }

// CitationRef points into a source at a named anchor, written source#anchor, and can
// carry after it how the entry was graded as evidence for what cites it, as in
// source#anchor OPD. The anchor is a key in that source's citations, and the rule for
// it is only that it carry no whitespace and no second hash.
type CitationRef struct {
	Source Slug
	Anchor string
	Grade  Grade
}

// ParseCitationRef reads a citation as written, source#anchor and a grade after it.
func ParseCitationRef(s string) (CitationRef, error) {
	ref, grade, graded := strings.Cut(s, " ")
	source, anchor, found := strings.Cut(ref, "#")
	if !found {
		return CitationRef{}, fmt.Errorf("citation %q: expected source#anchor", s)
	}
	slug, err := ParseSlug(source)
	if err != nil {
		return CitationRef{}, fmt.Errorf("citation %q: %w", s, err)
	}
	if anchor == "" || strings.ContainsAny(anchor, "# \t\n") {
		return CitationRef{}, fmt.Errorf("citation %q: %q is not an anchor", s, anchor)
	}
	c := CitationRef{Source: slug, Anchor: anchor}
	if graded {
		if c.Grade, err = ParseGrade(grade); err != nil {
			return CitationRef{}, fmt.Errorf("citation %q: %w", s, err)
		}
	}
	return c, nil
}

func (c CitationRef) String() string {
	ref := string(c.Source) + "#" + c.Anchor
	if c.Grade.IsZero() {
		return ref
	}
	return ref + " " + c.Grade.String()
}

// Ungraded is the entry cited, whatever it was cited as. One entry is graded
// differently by each fact that cites it, and is still the one entry.
func (c CitationRef) Ungraded() CitationRef { return CitationRef{Source: c.Source, Anchor: c.Anchor} }

func (c CitationRef) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

func (c *CitationRef) UnmarshalText(b []byte) error {
	parsed, err := ParseCitationRef(string(b))
	if err != nil {
		return err
	}
	*c = parsed
	return nil
}

// Grade is the Genealogical Proof Standard's three questions of a piece of evidence,
// answered by a letter each: whether the source is original, derivative or authored,
// whether the information is primary, secondary or undetermined, and whether the
// evidence is direct, indirect or negative. Weight is how far it can be trusted for
// the fact citing it, a digit from 0, unreliable, to 3, certain, and the researcher's
// judgement of what the letters do not cover. A zero byte is a question not assessed,
// written -, a weight not assessed is left off, and a grade assessing nothing is not
// written at all.
type Grade struct {
	Source      byte
	Information byte
	Evidence    byte
	Weight      byte
}

var gradeLetters = [4]string{"ODA", "PSU", "DIN", "0123"}

// ParseGrade reads a grade as written, three letters or - and a weight or none.
func ParseGrade(s string) (Grade, error) {
	if len(s) != 3 && len(s) != 4 {
		return Grade{}, fmt.Errorf("grade %q: expected three letters, one each of %s, %s and %s, or -, and a weight from 0 to 3 or none",
			s, gradeLetters[0], gradeLetters[1], gradeLetters[2])
	}
	var answers [4]byte
	for i := range len(s) {
		switch {
		case s[i] == '-':
		case strings.IndexByte(gradeLetters[i], s[i]) >= 0:
			answers[i] = s[i]
		default:
			return Grade{}, fmt.Errorf("grade %q: %q is not one of %s or -", s, s[i], gradeLetters[i])
		}
	}
	return Grade{Source: answers[0], Information: answers[1], Evidence: answers[2], Weight: answers[3]}, nil
}

// IsZero reports whether the grade assesses nothing.
func (g Grade) IsZero() bool { return g == Grade{} }

func (g Grade) String() string {
	out := []byte{g.Source, g.Information, g.Evidence}
	if g.Weight != 0 {
		out = append(out, g.Weight)
	}
	for i, b := range out {
		if b == 0 {
			out[i] = '-'
		}
	}
	return string(out)
}

// PlaceName is a name a place was known by. An undated one is a spelling to match on;
// a dated one is the name the place actually bore when a given record was written, and
// the difference is worth keeping. Lang is the language the name is in, as a language
// tag: Florence is the English name of Firenze and Florenz the German. Type is an open
// word for a spelling its language does not account for, `pinyin` for a romanization or
// `kana` for a spelling by sound.
type PlaceName struct {
	Text    string
	Lang    Language
	Type    string
	From    Date
	Until   Date
	Note    string
	Sources []CitationRef
}

// Qualified reports whether the name needs more than a bare scalar to express.
func (n PlaceName) Qualified() bool {
	return n.Lang != "" || n.Type != "" || !n.From.IsZero() || !n.Until.IsZero() ||
		n.Note != "" || len(n.Sources) > 0
}

// LinkStatus is what was concluded about a child's link to a family. Nothing is derived
// from proven or challenged; a disproven link is kept with what disproved it and is no
// parentage.
type LinkStatus string

// The statuses a link can be given, the empty one saying nothing was concluded.
const (
	Proven     LinkStatus = "proven"
	Challenged LinkStatus = "challenged"
	Disproven  LinkStatus = "disproven"
)

// ParseLinkStatus reads a link's status as written.
func ParseLinkStatus(s string) (LinkStatus, error) {
	switch status := LinkStatus(strings.TrimSpace(s)); status {
	case "", Proven, Challenged, Disproven:
		return status, nil
	}
	return "", fmt.Errorf("status %q: expected proven, challenged or disproven", s)
}

// Child is somebody a family lists as its child: a person by their record, Unknown for
// a child the record says there was and does not name, or a name nobody has a record
// of, which counts as Unknown. The zero Relationship is a birth child of every partner.
type Child struct {
	ID       Slug
	Name     PersonName
	Relation Relationship
	Status   LinkStatus
	Note     string
	Sources  []CitationRef
}

// Relationship is what a child was to a family's partners: one word for every one of
// them, or a word for each where they differ, keyed by the partner's Who and kept in the
// order written.
type Relationship struct {
	Word     Relation
	Partners []PartnerRelation
}

// PartnerRelation is a child's relation to one partner of the family.
type PartnerRelation struct {
	Partner  Slug
	Relation Relation
}

// IsZero reports whether nothing is said of the relationship.
func (r Relationship) IsZero() bool { return r.Word == "" && len(r.Partners) == 0 }

// To is the relation to one partner, as written.
func (r Relationship) To(partner Slug) Relation {
	for _, p := range r.Partners {
		if p.Partner == partner {
			return p.Relation
		}
	}
	if r.Word == "" {
		return Birth
	}
	return r.Word
}

// Clone copies the relationship so that the copy shares nothing with it.
func (r Relationship) Clone() Relationship {
	r.Partners = slices.Clone(r.Partners)
	return r
}

// ChildrenOf makes the children a family lists by their slugs alone.
func ChildrenOf(slugs ...Slug) []Child {
	out := make([]Child, len(slugs))
	for i, s := range slugs {
		out[i] = Child{ID: s}
	}
	return out
}

// Who is the child as anything worked out from the family sees them.
func (c Child) Who() Slug {
	if c.ID == "" {
		return Unknown
	}
	return c.ID
}

// Somebody reports whether the child is a person with a record.
func (c Child) Somebody() bool { return c.ID != "" && !c.ID.Reserved() }

// To is the child's relation to a partner, as written.
func (c Child) To(partner Slug) Relation { return c.Relation.To(partner) }

// Qualified reports whether the child needs more than a bare slug to express.
func (c Child) Qualified() bool {
	return c.ID == "" || !c.Relation.IsZero() || c.Status != "" ||
		c.Note != "" || len(c.Sources) > 0
}

// The types of family origins knows. The list is open, as event types are: a
// handfasting or a concubinage is written in its own word rather than forced into one of
// these, and a family is left untyped where nothing says which.
const (
	Married    = "married"
	CivilUnion = "civil-union"
	Unmarried  = "unmarried"
)

// IDs maps the name of a service to the identifiers a record has there — familysearch,
// wikitree, wikidata, geonames, and whatever else, the list being open.
type IDs map[string]Identifiers

// Identifiers are a record's identifiers at one service, the one it goes by now first
// and those of the records merged into it after.
type Identifiers []Identifier

// Identifier is one identifier a record has at a service, with what is said of it.
type Identifier struct {
	ID      string
	Note    string
	Sources []CitationRef
}

// IdentifiersOf makes identifiers of the bare strings, in the order given.
func IdentifiersOf(ids ...string) Identifiers {
	out := make(Identifiers, len(ids))
	for i, id := range ids {
		out[i] = Identifier{ID: id}
	}
	return out
}

// Qualified reports whether the identifier says more than itself.
func (i Identifier) Qualified() bool { return i.Note != "" || len(i.Sources) > 0 }

// SameAs is a record of its own kind a record is said to be, by its slug, or by its
// name where it has no record here, never both. It is a claim waiting to be settled,
// by merging the two or by taking it out.
type SameAs struct {
	ID      Slug
	Name    string
	Note    string
	Sources []CitationRef
}

// Qualified reports whether the claim says more than a slug can.
func (s SameAs) Qualified() bool { return s.Name != "" || s.Note != "" || len(s.Sources) > 0 }

// Link is where something is online or how it is reached, by a URI whose scheme says
// which: a website, a telephone as tel: and the number as written, an email as
// mailto:, a fax as a tel: typed fax. Type is an open word.
type Link struct {
	URL   string
	Type  string
	Title string
	Note  string
}

// LinksOf makes links of the bare URLs, in the order given.
func LinksOf(urls ...string) []Link {
	out := make([]Link, len(urls))
	for i, url := range urls {
		out[i] = Link{URL: url}
	}
	return out
}

// Qualified reports whether the link says more than where it leads.
func (l Link) Qualified() bool { return l.Type != "" || l.Title != "" || l.Note != "" }

// Address is where something is, as its lines, its parts or both. The lines are the
// address as written on an envelope and the parts what each piece of it is, and neither
// is rebuilt from the other: the lines do not say which is the city, and the parts do
// not say how the lines were set out.
type Address struct {
	Lines    string
	Street   string
	Locality string
	City     string
	Parish   string
	County   string
	State    string
	Postal   string
	Country  string
	Date     Date
	Links    []Link
	Note     string
	Sources  []CitationRef
}

// Qualified reports whether the address says more than its lines.
func (a Address) Qualified() bool {
	return a.Street != "" || a.Locality != "" || a.City != "" || a.Parish != "" ||
		a.County != "" || a.State != "" || a.Postal != "" || a.Country != "" ||
		!a.Date.IsZero() || len(a.Links) > 0 || a.Note != "" || len(a.Sources) > 0
}

// IsZero reports whether the address says nothing at all.
func (a Address) IsZero() bool { return a.Lines == "" && !a.Qualified() }

// PlaceRef is a containing place, optionally bounded by the years the containment held.
// A birth in the Granducato di Toscana stays a birth there after 1861.
type PlaceRef struct {
	Place   Slug
	From    Date
	Until   Date
	Note    string
	Sources []CitationRef
}

// Qualified reports whether the containment says more than which place it is.
func (p PlaceRef) Qualified() bool {
	return !p.From.IsZero() || !p.Until.IsZero() || p.Note != "" || len(p.Sources) > 0
}

// Event is something that happened, to a person or to a couple, or a fact held — a
// title, an occupation, a residence. Its Value is the event's own words, the one text
// Gramps and Family Tree Maker give every event: what was held where it is a fact, and
// otherwise what the record says of it. The set of types is open: real records produce
// events no fixed list anticipates.
type Event struct {
	Private bool // left out of a hidden view

	Type  string
	Value string
	Date  Date

	// Sort is where the event goes in order when its date does not say, or there is
	// none: the researcher's placing, which says nothing of when it happened.
	Sort Date

	Place Slug

	// Address is where in the place it happened: a residence's house, the church a
	// wedding was in.
	Address Address

	// Age is the age the record gives whom the event is about, and Cause the cause the
	// record gives, of a death above all.
	Age   StatedAge
	Cause StatedCause

	// With are the people the record names besides whom the event is about.
	With []Named

	// Family is, on a person's event, the family the record ties it to: the one a
	// sealing to parents seals them to. It makes nobody a parent.
	Family     Slug
	Attributes []Attribute

	// Media are the pictures of it, kept beside the record holding the event or
	// referred to by their path.
	Media   []Media
	Links   []Link
	Note    string
	Sources []CitationRef

	// Tags are the researcher's own groupings, words as written.
	Tags []string

	// IDs are what an export identified the event by, as a record's are.
	IDs IDs
}

// Placing is the date that puts the event in order: its sort date where it has one, and
// its date where not. Whatever asks when the event happened asks Date.
func (e Event) Placing() Date {
	if !e.Sort.IsZero() {
		return e.Sort
	}
	return e.Date
}

// Named is somebody an event names in a role — the godfather at a baptism, a witness at
// a marriage — by their record where they have one and by their name where they have
// none, never both. Their own page derives the link back, so it is written once.
//
// Person is Unknown for somebody the record says was there and does not name.
type Named struct {
	Person     Slug
	Name       PersonName
	Role       string
	Age        Age
	Attributes []Attribute
	Note       string
	Sources    []CitationRef
}

// Attribute is something a record says of what it is on that no field holds: the
// agency that registered a birth, a father's age, that a birth was stillborn. It is
// what the record says, and nothing is worked out from it.
type Attribute struct {
	Type    string
	Value   string
	Date    Date
	Note    string
	Sources []CitationRef
}

// NonEvent is something that did not happen, over its date or, without one, ever: a
// finding with its own sources, kept apart from the events so that nothing reading
// them can take it for one.
type NonEvent struct {
	Type    string
	Date    Date
	Note    string
	Sources []CitationRef
}

// StatedAge is the age a record gives, and what is said of it and what states it where
// anything is.
type StatedAge struct {
	Value   Age
	Note    string
	Sources []CitationRef
}

// Qualified reports whether the age needs more than the age to express.
func (a StatedAge) Qualified() bool { return a.Note != "" || len(a.Sources) > 0 }

// StatedCause is the cause a record gives, and what is said of it and what states it
// where anything is.
type StatedCause struct {
	Value   string
	Note    string
	Sources []CitationRef
}

// Qualified reports whether the cause needs more than its words to express.
func (c StatedCause) Qualified() bool { return c.Note != "" || len(c.Sources) > 0 }

// Person carries no parents, no families and no children. All three are derived from the
// families listing the person, so there is one source of truth and nothing to keep in
// step.
type Person struct {
	Slug Slug

	// Private marks a record as not for sharing. A hidden view leaves it out, and
	// nothing else reads it.
	Private bool
	Living  Living

	Name        PersonName
	AlsoKnownAs []PersonName
	Sex         StatedSex
	Events      []Event
	Not         []NonEvent
	Body        string

	// Sources are what the record rests on as a whole, where no one fact does.
	Sources []CitationRef
	Tags    []string
	SameAs  []SameAs
	IDs     IDs

	// Portrait names the attached image that is a picture of this person, when the
	// first one under photos/ is not the one wanted, and the part of it that is them
	// when the picture is of several people. Its file is a path into Files.
	Portrait Image
	Media    []Media
	Links    []Link

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Family is the people a child was born to or raised by, with the partnership between
// them where there was one, and the children, in the order the record gives them. It
// exists so that facts about a couple — the marriage date and place, the divorce, the
// citations evidencing them — and which children are whose live in one place rather
// than being repeated across the people in it.
type Family struct {
	Slug     Slug
	Private  bool
	Partners []Partner
	Type     string
	Children []Child
	Events   []Event
	Not      []NonEvent
	Media    []Media
	Links    []Link
	Body     string
	Sources  []CitationRef
	Tags     []string
	SameAs   []SameAs
	IDs      IDs

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// The two words that stand where a record would. Neither is ever a record's slug, so
// that a partner written as one cannot be mistaken for somebody.
const (
	// Unknown is somebody there was and nobody knows: the father of a child baptised as
	// illegitimate, the husband of a widow whose marriage record is lost.
	Unknown Slug = "unknown"
	// None is nobody at all: the partner a woman who adopted alone did not have.
	None Slug = "none"
)

// Reserved reports whether a slug is one of the two words, and so names no record.
func (s Slug) Reserved() bool { return s == Unknown || s == None }

// Partner is one of a family's partners: a person's record, Unknown or None, or
// somebody the record names and nobody has a record of, by their name. A note says
// something of that partner's place in this family alone.
type Partner struct {
	Person Slug
	Name   PersonName
	Note   string
}

// PartnersOf makes partners of slugs, each a person's record or one of the two words.
func PartnersOf(slugs ...Slug) []Partner {
	partners := make([]Partner, len(slugs))
	for i, slug := range slugs {
		partners[i] = Partner{Person: slug}
	}
	return partners
}

// Who is the partner as everything worked out from the family counts them: their
// record, or one of the two words, a partner known only by name being Unknown.
func (p Partner) Who() Slug {
	if p.Person == "" {
		return Unknown
	}
	return p.Person
}

// Somebody reports whether the partner is a person with a record.
func (p Partner) Somebody() bool { return p.Person != "" && !p.Person.Reserved() }

// Qualified reports whether the partner says more than a slug can.
func (p Partner) Qualified() bool { return !p.Name.IsZero() || p.Note != "" }

// Coordinates are decimal degrees, north and east positive. Wording is what a record
// gave that cannot be read as two angles, kept in place of them, and coordinates that
// are only wording put the place nowhere.
type Coordinates struct {
	Latitude, Longitude float64
	Wording             string
}

// Placed reports whether the coordinates are two angles rather than a record's words.
func (c Coordinates) Placed() bool { return c.Wording == "" }

// Place holds its own hierarchy as data. Places are kept side by side on disk, whatever
// lies within what. Name is the name the place goes by where it is, and its dates when it
// has gone by it.
//
// Places begin and end: a hamlet absorbed into a neighbouring comune, a state annexed
// out of existence. From and Until bound the place itself, and references to it from
// before that date stay correct. Sources are what says it existed, and when it began and
// ended.
type Place struct {
	Slug        Slug
	Private     bool
	Name        PlaceName
	AlsoKnownAs []PlaceName
	Type        string
	Within      []PlaceRef
	Coordinates *Coordinates

	// Addresses are where the place lies in a reckoning other than what it lies
	// within: a street, a parish, a county.
	Addresses []Address
	From      Date
	Until     Date
	Media     []Media
	Links     []Link
	Sources   []CitationRef
	Tags      []string
	SameAs    []SameAs
	Body      string
	IDs       IDs

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Citation is a pointer into a source: what kind of entry it is, where in the source it
// is, when the entry was written, the transcription, who it names, and the scans it
// came off.
//
// Event is the kind of entry, in the words an event's type is written in, and need not
// be the fact citing it: a burial entry is cited for a death. Page is the locator
// somebody reads, "c. 88r, n. 112", and not the anchor, which references use and which
// has to stay short and free of spaces. Date is when the entry was written, since a
// death registered two days after it happened is a different date from the death. With
// is who the entry names and as what, each in the shape of somebody an event names but
// citing nothing, being in the entry cited; the entry is one and the people citing it
// many. Images are the scans, each a media entry whose region marks the entry on the
// page. IDs are what an export identified the entry by, as a record's are.
type Citation struct {
	Event      string
	Page       string
	Date       Date
	Text       string
	With       []Named
	Images     []Media
	Links      []Link
	Attributes []Attribute
	Note       string
	Tags       []string
	IDs        IDs
}

// Image is a picture beside a record, and the part of it that matters where that is
// marked: where on a scan the entry is, or which of the people in a photograph is the
// one it is the portrait of.
type Image struct {
	File   string
	Region *Region
}

// Qualified reports whether the image says more than which file it is.
func (i Image) Qualified() bool { return i.Region != nil }

// Region is part of a picture, as four fractions of it, so that it survives the picture
// being scaled.
type Region struct {
	Left, Top, Right, Bottom float64
}

// Source is a thing that exists in the world. Its citations are named anchors within
// it, so no separate citation entity is needed and the transcription lives once, beside
// the scan.
//
// ShortTitle is the abbreviation the source is cited by, and names it where it has no
// title. Covers is what it records, Repositories where it is kept, and Text its own
// words where they are short enough to copy whole, apart from the body, which is the
// researcher's.
type Source struct {
	Slug         Slug
	Private      bool
	Title        string
	ShortTitle   string
	Type         string
	Author       string
	Publication  string
	Covers       []Coverage
	Repositories []Holding
	Text         string
	Attributes   []Attribute
	Citations    map[string]Citation
	Media        []Media
	Links        []Link
	Tags         []string
	SameAs       []SameAs
	Body         string
	IDs          IDs

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Coverage is what a register or an index records: the kinds of event, in the words an
// event's type is written in, over a period and for a place. A baptismal register of one
// parish covers baptisms there between two years, and is where to look for anybody born
// there then who is not yet cited to it.
type Coverage struct {
	Events []string
	Date   Date
	Place  Slug
	Note   string
}

// Holding is a repository a source is kept in, the call number it is kept under there
// and the medium it is kept in, an open word: book, microfilm, electronic. A source with
// two call numbers in one archive is held there twice.
type Holding struct {
	Repository Slug
	CallNumber string
	Medium     string
	Note       string
}

// Qualified reports whether the entry says more than which repository it is.
func (h Holding) Qualified() bool { return h.CallNumber != "" || h.Medium != "" || h.Note != "" }

// Repository is the archive, library, parish office or website a source is kept in, a
// record of its own so that one archive is written once and not in every source it
// keeps. Addresses are where it is and has been; what only somebody visiting needs to
// know is the body.
type Repository struct {
	Slug      Slug
	Private   bool
	Name      string
	Type      string
	Addresses []Address
	Media     []Media
	Links     []Link
	Tags      []string
	SameAs    []SameAs
	Body      string
	IDs       IDs

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Note is what is about no record in particular: a research log, a list of things to
// do, a letter to an archive. Its slug is made from its title. An event it holds is
// about nobody the corpus has a record of, a wedding known only because a witness was
// there, and names whoever it names in With.
type Note struct {
	Slug    Slug
	Private bool
	Title   string
	Events  []Event
	Media   []Media
	Links   []Link
	Sources []CitationRef
	Tags    []string
	SameAs  []SameAs
	Body    string
	IDs     IDs

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Status is where a hypothesis stands.
type Status string

// The statuses a hypothesis can have.
const (
	Open     Status = "open"
	Accepted Status = "accepted"
	Rejected Status = "rejected"
)

// ParseStatus reads a hypothesis's status as written.
func ParseStatus(s string) (Status, error) {
	switch st := Status(strings.TrimSpace(s)); st {
	case Open, Accepted, Rejected:
		return st, nil
	default:
		return "", fmt.Errorf("status %q: expected open, accepted or rejected", s)
	}
}

// Confidence is how much the proposer believes the hypothesis.
type Confidence string

// The confidences a proposer can state.
const (
	Probable Confidence = "probable"
	Possible Confidence = "possible"
	Doubtful Confidence = "doubtful"
)

// ParseConfidence reads a confidence as written.
func ParseConfidence(s string) (Confidence, error) {
	switch c := Confidence(strings.TrimSpace(s)); c {
	case Probable, Possible, Doubtful:
		return c, nil
	default:
		return "", fmt.Errorf("confidence %q: expected probable, possible or doubtful", s)
	}
}

// Hypothesis is a proposed departure from the corpus, expressed as edits that can be
// applied, tested and reversed mechanically.
//
// Base pins the commit the edits apply to. Edit paths are positional and the corpus
// moves, so the edits are never re-resolved against a later state: the base is
// reconstructed, the edits applied where the paths are valid by construction, and the
// result diffed. Applying a stale hypothesis is a rebase, and drift shows up as a
// conflict rather than as a silent misapplication.
type Hypothesis struct {
	Slug        Slug
	Status      Status
	Base        string
	Confidence  Confidence
	ProposedBy  string
	Supports    []CitationRef
	Contradicts []CitationRef
	Edits       []Edit
	Body        string

	// Files are the paths of attached files, relative to the entity's directory and in
	// sorted order. An entity with nothing attached to it yet has none.
	Files []string
}

// Project is the corpus-wide header, kept in origins.yaml: what the tree is, who keeps
// it and for whom.
type Project struct {
	Name        string
	Description string
	MainPerson  Slug
	Lang        string
	Copyright   string
	Researchers []Researcher

	// Bookmarks are records the researcher keeps to hand, in their order.
	Bookmarks []EntityRef

	// Tags gives a tag the color and priority it is shown with. A tag needs no entry
	// here to be used.
	Tags map[string]TagStyle

	FormatVersion int
}

// Researcher is somebody who compiled the tree.
type Researcher struct {
	Name      string
	Addresses []Address
	Links     []Link
	Note      string
}

// TagStyle is how a tag is shown, as the researcher gave it.
type TagStyle struct {
	Color    string
	Priority string
}

// IsZero reports whether nothing at all was read into the project, which is what a
// manifest that failed to decode leaves.
func (p Project) IsZero() bool {
	return p.Name == "" && p.Description == "" && p.MainPerson == "" && p.Lang == "" &&
		p.Copyright == "" && p.Researchers == nil && p.Bookmarks == nil && p.Tags == nil &&
		p.FormatVersion == 0
}

// Clone copies the project deeply enough that changing one cannot be seen in the other.
func (p Project) Clone() Project {
	out := p
	if p.Researchers != nil {
		out.Researchers = make([]Researcher, len(p.Researchers))
		for i, r := range p.Researchers {
			r.Addresses = cloneAddresses(r.Addresses)
			r.Links = slices.Clone(r.Links)
			out.Researchers[i] = r
		}
	}
	out.Bookmarks = slices.Clone(p.Bookmarks)
	out.Tags = maps.Clone(p.Tags)
	return out
}

// FormatVersion is the version a corpus origins writes is written in, so that the day
// the canonical form changes a reader can tell which of the two it is holding rather
// than guessing. It is one number and not one per writer: a corpus imported and a
// corpus started from nothing are the same format, or the field says nothing at all.
const FormatVersion = 1
