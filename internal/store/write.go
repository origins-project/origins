package store

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/origins-project/origins/internal/model"
)

// Write lays a corpus out on disk, canonically, every entity in the one place PathOf
// gives it.
//
// The directory must be empty or absent. Writing into a corpus that already exists would
// mean merging two of them — deciding that this Giovanni Rossi is that one — and that is
// a research judgement rather than a file operation.
func Write(root string, corpus *model.Corpus) error {
	if err := Vacant(root); err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	project, err := EmitProject(corpus.Project)
	if err != nil {
		return fmt.Errorf("%s: %w", ProjectFile, err)
	}
	if err := os.WriteFile(filepath.Join(root, ProjectFile), project, 0o644); err != nil {
		return err
	}

	var problems []error
	write := func(entity any) {
		relative, err := PathOf(entity)
		if err != nil {
			problems = append(problems, err)
			return
		}
		content, err := Emit(entity)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", relative, err))
			return
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			problems = append(problems, err)
			return
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			problems = append(problems, err)
		}
	}

	for _, kind := range model.Kinds {
		for _, slug := range corpus.SlugsOf(kind) {
			write(corpus.Entity(slug))
		}
	}
	return errors.Join(problems...)
}

// Attach copies the file at from beside a record of the corpus at root, as name: a path
// relative to the record's directory, as the record's Files and media name it.
func Attach(root string, entity any, name, from string) error {
	relative, err := PathOf(entity)
	if err != nil {
		return err
	}
	target := filepath.Join(root, filepath.FromSlash(Location(relative)), filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	source, err := os.Open(from)
	if err != nil {
		return err
	}
	defer source.Close()
	copied, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(copied, source); err != nil {
		copied.Close()
		return err
	}
	return copied.Close()
}

// Vacant says whether root is somewhere Write will put a corpus: a directory that is
// empty, or one that does not exist yet. It is asked before anything is read, so that a
// long export is not parsed only to be refused somewhere to go.
func Vacant(root string) error {
	found, err := os.ReadDir(root)
	switch {
	case os.IsNotExist(err):
		return nil
	case err != nil:
		return err
	case len(found) > 0:
		return fmt.Errorf("%s is not empty: origins writes an imported corpus into a new "+
			"directory, because merging one tree into another is a research decision", root)
	}
	return nil
}
