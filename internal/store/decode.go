package store

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/origins-project/origins/internal/model"
)

// The types here exist so that model stays free of YAML. They are the wire forms of the
// model types, one per value that needs parsing out of a scalar or out of the
// scalar-or-qualified pattern, and nothing else in the package is allowed to know how
// the corpus is spelled on disk.

// scalar decodes a plain scalar node through a parse function, which is how every
// value with its own grammar — dates, slugs, citation references — enters the corpus.
// Reading node.Value rather than letting YAML infer a type matters: 1834 is a year, not
// an integer, and 1750/51 is a year too.
func scalar[T any](node *yaml.Node, parse func(string) (T, error)) (T, error) {
	var zero T
	if node.Kind != yaml.ScalarNode {
		return zero, fmt.Errorf("line %d: expected a value", node.Line)
	}
	value, err := parse(node.Value)
	if err != nil {
		return zero, fmt.Errorf("line %d: %w", node.Line, err)
	}
	return value, nil
}

// onlyFields rejects a mapping carrying anything the caller does not know about. The
// corpus is re-emitted canonically on every write, so a field origins does not
// understand would be silently deleted; refusing to read it is the honest answer.
func onlyFields(node *yaml.Node, allowed ...string) error {
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if !slices.Contains(allowed, key.Value) {
			return fmt.Errorf("line %d: unknown field %q, expected one of %s",
				key.Line, key.Value, strings.Join(allowed, ", "))
		}
	}
	return nil
}

type slug model.Slug

func (s *slug) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseSlug)
	if err != nil {
		return err
	}
	*s = slug(value)
	return nil
}

type date model.Date

func (d *date) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseDate)
	if err != nil {
		return err
	}
	*d = date(value)
	return nil
}

type age model.Age

func (a *age) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseAge)
	if err != nil {
		return err
	}
	*a = age(value)
	return nil
}

type citationRef model.CitationRef

func (c *citationRef) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseCitationRef)
	if err != nil {
		return err
	}
	*c = citationRef(value)
	return nil
}

type sex model.Sex

func (s *sex) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseSex)
	if err != nil {
		return err
	}
	*s = sex(value)
	return nil
}

// flagged is a field that is the word true or is left out. Every scalar in a corpus is
// text, so yes, on and 1 are not read as true; and false is what leaving the field out
// already says, which the emitter can only write one way.
type flagged bool

func (f *flagged) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, func(s string) (bool, error) {
		if s != "true" {
			return false, fmt.Errorf("%q is not a flag: a flag is true, or left out", s)
		}
		return true, nil
	})
	if err != nil {
		return err
	}
	*f = flagged(value)
	return nil
}

type living model.Living

func (l *living) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseLiving)
	if err != nil {
		return err
	}
	*l = living(value)
	return nil
}

// ids is a mapping from the name of a service to the identifiers the record has there,
// the one it goes by now first. A service is named the way a slug is written, so that
// ids.familysearch is a path.
type ids model.IDs

func (i *ids) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: ids is a mapping from a service to the identifiers it gives", node.Line)
	}
	read := make(model.IDs, len(node.Content)/2)
	for k := 0; k+1 < len(node.Content); k += 2 {
		key, value := node.Content[k], node.Content[k+1]
		if err := serviceName(key.Value); err != nil {
			return fmt.Errorf("line %d: %w", key.Line, err)
		}
		var list identifiers
		if err := value.Decode(&list); err != nil {
			return err
		}
		if len(list) > 0 {
			read[key.Value] = model.Identifiers(list)
		}
	}
	if len(read) > 0 {
		*i = ids(read)
	}
	return nil
}

// serviceName is the rule for a key of ids, which the emitter holds to as well.
func serviceName(name string) error {
	if _, err := model.ParseSlug(name); err != nil {
		return fmt.Errorf("%q is not the name of a service: expected lowercase words joined by hyphens", name)
	}
	return nil
}

// identifiers accepts the scalar-or-qualified pattern at the level of the field as well
// as the entry, so that `familysearch: XXXX-YYY` and a list of two read.
type identifiers []model.Identifier

func (l *identifiers) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		var entries []identifier
		if err := node.Decode(&entries); err != nil {
			return err
		}
		*l = convert(entries, func(i identifier) model.Identifier { return model.Identifier(i) })
		return nil
	}
	var one identifier
	if err := one.UnmarshalYAML(node); err != nil {
		return err
	}
	*l = identifiers{model.Identifier(one)}
	return nil
}

type identifier model.Identifier

func (i *identifier) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		ID      string        `yaml:"id"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.ID = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "id", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: an identifier is text, or a mapping with id", node.Line)
	}
	if strings.TrimSpace(qualified.ID) == "" {
		return fmt.Errorf("line %d: %s", node.Line, emptyIdentifier)
	}
	*i = identifier{ID: qualified.ID, Note: qualified.Note, Sources: citations(qualified.Sources)}
	return nil
}

const emptyIdentifier = "an identifier cannot be empty"

// tag is one of the researcher's own groupings, a word as written.
type tag string

func (t *tag) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, func(s string) (string, error) {
		if strings.TrimSpace(s) == "" {
			return "", errors.New(emptyTag)
		}
		return s, nil
	})
	if err != nil {
		return err
	}
	*t = tag(value)
	return nil
}

const emptyTag = "a tag cannot be empty"

func tags(list []tag) []string {
	return convert(list, func(t tag) string { return string(t) })
}

// sameAs is a record a record is said to be: its slug, or a mapping of the slug or the
// name of one with no record here, a note and sources.
type sameAs model.SameAs

func (s *sameAs) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		ID      slug          `yaml:"id"`
		Name    string        `yaml:"name"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := node.Decode(&fields.ID); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "id", "name", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&fields); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a record this one is the same as is a slug, or a mapping with id or name", node.Line)
	}
	one := model.SameAs{
		ID:      model.Slug(fields.ID),
		Name:    fields.Name,
		Note:    fields.Note,
		Sources: citations(fields.Sources),
	}
	if err := whoIsSame(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*s = sameAs(one)
	return nil
}

func sameAsList(list []sameAs) []model.SameAs {
	return convert(list, func(s sameAs) model.SameAs { return model.SameAs(s) })
}

// whoIsSame is the rule for a record said to be the same, which the emitter holds to as
// well: a record or a name, never both, and never one of the words for nobody, a record
// being somebody or something in particular.
func whoIsSame(s model.SameAs) error {
	switch {
	case s.ID.Reserved():
		return fmt.Errorf("a record is the same as a record or a name, and never %s", s.ID)
	case s.ID != "" && strings.TrimSpace(s.Name) != "":
		return fmt.Errorf("%s is named by its record, and takes no name beside it", s.ID)
	case s.ID == "" && strings.TrimSpace(s.Name) == "":
		return errors.New("a record this one is the same as has an id or a name")
	}
	return nil
}

// link is where something is online or how it is reached, as its URL or the URL with
// what kind of link it is, a title and a note.
type link model.Link

func (l *link) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		URL   string `yaml:"url"`
		Type  string `yaml:"type"`
		Title string `yaml:"title"`
		Note  string `yaml:"note"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.URL = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "url", "type", "title", "note"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a link is a URL, or a mapping with url", node.Line)
	}
	if strings.TrimSpace(qualified.URL) == "" {
		return fmt.Errorf("line %d: %s", node.Line, emptyLink)
	}
	*l = link(qualified)
	return nil
}

const emptyLink = "a link names the URL it is"

func links(list []link) []model.Link {
	return convert(list, func(l link) model.Link { return model.Link(l) })
}

// address is where something is, as the lines written on an envelope, or a mapping of
// the lines, the parts, its links, a date, a note and sources.
type address model.Address

var addressParts = []string{"lines", "street", "locality", "city", "parish", "county", "state",
	"postal", "country", "date", "links", "note", "sources"}

func (a *address) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Lines    string        `yaml:"lines"`
		Street   string        `yaml:"street"`
		Locality string        `yaml:"locality"`
		City     string        `yaml:"city"`
		Parish   string        `yaml:"parish"`
		County   string        `yaml:"county"`
		State    string        `yaml:"state"`
		Postal   string        `yaml:"postal"`
		Country  string        `yaml:"country"`
		Date     date          `yaml:"date"`
		Links    []link        `yaml:"links"`
		Note     string        `yaml:"note"`
		Sources  []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Lines = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, addressParts...); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: an address is its lines, or a mapping of lines and parts", node.Line)
	}
	read := model.Address{
		Lines:    qualified.Lines,
		Street:   qualified.Street,
		Locality: qualified.Locality,
		City:     qualified.City,
		Parish:   qualified.Parish,
		County:   qualified.County,
		State:    qualified.State,
		Postal:   qualified.Postal,
		Country:  qualified.Country,
		Date:     model.Date(qualified.Date),
		Links:    links(qualified.Links),
		Note:     qualified.Note,
		Sources:  citations(qualified.Sources),
	}
	if strings.TrimSpace(read.Lines) == "" && !read.Qualified() {
		return fmt.Errorf("line %d: %s", node.Line, emptyAddress)
	}
	*a = address(read)
	return nil
}

const emptyAddress = "an address says where: its lines or its parts"

func addresses(list []address) []model.Address {
	return convert(list, func(a address) model.Address { return model.Address(a) })
}

type status model.Status

func (s *status) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseStatus)
	if err != nil {
		return err
	}
	*s = status(value)
	return nil
}

type confidence model.Confidence

func (c *confidence) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseConfidence)
	if err != nil {
		return err
	}
	*c = confidence(value)
	return nil
}

type coordinates model.Coordinates

func (c *coordinates) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, parseCoordinates)
	if err != nil {
		return err
	}
	*c = coordinates(value)
	return nil
}

// parseCoordinates reads two angles, or a record's words in parentheses where it gave
// none that can be read as two.
func parseCoordinates(s string) (model.Coordinates, error) {
	if words, ok := strings.CutPrefix(strings.TrimSpace(s), "("); ok {
		words, closed := strings.CutSuffix(words, ")")
		wording := strings.Join(strings.Fields(words), " ")
		if !closed || wording == "" {
			return model.Coordinates{}, fmt.Errorf("coordinates %q: expected words in parentheses", s)
		}
		return model.Coordinates{Wording: wording}, nil
	}
	north, east, found := strings.Cut(s, ",")
	if !found {
		return model.Coordinates{}, fmt.Errorf("coordinates %q: expected latitude, longitude", s)
	}
	latitude, err := strconv.ParseFloat(strings.TrimSpace(north), 64)
	if err != nil || !(-90 <= latitude && latitude <= 90) {
		return model.Coordinates{}, fmt.Errorf("coordinates %q: %q is not a latitude", s, strings.TrimSpace(north))
	}
	longitude, err := strconv.ParseFloat(strings.TrimSpace(east), 64)
	if err != nil || !(-180 <= longitude && longitude <= 180) {
		return model.Coordinates{}, fmt.Errorf("coordinates %q: %q is not a longitude", s, strings.TrimSpace(east))
	}
	return model.Coordinates{Latitude: latitude, Longitude: longitude}, nil
}

type region model.Region

// UnmarshalYAML reads the four fractions in the order they are written. Whether they lie
// within the picture and the right way round is check's to say, since a region is read
// before the picture it marks is looked at.
func (r *region) UnmarshalYAML(node *yaml.Node) error {
	const shape = "a region is four fractions of the picture: [left, top, right, bottom]"
	if node.Kind != yaml.SequenceNode || len(node.Content) != 4 {
		return fmt.Errorf("line %d: %s", node.Line, shape)
	}
	var edges [4]float64
	for i, edge := range node.Content {
		value, err := scalar(edge, func(s string) (float64, error) {
			f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
			if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
				return 0, fmt.Errorf("%q is not a fraction: %s", s, shape)
			}
			return f, nil
		})
		if err != nil {
			return err
		}
		edges[i] = value
	}
	*r = region{Left: edges[0], Top: edges[1], Right: edges[2], Bottom: edges[3]}
	return nil
}

// The fields below carry the scalar-or-qualified pattern: a bare scalar in the
// ordinary case, a mapping where the plain form would lose something. It is the
// format's one general escape hatch, and reading both forms is what keeps the common
// case short.

// statedSex is a sex as a letter, or qualified with what states it.
type statedSex model.StatedSex

func (s *statedSex) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Value   sex           `yaml:"value"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := qualified.Value.UnmarshalYAML(node); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "value", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
		if model.Sex(qualified.Value) == model.SexUnknown {
			return fmt.Errorf("line %d: %s", node.Line, uncitedSex)
		}
	default:
		return fmt.Errorf("line %d: a sex is a letter, or a mapping with value", node.Line)
	}
	*s = statedSex{Value: model.Sex(qualified.Value), Sources: citations(qualified.Sources)}
	return nil
}

// uncitedSex is why a sex the sources do not state cannot be cited: what would the
// citation be evidence of?
const uncitedSex = "a sex with sources is one the sources state, so its value is M, F or X"

// statedAge is the age a record gives as the age, or qualified with a note and what
// states it.
type statedAge model.StatedAge

func (a *statedAge) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Value   age           `yaml:"value"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := qualified.Value.UnmarshalYAML(node); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "value", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
		if qualified.Value == "" {
			return fmt.Errorf("line %d: %s", node.Line, unstatedAge)
		}
	default:
		return fmt.Errorf("line %d: an age is an age, or a mapping with value", node.Line)
	}
	*a = statedAge{Value: model.Age(qualified.Value), Note: qualified.Note, Sources: citations(qualified.Sources)}
	return nil
}

// statedCause is the cause a record gives as its words, or qualified with a note and
// what states it.
type statedCause model.StatedCause

func (c *statedCause) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Value   string        `yaml:"value"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Value = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "value", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
		if strings.TrimSpace(qualified.Value) == "" {
			return fmt.Errorf("line %d: %s", node.Line, unstatedCause)
		}
	default:
		return fmt.Errorf("line %d: a cause is text, or a mapping with value", node.Line)
	}
	*c = statedCause{Value: qualified.Value, Note: qualified.Note, Sources: citations(qualified.Sources)}
	return nil
}

// An age or a cause said something of, or cited, is one the record gives: what would
// the note be about, or the citation evidence of?
const (
	unstatedAge   = "an age with a note or sources has its value"
	unstatedCause = "a cause with a note or sources has its value"
)

// attribute is something a record says that no field holds: a mapping, because its
// type is what says what the value is.
type attribute model.Attribute

func (a *attribute) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: an attribute is a mapping with type", node.Line)
	}
	if err := onlyFields(node, "type", "value", "date", "note", "sources"); err != nil {
		return err
	}
	var fields struct {
		Type    string        `yaml:"type"`
		Value   string        `yaml:"value"`
		Date    date          `yaml:"date"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	if strings.TrimSpace(fields.Type) == "" {
		return fmt.Errorf("line %d: %s", node.Line, untypedAttribute)
	}
	*a = attribute{
		Type:    fields.Type,
		Value:   fields.Value,
		Date:    model.Date(fields.Date),
		Note:    fields.Note,
		Sources: citations(fields.Sources),
	}
	return nil
}

const untypedAttribute = "an attribute has a type, which says what its value is"

func attributes(list []attribute) []model.Attribute {
	return convert(list, func(a attribute) model.Attribute { return model.Attribute(a) })
}

// nonEvent is something that did not happen: a mapping with the type of event it was
// not, over its date or ever.
type nonEvent model.NonEvent

func (n *nonEvent) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: what did not happen is a mapping with type", node.Line)
	}
	if err := onlyFields(node, "type", "date", "note", "sources"); err != nil {
		return err
	}
	var fields struct {
		Type    string        `yaml:"type"`
		Date    date          `yaml:"date"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	if strings.TrimSpace(fields.Type) == "" {
		return fmt.Errorf("line %d: %s", node.Line, untypedNonEvent)
	}
	*n = nonEvent{
		Type:    fields.Type,
		Date:    model.Date(fields.Date),
		Note:    fields.Note,
		Sources: citations(fields.Sources),
	}
	return nil
}

const untypedNonEvent = "what did not happen has the type of event it was not"

func nonEvents(list []nonEvent) []model.NonEvent {
	return convert(list, func(n nonEvent) model.NonEvent { return model.NonEvent(n) })
}

// image is a picture beside a record as its path, or the path with the region of it
// that matters: where on a scan the entry is, or who in a photograph a portrait is.
type image model.Image

func (i *image) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		File   string  `yaml:"file"`
		Region *region `yaml:"region"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.File = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "file", "region"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: an image is a path, or a mapping with file", node.Line)
	}
	if strings.TrimSpace(qualified.File) == "" {
		return fmt.Errorf("line %d: an image names the file it is", node.Line)
	}
	*i = image{File: qualified.File, Region: (*model.Region)(qualified.Region)}
	return nil
}

// depicted is somebody a picture shows as their slug, or the slug with the part of the
// picture that is them.
type depicted model.Depicted

func (d *depicted) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Person     string      `yaml:"person"`
		Region     *region     `yaml:"region"`
		Attributes []attribute `yaml:"attributes"`
		Note       string      `yaml:"note"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Person = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "person", "region", "attributes", "note"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: somebody a picture shows is a slug, or a mapping with person", node.Line)
	}
	person, err := model.ParseSlug(qualified.Person)
	if err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*d = depicted{
		Person:     person,
		Region:     (*model.Region)(qualified.Region),
		Attributes: attributes(qualified.Attributes),
		Note:       qualified.Note,
	}
	return nil
}

// media is what one of a record's files is: its path, or a mapping of the path and what
// is said of the file.
type media model.Media

func (m *media) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		Private    flagged       `yaml:"private"`
		File       string        `yaml:"file"`
		Type       string        `yaml:"type"`
		Title      string        `yaml:"title"`
		Date       date          `yaml:"date"`
		Place      slug          `yaml:"place"`
		Region     *region       `yaml:"region"`
		Depicts    []depicted    `yaml:"depicts"`
		Attributes []attribute   `yaml:"attributes"`
		Note       string        `yaml:"note"`
		Sources    []citationRef `yaml:"sources"`
		Tags       []tag         `yaml:"tags"`
		IDs        ids           `yaml:"ids"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		fields.File = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "private", "file", "type", "title", "date", "place", "region", "depicts",
			"attributes", "note", "sources", "tags", "ids"); err != nil {
			return err
		}
		if err := node.Decode(&fields); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a media entry is a path, or a mapping with file", node.Line)
	}
	if strings.TrimSpace(fields.File) == "" {
		return fmt.Errorf("line %d: %s", node.Line, unnamedMedia)
	}
	*m = media{
		Private:    bool(fields.Private),
		File:       fields.File,
		Type:       fields.Type,
		Title:      fields.Title,
		Date:       model.Date(fields.Date),
		Place:      model.Slug(fields.Place),
		Region:     (*model.Region)(fields.Region),
		Depicts:    convert(fields.Depicts, func(d depicted) model.Depicted { return model.Depicted(d) }),
		Attributes: attributes(fields.Attributes),
		Note:       fields.Note,
		Sources:    citations(fields.Sources),
		Tags:       tags(fields.Tags),
		IDs:        model.IDs(fields.IDs),
	}
	if model.Media(*m).Refers() && model.Media(*m).Says() {
		return fmt.Errorf("line %d: %s", node.Line, referringMedia)
	}
	return nil
}

// unnamedMedia is why a media entry has to name its file: it says what a file is, and
// without one it is about nothing.
const unnamedMedia = "a media entry names the file it is about"

// referringMedia is why an entry whose path leads into another record's directory says
// so little: what the picture is, and who is in it, is said once, by the record keeping
// it.
const referringMedia = "an entry referring to a picture another record keeps says only its own " +
	"region, attributes, note, sources and private, the record keeping it saying what the picture is"

func mediaEntries(list []media) []model.Media {
	return convert(list, func(m media) model.Media { return model.Media(m) })
}

// child is somebody a family lists: a slug, or a mapping of id or name, what they were
// to the partners, how well the link stands, a note and its sources.
type child model.Child

func (c *child) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		ID       slug          `yaml:"id"`
		Name     unflaggedName `yaml:"name"`
		Relation relationship  `yaml:"relation"`
		Status   linkStatus    `yaml:"status"`
		Note     string        `yaml:"note"`
		Sources  []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := node.Decode(&fields.ID); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "id", "name", "relation", "status", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&fields); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a child is a slug, or a mapping with id or name", node.Line)
	}
	one := model.Child{
		ID:       model.Slug(fields.ID),
		Name:     model.PersonName(fields.Name),
		Relation: model.Relationship(fields.Relation),
		Status:   model.LinkStatus(fields.Status),
		Note:     fields.Note,
		Sources:  citations(fields.Sources),
	}
	if err := whoChild(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*c = child(one)
	return nil
}

// whoChild is the rule for a child, which the emitter holds to as well: a record,
// unknown, or a name, and never a name beside a slug.
func whoChild(c model.Child) error {
	switch {
	case c.ID != "" && !c.Name.IsZero():
		return fmt.Errorf("child %s takes no name beside it", c.ID)
	case c.ID == "" && c.Name.IsZero():
		return errors.New("a child is a person, unknown, or a name")
	case c.ID == model.None:
		return errors.New("a child is a person, unknown, or a name, never none")
	case c.Relation.Word != "" && c.Relation.Partners != nil:
		return errors.New("a child's relation is one word or one per partner, not both")
	}
	return nil
}

// relationship is a child's relation, one word for every partner or a mapping from each
// partner to theirs, kept in the order it is written.
type relationship model.Relationship

func (r *relationship) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		word, err := relationWord(node)
		r.Word = word
		return err
	case yaml.MappingNode:
		if len(node.Content) == 0 {
			return fmt.Errorf("line %d: a relation by partner names a partner", node.Line)
		}
		for i := 0; i < len(node.Content); i += 2 {
			partner, err := model.ParseSlug(node.Content[i].Value)
			if err != nil {
				return fmt.Errorf("line %d: %w", node.Content[i].Line, err)
			}
			word, err := relationWord(node.Content[i+1])
			if err != nil {
				return err
			}
			r.Partners = append(r.Partners, model.PartnerRelation{Partner: partner, Relation: word})
		}
		return nil
	}
	return fmt.Errorf("line %d: a relation is a word, or a mapping from each partner to one", node.Line)
}

func relationWord(node *yaml.Node) (model.Relation, error) {
	if node.Kind != yaml.ScalarNode || strings.TrimSpace(node.Value) == "" {
		return "", fmt.Errorf("line %d: a relation is a word", node.Line)
	}
	return model.Relation(node.Value), nil
}

type linkStatus model.LinkStatus

func (s *linkStatus) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseLinkStatus)
	if err != nil {
		return err
	}
	*s = linkStatus(value)
	return nil
}

// personNameParts are the fields of a name in parts, in the order they are written.
var personNameParts = []string{"prefix", "given", "call", "nickname", "family-nickname",
	"surname-prefix", "surname", "suffix", "group"}

type personName model.PersonName

func (n *personName) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		if strings.TrimSpace(node.Value) == "" {
			return fmt.Errorf("line %d: a name cannot be empty", node.Line)
		}
		*n = personName{Text: node.Value}
		return nil
	case yaml.MappingNode:
	default:
		return fmt.Errorf("line %d: a name is text, or a mapping of its parts", node.Line)
	}
	fields := append(append([]string{"private", "text"}, personNameParts...),
		"lang", "type", "display", "sort", "from", "until", "note", "sources")
	if err := onlyFields(node, fields...); err != nil {
		return err
	}
	var parts struct {
		Private        flagged       `yaml:"private"`
		Text           string        `yaml:"text"`
		Prefix         string        `yaml:"prefix"`
		Given          string        `yaml:"given"`
		Call           string        `yaml:"call"`
		Nickname       string        `yaml:"nickname"`
		FamilyNickname string        `yaml:"family-nickname"`
		SurnamePrefix  string        `yaml:"surname-prefix"`
		Surname        surnames      `yaml:"surname"`
		Suffix         string        `yaml:"suffix"`
		Group          string        `yaml:"group"`
		Lang           language      `yaml:"lang"`
		Type           string        `yaml:"type"`
		Display        string        `yaml:"display"`
		Sort           string        `yaml:"sort"`
		From           date          `yaml:"from"`
		Until          date          `yaml:"until"`
		Note           string        `yaml:"note"`
		Sources        []citationRef `yaml:"sources"`
	}
	if err := node.Decode(&parts); err != nil {
		return err
	}
	name := model.PersonName{
		Private:        bool(parts.Private),
		Text:           parts.Text,
		Prefix:         parts.Prefix,
		Given:          parts.Given,
		Call:           parts.Call,
		Nickname:       parts.Nickname,
		FamilyNickname: parts.FamilyNickname,
		SurnamePrefix:  parts.SurnamePrefix,
		Surname:        model.Surnames(convert(parts.Surname, func(s surname) model.Surname { return model.Surname(s) })),
		Suffix:         parts.Suffix,
		Group:          parts.Group,
		Lang:           model.Language(parts.Lang),
		Type:           parts.Type,
		Display:        parts.Display,
		Sort:           parts.Sort,
		From:           model.Date(parts.From),
		Until:          model.Date(parts.Until),
		Note:           parts.Note,
		Sources:        citations(parts.Sources),
	}
	if strings.TrimSpace(name.Text) == "" && !name.HasParts() {
		return fmt.Errorf("line %d: a name has its text or at least one of %s",
			node.Line, strings.Join(personNameParts, ", "))
	}
	*n = personName(name)
	return nil
}

// surnames accepts the scalar-or-qualified pattern at the level of the field as well as
// the entry, so that `surname: Rossi` and a list of two read.
type surnames []surname

func (s *surnames) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		var entries []surname
		if err := node.Decode(&entries); err != nil {
			return err
		}
		*s = entries
		return nil
	}
	var one surname
	if err := one.UnmarshalYAML(node); err != nil {
		return err
	}
	*s = []surname{one}
	return nil
}

type surname model.Surname

func (s *surname) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Name      string  `yaml:"name"`
		Prefix    string  `yaml:"prefix"`
		Connector string  `yaml:"connector"`
		Origin    string  `yaml:"origin"`
		Primary   flagged `yaml:"primary"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Name = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "name", "prefix", "connector", "origin", "primary"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a surname is text, or a mapping with name", node.Line)
	}
	if strings.TrimSpace(qualified.Name) == "" {
		return fmt.Errorf("line %d: a surname cannot be empty", node.Line)
	}
	*s = surname{
		Name:      qualified.Name,
		Prefix:    qualified.Prefix,
		Connector: qualified.Connector,
		Origin:    qualified.Origin,
		Primary:   bool(qualified.Primary),
	}
	return nil
}

// unflaggedName is a name that stands for somebody rather than being one more name they
// were known by: the one a person is filed under, and the one an event gives somebody
// with no record. Neither takes a flag, the emitter holding to it as well.
type unflaggedName personName

func (n *unflaggedName) UnmarshalYAML(node *yaml.Node) error {
	var name personName
	if err := name.UnmarshalYAML(node); err != nil {
		return err
	}
	if name.Private {
		return fmt.Errorf("line %d: %w", node.Line, errFlaggedName)
	}
	*n = unflaggedName(name)
	return nil
}

var errFlaggedName = errors.New("only a name in also-known-as is private: the name a person " +
	"is filed under is hidden with them, and a name an event gives with the event")

// named is somebody an event names: a mapping, because the role is what says why they
// are there, holding their record's slug or their name and never both.
type named model.Named

func (n *named) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: somebody an event names is a mapping with person or name, and role", node.Line)
	}
	if err := onlyFields(node, "person", "name", "role", "age", "attributes", "note", "sources"); err != nil {
		return err
	}
	var fields struct {
		Person     slug          `yaml:"person"`
		Name       unflaggedName `yaml:"name"`
		Role       string        `yaml:"role"`
		Age        age           `yaml:"age"`
		Attributes []attribute   `yaml:"attributes"`
		Note       string        `yaml:"note"`
		Sources    []citationRef `yaml:"sources"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	one := model.Named{
		Person:     model.Slug(fields.Person),
		Name:       model.PersonName(fields.Name),
		Role:       fields.Role,
		Age:        model.Age(fields.Age),
		Attributes: attributes(fields.Attributes),
		Note:       fields.Note,
		Sources:    citations(fields.Sources),
	}
	if err := whoIsNamed(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*n = named(one)
	return nil
}

// namedIn is somebody a citation's entry names, as an event names them but without
// sources of their own, the entry being what says so.
type namedIn named

func (n *namedIn) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.MappingNode {
		if err := onlyFields(node, "person", "name", "role", "age", "attributes", "note"); err != nil {
			return err
		}
	}
	return (*named)(n).UnmarshalYAML(node)
}

// coverage is what a source records: the events, over which period and for which place,
// a mapping of whichever of them is known and a note.
type coverage model.Coverage

func (c *coverage) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: what a source covers is a mapping of events, date and place", node.Line)
	}
	if err := onlyFields(node, "events", "date", "place", "note"); err != nil {
		return err
	}
	var fields struct {
		Events []string `yaml:"events"`
		Date   date     `yaml:"date"`
		Place  slug     `yaml:"place"`
		Note   string   `yaml:"note"`
	}
	if err := node.Decode(&fields); err != nil {
		return err
	}
	one := model.Coverage{
		Events: convert(fields.Events, func(s string) string { return s }),
		Date:   model.Date(fields.Date),
		Place:  model.Slug(fields.Place),
		Note:   fields.Note,
	}
	if err := coversSomething(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*c = coverage(one)
	return nil
}

// coversSomething is the rule for what a source covers, which the emitter holds to as
// well: an event it records, a period or a place, a note alone being about nothing, and
// no event without a word for it.
func coversSomething(c model.Coverage) error {
	if slices.ContainsFunc(c.Events, func(e string) bool { return strings.TrimSpace(e) == "" }) {
		return errors.New("an event a source covers is a word for it")
	}
	if len(c.Events) == 0 && c.Date.IsZero() && c.Place == "" {
		return errors.New("what a source covers is events, a date or a place")
	}
	return nil
}

// holding is where a source is kept: the repository's slug, or a mapping of the
// repository, the call number it is kept under there, the medium and a note. The
// repository may be left out where only the call number is known.
type holding model.Holding

func (h *holding) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		Repository slug   `yaml:"repository"`
		CallNumber string `yaml:"call-number"`
		Medium     string `yaml:"medium"`
		Note       string `yaml:"note"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := node.Decode(&fields.Repository); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "repository", "call-number", "medium", "note"); err != nil {
			return err
		}
		if err := node.Decode(&fields); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: where a source is kept is a slug, or a mapping with repository or call-number", node.Line)
	}
	one := model.Holding{
		Repository: model.Slug(fields.Repository),
		CallNumber: fields.CallNumber,
		Medium:     fields.Medium,
		Note:       fields.Note,
	}
	if one.Repository == "" && one.CallNumber == "" {
		return fmt.Errorf("line %d: %s", node.Line, nowhereKept)
	}
	*h = holding(one)
	return nil
}

// nowhereKept is why an entry of where a source is kept names a repository or a call
// number: a medium or a note alone does not say where.
const nowhereKept = "where a source is kept names a repository, a call number or both"

// whoIsNamed is the rule for somebody an event names, which the emitter holds to as
// well: a record, unknown or a name, since a name beside either would be a second
// account of who they were, and a role, since that is what the event says of them.
// Nobody at all is not somebody the record names.
func whoIsNamed(n model.Named) error {
	switch {
	case n.Person == model.None:
		return errors.New("somebody an event names is a person, unknown or a name, and never none")
	case n.Person != "" && !n.Name.IsZero():
		return fmt.Errorf("%s is named by their record, and takes no name beside it", n.Person)
	case n.Person == "" && n.Name.IsZero():
		return errors.New("somebody an event names has a person or a name")
	case strings.TrimSpace(n.Role) == "":
		return errors.New("somebody an event names has a role: godfather, witness, informant")
	}
	return nil
}

// partner is one of a family's partners: a slug, which is a person's record or one of
// the two words for nobody known and nobody at all, or a mapping holding the record or
// the name the record gives somebody nobody has a record of, and a note.
type partner model.Partner

func (p *partner) UnmarshalYAML(node *yaml.Node) error {
	var fields struct {
		Person slug          `yaml:"person"`
		Name   unflaggedName `yaml:"name"`
		Note   string        `yaml:"note"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		if err := node.Decode(&fields.Person); err != nil {
			return err
		}
	case yaml.MappingNode:
		if err := onlyFields(node, "person", "name", "note"); err != nil {
			return err
		}
		if err := node.Decode(&fields); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a partner is a slug, or a mapping with person or name", node.Line)
	}
	one := model.Partner{
		Person: model.Slug(fields.Person),
		Name:   model.PersonName(fields.Name),
		Note:   fields.Note,
	}
	if err := whoPartners(one); err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*p = partner(one)
	return nil
}

// whoPartners is the rule for a partner, which the emitter holds to as well: a record
// or a word, or a name, and never a name beside either, which would be a second account
// of who they were.
func whoPartners(p model.Partner) error {
	switch {
	case p.Person != "" && !p.Name.IsZero():
		return fmt.Errorf("partner %s takes no name beside it", p.Person)
	case p.Person == "" && p.Name.IsZero():
		return errors.New("a partner is a person, unknown or none, or a name")
	}
	return nil
}

type language model.Language

func (l *language) UnmarshalYAML(node *yaml.Node) error {
	value, err := scalar(node, model.ParseLanguage)
	if err != nil {
		return err
	}
	*l = language(value)
	return nil
}

type placeName model.PlaceName

const unnamedPlace = "a name with a language, a type, dates, a note or sources has its words"

func (n *placeName) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Name    string        `yaml:"name"`
		Lang    language      `yaml:"lang"`
		Type    string        `yaml:"type"`
		From    date          `yaml:"from"`
		Until   date          `yaml:"until"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Name = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "name", "lang", "type", "from", "until", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a name is text, or a mapping with name", node.Line)
	}
	if strings.TrimSpace(qualified.Name) == "" {
		return fmt.Errorf("line %d: a name cannot be empty", node.Line)
	}
	*n = placeName{
		Text:    qualified.Name,
		Lang:    model.Language(qualified.Lang),
		Type:    qualified.Type,
		From:    model.Date(qualified.From),
		Until:   model.Date(qualified.Until),
		Note:    qualified.Note,
		Sources: citations(qualified.Sources),
	}
	return nil
}

type placeRef model.PlaceRef

func (p *placeRef) UnmarshalYAML(node *yaml.Node) error {
	var qualified struct {
		Place   string        `yaml:"place"`
		From    date          `yaml:"from"`
		Until   date          `yaml:"until"`
		Note    string        `yaml:"note"`
		Sources []citationRef `yaml:"sources"`
	}
	switch node.Kind {
	case yaml.ScalarNode:
		qualified.Place = node.Value
	case yaml.MappingNode:
		if err := onlyFields(node, "place", "from", "until", "note", "sources"); err != nil {
			return err
		}
		if err := node.Decode(&qualified); err != nil {
			return err
		}
	default:
		return fmt.Errorf("line %d: a containing place is a slug, or a mapping with place", node.Line)
	}

	place, err := model.ParseSlug(qualified.Place)
	if err != nil {
		return fmt.Errorf("line %d: %w", node.Line, err)
	}
	*p = placeRef{
		Place:   place,
		From:    model.Date(qualified.From),
		Until:   model.Date(qualified.Until),
		Note:    qualified.Note,
		Sources: citations(qualified.Sources),
	}
	return nil
}

// within accepts the scalar-or-qualified pattern at the level of the field as well as
// the entry, so that `within: toscana` and a dated list both read.
type within []placeRef

func (w *within) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.SequenceNode {
		var entries []placeRef
		if err := node.Decode(&entries); err != nil {
			return err
		}
		*w = entries
		return nil
	}
	var one placeRef
	if err := one.UnmarshalYAML(node); err != nil {
		return err
	}
	*w = []placeRef{one}
	return nil
}

func places(refs []placeRef) []model.PlaceRef {
	return convert(refs, func(r placeRef) model.PlaceRef { return model.PlaceRef(r) })
}

func partners(list []partner) []model.Partner {
	return convert(list, func(p partner) model.Partner { return model.Partner(p) })
}

func children(list []child) []model.Child {
	return convert(list, func(c child) model.Child { return model.Child(c) })
}

func personNames(refs []personName) []model.PersonName {
	return convert(refs, func(r personName) model.PersonName { return model.PersonName(r) })
}

func placeNames(refs []placeName) []model.PlaceName {
	return convert(refs, func(r placeName) model.PlaceName { return model.PlaceName(r) })
}

func citations(refs []citationRef) []model.CitationRef {
	return convert(refs, func(r citationRef) model.CitationRef { return model.CitationRef(r) })
}

// convert maps a slice of wire types onto the model. An empty sequence becomes a nil
// slice, because `also-known-as: []` and no field at all say the same thing and the
// emitter has to be able to write only one of them.
func convert[In, Out any](in []In, one func(In) Out) []Out {
	if len(in) == 0 {
		return nil
	}
	out := make([]Out, len(in))
	for i, v := range in {
		out[i] = one(v)
	}
	return out
}
