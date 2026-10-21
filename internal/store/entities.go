package store

import (
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

type eventDoc struct {
	Private    flagged       `yaml:"private"`
	Type       string        `yaml:"type"`
	Value      string        `yaml:"value"`
	Date       date          `yaml:"date"`
	Sort       date          `yaml:"sort-date"`
	Place      slug          `yaml:"place"`
	Address    *address      `yaml:"address"`
	Age        statedAge     `yaml:"age"`
	Cause      statedCause   `yaml:"cause"`
	With       []named       `yaml:"with"`
	Family     slug          `yaml:"family"`
	Attributes []attribute   `yaml:"attributes"`
	Media      []media       `yaml:"media"`
	Links      []link        `yaml:"links"`
	Note       string        `yaml:"note"`
	Sources    []citationRef `yaml:"sources"`
	Tags       []tag         `yaml:"tags"`
	IDs        ids           `yaml:"ids"`
}

func events(docs []eventDoc) []model.Event {
	return convert(docs, func(d eventDoc) model.Event {
		var at model.Address
		if d.Address != nil {
			at = model.Address(*d.Address)
		}
		return model.Event{
			Private:    bool(d.Private),
			Type:       d.Type,
			Value:      d.Value,
			Date:       model.Date(d.Date),
			Sort:       model.Date(d.Sort),
			Place:      model.Slug(d.Place),
			Address:    at,
			Age:        model.StatedAge(d.Age),
			Cause:      model.StatedCause(d.Cause),
			With:       convert(d.With, func(n named) model.Named { return model.Named(n) }),
			Family:     model.Slug(d.Family),
			Attributes: attributes(d.Attributes),
			Media:      mediaEntries(d.Media),
			Links:      links(d.Links),
			Note:       d.Note,
			Sources:    citations(d.Sources),
			Tags:       tags(d.Tags),
			IDs:        model.IDs(d.IDs),
		}
	})
}

type personDoc struct {
	Private     flagged       `yaml:"private"`
	Living      living        `yaml:"living"`
	Name        unflaggedName `yaml:"name"`
	AlsoKnownAs []personName  `yaml:"also-known-as"`
	Sex         statedSex     `yaml:"sex"`
	Portrait    image         `yaml:"portrait"`
	Events      []eventDoc    `yaml:"events"`
	Not         []nonEvent    `yaml:"not"`
	Media       []media       `yaml:"media"`
	Links       []link        `yaml:"links"`
	Sources     []citationRef `yaml:"sources"`
	Tags        []tag         `yaml:"tags"`
	SameAs      []sameAs      `yaml:"same-as"`
	IDs         ids           `yaml:"ids"`
}

type familyDoc struct {
	Private  flagged       `yaml:"private"`
	Partners []partner     `yaml:"partners"`
	Type     string        `yaml:"type"`
	Children []child       `yaml:"children"`
	Events   []eventDoc    `yaml:"events"`
	Not      []nonEvent    `yaml:"not"`
	Media    []media       `yaml:"media"`
	Links    []link        `yaml:"links"`
	Sources  []citationRef `yaml:"sources"`
	Tags     []tag         `yaml:"tags"`
	SameAs   []sameAs      `yaml:"same-as"`
	IDs      ids           `yaml:"ids"`
}

type placeDoc struct {
	Private     flagged       `yaml:"private"`
	Name        *placeName    `yaml:"name"`
	AlsoKnownAs []placeName   `yaml:"also-known-as"`
	Type        string        `yaml:"type"`
	Within      within        `yaml:"within"`
	Coordinates *coordinates  `yaml:"coordinates"`
	Addresses   []address     `yaml:"addresses"`
	From        date          `yaml:"from"`
	Until       date          `yaml:"until"`
	Media       []media       `yaml:"media"`
	Links       []link        `yaml:"links"`
	Sources     []citationRef `yaml:"sources"`
	Tags        []tag         `yaml:"tags"`
	SameAs      []sameAs      `yaml:"same-as"`
	IDs         ids           `yaml:"ids"`
}

type citationDoc struct {
	Event      string      `yaml:"event"`
	Page       string      `yaml:"page"`
	Date       date        `yaml:"date"`
	Text       string      `yaml:"text"`
	With       []namedIn   `yaml:"with"`
	Images     []media     `yaml:"images"`
	Links      []link      `yaml:"links"`
	Attributes []attribute `yaml:"attributes"`
	Note       string      `yaml:"note"`
	Tags       []tag       `yaml:"tags"`
	IDs        ids         `yaml:"ids"`
}

func (d citationDoc) citation() model.Citation {
	return model.Citation{
		Event:      d.Event,
		Page:       d.Page,
		Date:       model.Date(d.Date),
		Text:       d.Text,
		With:       convert(d.With, func(n namedIn) model.Named { return model.Named(n) }),
		Images:     mediaEntries(d.Images),
		Links:      links(d.Links),
		Attributes: attributes(d.Attributes),
		Note:       d.Note,
		Tags:       tags(d.Tags),
		IDs:        model.IDs(d.IDs),
	}
}

type sourceDoc struct {
	Private      flagged                `yaml:"private"`
	Title        string                 `yaml:"title"`
	Type         string                 `yaml:"type"`
	Author       string                 `yaml:"author"`
	Publication  string                 `yaml:"publication"`
	ShortTitle   string                 `yaml:"short-title"`
	Covers       []coverage             `yaml:"covers"`
	Repositories []holding              `yaml:"repositories"`
	Attributes   []attribute            `yaml:"attributes"`
	Text         string                 `yaml:"text"`
	Citations    map[string]citationDoc `yaml:"citations"`
	Media        []media                `yaml:"media"`
	Links        []link                 `yaml:"links"`
	Tags         []tag                  `yaml:"tags"`
	SameAs       []sameAs               `yaml:"same-as"`
	IDs          ids                    `yaml:"ids"`
}

type repositoryDoc struct {
	Private   flagged   `yaml:"private"`
	Name      string    `yaml:"name"`
	Type      string    `yaml:"type"`
	Addresses []address `yaml:"addresses"`
	Media     []media   `yaml:"media"`
	Links     []link    `yaml:"links"`
	Tags      []tag     `yaml:"tags"`
	SameAs    []sameAs  `yaml:"same-as"`
	IDs       ids       `yaml:"ids"`
}

type noteDoc struct {
	Private flagged       `yaml:"private"`
	Title   string        `yaml:"title"`
	Events  []eventDoc    `yaml:"events"`
	Media   []media       `yaml:"media"`
	Links   []link        `yaml:"links"`
	Sources []citationRef `yaml:"sources"`
	Tags    []tag         `yaml:"tags"`
	SameAs  []sameAs      `yaml:"same-as"`
	IDs     ids           `yaml:"ids"`
}

type hypothesisDoc struct {
	Status      status        `yaml:"status"`
	Base        string        `yaml:"base"`
	Confidence  confidence    `yaml:"confidence"`
	ProposedBy  string        `yaml:"proposed-by"`
	Supports    []citationRef `yaml:"supports"`
	Contradicts []citationRef `yaml:"contradicts"`
	Edits       []editDoc     `yaml:"edits"`
}

type projectDoc struct {
	Project struct {
		Name        string              `yaml:"name"`
		Description string              `yaml:"description"`
		MainPerson  slug                `yaml:"main-person"`
		Lang        string              `yaml:"lang"`
		Copyright   string              `yaml:"copyright"`
		Researchers []researcher        `yaml:"researchers"`
		Bookmarks   []bookmark          `yaml:"bookmarks"`
		Tags        map[string]tagStyle `yaml:"tags"`
	} `yaml:"project"`
	Origins struct {
		FormatVersion int `yaml:"format-version"`
	} `yaml:"origins"`
}

// researcher is somebody who compiled the tree: a name, addresses and links as a
// repository has them, and a note.
type researcher model.Researcher

func (r *researcher) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: a researcher is a mapping with a name", node.Line)
	}
	if err := onlyFields(node, "name", "addresses", "links", "note"); err != nil {
		return err
	}
	var fields struct {
		Name      string    `yaml:"name"`
		Addresses []address `yaml:"addresses"`
		Links     []link    `yaml:"links"`
		Note      string    `yaml:"note"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	one := model.Researcher{
		Name:      fields.Name,
		Addresses: addresses(fields.Addresses),
		Links:     links(fields.Links),
		Note:      fields.Note,
	}
	if err := whoResearches(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*r = researcher(one)
	return nil
}

// whoResearches is the rule for a researcher, which the emitter holds to as well. The
// manifest is not a record, so nothing in it is cited.
func whoResearches(r model.Researcher) error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("a researcher has a name")
	}
	for _, a := range r.Addresses {
		if len(a.Sources) > 0 {
			return fmt.Errorf("%s: a researcher's address cites nothing", r.Name)
		}
	}
	return nil
}

// bookmark is a record kept to hand, written as its kind's directory and its slug.
type bookmark model.EntityRef

func (b *bookmark) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, parseBookmark)
	if err != nil {
		return err
	}
	*b = bookmark(value)
	return nil
}

func parseBookmark(s string) (model.EntityRef, error) {
	dir, name, _ := strings.Cut(s, "/")
	kind := model.KindIn(dir)
	if kind == "" {
		return model.EntityRef{}, fmt.Errorf("bookmark %q: expected a kind's directory and a slug, as people/rossi-giovanni", s)
	}
	slug, err := model.ParseSlug(name)
	if err != nil {
		return model.EntityRef{}, fmt.Errorf("bookmark %q: %w", s, err)
	}
	return model.EntityRef{Kind: kind, Slug: slug}, nil
}

func bookmarkText(b model.EntityRef) (string, error) {
	text := b.Kind.Dir() + "/" + string(b.Slug)
	if _, err := parseBookmark(text); err != nil {
		return "", err
	}
	return text, nil
}

type tagStyle struct {
	Color    string `yaml:"color"`
	Priority string `yaml:"priority"`
}

const emptyTagName = "a tag in the project is named"

// unmarshal decodes frontmatter strictly. A field origins does not recognise is an
// error rather than something to skip, because the corpus is re-emitted canonically on
// every write and an unread field would vanish from the file.
//
// The frontmatter is fed to the parser with a blank line in front of it, standing in
// for the opening fence, so that every line number the parser reports is the line
// number in the file the reader has open.
func unmarshal(frontmatter []byte, into any) error {
	aligned := append([]byte("\n"), frontmatter...)
	decoder := yaml.NewDecoder(bytes.NewReader(aligned))
	decoder.KnownFields(true)
	if err := decoder.Decode(into); err != nil {
		return readable(err)
	}
	return nil
}

// decodeRecord decodes a record's frontmatter. A record with nothing in it has
// frontmatter of no fields at all, which the parser reads as no document rather than
// as an empty one.
func decodeRecord(frontmatter []byte, into any) error {
	if len(bytes.TrimSpace(frontmatter)) == 0 {
		return nil
	}
	return unmarshal(frontmatter, into)
}

var unknownField = regexp.MustCompile(`^(line \d+: )?field (\S+) not found in type \S+$`)

// readable turns the YAML package's report into one a reader of the corpus can act on.
// Its own wording names the Go type the frontmatter was being decoded into, which is an
// implementation detail and not something the file can be fixed by.
func readable(err error) error {
	var typeError *yaml.TypeError
	if !errors.As(err, &typeError) {
		return err
	}
	messages := make([]string, 0, len(typeError.Errors))
	for _, message := range typeError.Errors {
		messages = append(messages, unknownField.ReplaceAllString(message, `${1}unknown field "$2"`))
	}
	return errors.New(strings.Join(messages, "; "))
}

func decodePerson(s model.Slug, frontmatter []byte, body string) (*model.Person, error) {
	var doc personDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	return &model.Person{
		Private:     bool(doc.Private),
		Living:      model.Living(doc.Living),
		Slug:        s,
		Name:        model.PersonName(doc.Name),
		AlsoKnownAs: personNames(doc.AlsoKnownAs),
		Sex:         model.StatedSex(doc.Sex),
		Portrait:    model.Image(doc.Portrait),
		Events:      events(doc.Events),
		Not:         nonEvents(doc.Not),
		Media:       mediaEntries(doc.Media),
		Links:       links(doc.Links),
		Body:        body,
		Sources:     citations(doc.Sources),
		Tags:        tags(doc.Tags),
		SameAs:      sameAsList(doc.SameAs),
		IDs:         model.IDs(doc.IDs),
	}, nil
}

func decodeFamily(s model.Slug, frontmatter []byte, body string) (*model.Family, error) {
	var doc familyDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	return &model.Family{
		Private:  bool(doc.Private),
		Slug:     s,
		Partners: partners(doc.Partners),
		Type:     doc.Type,
		Children: children(doc.Children),
		Events:   events(doc.Events),
		Not:      nonEvents(doc.Not),
		Media:    mediaEntries(doc.Media),
		Links:    links(doc.Links),
		Body:     body,
		Sources:  citations(doc.Sources),
		Tags:     tags(doc.Tags),
		SameAs:   sameAsList(doc.SameAs),
		IDs:      model.IDs(doc.IDs),
	}, nil
}

func decodePlace(s model.Slug, frontmatter []byte, body string) (*model.Place, error) {
	var doc placeDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	place := &model.Place{
		Private:     bool(doc.Private),
		Slug:        s,
		AlsoKnownAs: placeNames(doc.AlsoKnownAs),
		Type:        doc.Type,
		Within:      places(doc.Within),
		Addresses:   addresses(doc.Addresses),
		From:        model.Date(doc.From),
		Until:       model.Date(doc.Until),
		Media:       mediaEntries(doc.Media),
		Links:       links(doc.Links),
		Sources:     citations(doc.Sources),
		Tags:        tags(doc.Tags),
		SameAs:      sameAsList(doc.SameAs),
		Body:        body,
		IDs:         model.IDs(doc.IDs),
	}
	if doc.Name != nil {
		place.Name = model.PlaceName(*doc.Name)
	}
	if doc.Coordinates != nil {
		at := model.Coordinates(*doc.Coordinates)
		place.Coordinates = &at
	}
	return place, nil
}

func decodeSource(s model.Slug, frontmatter []byte, body string) (*model.Source, error) {
	var doc sourceDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	source := &model.Source{
		Private:      bool(doc.Private),
		Slug:         s,
		Title:        doc.Title,
		ShortTitle:   doc.ShortTitle,
		Type:         doc.Type,
		Author:       doc.Author,
		Publication:  doc.Publication,
		Covers:       convert(doc.Covers, func(c coverage) model.Coverage { return model.Coverage(c) }),
		Repositories: convert(doc.Repositories, func(h holding) model.Holding { return model.Holding(h) }),
		Attributes:   attributes(doc.Attributes),
		Text:         doc.Text,
		Media:        mediaEntries(doc.Media),
		Links:        links(doc.Links),
		Tags:         tags(doc.Tags),
		SameAs:       sameAsList(doc.SameAs),
		Body:         body,
		IDs:          model.IDs(doc.IDs),
	}
	if len(doc.Citations) > 0 {
		source.Citations = make(map[string]model.Citation, len(doc.Citations))
		for anchor, citation := range doc.Citations {
			if _, err := model.ParseCitationRef(string(s) + "#" + anchor); err != nil {
				return nil, err
			}
			source.Citations[anchor] = citation.citation()
		}
	}
	return source, nil
}

func decodeRepository(s model.Slug, frontmatter []byte, body string) (*model.Repository, error) {
	var doc repositoryDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	return &model.Repository{
		Private:   bool(doc.Private),
		Slug:      s,
		Name:      doc.Name,
		Type:      doc.Type,
		Addresses: addresses(doc.Addresses),
		Media:     mediaEntries(doc.Media),
		Links:     links(doc.Links),
		Tags:      tags(doc.Tags),
		SameAs:    sameAsList(doc.SameAs),
		Body:      body,
		IDs:       model.IDs(doc.IDs),
	}, nil
}

func decodeNote(s model.Slug, frontmatter []byte, body string) (*model.Note, error) {
	var doc noteDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	return &model.Note{
		Private: bool(doc.Private),
		Slug:    s,
		Title:   doc.Title,
		Events:  events(doc.Events),
		Media:   mediaEntries(doc.Media),
		Links:   links(doc.Links),
		Sources: citations(doc.Sources),
		Tags:    tags(doc.Tags),
		SameAs:  sameAsList(doc.SameAs),
		Body:    body,
		IDs:     model.IDs(doc.IDs),
	}, nil
}

// ParseHypothesis reads a hypothesis's file, frontmatter and body, as the corpus reads
// the one kept under slug.
func ParseHypothesis(slug model.Slug, content []byte) (*model.Hypothesis, error) {
	frontmatter, body, err := Split(content)
	if err != nil {
		return nil, err
	}
	return decodeHypothesis(slug, frontmatter, string(body))
}

func decodeHypothesis(s model.Slug, frontmatter []byte, body string) (*model.Hypothesis, error) {
	var doc hypothesisDoc
	if err := decodeRecord(frontmatter, &doc); err != nil {
		return nil, err
	}
	edits := make([]model.Edit, 0, len(doc.Edits))
	for i, e := range doc.Edits {
		edit, err := e.edit()
		if err != nil {
			return nil, fmt.Errorf("edits[%d]: %w", i, err)
		}
		edits = append(edits, edit)
	}
	return &model.Hypothesis{
		Slug:        s,
		Status:      model.Status(doc.Status),
		Base:        doc.Base,
		Confidence:  model.Confidence(doc.Confidence),
		ProposedBy:  doc.ProposedBy,
		Supports:    citations(doc.Supports),
		Contradicts: citations(doc.Contradicts),
		Edits:       edits,
		Body:        body,
	}, nil
}

func decodeProject(content []byte) (model.Project, error) {
	var doc projectDoc
	err := unmarshal(content, &doc)
	if err != nil {
		return model.Project{}, err
	}
	project := model.Project{
		Name:          doc.Project.Name,
		Description:   doc.Project.Description,
		MainPerson:    model.Slug(doc.Project.MainPerson),
		Lang:          doc.Project.Lang,
		Copyright:     doc.Project.Copyright,
		Researchers:   convert(doc.Project.Researchers, func(r researcher) model.Researcher { return model.Researcher(r) }),
		Bookmarks:     convert(doc.Project.Bookmarks, func(b bookmark) model.EntityRef { return model.EntityRef(b) }),
		FormatVersion: doc.Origins.FormatVersion,
	}
	if project.Tags, err = tagStyles(doc.Project.Tags); err != nil {
		return model.Project{}, err
	}
	return project, nil
}

// tagStyles reads the styles the manifest gives its tags, each under the tag as written.
func tagStyles(wire map[string]tagStyle) (map[string]model.TagStyle, error) {
	var styles map[string]model.TagStyle
	for name, style := range wire {
		if strings.TrimSpace(name) == "" {
			return nil, errors.New(emptyTagName)
		}
		if styles == nil {
			styles = map[string]model.TagStyle{}
		}
		styles[name] = model.TagStyle(style)
	}
	return styles, nil
}
