package gedcom

import (
	"net/url"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// ids reads the identifiers a record has elsewhere: 7.0's EXID, under the service its
// TYPE is the address of; REFN, a number somebody chose and may be filing their papers
// under, under the name its TYPE gives or as refn; _FSFTID, the identifier in
// FamilySearch's Family Tree, which exports from there and from Ancestry write; AFN, its
// number in the Ancestral File that came before it, under afn; UID and its older spelling
// _UID, under uid; and RIN, the number the exporting program filed the record under,
// under rin. A 7.0 event has a UID too, and is read the same way.
func (c *converter) ids(record *Node) model.IDs {
	var ids model.IDs
	for _, node := range record.Children {
		var name string
		switch node.Tag {
		case "EXID":
			name = service(node.First("TYPE").Text(), "exid")
		case "REFN":
			name = service(node.First("TYPE").Text(), "refn")
		case "_FSFTID":
			name = "familysearch"
		case "AFN":
			name = "afn"
		case "UID", "_UID":
			name = "uid"
		case "RIN":
			name = "rin"
		default:
			continue
		}
		var kept bool
		if ids, kept = identify(ids, name, strings.TrimSpace(node.Text())); kept {
			node.Mark().Take("TYPE")
		}
	}
	return ids
}

// service names what an identifier is kept by, as the corpus writes the name of one. An
// EXID's TYPE is an address, https://www.familysearch.org/tree/person/, and the service
// is the name the address is under; anything else is named by its own words.
func service(named, otherwise string) string {
	if address, err := url.Parse(strings.TrimSpace(named)); err == nil && address.Hostname() != "" {
		named, _, _ = strings.Cut(strings.TrimPrefix(address.Hostname(), "www."), ".")
	}
	if slug, err := model.Slugify(named); err == nil {
		return string(slug)
	}
	return otherwise
}

// identify adds an identifier, reporting whether the record holds it now. Where a record
// gives two at the same service the first is kept and the second is not: a list of them
// says which the service merged away, and two given side by side say nothing of that.
func identify(ids model.IDs, service, id string) (model.IDs, bool) {
	if id == "" {
		return ids, false
	}
	if ids == nil {
		ids = model.IDs{}
	}
	switch existing, taken := ids[service]; {
	case !taken:
		ids[service] = model.IdentifiersOf(id)
	case existing[0].ID != id:
		return ids, false
	}
	return ids, true
}
