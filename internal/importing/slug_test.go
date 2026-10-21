package importing

import (
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// A slug is spelled in the letters a-z and the digits, and accents fold away, so a name
// in Latin letters is filed under itself and a name in any other script leaves nothing
// to file it by, its digits and its Latin words aside. An importer files such a record
// under something else it was given.
func TestASlugIsSpelledInLatinLettersAndDigits(t *testing.T) {
	for _, c := range []struct{ script, name, slug string }{
		{"Latin", "Sólveig Gunnarsdóttir", "solveig-gunnarsdottir"},
		{"Latin", "Jean-Baptiste Verdier", "jean-baptiste-verdier"},
		{"Latin", "Đặng Thị Hoa", "dang-thi-hoa"},
		{"Cyrillic", "Иван Петрович Ершов", ""},
		{"Greek", "Μαρία Παπαδοπούλου", ""},
		{"Arabic", "محمود بن علي الحلبي", ""},
		{"Hebrew", "רחל מודיאנו", ""},
		{"Chinese", "陳玉英", ""},
		{"Japanese", "佐藤 由美子", ""},
		{"Japanese", "さとう ゆみこ", ""},
		{"Korean", "박서연", ""},
		{"Devanagari", "रमेश विठ्ठल", ""},
		{"Cyrillic", "Ревизская сказка, Нижнетагильский завод, 1834", "1834"},
		{"Arabic", "سجل عقود الزواج، ١٣٠٢", ""},
		{"Cyrillic", "Ершов (Ershov)", "ershov"},
		{"Arabic, romanized", "ʿĀʾisha bint Ḥasan al-Ḥalabī", "aisha-bint-hasan-al-halabi"},
	} {
		base, err := model.Slugify(c.name)
		if c.slug == "" {
			if err == nil {
				t.Errorf("%s: %q gave %q", c.script, c.name, base)
			}
			continue
		}
		namer := NewNamer(model.NewCorpus(), nil)
		if got := namer.Assign(Shorten(base), nil, c.name); err != nil || got != model.Slug(c.slug) {
			t.Errorf("%s: %q gave %q, %v, wanted %q", c.script, c.name, got, err, c.slug)
		}
	}
}
