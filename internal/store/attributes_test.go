package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gitattributes(t *testing.T, root string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, AttributesFile))
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

func TestACorpusWithNoAttributesIsGivenThem(t *testing.T) {
	root := t.TempDir()
	written, _, err := DeclareEndings(root)
	if err != nil {
		t.Fatal(err)
	}
	if !written {
		t.Error("nothing was written")
	}
	if got, want := gitattributes(t, root), strings.Join(declared, "\n")+"\n"; got != want {
		t.Errorf("wrote\n%s\nwant\n%s", got, want)
	}
}

func TestACorpusThatDeclaresItsEndingsIsLeftAlone(t *testing.T) {
	root := t.TempDir()
	if _, _, err := DeclareEndings(root); err != nil {
		t.Fatal(err)
	}
	before := gitattributes(t, root)
	written, _, err := DeclareEndings(root)
	if err != nil {
		t.Fatal(err)
	}
	if written || gitattributes(t, root) != before {
		t.Error("a second declaration wrote the file again")
	}
}

// Somebody's own attributes are theirs, and the declaration goes after them: a later
// line overrides an earlier one, and one of theirs that already says what origins would
// is not said twice. A last line with no newline is finished before anything follows it,
// or the first line added would be joined onto it.
func TestAttributesAlreadyThereAreAddedTo(t *testing.T) {
	root := t.TempDir()
	theirs := "*.jpg binary\n" + declared[1] + "\n*.md -text"
	if err := os.WriteFile(filepath.Join(root, AttributesFile), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}
	written, _, err := DeclareEndings(root)
	if err != nil {
		t.Fatal(err)
	}
	if !written {
		t.Error("nothing was written")
	}
	want := theirs + "\n" + declared[0] + "\n" + strings.Join(declared[2:], "\n") + "\n"
	if got := gitattributes(t, root); got != want {
		t.Errorf("wrote\n%s\nwant\n%s", got, want)
	}
}

func TestADeclarationIsTakenBack(t *testing.T) {
	t.Run("a file that was there", func(t *testing.T) {
		root := t.TempDir()
		theirs := "*.jpg binary\n"
		if err := os.WriteFile(filepath.Join(root, AttributesFile), []byte(theirs), 0o644); err != nil {
			t.Fatal(err)
		}
		_, undo, err := DeclareEndings(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := undo(); err != nil {
			t.Fatal(err)
		}
		if got := gitattributes(t, root); got != theirs {
			t.Errorf("put back as\n%s", got)
		}
	})
	t.Run("a file that was not", func(t *testing.T) {
		root := t.TempDir()
		_, undo, err := DeclareEndings(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := undo(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(root, AttributesFile)); !os.IsNotExist(err) {
			t.Errorf("the file is still there: %v", err)
		}
	})
}
