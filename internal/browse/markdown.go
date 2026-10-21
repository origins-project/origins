package browse

import (
	"bytes"
	"html"
	"regexp"
	"slices"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"

	"github.com/origins-project/origins/internal/model"
)

// markdown renders prose as GitHub-flavoured markdown.
//
// Two things cannot be left to goldmark. A relative link is written from the record's
// own directory, where ../../people/rossi-giovanni/README.md is the file of a record
// and photos/1890-portrait.jpg a file beside it, and the browser is looking at neither.
// And HTML in prose is kept only where it is one of the tags the format lists, so that
// a body cannot run a script.
type markdown struct {
	converter goldmark.Markdown
}

func newMarkdown() *markdown {
	return &markdown{converter: goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(),
			renderer.WithNodeRenderers(util.Prioritized(&rawHTML{}, 100)),
		),
	)}
}

// render is prose written on the record owner as HTML.
func (m *markdown) render(owner model.EntityRef, prose string) string {
	if strings.TrimSpace(prose) == "" {
		return ""
	}
	source := []byte(prose)
	document := m.converter.Parser().Parse(text.NewReader(source))
	fitToPage(document, owner)

	var out bytes.Buffer
	if err := m.converter.Renderer().Render(&out, source, document); err != nil {
		// goldmark's renderer fails only on a failing writer, and a bytes.Buffer is not
		// one; the prose unrendered still beats nothing
		return html.EscapeString(prose)
	}
	return out.String()
}

// fitToPage adjusts prose written as a document of its own to sit inside a page.
//
// A relative destination becomes the route of the record or the file it leads to;
// an absolute URL, a path from the root and a fragment are left exactly as written.
// And every heading drops a level, because the page already carries the record's name
// as its first heading.
func fitToPage(document ast.Node, owner model.EntityRef) {
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Image:
			n.Destination = routed(n.Destination, owner)
		case *ast.Link:
			n.Destination = routed(n.Destination, owner)
		case *ast.Heading:
			n.Level = min(n.Level+1, 6)
		}
		return ast.WalkContinue, nil
	})
}

func routed(destination []byte, owner model.EntityRef) []byte {
	link, ok := model.Leads(owner, string(destination))
	if !ok || link.Record.Slug == "" {
		return destination
	}
	to := RecordRoute(link.Record)
	if link.File != "" {
		to = FileRoute(link.Record, link.File)
	}
	if link.Anchor != "" {
		to += "#" + link.Anchor
	}
	return util.URLEscape([]byte(to), true)
}

// RecordRoute is where the application shows a record.
func RecordRoute(ref model.EntityRef) string {
	return "/" + ref.String()
}

// FileRoute is where a file attached to a record is served.
func FileRoute(ref model.EntityRef, file string) string {
	return "/files/" + ref.String() + "/" + strings.TrimPrefix(file, "/")
}

// rawHTML renders the HTML written in prose, keeping the tags the format lists and
// showing any other as its text.
type rawHTML struct{}

func (r *rawHTML) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindRawHTML, r.inline)
	reg.Register(ast.KindHTMLBlock, r.block)
}

func (r *rawHTML) inline(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkSkipChildren, nil
	}
	n := node.(*ast.RawHTML)
	var written bytes.Buffer
	for i := 0; i < n.Segments.Len(); i++ {
		segment := n.Segments.At(i)
		written.Write(segment.Value(source))
	}
	_, _ = w.WriteString(sanitised(written.String()))
	return ast.WalkSkipChildren, nil
}

func (r *rawHTML) block(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.HTMLBlock)
	if !entering {
		if n.HasClosure() {
			_, _ = w.WriteString(sanitised(string(n.ClosureLine.Value(source))))
		}
		return ast.WalkContinue, nil
	}
	var written bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		line := lines.At(i)
		written.Write(line.Value(source))
	}
	_, _ = w.WriteString(sanitised(written.String()))
	return ast.WalkContinue, nil
}

// kept are the tags prose may write as HTML, and the attributes each may carry.
var kept = map[string][]string{
	"u":    nil,
	"sup":  nil,
	"sub":  nil,
	"mark": nil,
	"pre":  nil,
	"span": {"style"},
}

var (
	tag       = regexp.MustCompile(`<(/?)([A-Za-z][A-Za-z0-9-]*)((?:\s+[^<>]*?)?)\s*(/?)>`)
	attribute = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9-]*)\s*=\s*("[^"]*"|'[^']*')`)
	anyTag    = regexp.MustCompile(`<[^>]*>?`)
)

// sanitised is HTML from prose with every tag the format does not list, and every
// attribute a listed one may not carry, shown as text.
func sanitised(written string) string {
	var out strings.Builder
	for len(written) > 0 {
		at := anyTag.FindStringIndex(written)
		if at == nil {
			out.WriteString(written)
			break
		}
		out.WriteString(written[:at[0]])
		one := written[at[0]:at[1]]
		if allowed(one) {
			out.WriteString(one)
		} else {
			out.WriteString(html.EscapeString(one))
		}
		written = written[at[1]:]
	}
	return out.String()
}

func allowed(one string) bool {
	parts := tag.FindStringSubmatch(one)
	if parts == nil || parts[0] != one {
		return false
	}
	closing, name, attributes := parts[1] == "/", strings.ToLower(parts[2]), parts[3]
	may, ok := kept[name]
	if !ok {
		return false
	}
	if closing {
		return strings.TrimSpace(attributes) == ""
	}
	for rest := attributes; strings.TrimSpace(rest) != ""; {
		found := attribute.FindStringSubmatch(rest)
		if found == nil || !slices.Contains(may, strings.ToLower(found[1])) {
			return false
		}
		rest = rest[len(found[0]):]
	}
	return true
}
