package history

// Patch writes what has been done to the working tree since from, a tree or a commit,
// as a patch, so that it can be carried to a state that is not this one. An empty from
// is the last commit, or nothing at all where there has not been one.
//
// Only the paths it is given, as a commit takes only the paths it is given. The index
// is where the question is put, a file git has never seen being invisible to a diff
// otherwise and staging the one way to put a new one in front of it. That is safe
// here and would not be in the corpus somebody is keeping: what this is pointed at is
// a checkout made to be read once and thrown away.
//
// The paths are relative to the corpus rather than to the repository, because the
// corpus is the thing and where in a repository it sits is incidental — a patch made
// where the corpus is the whole repository has to apply where it is three directories
// down. The patch carries context, which is what a three-way merge works with; it
// carries renames, a rename being an operation origins has a word for rather than a
// deletion beside a creation; and it carries the bytes of a binary file, a photograph
// following the record it belongs to being part of what a change says.
//
// Everything a reader's git configuration could reach in and alter is written out.
// The prefixes, the colour and the amount of context are all settings somebody has,
// and a patch that comes out differently on two machines is a patch that applies on
// one of them.
func (r *Repo) Patch(from string, paths []string) ([]byte, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	if _, err := r.git(append([]string{"add", "-A", "--"}, paths...)...); err != nil {
		return nil, err
	}
	diff := []string{"diff", "--cached"}
	if from != "" {
		diff = append(diff, from)
	}
	return r.output(append(append(diff,
		"--relative", "--binary", "--find-renames", "--unified=3",
		"--no-color", "--no-ext-diff", "--no-textconv", "--src-prefix=a/", "--dst-prefix=b/", "--",
	), paths...)...)
}

// Tree stages the whole of the corpus and names what is staged, so that a patch can be
// made from a state no commit holds.
//
// A tree and not a commit, because what it names is a checkout rearranged to be read
// once and thrown away. A commit would want an identity for a state nobody chose to
// record, and a tree wants nothing.
func (r *Repo) Tree() (string, error) {
	if _, err := r.git("add", "-A", "--", "."); err != nil {
		return "", err
	}
	return r.git("write-tree")
}
