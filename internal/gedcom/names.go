package gedcom

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/origins-project/origins/internal/model"
)

// readName reads a NAME line into the parts of a name.
//
// The payload is the whole name with the surname marked by slashes — "Lt. Cmndr. Joseph
// /de Allen/ jr." — and the subtags beneath it name the parts of that payload rather
// than adding to it. So the payload is divided at its slashes, the words before them
// the given names and the words after the suffix, and a subtag says where the payload
// is not divided finely enough: NPFX takes a title off the front of the given names and
// SPFX a prefix off the surname. A subtag that differs from the payload only in case is
// an index key, exporters writing SURN in capitals, and the payload's spelling stands.
// A subtag given more than once is a part in pieces, joined in order — two GIVN being
// two given names — save NICK, two nicknames joined by a comma, and SURN, each a surname
// of the list, the SPFX in the same position its prefix and any left over the last's.
//
// Where the payload says what the parts do not, it is the name's text beside them, as
// the record wrote it, slashes and all. A name with no slashes and no subtags is that
// text alone, rather than divided on a guess that the last word is the family name.
// That guess is wrong for a Spanish double surname, wrong for a Hungarian name written
// family-first, and wrong for anyone recorded under a patronymic.
func readName(node *Node) model.PersonName {
	name := model.PersonName{
		Prefix:   pieces(node, "NPFX", " "),
		Given:    pieces(node, "GIVN", " "),
		Nickname: pieces(node, "NICK", ", "),
		Suffix:   pieces(node, "NSFX", " "),
	}
	surnames, prefixes := each(node, "SURN"), each(node, "SPFX")
	if len(surnames) > 1 {
		for _, surname := range surnames {
			name.Surname = append(name.Surname, model.Surname{Name: surname})
		}
		for i, prefix := range prefixes {
			last := &name.Surname[min(i, len(surnames)-1)]
			last.Prefix = strings.TrimSpace(last.Prefix + " " + prefix)
		}
	} else {
		name.SurnamePrefix = strings.Join(prefixes, " ")
		name.Surname = model.SurnameOf(strings.Join(surnames, " "))
	}

	line := collapse(node.Text())
	if before, marked, after, marks := cutSurname(line); marks {
		before, marked, after = named(before), named(marked), named(after)
		name.Given = spelled(name.Given, strings.TrimSpace(strings.TrimPrefix(before, name.Prefix)))
		if len(name.Surname) <= 1 {
			name.Surname = model.SurnameOf(spelled(name.PrimarySurname().Name, strings.TrimSpace(strings.TrimPrefix(marked, name.SurnamePrefix))))
		}
		name.Suffix = spelled(name.Suffix, after)
	}
	if !name.HasParts() {
		return model.PersonName{Text: named(line)}
	}
	if saysMore(line, name) {
		name.Text = line
	}
	return name
}

// each is the words of every tag under node that has any, in order, marking them all
// read.
func each(node *Node, tag string) []string {
	var out []string
	for _, piece := range node.TakeAll(tag) {
		if word := named(piece.Text()); word != "" {
			out = append(out, word)
		}
	}
	return out
}

func pieces(node *Node, tag, sep string) string { return strings.Join(each(node, tag), sep) }

// spelled is a part of a name as its subtag gives it, or as the payload spells it where
// the subtag is silent or says the same words in other capitals.
func spelled(subtag, payload string) string {
	if subtag == "" || strings.EqualFold(subtag, payload) {
		return payload
	}
	return subtag
}

// cutSurname splits a name payload at the pair of slashes marking the surname.
func cutSurname(text string) (before, surname, after string, marks bool) {
	open := strings.Index(text, "/")
	if open < 0 {
		return "", "", "", false
	}
	close := strings.Index(text[open+1:], "/")
	if close < 0 {
		return "", "", "", false
	}
	return text[:open], text[open+1 : open+1+close], text[open+close+2:], true
}

// saysMore reports a payload with a word in it that no part of the name has, setting
// capitals and the punctuation around a word aside.
func saysMore(line string, name model.PersonName) bool {
	parts := map[string]bool{}
	held := []string{name.Prefix, name.Given, name.Nickname, name.SurnamePrefix, name.Suffix}
	for _, surname := range name.Surname {
		held = append(held, surname.Prefix, surname.Name)
	}
	for _, word := range words(strings.Join(held, " ")) {
		parts[word] = true
	}
	for _, word := range words(line) {
		if !parts[word] {
			return true
		}
	}
	return false
}

// words are the words of a name, in lower case and without the punctuation around them.
func words(s string) []string {
	var out []string
	for _, word := range strings.FieldsFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == '/' }) {
		word = strings.TrimFunc(word, func(r rune) bool { return unicode.IsPunct(r) || unicode.IsSymbol(r) })
		if word != "" {
			out = append(out, strings.ToLower(word))
		}
	}
	return out
}

// names reads a NAME into person with every other writing of it beneath: a TRAN, the
// name in another language or script, with its LANG, and 5.5.1's FONE and ROMN, a
// phonetic and a romanized spelling, in the language their method gives. Each is
// another name the person went by, of the kind the name it writes again is. So is each
// of the extension tags Gramps reads under a NAME, of the kind taken gives it. The first
// NAME of a record is the one the person is filed under, and a RESN on it is on the
// person, the name being hidden with them; on any other it is the name's own.
func (c *converter) names(person *model.Person, node *Node, slug model.Slug, primary bool) {
	name := c.name(node, slug)
	if primary {
		person.Name = name
		person.Private = c.private(node) || name.Private || person.Private
		name.Private = false
	} else {
		name.Private = c.private(node) || name.Private
		alsoKnownAs(person, name)
	}
	c.home(node, slug, nameNote(person, name))

	for _, written := range node.Children {
		var other model.PersonName
		switch written.Tag {
		case "TRAN":
			lang := language(written.First("LANG"))
			other = c.name(written, slug)
			other.Lang = lang
		case "FONE", "ROMN":
			lang, said := method(written)
			other = c.name(written, slug)
			other.Lang, other.Type = lang, ""
			other.Note = strings.TrimSpace(said + "\n\n" + other.Note)
		case "_MARNM", "_MAR", "_MARN", "_ADPN", "_AKA", "_AKAN", "AKA":
			if other = c.name(written, slug); other.IsZero() {
				continue
			}
			if other.Type == "" {
				other.Type = taken[written.Tag]
			}
			if taken[written.Tag] != "aka" && !other.HasParts() {
				other = surnamed(name, other)
			}
		default:
			continue
		}
		if other.IsZero() {
			continue
		}
		written.Mark()
		if other.Type == "" {
			other.Type = name.Type
		}
		alsoKnownAs(person, other)
		c.home(written, slug, nameNote(person, other))
	}
	c.aliases(person, node, slug)
}

// name reads a NAME, or another writing of one, with what it says about itself: the
// kind of name it is, what states it and the notes on it.
func (c *converter) name(node *Node, subject model.Slug) model.PersonName {
	name := readName(node)
	if name.IsZero() {
		return name
	}
	name.Type = nameType(node.Take("TYPE"))
	c.spoken(&name, node)
	name.Sources = c.citations(node, subject, "name")
	name.Note = strings.Join(c.noteTexts(node), "\n\n")
	return name
}

// spoken reads what origins writes of a name that 7.0 has no structure for: the words
// the record wrote beside the parts, which a file origins wrote says only so, the given
// name the person went by, the patterns it is written and sorted by, when it was borne,
// its language and whether it is private.
func (c *converter) spoken(name *model.PersonName, node *Node) {
	if text := c.take(node, "_TEXT"); text != nil {
		name.Text = collapse(text.Text())
	} else if c.mine && name.HasParts() {
		name.Text = ""
	}
	name.Call = c.extension(node, "_CALL")
	name.FamilyNickname = c.extension(node, "_FAMILY_NICKNAME")
	name.Group = c.extension(node, "_GROUP")
	name.Display = c.extension(node, "_DISPLAY")
	name.Sort = c.extension(node, "_SORT")
	for _, bound := range []struct {
		tag  string
		date *model.Date
	}{{"_FROM", &name.From}, {"_UNTIL", &name.Until}} {
		if line := c.take(node, bound.tag); line != nil {
			date, read := readDateNode(line)
			if !read {
				c.notes.Add("a date could not be read and was kept as words: %q", date.Wording)
			}
			*bound.date = date
		}
	}
	if lang := node.First("_LANG"); c.ours["_LANG"] && lang != nil {
		if parsed, err := model.ParseLanguage(collapse(lang.Text())); err == nil {
			name.Lang = parsed
			lang.Mark()
		}
	}
	name.Private = c.flagged(node)
}

// nameType is the kind of name a TYPE says, in the words the corpus writes: the values
// both versions enumerate — aka, birth, immigrant, maiden, married, professional — in
// origins' words, and whatever else an exporter wrote as written, the list being open.
// 7.0's OTHER is what its PHRASE says.
func nameType(node *Node) string {
	word := collapse(node.Text())
	if strings.EqualFold(word, "OTHER") {
		if phrase := collapse(node.First("PHRASE").Text()); phrase != "" {
			node.Take("PHRASE")
			return phrase
		}
	}
	if lower := strings.ToLower(word); nameTypes[lower] {
		return lower
	}
	return word
}

var nameTypes = map[string]bool{
	"aka": true, "birth": true, "immigrant": true, "maiden": true, "married": true, "professional": true,
	"other": true,
}

// methods are the languages the spelling methods 5.5.1 names write a name in.
var methods = map[string]model.Language{
	"kana": "ja-Kana", "hangul": "ko-Hang", "pinyin": "zh-Latn-pinyin", "romaji": "ja-Latn",
	"wadegiles": "zh-Latn-wadegile",
}

// method is the language a FONE or a ROMN writes a name in, from the method its TYPE
// names: one 5.5.1 names, or one of the file's own as a private-use tag. A method too
// long to be one, or none at all, is said in words for the name's note instead.
func method(written *Node) (model.Language, string) {
	how := "phonetic"
	private := "und-x-"
	if written.Tag == "ROMN" {
		how, private = "romanized", "und-Latn-x-"
	}
	word := collapse(written.Take("TYPE").Text())
	if lang, ok := methods[strings.ToLower(word)]; ok {
		return lang, ""
	}
	if word == "" {
		return "", how
	}
	if !strings.ContainsAny(word, " -") {
		if lang, err := model.ParseLanguage(private + strings.ToLower(word)); err == nil {
			return lang, ""
		}
	}
	return "", how + ", by " + word
}

// taken are the kinds of name the extension tags Gramps reads under a NAME give, in the
// words its XML gives them: a married name, as Personal Ancestral File, Ancestral Quest,
// geni.com and Brother's Keeper write one, an adopted name, and a name somebody was also
// known by. That last is read as any NAME without slashes is, whole, where Gramps makes
// a single word a nickname and divides more on a guess that the last is the surname.
var taken = map[string]string{
	"_MARNM": "married", "_MAR": "married", "_MARN": "married", "_ADPN": "Adopted",
	"_AKA": "aka", "_AKAN": "aka", "AKA": "aka",
}

// surnamed is a married or an adopted name taken in place of name, written with no
// slashes: the surname taken, so the name's own given names, title and suffix with that
// surname, as Gramps reads one of a single word. Exporters write "de Percy" as readily
// as "Lucas", so the payload is the surname whatever its length, and whole, its prefix
// not cut off on a guess.
func surnamed(name, other model.PersonName) model.PersonName {
	other.Prefix, other.Given, other.Call, other.Nickname, other.Suffix =
		name.Prefix, name.Given, name.Call, name.Nickname, name.Suffix
	other.Surname, other.Text = model.SurnameOf(other.Text), ""
	return other
}

// nameNote is the path of the note on the name a person holds as name, or "" where they
// hold no such name.
func nameNote(person *model.Person, name model.PersonName) string {
	if name.IsZero() {
		return ""
	}
	if reflect.DeepEqual(name, person.Name) {
		return "name.note"
	}
	for i, known := range person.AlsoKnownAs {
		if reflect.DeepEqual(known, name) {
			return fmt.Sprintf("also-known-as[%d].note", i)
		}
	}
	return ""
}

// aliases reads the ALIA tags under a record or under one of its names. ALIA points at
// another record said to be the same person, which is an entry of same-as, its PHRASE
// the note; under @VOID@ its PHRASE is all the file says of somebody it has no record of,
// and the entry's name. A few dialects write a name on it instead, which is another
// name the person went by.
func (c *converter) aliases(person *model.Person, node *Node, slug model.Slug) {
	for _, alias := range node.All("ALIA") {
		phrase := collapse(alias.First("PHRASE").Text())
		var claim model.SameAs
		switch xref := pointer(alias.Value); {
		case voided(alias.Value):
			if phrase == "" {
				continue
			}
			claim.Name = phrase
		case xref != "":
			other, ok := c.person[xref]
			if !ok {
				c.notes.Add("an ALIA names an individual who is not in the file")
				continue
			}
			claim.ID, claim.Note = other, phrase
		default:
			alias.Mark()
			name := readName(alias)
			alsoKnownAs(person, name)
			c.home(alias, slug, nameNote(person, name))
			continue
		}
		alias.Mark().Take("PHRASE")
		if !slices.ContainsFunc(person.SameAs, func(known model.SameAs) bool { return reflect.DeepEqual(known, claim) }) {
			person.SameAs = append(person.SameAs, claim)
		}
	}
}

// alsoKnownAs records another name for someone, unless it is one already recorded: two
// names are one only where they agree in every field, so a second spelled as the first
// and cited differently is kept.
func alsoKnownAs(person *model.Person, name model.PersonName) {
	if name.IsZero() || reflect.DeepEqual(name, person.Name) {
		return
	}
	for _, known := range person.AlsoKnownAs {
		if reflect.DeepEqual(known, name) {
			return
		}
	}
	person.AlsoKnownAs = append(person.AlsoKnownAs, name)
}

// research reads ANCI and DESI, a submitter's interest in the person's ancestors or
// descendants, as tasks in the person's body naming who has it.
func (c *converter) research(record *Node) []string {
	var tasks []string
	for _, line := range record.Children {
		var task string
		switch line.Tag {
		case "ANCI":
			task = "- [ ] Research the ancestors"
		case "DESI":
			task = "- [ ] Research the descendants"
		default:
			continue
		}
		submitter := c.xref[pointer(line.Value)]
		if submitter == nil || submitter.Tag != "SUBM" {
			continue
		}
		if who := collapse(submitter.First("NAME").Text()); who != "" {
			task += " (" + who + ")"
		}
		line.Mark()
		tasks = append(tasks, task)
	}
	if len(tasks) == 0 {
		return nil
	}
	return []string{strings.Join(tasks, "\n")}
}
