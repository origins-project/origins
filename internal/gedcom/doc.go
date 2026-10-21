// Package gedcom reads GEDCOM 5.5.1 and maps it onto the model.
//
// GEDCOM is an import format, not the model, so nothing here escapes into the rest of
// origins: the package takes bytes and returns a [model.Corpus]. It is deliberately
// tolerant of the dialects real exports emit, because the alternative is a tool that
// refuses the only files anyone actually has.
//
// Tolerance has a limit, and it is drawn at the point where guessing would corrupt
// rather than merely lose. Structure is repaired silently — a skipped level, a missing
// terminator, a line nobody bothered to fold. Text is never guessed at: a file whose
// declared encoding cannot be decoded is refused rather than read approximately, since
// a name spelled wrong is worse than a name not imported.
//
// Everything that could not be carried onto the model is reported rather than dropped
// quietly. An import that silently discards a third of a file looks exactly like one
// that worked.
package gedcom
