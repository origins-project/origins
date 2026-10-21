package api

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/origins-project/origins/internal/gedcom"
	"github.com/origins-project/origins/internal/gramps"
	"github.com/origins-project/origins/internal/history"
	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Imported is what an import brought in, and where it was recorded.
type Imported struct {
	Corpus  string   `json:"corpus"`
	Counts  Counts   `json:"counts"`
	Notes   []string `json:"notes"`
	Commit  string   `json:"commit"`
	History string   `json:"history"`
	Started bool     `json:"started"`

	// Subject is the subject line of the import's commit.
	Subject string `json:"-"`
}

// Importing is what an import is asked: the format and the file, where to write the
// corpus and what to call it, where else the pictures may be, and who chooses between
// two slugs, nil for origins to choose.
type Importing struct {
	Format string
	File   string
	Into   string
	Name   string
	Media  string
	Choose importing.Chooser
}

// Import reads a GEDCOM export or a Gramps file into a new corpus.
//
// The corpus is written fresh rather than merged into an existing one, and it is
// written canonically like everything else origins writes — there is no path by which
// bytes from another tool reach a corpus unread.
//
// What could not be carried is reported rather than passed over. An import that
// silently dropped a third of the file would look exactly like one that worked, and the
// only moment anyone can catch that is now, with the export still to hand.
//
// The whole of it lands as one commit, thirty thousand people and all, because a tree
// arriving from elsewhere is one act of research and not thirty thousand. A corpus
// written and not recorded comes back with its report and an Unrecorded beside it.
func Import(asked Importing) (report *Imported, err error) {
	if asked.Format != "gedcom" && asked.Format != "gramps" {
		return nil, fmt.Errorf("cannot import %q: origins reads gedcom and gramps", asked.Format)
	}
	if asked.Media != "" {
		if info, err := os.Stat(asked.Media); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("--media %s is not a directory", asked.Media)
		}
	}
	root := asked.Into
	if root == "" {
		root = strings.TrimSuffix(filepath.Base(asked.File), filepath.Ext(asked.File))
	}

	// a directory origins chose is one the reader may not know it chose, and either way
	// the answer is somewhere else rather than this command run again
	if err := store.Vacant(root); err != nil {
		if asked.Into == "" {
			return nil, fmt.Errorf("%v\n  %s is the name of %s without its extension; "+
				"--into <dir> puts the corpus somewhere else", err, root, filepath.Base(asked.File))
		}
		return nil, fmt.Errorf("%v\n  --into <dir> puts the corpus somewhere else", err)
	}

	read := fromGEDCOM
	if asked.Format == "gramps" {
		read = fromGramps
	}
	conversion, err := read(asked.File, asked.Media, asked.Choose)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, conversion.done()) }()
	corpus := conversion.corpus
	// the name an export of origins' own says is the name, unless another is asked
	if asked.Name != "" {
		corpus.Project.Name = asked.Name
	}
	if corpus.Project.Name == "" {
		corpus.Project.Name = filepath.Base(root)
	}

	if err := store.Write(root, corpus); err != nil {
		return nil, err
	}
	for _, attached := range conversion.attachments {
		if err := store.Attach(root, corpus.Entity(attached.Record.Slug), attached.File, attached.From); err != nil {
			return nil, err
		}
	}

	report = &Imported{
		Corpus: root,
		Counts: counted(corpus),
		Notes:  conversion.notes,
	}
	if report.Notes == nil {
		report.Notes = []string{}
	}
	report.Subject = "Import " + filepath.Base(asked.File) + ": " + CountedIn(report.Counts)
	failed, released := recording(report, conversion.described, conversion.op)
	if failed != nil && report.Commit == "" {
		failed = &Unrecorded{Root: root, Err: failed}
	}
	return report, errors.Join(failed, released)
}

// conversion is an export read onto the model, with what the import's commit says of it.
//
// done gives up what reading it took, which for a package is the directory it was
// extracted into, and so is called only once the pictures have been copied out of it.
type conversion struct {
	corpus      *model.Corpus
	attachments []importing.Attachment
	notes       []string
	described   string
	op          string
	done        func() error
}

// fromGEDCOM reads a GEDCOM export, its pictures found where its own references lead,
// or a GEDZIP, its dataset read with the pictures packed beside it.
func fromGEDCOM(file, _ string, choose importing.Chooser) (conversion, error) {
	dataset, done := file, func() error { return nil }
	if isZip(file) {
		var err error
		if dataset, done, err = unzipped(file); err != nil {
			return conversion{}, err
		}
	}
	export, err := os.Open(dataset)
	if err != nil {
		return conversion{}, errors.Join(err, done())
	}
	defer export.Close()
	records, err := gedcom.Parse(export)
	if err != nil {
		return conversion{}, errors.Join(fmt.Errorf("%s: %w", file, err), done())
	}
	corpus, attachments, notes := gedcom.Convert(records, choose, importing.Beside(filepath.Dir(dataset)))
	described := gedcom.Describe(records, filepath.Base(file))
	return conversion{corpus, attachments, notes, described, "gedcom.import", done}, nil
}

// fromGramps reads a Gramps file or package, its pictures found where Gramps would
// look for them and then in media, where that names a directory.
func fromGramps(file, media string, choose importing.Chooser) (conversion, error) {
	export, err := gramps.Open(file)
	if err != nil {
		if errors.As(err, new(*fs.PathError)) {
			return conversion{}, err
		}
		return conversion{}, fmt.Errorf("%s: %w", file, err)
	}
	dirs := export.Media
	if media != "" {
		dirs = append(dirs, media)
	}
	corpus, attachments, notes := gramps.Convert(export, choose, importing.Among(dirs))
	described := gramps.Describe(export, filepath.Base(file))
	return conversion{corpus, attachments, notes, described, "gramps.import", export.Close}, nil
}

// recording puts a corpus that has just been written into git as one commit, making
// the repository where there is none, and fills in what the report says about it.
//
// The corpus stays where it is whatever happens here. It is written and it is
// correct, and what failed is git: taking a tree of thirty thousand people back off
// the disk over an identity nobody has configured would throw away the part that took
// the work and keep the part that takes a minute.
//
// Nothing is validated. Everywhere else a change is held to the contradictions it
// introduces, and an import introduces none: what a real export contradicts is the
// export's, stated by whoever kept the tree, and a tool that refused to carry a file
// in until the file agreed with itself would refuse every file worth carrying. That is
// what the line telling the reader to run check is for.
//
// The commit names no subjects. Everywhere else the trailer says which few entities a
// change was about; here the answer is all of them, and writing it out would be naming
// the corpus one line at a time — a megabyte of trailers on a tree of any size, every
// one of them pointing at the same commit. Blame still reads a line of an imported file
// back to it, which is the answer somebody is actually looking for.
//
// A commit that was made and could not be marked leaves the sha behind, and the corpus
// is then recorded; the marking is a local note of where origins had got to and the
// trailer says the same thing where it matters. A lock not given up is handed back
// apart, being about the next origins rather than this corpus.
//
// The corpus declares its line endings before anything is recorded, so the commit
// carries the declaration and a clone made anywhere checks the corpus out as written.
func recording(report *Imported, body, op string) (failed, released error) {
	if _, _, err := store.DeclareEndings(report.Corpus); err != nil {
		return err, nil
	}
	repo, made, err := history.Start(report.Corpus)
	if err != nil {
		return err, nil
	}
	release, err := repo.Lock("origins import")
	if err != nil {
		return err, nil
	}
	defer func() { released = release() }()
	report.Started = made
	if report.History, err = repo.Root(); err != nil {
		return err, nil
	}

	// the corpus directory and nothing above it, so a tree written inside a larger
	// repository does not drag the rest of it into the commit
	sha, err := repo.Commit(history.Commit{
		Subject: report.Subject,
		Body:    body,
		Tool:    Tool,
		Op:      op,
	}, []string{"."})
	report.Commit = sha
	return err, nil
}

// CountedIn is what came in, in the words the report and the commit subject both use.
func CountedIn(c Counts) string {
	return strings.Join([]string{
		Tally(c.People, "person", "people"),
		Tally(c.Families, "family", "families"),
		Tally(c.Places, "place", "places"),
		Tally(c.Sources, "source", "sources"),
		Tally(c.Repositories, "repository", "repositories"),
	}, ", ")
}

// Tally is a count of something, in the singular where there is one of it.
func Tally(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return fmt.Sprintf("%d %s", n, many)
}
