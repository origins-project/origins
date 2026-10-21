package importing

import (
	"fmt"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Written is a node of the tree an export was parsed into, which the importer marks as
// it reads: a GEDCOM line with the lines under it, or an XML element.
type Written[N any] interface {
	comparable
	// Parts is what is under the node, in the order written.
	Parts() []N
	// Taken reports whether the importer read the node.
	Taken() bool
	// Loose labels the attributes on the node the importer did not read, as the report
	// counts them.
	Loose() []string
	// Label is what the node is, as the report counts it.
	Label() string
	// Opening is the node's own text as written, and Closing what closes it, each ending
	// in a newline; Whole is the node and everything under it as written.
	Opening() string
	Closing() string
	Whole() string
}

// Home is a piece of prose in the corpus: the record holding it and the field path
// model.Prose hands it out at, or, with no record, a researcher's note in the manifest.
type Home struct {
	Record model.Slug
	Path   string
}

// Keeper writes what an import did not read into the corpus, as it was written.
//
// Whatever no code reads is kept verbatim, with everything under it, in a fenced block
// in the note of the nearest thing above it that takes one, which the importer names
// as a home while it reads, and otherwise in the body of the record it is in. A record
// that is not read at all, or that made nothing in the corpus, becomes a note record
// of its own. So an import loses nothing whatever a program adds to its export, and
// what nobody has decided how to read is still there to be read.
type Keeper[N Written[N]] struct {
	language string
	homes    map[N][]Home
}

// NewKeeper keeps what an export did not read in blocks marked as language.
func NewKeeper[N Written[N]](language string) *Keeper[N] {
	return &Keeper[N]{language: language, homes: map[N][]Home{}}
}

// Home names the prose at path on record as where what is unread under node goes. A
// node may have several homes, where what it says was read into several records, and
// a home given twice is one.
func (k *Keeper[N]) Home(node N, record model.Slug, path string) {
	home := Home{Record: record, Path: path}
	for _, given := range k.homes[node] {
		if given == home {
			return
		}
	}
	k.homes[node] = append(k.homes[node], home)
}

// Keep walks roots, the records of the export in the order written, and writes every
// node left unread into the corpus, counting each in notes by its label. A root with no
// home of its own and something unread in it becomes a note record titled as title
// says, named through namer.
func (k *Keeper[N]) Keep(corpus *model.Corpus, namer *Namer, notes *Notes, roots []N, title func(N) string) {
	w := &walk[N]{keeper: k, notes: notes, blocks: map[Home][]string{}}
	for _, root := range roots {
		homes := k.homes[root]
		w.lazy = nil
		if len(homes) == 0 {
			// a note record is made only once something turns out to need one
			var made []Home
			w.lazy = func() []Home {
				if made == nil {
					made = []Home{{Record: k.note(corpus, namer, title(root)), Path: "body"}}
				}
				return made
			}
		}
		if !root.Taken() {
			w.count(root)
			w.put(homes, root.Whole())
			continue
		}
		w.visit(root, homes, nil)
	}
	w.write(corpus, namer)
}

// note makes an empty note record to keep what a root of the export left unread.
func (k *Keeper[N]) note(corpus *model.Corpus, namer *Namer, title string) model.Slug {
	base, err := model.Slugify(title)
	if err != nil {
		base = "note"
	}
	slug := namer.Assign(Shorten(base), nil, title)
	if slug == "" {
		slug = namer.Assign("note", nil, title)
	}
	corpus.Notes[slug] = &model.Note{Slug: slug, Title: title}
	return slug
}

type walk[N Written[N]] struct {
	keeper *Keeper[N]
	notes  *Notes
	lazy   func() []Home

	order  []Home
	blocks map[Home][]string
}

// visit writes what is unread under node, and under each part of it that was read,
// into homes, or into the nearest home of node's own. context is every node between
// the home and this one, whose openings and closings frame what is kept so that it
// says where it was.
func (w *walk[N]) visit(node N, homes []Home, context []N) {
	if own := w.keeper.homes[node]; len(own) > 0 {
		homes, context = own, nil
	} else {
		context = append(context[:len(context):len(context)], node)
	}

	var unread []N
	for _, part := range node.Parts() {
		if !part.Taken() {
			unread = append(unread, part)
		}
	}
	loose := node.Loose()
	if len(unread) > 0 || len(loose) > 0 {
		frame := context
		if len(loose) > 0 && (len(frame) == 0 || frame[len(frame)-1] != node) {
			frame = append(frame[:len(frame):len(frame)], node)
		}
		var text strings.Builder
		for _, n := range frame {
			text.WriteString(n.Opening())
		}
		for _, part := range unread {
			w.count(part)
			text.WriteString(part.Whole())
		}
		for i := len(frame) - 1; i >= 0; i-- {
			text.WriteString(frame[i].Closing())
		}
		for _, label := range loose {
			w.notes.Add("%s was kept as written", label)
		}
		w.put(homes, text.String())
	}

	for _, part := range node.Parts() {
		if part.Taken() {
			w.visit(part, homes, context)
		}
	}
}

func (w *walk[N]) count(node N) { w.notes.Add("%s was kept as written", node.Label()) }

func (w *walk[N]) put(homes []Home, text string) {
	if len(homes) == 0 && w.lazy != nil {
		homes = w.lazy()
	}
	block := fence(w.keeper.language, dedent(text))
	for _, home := range homes {
		if _, ok := w.blocks[home]; !ok {
			w.order = append(w.order, home)
		}
		if !slices.Contains(w.blocks[home], block) {
			w.blocks[home] = append(w.blocks[home], block)
		}
	}
}

// write appends the blocks to the prose they belong in, a record's body standing in
// for a path it does not have and a note record for a record the corpus does not have.
func (w *walk[N]) write(corpus *model.Corpus, namer *Namer) {
	var records []model.Slug
	homes := map[model.Slug][]Home{}
	for _, home := range w.order {
		if homes[home.Record] == nil {
			records = append(records, home.Record)
		}
		homes[home.Record] = append(homes[home.Record], home)
	}
	for _, record := range records {
		if record == "" {
			manifest(&corpus.Project, func(path string, text *string) {
				for _, home := range homes[record] {
					if home.Path == path {
						*text = appended(*text, w.blocks[home])
					}
				}
			})
			continue
		}
		entity := corpus.Entity(record)
		if entity == nil {
			entity = corpus.Notes[w.keeper.note(corpus, namer, string(record))]
		}
		written := map[string]bool{}
		model.Prose(entity, func(path string, text *string) {
			var more []string
			for _, home := range homes[record] {
				// the body comes last, and takes whatever had no prose of its own
				if home.Path == path || path == "body" && !written[home.Path] {
					more = append(more, w.blocks[home]...)
				}
			}
			if len(more) > 0 {
				*text = appended(*text, more)
			}
			written[path] = true
		})
	}
}

// manifest hands out the prose the manifest holds as model.Prose does a record's: each
// researcher's note, at researchers[i].note.
func manifest(project *model.Project, each func(path string, text *string)) {
	for i := range project.Researchers {
		each(fmt.Sprintf("researchers[%d].note", i), &project.Researchers[i].Note)
	}
}

func appended(text string, blocks []string) string {
	parts := blocks
	if strings.TrimSpace(text) != "" {
		parts = append([]string{strings.TrimRight(text, "\n")}, blocks...)
	}
	return strings.Join(parts, "\n\n")
}

// fence marks text as a block of language, with a fence longer than any run of
// backticks inside it.
func fence(language, text string) string {
	longest, run := 0, 0
	for _, r := range text {
		if r == '`' {
			run++
			longest = max(longest, run)
		} else {
			run = 0
		}
	}
	marks := strings.Repeat("`", max(3, longest+1))
	return marks + language + "\n" + strings.TrimRight(text, "\n") + "\n" + marks
}

// dedent takes off the indentation every line shares, so that what was kept from deep
// inside a file starts at the margin.
func dedent(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	common := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if common < 0 || indent < common {
			common = indent
		}
	}
	if common <= 0 {
		return text
	}
	for i, line := range lines {
		if len(line) >= common {
			lines[i] = line[common:]
		} else {
			lines[i] = strings.TrimLeft(line, " \t")
		}
	}
	return strings.Join(lines, "\n")
}
