package store

import (
	"bytes"
	"errors"

	"golang.org/x/text/unicode/norm"
)

var fence = []byte("---")

// Split separates the YAML frontmatter of an entity file from its markdown body. The
// body is not decoration — the argument for why this Giovanni is that Giovanni is the
// substance of genealogy — so it is returned rather than discarded.
//
// Leading and trailing blank lines are trimmed from the body and line endings are
// normalised to LF, which is what the corpus stores regardless of the platform it was
// edited on.
//
// Text is normalised to NFC here, at the one point every byte of a corpus passes
// through. Jos\u00E9 is two different byte sequences in NFC and NFD and editors differ over
// which they write, so without this the model would hold whichever form the input
// happened to use and the emitter could not be deterministic.
func Split(content []byte) (frontmatter, body []byte, err error) {
	content = bytes.TrimPrefix(content, []byte("\uFEFF"))
	content = bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	content = norm.NFC.Bytes(content)

	rest, ok := bytes.CutPrefix(content, append(fence, '\n'))
	if !ok {
		if bytes.Equal(bytes.TrimRight(content, "\n"), fence) {
			return nil, nil, errors.New("frontmatter is never closed")
		}
		return nil, nil, errors.New("does not start with a --- frontmatter fence")
	}

	closing := -1
	for offset := 0; offset < len(rest); {
		line := rest[offset:]
		if end := bytes.IndexByte(line, '\n'); end >= 0 {
			line = line[:end]
		}
		if bytes.Equal(line, fence) {
			closing = offset
			break
		}
		offset += len(line) + 1
	}
	if closing < 0 {
		return nil, nil, errors.New("frontmatter is never closed")
	}

	frontmatter = rest[:closing]
	body = rest[min(closing+len(fence)+1, len(rest)):]
	return frontmatter, bytes.Trim(body, "\n"), nil
}
