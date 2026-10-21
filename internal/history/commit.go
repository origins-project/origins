package history

import (
	"errors"
	"fmt"
	"strings"

	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

// Commit is one entry in a corpus's history: a line for whoever reads the log, and
// trailers for origins to read back.
//
// Trailers rather than a structured subject or a log kept beside the repository: git
// parses them natively, they survive a rebase, and `git log` still reads like prose.
// Origins-Hypothesis is the one the rest exists for — it joins a fact to the argument
// that settled it, which is what lets `why` answer at all.
type Commit struct {
	Subject    string              // the human line, the only part not written for a machine
	Body       string              // what more there is to say to a human, or nothing
	Tool       string              // what wrote it, as "origins 0.1.0"
	Op         string              // what it did, as <noun>.<verb>
	Entities   []model.EntityRef   // the entities it affected
	Renamed    []Renaming          // the names it gave up, where it is not itself a rename
	Events     []string            // the kinds of event it touched, where it touched any
	Sources    []model.CitationRef // the citations the facts rest on
	Corrects   string              // the commit this one revises, where it revises one
	Hypothesis model.Slug          // the hypothesis that decided it, where one did
	ProposedBy string              // who proposed that hypothesis, where the researcher did not
}

// Message is the commit message: the subject, a blank line, the body where there is
// one and a blank line after it, and the trailer block.
// The trailers keep the order they are documented in, so two commits recording the
// same thing read the same way.
func (c Commit) Message() string {
	trailers := []string{"Origins-Tool: " + c.Tool, "Origins-Op: " + c.Op}
	for _, entity := range c.Entities {
		trailers = append(trailers, "Origins-Subject: "+entity.String())
	}
	for _, renaming := range c.Renamed {
		trailers = append(trailers, "Origins-Renamed: "+renaming.String())
	}
	for _, event := range c.Events {
		trailers = append(trailers, "Origins-Event: "+event)
	}
	for _, source := range c.Sources {
		trailers = append(trailers, "Origins-Source: "+source.String())
	}
	if c.Corrects != "" {
		trailers = append(trailers, "Origins-Corrects: "+c.Corrects)
	}
	if c.Hypothesis != "" {
		trailers = append(trailers, "Origins-Hypothesis: "+string(c.Hypothesis))
	}
	if c.ProposedBy != "" {
		trailers = append(trailers, "Origins-Proposed-By: "+c.ProposedBy)
	}
	message := strings.TrimSpace(c.Subject) + "\n\n"
	if body := strings.TrimSpace(c.Body); body != "" {
		message += body + "\n\n"
	}
	return message + strings.Join(trailers, "\n") + "\n"
}

// valid reports a commit origins would not be able to read back. History that looks
// complete and is not is worse than history that is missing, because only the second
// kind gets noticed.
func (c Commit) valid() error {
	switch {
	case strings.TrimSpace(c.Subject) == "":
		return errors.New("a commit needs a subject line, since it is what the log shows")
	case c.Tool == "":
		return errors.New("a commit needs the tool that wrote it, which is how a foreign commit is told apart")
	case c.Op == "":
		return errors.New("a commit needs an operation, since it is what the history commands read")
	}
	for _, value := range append([]string{c.Subject, c.Tool, c.Op, c.Corrects, c.ProposedBy}, c.Events...) {
		if strings.ContainsAny(value, "\n\r") {
			return fmt.Errorf("%q covers more than one line, and a trailer is one line", value)
		}
	}
	for _, renaming := range c.Renamed {
		if err := renaming.valid(); err != nil {
			return err
		}
	}
	return nil
}

// Renaming is a name a commit gave up and the name that took it over.
//
// A rename and a merge say so in their subjects, which they name in pairs, and the
// history is read by those. Everything else that can change a name — a file renamed by
// hand and adopted, a hypothesis accepted that renames or merges, a commit origins did
// not write replayed as one it did — is filed under some other operation, and without
// this the log would read the name it arrives at as somebody new. Where the name taken
// over is already held the two records are one, which is what a merge's pairs mean.
type Renaming struct {
	From, To model.EntityRef
}

func (r Renaming) String() string { return r.From.String() + " " + r.To.String() }

func (r Renaming) valid() error {
	switch {
	case r.From.Kind != r.To.Kind:
		return fmt.Errorf("%s: a name is given up to something of the same kind", r)
	case r.From == r.To:
		return fmt.Errorf("%s: that is not a different name", r)
	}
	return nil
}

func parseRenaming(s string) (Renaming, error) {
	from, to, found := strings.Cut(s, " ")
	if !found {
		return Renaming{}, fmt.Errorf("%q names one entity and not two", s)
	}
	var (
		r   Renaming
		err error
	)
	if r.From, err = model.ParseEntityRef(from); err != nil {
		return Renaming{}, err
	}
	if r.To, err = model.ParseEntityRef(to); err != nil {
		return Renaming{}, err
	}
	return r, r.valid()
}

// Commit records a change as one commit containing exactly the paths it is given,
// authored as whoever git is configured as.
//
// Naming the paths rather than committing the index is what keeps two promises at
// once: bulk work is one commit and not thousands, and a dirty working tree beside the
// change is not swept into it. Anything already staged that the change did not touch
// stays staged.
//
// The commit is marked as origins' own work as it is made, so the next command can tell
// this history apart from history that moved underneath it. A sha alongside the error
// is a commit that was made and a marking that was not: the change is in history then,
// and taking it back off the disk would leave the tree holding its reverse.
func (r *Repo) Commit(c Commit, paths []string) (string, error) {
	if err := c.valid(); err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", errors.New("a commit with no files in it records nothing")
	}

	if err := r.unignored(paths); err != nil {
		return "", err
	}
	// A file git has not seen before cannot be named in a pathspec, so it is added
	// first; a file the change removed is staged as the removal by the same command.
	if _, err := r.git(append([]string{"add", "--"}, paths...)...); err != nil {
		return "", err
	}
	sha, made, err := r.land(c, paths)
	if err != nil && !made {
		return "", errors.Join(err, r.unstage(paths))
	}
	return sha, err
}

// Record records the corpus as it stands: everything in it that differs from what was
// last recorded, rather than only what origins wrote itself.
//
// It is how a change origins did not make gets into the history — an edit somebody
// typed into a file, or the content of commits that have been taken back off. What to
// commit cannot be worked out from what origins did, because origins did none of it,
// so the answer is the corpus: every path under it and nothing above it, which is what
// keeps a corpus kept in a subdirectory of a larger repository from dragging the rest
// of the repository in with it.
//
// A corpus with nothing to record still gets a commit. What the commit says is that
// origins has read what is here and answers for it, and that is worth saying even when
// nothing needed changing: it is the whole difference between a history origins can
// account for and one it cannot.
func (r *Repo) Record(c Commit) (string, error) {
	if err := c.valid(); err != nil {
		return "", err
	}
	whole := []string{"."}
	if err := r.unignored(whole); err != nil {
		return "", err
	}
	if _, err := r.git("add", "--all", "--", "."); err != nil {
		return "", err
	}
	sha, made, err := r.land(c, whole, "--allow-empty")
	if err != nil && !made {
		return "", errors.Join(err, r.unstage(whole))
	}
	return sha, err
}

// unignored refuses to stage paths when git has been told to ignore a file in them
// that the corpus is made of.
//
// Git leaves such a file out of a directory it is asked to add without a word, so the
// commit would say the corpus was recorded while the file stayed on this disk and went
// missing from every clone. Naming the file itself gets a refusal, but one that says
// nothing about why; this says which rule. A file git already tracks is not asked
// about, an ignore rule having no hold on it, and neither is anything the corpus is not
// made of, a note ignored beside the corpus being its owner's business.
func (r *Repo) unignored(paths []string) error {
	out, err := r.git(append([]string{"ls-files", "-z", "--others", "--ignored", "--exclude-standard", "--"},
		paths...)...)
	if err != nil {
		return err
	}
	var lost []string
	for _, file := range strings.Split(out, "\x00") {
		if file != "" && store.MadeOf(r.dir, file) {
			lost = append(lost, file)
		}
	}
	if len(lost) == 0 {
		return nil
	}

	const shown = 10
	why, err := r.git(append([]string{"check-ignore", "--verbose", "--"}, lost[:min(len(lost), shown)]...)...)
	if err != nil {
		return err
	}
	var said strings.Builder
	for _, line := range strings.Split(why, "\n") {
		rule, file, _ := strings.Cut(line, "\t")
		fmt.Fprintf(&said, "\n  %s, by %s", file, rule)
	}
	if len(lost) > shown {
		fmt.Fprintf(&said, "\n  and %d more", len(lost)-shown)
	}
	return fmt.Errorf("git is told to ignore files this corpus is made of, so a commit "+
		"would leave them out of the history and of every clone:%s\n"+
		"take the rule out, or give git the files with git add --force", said.String())
}

// land commits what is already staged and marks it as origins' own work. Every
// commit origins makes goes through here, which is what keeps the marking from being
// something a second way of committing could forget to do.
//
// The message reaches git on its standard input rather than as an argument. It names
// every entity the change touched, and an import or an adoption touches thousands: past
// a hundred and twenty-eight kilobytes Linux refuses to start a command with one
// argument that long, and the change would fail for being large.
//
// made says whether the commit was made, which is not the same question as whether
// everything went well. A sha and an error together mean the change is recorded and
// the marking is not, and the caller must not then take the change back off the disk.
func (r *Repo) land(c Commit, paths []string, flags ...string) (sha string, made bool, err error) {
	args := append([]string{"commit", "--file=-"}, flags...)
	if err := r.fed([]byte(c.Message()), append(append(args, "--"), paths...)...); err != nil {
		return "", false, err
	}
	sha, err = r.git("rev-parse", "HEAD")
	if err != nil {
		return "", true, err
	}
	if err := r.Mark(sha); err != nil {
		return sha, true, fmt.Errorf("the change was recorded as %s, but origins could not mark it as "+
			"its own, so it will read as a commit made by something else: %w", sha, err)
	}
	return sha, true, nil
}

// unstage puts the index back after a commit git would not make. Without it a refused
// commit leaves the change staged, which from the outside is indistinguishable from a
// change somebody meant to make and has not finished.
func (r *Repo) unstage(paths []string) error {
	if _, err := r.git("rev-parse", "--verify", "-q", "HEAD"); err != nil {
		// nothing has been committed yet, so there is no state to put them back to.
		// -r because a path here can be the corpus directory itself, which is what an
		// import and an adoption both name, and git will not take a directory off the
		// index without being told that a directory is what it is.
		_, err := r.git(append([]string{"rm", "--cached", "-r", "-q", "--force", "--ignore-unmatch", "--"},
			paths...)...)
		return err
	}
	_, err := r.git(append([]string{"reset", "-q", "HEAD", "--"}, paths...)...)
	return err
}
