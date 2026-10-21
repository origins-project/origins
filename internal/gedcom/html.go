package gedcom

import (
	"regexp"
	"strings"
)

// markdown writes a note given as text/html as markdown. Markup markdown has a mark for
// becomes the mark — bold, italic, struck through, a link, a paragraph, a line break —
// and what it has none for stays in it as HTML, as markdown allows: <u>, <sup>, <sub> and
// any tag origins does not know, written as the file has it, so that nothing the note
// said is lost and browse decides what it shows. Whitespace is collapsed as 7.0 says
// HTML's is, and entities stay as written, markdown reading them as HTML does.
func markdown(html string) string {
	var b strings.Builder
	var hrefs []string
	text := func(s string) { b.WriteString(space.ReplaceAllString(s, " ")) }
	for html != "" {
		open := strings.IndexByte(html, '<')
		if open < 0 {
			text(html)
			break
		}
		closing := strings.IndexByte(html[open:], '>')
		if closing < 0 {
			text(html)
			break
		}
		text(html[:open])
		tag := html[open : open+closing+1]
		html = html[open+closing+1:]

		end, name := tagName(tag)
		switch name {
		case "p":
			b.WriteString("\n\n")
		case "br":
			b.WriteString("\\\n")
		case "b", "strong":
			b.WriteString("**")
		case "i", "em":
			b.WriteString("*")
		case "s", "strike", "del":
			b.WriteString("~~")
		case "u", "sup", "sub", "mark":
			if end {
				b.WriteString("</" + name + ">")
			} else {
				b.WriteString("<" + name + ">")
			}
		case "a":
			if !end {
				address := href.FindStringSubmatch(tag)
				if address == nil {
					b.WriteString(tag)
					hrefs = append(hrefs, "")
					continue
				}
				b.WriteString("[")
				hrefs = append(hrefs, address[1]+address[2]+address[3])
				continue
			}
			if len(hrefs) == 0 {
				b.WriteString(tag)
				continue
			}
			dest := hrefs[len(hrefs)-1]
			hrefs = hrefs[:len(hrefs)-1]
			if dest == "" {
				b.WriteString(tag)
			} else {
				b.WriteString("](" + destination(dest) + ")")
			}
		default:
			b.WriteString(tag)
		}
	}
	return paragraphs(b.String())
}

var (
	// href is a link's address, in one of the three groups however it is quoted
	href  = regexp.MustCompile(`(?i)\bhref\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	space = regexp.MustCompile(`[ \t\r\n]+`)
)

// tagName reads a tag: whether it closes an element, and the element's name in lower
// case, "" where it is no element at all — a comment, a declaration.
func tagName(tag string) (bool, string) {
	inner := strings.TrimSpace(tag[1 : len(tag)-1])
	end := strings.HasPrefix(inner, "/")
	inner = strings.TrimPrefix(inner, "/")
	n := 0
	for n < len(inner) && (inner[n] >= 'a' && inner[n] <= 'z' || inner[n] >= 'A' && inner[n] <= 'Z' || n > 0 && inner[n] >= '0' && inner[n] <= '9') {
		n++
	}
	return end, strings.ToLower(inner[:n])
}

// destination writes an address as a link's destination, in angle brackets where it has
// what would end one otherwise.
func destination(address string) string {
	if strings.ContainsAny(address, " ()<>") {
		return "<" + strings.NewReplacer("<", "%3C", ">", "%3E").Replace(address) + ">"
	}
	return address
}

// paragraphs tidies what markdown wrote: no line begins or ends with a space, and
// paragraphs are a blank line apart, however many the HTML opened and closed between
// them.
func paragraphs(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimLeft(line, " ")
		if !strings.HasSuffix(lines[i], "\\") {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
	}
	text = strings.Join(lines, "\n")
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	return strings.Trim(text, "\n")
}

// plain is a note given as text/html as plain text, the way 7.0 says to make it, for
// what has to be a line of words: a note record's title.
func plain(html string) string {
	html = strings.Join(strings.Fields(html), " ")
	html = breaks.ReplaceAllString(html, "\n")
	html = tags.ReplaceAllString(html, "")
	return strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&").Replace(html)
}

var (
	breaks = regexp.MustCompile(`(?i)</?(?:p|br)\b[^>]*>`)
	tags   = regexp.MustCompile(`<[^>]*>`)
)
