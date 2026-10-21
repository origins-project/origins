package gedcom

import "strings"

// private reads the RESN under node, reporting whether it makes what node is private.
// CONFIDENTIAL and PRIVACY both do, the two versions agreeing apart from 7.0 allowing a
// list of them. Only where the model can mark the thing read private is it asked, and
// the RESN is marked read only where every word of it is one origins carries, so that
// any other is kept as written.
func (c *converter) private(node *Node) bool {
	resn := node.First("RESN")
	if resn == nil {
		return false
	}
	private, known := false, true
	for _, word := range strings.Split(resn.Text(), ",") {
		switch strings.ToUpper(strings.TrimSpace(word)) {
		case "CONFIDENTIAL", "PRIVACY":
			private = true
		case "LOCKED", "":
		default:
			known = false
		}
	}
	if known {
		resn.Mark()
	}
	return private
}

// locked reports a RESN saying only that a record is not to be changed, which says
// nothing about who may see it, origins keeping a change in the history rather than
// refusing one.
func locked(resn *Node) bool {
	for _, word := range strings.Split(resn.Text(), ",") {
		if !strings.EqualFold(strings.TrimSpace(word), "LOCKED") {
			return false
		}
	}
	return true
}
