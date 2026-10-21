// Mkcorpora writes the three corpora the application's browser tests start origins
// browse on, into <dir>/empty, <dir>/sparse and <dir>/large, removing whatever was
// there first:
//
//	go run ./tools/mkcorpora.go <dir>
//
// The empty corpus is the manifest alone. The sparse one is a record of each kind with
// only what the reader requires, and check requires, and nothing else. The large one is
// five thousand people over twelve generations, born into sixteen cultures that each
// name them in its own way and in its own script, with places, archives, registers and
// a citation for every event in the register's own words, drawn from one seed.
//
// What it wrote is left alone where this program is the one that wrote it and each
// corpus is as it was left, which <dir>/stamp records: the digest of this program's
// own executable, which changes with any code built into it, and each corpus's refs,
// with nothing changed in its files and no draft begun. Making the large one takes
// minutes on a system that scans every file opened.
//
// Each is made as a researcher's would be: started by api.Init, its records written
// through store so that they are canonical, and recorded by api.Adopt, which would
// rewrite a file that was not and refuse a corpus check finds wrong — so this refuses
// both. A hypothesis pins the commit its edits apply to, so the hypotheses come after
// the records they change: the settled one adopted, the open one proposed through
// api.ProposeStated as an agent would propose it. Nothing of the machine's git
// configuration reaches the repositories, and the identity and the clock are fixed, so
// every run makes the same history down to the shas.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/origins-project/origins/internal/api"
	"github.com/origins-project/origins/internal/importing"
	"github.com/origins-project/origins/internal/model"
	"github.com/origins-project/origins/internal/store"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run ./tools/mkcorpora.go <dir>")
		os.Exit(2)
	}
	out, err := filepath.Abs(os.Args[1])
	if err == nil {
		var config string
		config, err = isolate(out)
		defer os.Remove(config)
	}
	corpora := []struct {
		name  string
		build func(*recorder) error
	}{
		{"empty", func(*recorder) error { return nil }},
		{"sparse", sparse},
		{"large", large},
	}
	var names []string
	for _, corpus := range corpora {
		names = append(names, corpus.name)
	}
	var made string
	if err == nil {
		made, err = stamp(out, names)
		if written, _ := os.ReadFile(filepath.Join(out, "stamp")); err == nil && made != "" && string(written) == made {
			fmt.Printf("unchanged  %s\n", out)
			return
		}
	}
	for _, corpus := range corpora {
		if err != nil {
			break
		}
		root := filepath.Join(out, corpus.name)
		if err = os.RemoveAll(root); err != nil {
			break
		}
		r := &recorder{root: root, clock: time.Unix(1551398400, 0).UTC()}
		if err = r.init(strings.ToUpper(corpus.name[:1]) + corpus.name[1:]); err == nil {
			err = corpus.build(r)
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", corpus.name, err)
			break
		}
		fmt.Printf("%s  %s\n", r.head[:12], root)
	}
	if err == nil {
		made, err = stamp(out, names)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(out, "stamp"), []byte(made), 0o644)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkcorpora:", err)
		os.Exit(1)
	}
}

// isolate keeps the environment out of the history: every GIT_ variable is dropped, a
// configuration of this program's own stands in for the user's and the system's, and git
// is not let find a repository above the corpora, which may well be inside origins' own.
// It hands back the configuration's file, to be removed when the corpora are made.
func isolate(out string) (string, error) {
	for _, kv := range os.Environ() {
		if name, _, _ := strings.Cut(kv, "="); strings.HasPrefix(name, "GIT_") {
			os.Unsetenv(name)
		}
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return "", err
	}
	config := filepath.Join(os.TempDir(), fmt.Sprintf("mkcorpora-%d.gitconfig", os.Getpid()))
	if err := os.WriteFile(config, []byte(gitConfig), 0o644); err != nil {
		return "", err
	}
	for name, value := range map[string]string{
		"GIT_CONFIG_GLOBAL":       config,
		"GIT_CONFIG_NOSYSTEM":     "1",
		"GIT_CEILING_DIRECTORIES": out,
		"GIT_AUTHOR_NAME":         identity,
		"GIT_AUTHOR_EMAIL":        email,
		"GIT_COMMITTER_NAME":      identity,
		"GIT_COMMITTER_EMAIL":     email,
	} {
		os.Setenv(name, value)
	}
	return config, nil
}

// stamp says what made the corpora in out and where each stands, or nothing where one
// is missing, has a change in its files or a draft begun.
func stamp(out string, names []string) (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	program, err := os.ReadFile(self)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(program)
	made := "mkcorpora " + hex.EncodeToString(digest[:]) + "\n"
	for _, name := range names {
		root := filepath.Join(out, name)
		if _, err := os.Stat(filepath.Join(root, ".git", "origins-draft.yaml")); err == nil {
			return "", nil
		}
		changed, err := exec.Command("git", "-C", root, "status", "--porcelain").Output()
		if err != nil || len(changed) > 0 {
			return "", nil
		}
		refs, err := exec.Command("git", "-C", root, "for-each-ref", "--format=%(refname) %(objectname)").Output()
		if err != nil {
			return "", nil
		}
		made += name + "\n" + string(refs)
	}
	return made, nil
}

const (
	identity  = "Rosa Bianchi"
	email     = "rosa@example.invalid"
	agent     = "research-agent" // who proposed the large corpus's open hypothesis
	gitConfig = "[user]\n\tname = " + identity + "\n\temail = " + email + "\n" +
		"[core]\n\tautocrlf = false\n[init]\n\tdefaultBranch = main\n[commit]\n\tgpgsign = false\n"
)

// recorder is one corpus being made, and the clock its commits are made at, an hour
// apart.
type recorder struct {
	root  string
	clock time.Time
	head  string
}

func (r *recorder) tick() {
	r.clock = r.clock.Add(time.Hour)
	stamp := fmt.Sprintf("@%d +0000", r.clock.Unix())
	os.Setenv("GIT_AUTHOR_DATE", stamp)
	os.Setenv("GIT_COMMITTER_DATE", stamp)
}

func (r *recorder) init(name string) error {
	r.tick()
	begun, err := api.Init(r.root, name)
	if err != nil {
		return err
	}
	r.head = begun.Commit
	return nil
}

// write lays entities out where store says they are kept, as store writes them.
func (r *recorder) write(entities ...any) error {
	for _, entity := range entities {
		relative, err := store.PathOf(entity)
		if err != nil {
			return err
		}
		content, err := store.Emit(entity)
		if err != nil {
			return fmt.Errorf("%s: %w", relative, err)
		}
		path := filepath.Join(r.root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (r *recorder) project(p model.Project) error {
	p.FormatVersion = model.FormatVersion
	content, err := store.EmitProject(p)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.root, store.ProjectFile), content, 0o644)
}

// adopt records what has been written, and refuses a file adopting had to rewrite,
// since that would be a file store did not write canonically.
func (r *recorder) adopt(subject string) error {
	r.tick()
	adoption, err := api.Adopt(api.Asking{Root: r.root, Command: "adopt", Message: subject})
	if err != nil {
		return err
	}
	if adoption == nil {
		return fmt.Errorf("%q: nothing had changed", subject)
	}
	if len(adoption.Emitted) > 0 {
		return fmt.Errorf("%q: adopting rewrote %s", subject, strings.Join(adoption.Emitted, ", "))
	}
	r.head = adoption.Commit
	return nil
}

func entity(kind model.Kind, slug model.Slug) model.EntityRef {
	return model.EntityRef{Kind: kind, Slug: slug}
}

// ---------------------------------------------------------------------- sparse

// sparse is a record of each kind holding nothing the reader can do without: a family
// has its two partners and a hypothesis an edit, which check requires, and a base to
// apply it at, which supposing it does.
func sparse(r *recorder) error {
	person := &model.Person{Slug: "rossi-giovanni"}
	family := &model.Family{Partners: model.PartnersOf(person.Slug, model.Unknown)}
	slug, err := model.FamilySlug(family.Partners)
	if err != nil {
		return err
	}
	family.Slug = slug
	if err := r.writeAndAdopt([]any{
		person,
		family,
		&model.Place{Slug: "firenze"},
		&model.Source{Slug: "parish-register"},
		&model.Repository{Slug: "archivio"},
		&model.Note{Slug: "research"},
	}, "Write a record of each kind"); err != nil {
		return err
	}
	return r.writeAndAdopt([]any{&model.Hypothesis{
		Slug:   "giovanni-rossi",
		Status: model.Open,
		Base:   r.head,
		Edits: []model.Edit{model.Set{
			Entity: entity(model.KindPerson, person.Slug),
			Path:   "name",
			Value:  "Giovanni Rossi",
		}},
	}}, "Write a hypothesis")
}

// ----------------------------------------------------------------------- large

const (
	people      = 5000
	generations = 12
	growth      = 1.38 // each generation of descendants over the one before
	lastYear    = 2024
)

// word is a name as its culture writes it, and in Latin letters where it writes in
// another script.
type word struct{ text, latin string }

func (w word) roman() string {
	if w.latin != "" {
		return w.latin
	}
	return w.text
}

// pair is a surname as a man and as a woman bear it, where the two differ.
type pair struct{ male, female word }

func (p pair) of(male bool) word {
	if male || p.female == (word{}) {
		return p.male
	}
	return p.female
}

// form is how a culture builds a name.
type form int

const (
	inherited    form = iota // a given name and the father's surname
	middle                   // two given names, the second a middle name
	called                   // two given names, going by the second
	both                     // the father's surname, then the mother's
	reversed                 // the mother's surname, then the father's, filed under his
	fatherly                 // the father's name made a patronymic, and no surname
	russian                  // a patronymic between the given name and the surname
	nasab                    // the father's name after ibn or bint, then the nisba
	marathi                  // the father's given name, then the surname
	generational             // a given name whose first half siblings share
	daily                    // the name of the day of the week of birth
)

// culture is where people are born, with how they are named, the region and the towns,
// the registers that record them there and what a register says.
type culture struct {
	lang, latn   model.Language // the names' language, and its romanization where not in Latin letters
	form         form
	surnameFirst bool
	region       word
	regionType   string
	towns        []spot
	men, women   []word
	surnames     []pair

	// generations, sons and daughters are the halves of a generational given name,
	// written romanized with joiner between them
	generations, sons, daughters []word
	joiner                       string

	older, civil register
	since        int // the year the state began registering, and the older register stops being cited

	born, died [2]string // what a register says of a birth and of a death, of a man and of a woman
	married    string
	dated      func(time.Time) string

	first int // where its towns start among the world's
}

// register is a culture's kind of register, its title and its archive's name written
// round the name of a town.
type register struct {
	title, archive word
	kind           string
	events         [3]string // what it records of a birth, a marriage and a death
}

type spot struct {
	name     word
	lat, lon float64
}

func latin(names string) []word {
	var out []word
	for _, name := range strings.Fields(names) {
		out = append(out, word{text: name})
	}
	return out
}

// scripted is names given each as itself, then in Latin letters.
func scripted(names ...string) []word {
	var out []word
	for i := 0; i < len(names); i += 2 {
		out = append(out, word{names[i], names[i+1]})
	}
	return out
}

func unisex(names []word) []pair {
	var out []pair
	for _, name := range names {
		out = append(out, pair{male: name})
	}
	return out
}

// ov is Russian surnames in -ов, -ев and -ин, which a woman bears with an -а.
func ov(names ...string) []pair {
	var out []pair
	for _, name := range scripted(names...) {
		out = append(out, pair{name, word{name.text + "а", name.latin + "a"}})
	}
	return out
}

// gendered is surnames given as a man's, then a woman's, each itself and then in Latin
// letters.
func gendered(names ...string) []pair {
	var out []pair
	for i := 0; i < len(names); i += 4 {
		out = append(out, pair{word{names[i], names[i+1]}, word{names[i+2], names[i+3]}})
	}
	return out
}

func spots(at ...any) []spot {
	var out []spot
	for i := 0; i < len(at); i += 4 {
		name := word{text: at[i].(string), latin: at[i+1].(string)}
		out = append(out, spot{name, at[i+2].(float64), at[i+3].(float64)})
	}
	return out
}

// writes says a day by pattern, given the day, the month and the year, the month by its
// name where months names them and as a number where not.
func writes(pattern, months string) func(time.Time) string {
	names := strings.Fields(months)
	return func(t time.Time) string {
		var month any = int(t.Month())
		if len(names) > 0 {
			month = names[t.Month()-1]
		}
		return fmt.Sprintf(pattern, t.Day(), month, t.Year())
	}
}

func alike(s string) [2]string { return [2]string{s, s} }

var rites = [3]string{"baptism", "marriage", "burial"}
var vitals = [3]string{"birth", "marriage", "death"}

var english = struct {
	born, died [2]string
	married    string
	dated      func(time.Time) string
}{alike("%[1]s, born %[2]s."), alike("%[1]s, died %[2]s."), "%[1]s and %[2]s, married %[3]s.",
	writes("%[1]d %[2]v %[3]d", "January February March April May June July August September October November December")}

var cultures = []*culture{
	{
		lang: "it", region: word{text: "Toscana"}, regionType: "region",
		towns: spots("Firenze", "", 43.7696, 11.2558, "Prato", "", 43.8777, 11.1022, "Pistoia", "", 43.9336, 10.9173),
		men: latin("Giovanni Giuseppe Antonio Francesco Luigi Pietro Angelo Domenico Carlo Paolo " +
			"Lorenzo Andrea Michele Stefano Bartolomeo Jacopo Cosimo Niccolò"),
		women: latin("Maria Anna Lucia Caterina Teresa Rosa Giovanna Margherita Elisabetta Francesca " +
			"Assunta Annunziata Maddalena Agnese Ottavia Clelia Costanza Livia"),
		surnames: unisex(latin("Rossi Bianchi Verdi Conti Ferri Galli Moretti Sarti Pieri Mazzoni " +
			"Bruni Fabbri Lombardi Marchetti Giusti Bellini Neri Pellegrini")),
		older: register{word{text: "Parrocchia di %s — Registri dei battesimi, dei matrimoni e dei morti"},
			word{text: "Archivio Diocesano di %s"}, "parish-register", rites},
		civil: register{word{text: "Comune di %s — Registri dello stato civile"},
			word{text: "Archivio di Stato di %s"}, "civil-registration", vitals},
		since: 1866,
		born:  [2]string{"%[1]s, nato il %[2]s.", "%[1]s, nata il %[2]s."},
		died:  [2]string{"%[1]s, morto il %[2]s.", "%[1]s, morta il %[2]s."}, married: "%[1]s e %[2]s, sposati il %[3]s.",
		dated: writes("%[1]d %[2]v %[3]d", "gennaio febbraio marzo aprile maggio giugno luglio agosto settembre ottobre novembre dicembre"),
	},
	{
		lang: "es", form: both, region: word{text: "Andalucía"}, regionType: "region",
		towns: spots("Sevilla", "", 37.3891, -5.9845, "Carmona", "", 37.4712, -5.6461, "Écija", "", 37.5422, -5.0826),
		men:   latin("José Juan Francisco Antonio Manuel Pedro Miguel Diego Alonso Rafael Fernando Luis"),
		women: latin("María Ana Josefa Isabel Francisca Teresa Carmen Juana Rosa Dolores Inés Catalina"),
		surnames: unisex(latin("García Fernández González Rodríguez López Martínez Sánchez Pérez Gómez " +
			"Ruiz Marchena Romero Navarro Benítez")),
		older: register{word{text: "Parroquia de %s — Libros de bautismos, matrimonios y defunciones"},
			word{text: "Archivo General del Arzobispado de %s"}, "parish-register", rites},
		civil: register{word{text: "Registro Civil de %s"},
			word{text: "Archivo Histórico Provincial de %s"}, "civil-registration", vitals},
		since: 1871,
		born:  [2]string{"%[1]s, nacido el %[2]s.", "%[1]s, nacida el %[2]s."},
		died:  [2]string{"%[1]s, fallecido el %[2]s.", "%[1]s, fallecida el %[2]s."}, married: "%[1]s y %[2]s, casados el %[3]s.",
		dated: writes("%[1]d de %[2]v de %[3]d", "enero febrero marzo abril mayo junio julio agosto septiembre octubre noviembre diciembre"),
	},
	{
		lang: "pt", form: reversed, region: word{text: "Estremadura"}, regionType: "province",
		towns: spots("Lisboa", "", 38.7223, -9.1393, "Sintra", "", 38.8029, -9.3817, "Setúbal", "", 38.5244, -8.8882),
		men:   latin("João José António Manuel Francisco Joaquim Pedro Luís Domingos Bernardo"),
		women: latin("Maria Ana Joana Rosa Teresa Helena Isabel Inês Francisca Gertrudes"),
		surnames: unisex(latin("Silva Santos Ferreira Pereira Oliveira Costa Rodrigues Martins Sousa " +
			"Galvão Carvalho Almeida")),
		older: register{word{text: "Paróquia de %s — Livros de baptismos, casamentos e óbitos"},
			word{text: "Arquivo Distrital de %s"}, "parish-register", rites},
		civil: register{word{text: "Conservatória do Registo Civil de %s"},
			word{text: "Arquivo Municipal de %s"}, "civil-registration", vitals},
		since: 1911,
		born:  [2]string{"%[1]s, nascido a %[2]s.", "%[1]s, nascida a %[2]s."},
		died:  [2]string{"%[1]s, falecido a %[2]s.", "%[1]s, falecida a %[2]s."}, married: "%[1]s e %[2]s, casados a %[3]s.",
		dated: writes("%[1]d de %[2]v de %[3]d", "janeiro fevereiro março abril maio junho julho agosto setembro outubro novembro dezembro"),
	},
	{
		lang: "en", form: middle, region: word{text: "Yorkshire"}, regionType: "county",
		towns: spots("York", "", 53.9600, -1.0873, "Leeds", "", 53.8008, -1.5491, "Whitby", "", 54.4858, -0.6206),
		men:   latin("John William Thomas James George Henry Richard Robert Edward Charles Joseph Samuel"),
		women: latin("Mary Elizabeth Ann Sarah Jane Margaret Hannah Martha Alice Emily Ellen Susannah"),
		surnames: unisex(latin("Smith Taylor Brown Wilson Walker Wood Robinson Thompson Wright Hall " +
			"Atkinson Hargreaves")),
		older: register{word{text: "Parish of %s — Registers of baptisms, marriages and burials"},
			word{text: "%s Diocesan Archives"}, "parish-register", rites},
		civil: register{word{text: "General Register Office — %s registration district"},
			word{text: "%s Record Office"}, "civil-registration", vitals},
		since: 1837,
		born:  english.born, died: english.died, married: english.married, dated: english.dated,
	},
	{
		lang: "de", form: called, region: word{text: "Württemberg"}, regionType: "region",
		towns: spots("Stuttgart", "", 48.7758, 9.1829, "Esslingen", "", 48.7406, 9.3108, "Tübingen", "", 48.5216, 9.0576),
		men:   latin("Johann Georg Friedrich Wilhelm Karl Heinrich Jakob Christian Gottlieb Ludwig Michael Andreas"),
		women: latin("Anna Maria Katharina Margarethe Christina Barbara Elisabeth Rosina Friederike Luise Dorothea Regina"),
		surnames: unisex(latin("Müller Schmid Schneider Fischer Weber Wagner Bauer Lang Keller Maier " +
			"Hofmann Kurz")),
		older: register{word{text: "Evangelische Kirchengemeinde %s — Kirchenbücher"},
			word{text: "Landeskirchliches Archiv %s"}, "parish-register", rites},
		civil: register{word{text: "Standesamt %s — Personenstandsregister"},
			word{text: "Stadtarchiv %s"}, "civil-registration", vitals},
		since: 1876,
		born:  alike("%[1]s, geboren am %[2]s."), died: alike("%[1]s, gestorben am %[2]s."),
		married: "%[1]s und %[2]s, getraut am %[3]s.",
		dated:   writes("%[1]d. %[2]v %[3]d", "Januar Februar März April Mai Juni Juli August September Oktober November Dezember"),
	},
	{
		lang: "is", form: fatherly, region: word{text: "Ísland"}, regionType: "country",
		towns: spots("Reykjavík", "", 64.1466, -21.9426, "Akureyri", "", 65.6885, -18.1262, "Ísafjörður", "", 66.0750, -23.1240),
		men:   latin("Jón Gunnar Ólafur Einar Magnús Bjarni Stefán Páll Árni Helgi Kristján Jóhann"),
		women: latin("Guðrún Sigríður Kristín Margrét Helga Anna Ingibjörg Jóhanna Sólveig Halldóra Katrín Ragnheiður"),
		older: register{word{text: "Prestsþjónustubók — %s"},
			word{text: "Biskupsskjalasafn — %s"}, "parish-register", rites},
		civil: register{word{text: "Þjóðskrá — %s"},
			word{text: "Héraðsskjalasafn — %s"}, "civil-registration", vitals},
		since: 1953,
		born:  [2]string{"%[1]s, fæddur %[2]s.", "%[1]s, fædd %[2]s."},
		died:  [2]string{"%[1]s, dáinn %[2]s.", "%[1]s, dáin %[2]s."}, married: "%[1]s og %[2]s, gefin saman %[3]s.",
		dated: writes("%[1]d. %[2]v %[3]d", "janúar febrúar mars apríl maí júní júlí ágúst september október nóvember desember"),
	},
	{
		lang: "ru", latn: "ru-Latn", form: russian, region: word{"Пермская губерния", "Permskaya guberniya"}, regionType: "governorate",
		towns: spots("Пермь", "Perm", 58.0105, 56.2502, "Нижний Тагил", "Nizhny Tagil", 57.9194, 59.9650, "Кунгур", "Kungur", 57.4284, 56.9439),
		men: scripted("Иван", "Ivan", "Алексей", "Aleksey", "Николай", "Nikolay", "Сергей", "Sergey",
			"Андрей", "Andrey", "Фёдор", "Fyodor", "Степан", "Stepan", "Борис", "Boris", "Тимофей", "Timofey",
			"Матвей", "Matvey", "Роман", "Roman", "Глеб", "Gleb", "Кирилл", "Kirill"),
		women: scripted("Мария", "Mariya", "Анна", "Anna", "Екатерина", "Yekaterina", "Елена", "Yelena",
			"Ольга", "Olga", "Татьяна", "Tatyana", "Наталья", "Natalya", "Анастасия", "Anastasiya",
			"Ксения", "Kseniya", "Евдокия", "Yevdokiya", "Аграфена", "Agrafena", "Пелагея", "Pelageya"),
		surnames: ov("Иванов", "Ivanov", "Петров", "Petrov", "Смирнов", "Smirnov", "Кузнецов", "Kuznetsov",
			"Попов", "Popov", "Соколов", "Sokolov", "Лебедев", "Lebedev", "Морозов", "Morozov",
			"Волков", "Volkov", "Зайцев", "Zaytsev", "Орлов", "Orlov", "Ершов", "Ershov"),
		older: register{word{"Метрические книги — %s", "Metricheskie knigi — %s"},
			word{"Государственный архив — %s", "Gosudarstvenny arkhiv — %s"}, "parish-register", rites},
		civil: register{word{"Записи актов гражданского состояния — %s", "Zapisi aktov grazhdanskogo sostoyaniya — %s"},
			word{"Архив ЗАГС — %s", "Arkhiv ZAGS — %s"}, "civil-registration", vitals},
		since: 1918,
		born:  [2]string{"%[1]s, родился %[2]s.", "%[1]s, родилась %[2]s."},
		died:  [2]string{"%[1]s, умер %[2]s.", "%[1]s, умерла %[2]s."}, married: "%[1]s и %[2]s, венчались %[3]s.",
		dated: writes("%[1]d %[2]v %[3]d года", "января февраля марта апреля мая июня июля августа сентября октября ноября декабря"),
	},
	{
		lang: "el", latn: "el-Latn", region: word{"Μακεδονία", "Makedonia"}, regionType: "region",
		towns: spots("Θεσσαλονίκη", "Thessaloniki", 40.6401, 22.9444, "Βέροια", "Veroia", 40.5240, 22.2020, "Κοζάνη", "Kozani", 40.3007, 21.7887),
		men: scripted("Γεώργιος", "Georgios", "Ιωάννης", "Ioannis", "Κωνσταντίνος", "Konstantinos",
			"Δημήτριος", "Dimitrios", "Νικόλαος", "Nikolaos", "Παναγιώτης", "Panagiotis", "Βασίλειος", "Vasileios",
			"Χρήστος", "Christos", "Αθανάσιος", "Athanasios", "Μιχαήλ", "Michail"),
		women: scripted("Μαρία", "Maria", "Ελένη", "Eleni", "Αικατερίνη", "Aikaterini", "Βασιλική", "Vasiliki",
			"Σοφία", "Sofia", "Αναστασία", "Anastasia", "Ευαγγελία", "Evangelia", "Δέσποινα", "Despoina",
			"Ειρήνη", "Eirini", "Παρασκευή", "Paraskevi"),
		surnames: gendered(
			"Παπαδόπουλος", "Papadopoulos", "Παπαδοπούλου", "Papadopoulou",
			"Παπαδάκης", "Papadakis", "Παπαδάκη", "Papadaki",
			"Γεωργίου", "Georgiou", "", "",
			"Νικολαΐδης", "Nikolaidis", "Νικολαΐδου", "Nikolaidou",
			"Οικονόμου", "Oikonomou", "", "",
			"Αντωνίου", "Antoniou", "", "",
			"Καραγιάννης", "Karagiannis", "Καραγιάννη", "Karagianni",
			"Βασιλείου", "Vasileiou", "", "",
			"Μακρής", "Makris", "Μακρή", "Makri",
			"Ιωαννίδης", "Ioannidis", "Ιωαννίδου", "Ioannidou"),
		older: register{word{"Ληξιαρχικά βιβλία ενορίας — %s", "Lixiarchika vivlia enorias — %s"},
			word{"Μητρόπολη — %s", "Mitropoli — %s"}, "parish-register", rites},
		civil: register{word{"Ληξιαρχείο — %s", "Lixiarcheio — %s"},
			word{"Γενικά Αρχεία του Κράτους — %s", "Genika Archeia tou Kratous — %s"}, "civil-registration", vitals},
		since: 1925,
		born:  alike("%[1]s, γεννήθηκε στις %[2]s."), died: alike("%[1]s, πέθανε στις %[2]s."),
		married: "%[1]s και %[2]s, παντρεύτηκαν στις %[3]s.",
		dated: writes("%[1]d %[2]v %[3]d", "Ιανουαρίου Φεβρουαρίου Μαρτίου Απριλίου Μαΐου Ιουνίου Ιουλίου "+
			"Αυγούστου Σεπτεμβρίου Οκτωβρίου Νοεμβρίου Δεκεμβρίου"),
	},
	{
		lang: "ar", latn: "ar-Latn", form: nasab, region: word{"بلاد الشام", "Bilād al-Shām"}, regionType: "region",
		towns: spots("حلب", "Ḥalab", 36.2021, 37.1343, "دمشق", "Dimashq", 33.5138, 36.2765, "حمص", "Ḥimṣ", 34.7324, 36.7137),
		men: scripted("محمد", "Muḥammad", "أحمد", "Aḥmad", "علي", "ʿAlī", "حسن", "Ḥasan", "حسين", "Ḥusayn",
			"عمر", "ʿUmar", "يوسف", "Yūsuf", "إبراهيم", "Ibrāhīm", "خليل", "Khalīl", "محمود", "Maḥmūd",
			"سليمان", "Sulaymān", "عبد الله", "ʿAbd Allāh"),
		women: scripted("فاطمة", "Fāṭima", "زينب", "Zaynab", "عائشة", "ʿĀʾisha", "خديجة", "Khadīja",
			"مريم", "Maryam", "آمنة", "Āmina", "سعاد", "Suʿād", "ليلى", "Laylā", "نور", "Nūr", "هدى", "Hudā"),
		surnames: unisex(scripted("الحلبي", "al-Ḥalabī", "الدمشقي", "al-Dimashqī", "البغدادي", "al-Baghdādī",
			"المصري", "al-Miṣrī", "الحمصي", "al-Ḥimṣī", "الأنصاري", "al-Anṣārī", "الحسيني", "al-Ḥusaynī",
			"التميمي", "al-Tamīmī", "الخطيب", "al-Khaṭīb", "النجار", "al-Najjār")),
		older: register{word{"سجلات المحكمة الشرعية — %s", "Sijillāt al-Maḥkama al-Sharʿiyya — %s"},
			word{"مركز الوثائق التاريخية — %s", "Markaz al-Wathāʾiq al-Tārīkhiyya — %s"}, "court-register", vitals},
		civil: register{word{"سجلات النفوس — %s", "Sijillāt al-Nufūs — %s"},
			word{"دائرة النفوس — %s", "Dāʾirat al-Nufūs — %s"}, "civil-registration", vitals},
		since: 1881,
		born:  [2]string{"%[1]s، وُلد في %[2]s.", "%[1]s، وُلدت في %[2]s."},
		died:  [2]string{"%[1]s، تُوفي في %[2]s.", "%[1]s، تُوفيت في %[2]s."}, married: "%[1]s و%[2]s، تزوجا في %[3]s.",
		dated: writes("%[1]d/%[2]v/%[3]d", ""),
	},
	{
		lang: "fa", latn: "fa-Latn", region: word{"عراق عجم", "Erāq-e Ajam"}, regionType: "region",
		towns: spots("تهران", "Tehrān", 35.6892, 51.3890, "قزوین", "Qazvin", 36.2797, 50.0049, "کاشان", "Kāshān", 33.9850, 51.4100),
		men: scripted("علی", "Ali", "حسین", "Hossein", "محمد", "Mohammad", "رضا", "Reza", "حسن", "Hassan",
			"مهدی", "Mehdi", "محمود", "Mahmoud", "احمد", "Ahmad", "عباس", "Abbas", "کریم", "Karim"),
		women: scripted("فاطمه", "Fatemeh", "زهرا", "Zahra", "مریم", "Maryam", "پروین", "Parvin",
			"معصومه", "Masoumeh", "شیرین", "Shirin", "رویا", "Roya", "صغری", "Soghra", "خدیجه", "Khadijeh", "عذرا", "Ozra"),
		surnames: unisex(scripted("رحیمی", "Rahimi", "حسینی", "Hosseini", "احمدی", "Ahmadi", "محمدی", "Mohammadi",
			"کریمی", "Karimi", "رضایی", "Rezaei", "موسوی", "Mousavi", "جعفری", "Jafari", "صادقی", "Sadeghi", "نوری", "Nouri")),
		older: register{word{"دفاتر محضر — %s", "Dafāter-e Mahzar — %s"},
			word{"سازمان اسناد ملی — %s", "Sāzmān-e Asnād-e Melli — %s"}, "notarial-register", vitals},
		civil: register{word{"ثبت احوال — %s", "Sabt-e Ahvāl — %s"},
			word{"اداره ثبت احوال — %s", "Edāre-ye Sabt-e Ahvāl — %s"}, "civil-registration", vitals},
		since: 1925,
		born:  alike("%[1]s، متولد %[2]s."), died: alike("%[1]s، درگذشته %[2]s."), married: "%[1]s و %[2]s، ازدواج %[3]s.",
		dated: writes("%[1]d/%[2]v/%[3]d", ""),
	},
	{
		lang: "he", latn: "he-Latn", region: word{"הגליל", "ha-Galil"}, regionType: "region",
		towns: spots("צפת", "Tsfat", 32.9646, 35.4960, "טבריה", "Tveriya", 32.7922, 35.5312, "פקיעין", "Peki'in", 32.9761, 35.3297),
		men: scripted("אברהם", "Avraham", "יצחק", "Yitzhak", "יעקב", "Yaakov", "משה", "Moshe", "דוד", "David",
			"שלמה", "Shlomo", "יוסף", "Yosef", "אהרן", "Aharon", "שמואל", "Shmuel", "חיים", "Haim"),
		women: scripted("שרה", "Sarah", "רבקה", "Rivka", "רחל", "Rachel", "לאה", "Leah", "מרים", "Miriam",
			"אסתר", "Esther", "חנה", "Hannah", "דבורה", "Devorah", "יהודית", "Yehudit", "שושנה", "Shoshana"),
		surnames: unisex(scripted("כהן", "Cohen", "לוי", "Levi", "מזרחי", "Mizrahi", "פרץ", "Peretz", "ביטון", "Biton",
			"אזולאי", "Azoulay", "אלבז", "Elbaz", "אברבנאל", "Abravanel", "נבון", "Navon", "אשכנזי", "Ashkenazi")),
		older: register{word{"פנקס הקהילה — %s", "Pinkas ha-Kehila — %s"},
			word{"הארכיון המרכזי לתולדות העם היהודי — %s", "ha-Arkhiyon ha-Merkazi le-Toldot ha-Am ha-Yehudi — %s"},
			"community-register", vitals},
		civil: register{word{"מרשם האוכלוסין — %s", "Mirsham ha-Ukhlusin — %s"},
			word{"ארכיון המדינה — %s", "Arkhiyon ha-Medina — %s"}, "civil-registration", vitals},
		since: 1948,
		born:  [2]string{"%[1]s, נולד ביום %[2]s.", "%[1]s, נולדה ביום %[2]s."},
		died:  [2]string{"%[1]s, נפטר ביום %[2]s.", "%[1]s, נפטרה ביום %[2]s."}, married: "%[1]s ו%[2]s, נישאו ביום %[3]s.",
		dated: writes("%[1]d.%[2]v.%[3]d", ""),
	},
	{
		lang: "zh-Hant", latn: "zh-Latn-pinyin", form: generational, surnameFirst: true,
		region: word{"浙江", "Zhèjiāng"}, regionType: "province",
		towns: spots("溫州", "Wēnzhōu", 27.9938, 120.6994, "瑞安", "Ruì'ān", 27.7781, 120.6553, "樂清", "Yuèqīng", 28.1124, 120.9614),
		surnames: unisex(scripted("陳", "Chén", "林", "Lín", "黃", "Huáng", "張", "Zhāng", "李", "Lǐ",
			"王", "Wáng", "吳", "Wú", "劉", "Liú", "蔡", "Cài", "楊", "Yáng")),
		generations: scripted("家", "Jiā", "玉", "Yù", "永", "Yǒng", "德", "Dé", "文", "Wén", "世", "Shì", "國", "Guó", "志", "Zhì"),
		sons:        scripted("明", "míng", "華", "huá", "強", "qiáng", "輝", "huī", "傑", "jié", "福", "fú", "成", "chéng", "生", "shēng"),
		daughters:   scripted("英", "yīng", "蘭", "lán", "珍", "zhēn", "秀", "xiù", "芳", "fāng", "梅", "méi", "鳳", "fèng", "美", "měi"),
		older: register{word{"族譜 — %s", "Zúpǔ — %s"},
			word{"宗祠 — %s", "Zōngcí — %s"}, "genealogy", vitals},
		civil: register{word{"戶籍 — %s", "Hùjí — %s"},
			word{"檔案館 — %s", "Dàng'ànguǎn — %s"}, "civil-registration", vitals},
		since: 1931,
		born:  alike("%[1]s，%[2]s生。"), died: alike("%[1]s，%[2]s卒。"), married: "%[1]s與%[2]s，%[3]s成婚。",
		dated: writes("%[3]d年%[2]v月%[1]d日", ""),
	},
	{
		lang: "ja", latn: "ja-Latn", surnameFirst: true, region: word{"岩手", "Iwate"}, regionType: "prefecture",
		towns: spots("盛岡", "Morioka", 39.7036, 141.1527, "遠野", "Tōno", 39.3275, 141.5336, "花巻", "Hanamaki", 39.3886, 141.1169),
		men: scripted("太郎", "Tarō", "一郎", "Ichirō", "健二", "Kenji", "清", "Kiyoshi", "茂", "Shigeru",
			"博", "Hiroshi", "正雄", "Masao", "勇", "Isamu", "三郎", "Saburō", "誠", "Makoto"),
		women: scripted("花子", "Hanako", "由美子", "Yumiko", "幸子", "Sachiko", "和子", "Kazuko", "節子", "Setsuko",
			"久子", "Hisako", "千代", "Chiyo", "文子", "Fumiko", "春子", "Haruko", "菊", "Kiku"),
		surnames: unisex(scripted("佐藤", "Satō", "鈴木", "Suzuki", "高橋", "Takahashi", "田中", "Tanaka",
			"渡辺", "Watanabe", "伊藤", "Itō", "山本", "Yamamoto", "中村", "Nakamura", "小林", "Kobayashi", "加藤", "Katō")),
		older: register{word{"宗門人別改帳 — %s", "Shūmon ninbetsu aratamechō — %s"},
			word{"寺院 — %s", "Jiin — %s"}, "religious-census", vitals},
		civil: register{word{"戸籍 — %s", "Koseki — %s"},
			word{"市役所 — %s", "Shiyakusho — %s"}, "civil-registration", vitals},
		since: 1872,
		born:  alike("%[1]s、%[2]s生。"), died: alike("%[1]s、%[2]s没。"), married: "%[1]sと%[2]s、%[3]s婚姻。",
		dated: writes("%[3]d年%[2]v月%[1]d日", ""),
	},
	{
		lang: "ko", latn: "ko-Latn", form: generational, surnameFirst: true, joiner: "-",
		region: word{"경상도", "Gyeongsang-do"}, regionType: "province",
		towns: spots("부산", "Busan", 35.1796, 129.0756, "경주", "Gyeongju", 35.8562, 129.2247, "대구", "Daegu", 35.8714, 128.6014),
		surnames: unisex(scripted("김", "Gim", "이", "I", "박", "Bak", "최", "Choe", "정", "Jeong",
			"강", "Gang", "조", "Jo", "윤", "Yun", "장", "Jang", "임", "Im")),
		generations: scripted("서", "Seo", "민", "Min", "현", "Hyeon", "준", "Jun", "영", "Yeong", "성", "Seong"),
		sons:        scripted("호", "ho", "수", "su", "철", "cheol", "우", "u", "석", "seok", "진", "jin"),
		daughters:   scripted("연", "yeon", "희", "hui", "숙", "suk", "경", "gyeong", "미", "mi", "정", "jeong"),
		older: register{word{"족보 — %s", "Jokbo — %s"},
			word{"종친회 — %s", "Jongchinhoe — %s"}, "genealogy", vitals},
		civil: register{word{"호적 — %s", "Hojeok — %s"},
			word{"국가기록원 — %s", "Gukga Girogwon — %s"}, "civil-registration", vitals},
		since: 1909,
		born:  alike("%[1]s, %[2]s 출생."), died: alike("%[1]s, %[2]s 사망."), married: "%[1]s·%[2]s, %[3]s 혼인.",
		dated: writes("%[3]d년 %[2]v월 %[1]d일", ""),
	},
	{
		lang: "mr", latn: "mr-Latn", form: marathi, region: word{"महाराष्ट्र", "Maharashtra"}, regionType: "region",
		towns: spots("पुणे", "Pune", 18.5204, 73.8567, "सातारा", "Satara", 17.6805, 74.0183, "कासेगाव", "Kasegaon", 17.1286, 74.2178),
		men: scripted("रमेश", "Ramesh", "विठ्ठल", "Vitthal", "गणेश", "Ganesh", "शंकर", "Shankar", "नारायण", "Narayan",
			"दत्तात्रय", "Dattatray", "बाळकृष्ण", "Balkrishna", "महादेव", "Mahadev", "सदाशिव", "Sadashiv", "केशव", "Keshav"),
		women: scripted("सीता", "Sita", "लक्ष्मी", "Lakshmi", "पार्वती", "Parvati", "गंगा", "Ganga", "राधा", "Radha",
			"सावित्री", "Savitri", "यमुना", "Yamuna", "सरस्वती", "Saraswati", "कमला", "Kamala", "आनंदी", "Anandi"),
		surnames: unisex(scripted("कुलकर्णी", "Kulkarni", "देशपांडे", "Deshpande", "जोशी", "Joshi", "पाटील", "Patil",
			"गोखले", "Gokhale", "आपटे", "Apte", "केळकर", "Kelkar", "भोसले", "Bhosale", "शिंदे", "Shinde", "पवार", "Pawar")),
		older: register{word{"वंशावळ — %s", "Vanshaval — %s"},
			word{"पुरोहित दप्तर — %s", "Purohit Daftar — %s"}, "genealogy", vitals},
		civil: register{word{"जन्म-मृत्यू नोंदणी — %s", "Janma-Mrutyu Nondani — %s"},
			word{"महाराष्ट्र राज्य अभिलेखागार — %s", "Maharashtra Rajya Abhilekhagar — %s"}, "civil-registration", vitals},
		since: 1886,
		born:  alike("%[1]s, जन्म %[2]s."), died: alike("%[1]s, मृत्यू %[2]s."), married: "%[1]s व %[2]s, विवाह %[3]s.",
		dated: writes("%[1]d/%[2]v/%[3]d", ""),
	},
	{
		lang: "ak", form: daily, region: word{text: "Asante"}, regionType: "region",
		towns: spots("Kumasi", "", 6.6885, -1.6244, "Obuasi", "", 6.2023, -1.6704, "Ejisu", "", 6.7230, -1.4701),
		// from Sunday to Saturday
		men:      latin("Kwasi Kwadwo Kwabena Kwaku Yaw Kofi Kwame"),
		women:    latin("Akosua Adwoa Abena Akua Yaa Afua Ama"),
		surnames: unisex(latin("Ofori Mensah Owusu Boateng Asante Osei Appiah Agyeman Darko Amoako")),
		older: register{word{text: "Basel Mission Church, %s — Registers of baptisms, marriages and burials"},
			word{text: "Presbyterian Church Archives, %s"}, "parish-register", rites},
		civil: register{word{text: "Registrar of Births and Deaths, %s"},
			word{text: "Public Records and Archives Administration, %s"}, "civil-registration", vitals},
		since: 1912,
		born:  english.born, died: english.died, married: english.married, dated: english.dated,
	},
}

// town is where people are born, marry and die, with the two registers that record
// them there.
type town struct {
	name             word
	slug             string
	culture          *culture
	parish, register *model.Source
	entries          map[string]int // citations made in a source in a year
}

type person struct {
	record     *model.Person
	culture    *culture
	given      word // what a child's patronymic is made of
	family     pair // the surname handed down, where the culture hands one down
	generation int
	male       bool
	born       time.Time
	died       time.Time // zero where they are living, or died after the corpus was made
	town       int
	families   []*family
}

type family struct {
	record          *model.Family
	husband, wife   *person
	married         time.Time
	town            int
	children        []*person
	generation      word // the half of a given name its children share, where they share one
	firstFree, last int  // the years a child could next be born in, and the last one
}

type world struct {
	rand     *rand.Rand
	corpus   *model.Corpus
	namer    *importing.Namer
	towns    []*town
	people   []*person
	families []*family
}

func large(r *recorder) error {
	w := &world{rand: rand.New(rand.NewPCG(1, 2)), corpus: model.NewCorpus()}
	w.namer = importing.NewNamer(w.corpus, nil)
	w.places()

	// generation 0 is a couple in each culture nobody descends from here, and each after
	// it is the children of the one before, as many as growth says, those who live to
	// marry marrying somebody from outside. The last is whatever is left of the five
	// thousand.
	var parents []*family
	for _, c := range cultures {
		husband := w.newPerson(0, true, w.date(1650, 1672), c.first+w.rand.IntN(len(c.towns)), nil)
		parents = append(parents, w.marry(husband))
	}
	target := float64(len(cultures)) * 2
	for g := 1; g < generations; g++ {
		count := int(target)
		if g == generations-1 {
			count = people - len(w.people)
		}
		children := w.bear(parents, count)
		parents = nil
		for _, child := range children {
			if g < generations-1 && w.rand.Float64() < 0.13 {
				w.dieYoung(child)
				continue
			}
			if g < generations-1 && w.rand.Float64() < 0.86 {
				parents = append(parents, w.marry(child))
			}
		}
		target *= growth
	}
	if len(w.people) != people {
		return fmt.Errorf("made %d people rather than %d", len(w.people), people)
	}
	for _, p := range w.people {
		w.die(p)
	}
	w.cite()
	notes := w.notes()

	// The registers go first, since every event cites them, then the people, their
	// families and the notes about them in one commit, as an import would land them, with
	// the manifest naming them.
	if err := r.writeAndAdopt(records(w.corpus, model.KindPlace, model.KindRepository, model.KindSource),
		"Write the places, the archives and the registers"); err != nil {
		return err
	}
	everyone := records(w.corpus, model.KindPerson, model.KindFamily, model.KindNote)
	main := w.families[0].children[0].record.Slug
	if err := r.project(model.Project{
		Name:        "Large",
		Description: "Five thousand people over twelve generations, made from one seed.",
		MainPerson:  main,
		Lang:        "en",
		Researchers: []model.Researcher{{Name: identity}},
		Bookmarks: []model.EntityRef{
			entity(model.KindPerson, main),
			entity(model.KindFamily, w.families[0].record.Slug),
			entity(model.KindSource, w.towns[0].parish.Slug),
			entity(model.KindNote, notes[0]),
		},
	}); err != nil {
		return err
	}
	if err := r.writeAndAdopt(everyone, "Write the people, their families and the research notes"); err != nil {
		return err
	}
	hypotheses := w.hypotheses(r.head)
	if err := r.writeAndAdopt(hypotheses[1:], "Write the settled hypothesis"); err != nil {
		return err
	}
	return r.propose(hypotheses[0].(*model.Hypothesis), agent)
}

// propose records h as proposed by whoever by names, as the machine interface would.
func (r *recorder) propose(h *model.Hypothesis, by string) error {
	r.tick()
	document, err := store.Emit(h)
	if err != nil {
		return err
	}
	proposal, err := api.ProposeStated(api.Asking{Root: r.root, Command: "hypothesis propose"},
		h.Slug, document, by)
	if err != nil {
		return err
	}
	r.head = proposal.Commit
	return nil
}

func (r *recorder) writeAndAdopt(entities []any, subject string) error {
	if err := r.write(entities...); err != nil {
		return err
	}
	return r.adopt(subject)
}

// records are the corpus's records of the kinds, kind by kind and in slug order.
func records(c *model.Corpus, kinds ...model.Kind) []any {
	var out []any
	for _, kind := range kinds {
		for _, slug := range c.SlugsOf(kind) {
			out = append(out, c.Entity(slug))
		}
	}
	return out
}

func (w *world) slug(base string, hints ...string) model.Slug {
	s, err := model.Slugify(base)
	if err != nil {
		panic(err)
	}
	return w.namer.Assign(s, hints, base)
}

// placeName is a name as its culture writes it, and the romanization beside it.
func placeName(c *culture, name word) (model.PlaceName, []model.PlaceName) {
	if c.latn == "" {
		return model.PlaceName{Text: name.text}, nil
	}
	return model.PlaceName{Text: name.text, Lang: c.lang},
		[]model.PlaceName{{Text: name.latin, Lang: c.latn, Type: "romanized"}}
}

// places is a region for each culture and three towns in it, and for each town the
// register kept before the state registered anybody and the state's, kept in two
// archives of the region's seat.
func (w *world) places() {
	for _, c := range cultures {
		c.first = len(w.towns)
		name, also := placeName(c, c.region)
		region := &model.Place{Slug: w.slug(c.region.roman()), Name: name, AlsoKnownAs: also, Type: c.regionType}
		w.corpus.Places[region.Slug] = region

		seat := c.towns[0].name
		archive := func(r register) *model.Repository {
			name := word{fmt.Sprintf(r.archive.text, seat.text), ""}
			if r.archive.latin != "" {
				name.latin = fmt.Sprintf(r.archive.latin, seat.latin)
			}
			repository := &model.Repository{Slug: w.slug(name.roman()), Name: name.text, Type: "archive"}
			w.corpus.Repositories[repository.Slug] = repository
			return repository
		}
		older, civil := archive(c.older), archive(c.civil)
		older.Addresses = []model.Address{{Lines: seat.text + "\n"}}
		civil.Links = model.LinksOf("https://www.example.org/archives/" + string(civil.Slug))

		for ti, at := range c.towns {
			kind := "town"
			if ti == 0 {
				kind = "city"
			}
			name, also := placeName(c, at.name)
			p := &model.Place{Slug: w.slug(at.name.roman()), Name: name, AlsoKnownAs: also, Type: kind,
				Within:      []model.PlaceRef{{Place: region.Slug}},
				Coordinates: &model.Coordinates{Latitude: at.lat, Longitude: at.lon}}
			w.corpus.Places[p.Slug] = p
			t := &town{name: at.name, slug: string(p.Slug), culture: c, entries: map[string]int{}}

			source := func(r register, held model.Holding, when string) *model.Source {
				title := word{fmt.Sprintf(r.title.text, at.name.text), ""}
				if r.title.latin != "" {
					title.latin = fmt.Sprintf(r.title.latin, at.name.latin)
				}
				s := &model.Source{Slug: w.slug(title.roman()), Title: title.text, Type: r.kind,
					Covers:       []model.Coverage{{Events: r.events[:], Date: date(when), Place: p.Slug}},
					Repositories: []model.Holding{held},
					Citations:    map[string]model.Citation{}}
				w.corpus.Sources[s.Slug] = s
				return s
			}
			t.parish = source(c.older, model.Holding{Repository: older.Slug,
				CallNumber: fmt.Sprintf("%s/%d", strings.ToUpper(t.slug[:3]), ti+1)}, fmt.Sprint("BEF ", c.since))
			t.register = source(c.civil, model.Holding{Repository: civil.Slug, Medium: "microfilm"},
				fmt.Sprint("FROM ", c.since))
			w.towns = append(w.towns, t)
		}
	}
}

func date(s string) model.Date {
	d, err := model.ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

func day(t time.Time) model.Date { return date(strings.ToUpper(t.Format("2 Jan 2006"))) }

func (w *world) date(from, to int) time.Time {
	return time.Date(from+w.rand.IntN(to-from+1), time.January, 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, 0, w.rand.IntN(365))
}

func (w *world) pick(from []word) word { return from[w.rand.IntN(len(from))] }

// given is a name of somebody's own, as their culture gives one to a child born into f,
// or to somebody whose parents are not here where f is nil.
func (w *world) given(c *culture, male bool, born time.Time, f *family) word {
	names := c.women
	if male {
		names = c.men
	}
	switch c.form {
	case daily:
		return names[born.Weekday()]
	case generational:
		shared := w.pick(c.generations)
		if f != nil {
			if f.generation == (word{}) {
				f.generation = shared
			}
			shared = f.generation
		}
		own := w.pick(c.daughters)
		if male {
			own = w.pick(c.sons)
		}
		return word{shared.text + own.text, shared.latin + c.joiner + own.latin}
	case middle, called:
		first, second := w.pick(names), w.pick(names)
		for second == first {
			second = w.pick(names)
		}
		return word{text: first.text + " " + second.text}
	}
	return w.pick(names)
}

// surname is one of a name's surnames, as its culture writes it and in Latin letters.
type surname struct {
	name, prefix word
	origin       string
	primary      bool
}

// names are what somebody is called as their culture writes it, and its romanization
// where that is another script.
func names(c *culture, given word, surnames []surname) (model.PersonName, *model.PersonName) {
	build := func(written func(word) string, lang model.Language, display string) model.PersonName {
		n := model.PersonName{Given: written(given), Lang: lang, Display: display}
		if c.form == called {
			n.Call = n.Given[strings.LastIndexByte(n.Given, ' ')+1:]
		}
		for _, s := range surnames {
			n.Surname = append(n.Surname, model.Surname{Name: written(s.name), Prefix: written(s.prefix),
				Origin: s.origin, Primary: s.primary && len(surnames) > 1})
		}
		return n
	}
	native, roman := "", ""
	if c.surnameFirst {
		native, roman = "{surname}{given}", "{surname} {given}"
	}
	name := build(func(w word) string { return w.text }, c.lang, native)
	if c.latn == "" {
		return name, nil
	}
	romanized := build(word.roman, c.latn, roman)
	return name, &romanized
}

// patronymic is a Russian patronymic made of the father's given name.
func patronymic(father word, male bool) word {
	text, latin := father.text+"ович", father.latin+"ovich"
	if strings.HasSuffix(father.text, "й") {
		text, latin = strings.TrimSuffix(father.text, "й")+"евич", father.latin+"evich"
	}
	if !male {
		text, latin = strings.TrimSuffix(text, "ич")+"на", strings.TrimSuffix(latin, "ich")+"na"
	}
	return word{text, latin}
}

// patronym is an Icelandic patronymic made of the father's given name, in the genitive.
func patronym(father word, male bool) word {
	of := father.text + "s"
	switch {
	case strings.HasSuffix(father.text, "ús"):
		of = father.text + "ar"
	case strings.HasSuffix(father.text, "ur"):
		of = strings.TrimSuffix(father.text, "ur") + "s"
	case strings.HasSuffix(father.text, "i"):
		of = strings.TrimSuffix(father.text, "i") + "a"
	}
	if male {
		return word{text: of + "son"}
	}
	return word{text: of + "dóttir"}
}

// newPerson is somebody born at a town, named as its culture names a child of f, or as
// somebody whose parents are not here where f is nil.
func (w *world) newPerson(generation int, male bool, born time.Time, at int, f *family) *person {
	c := w.towns[at].culture
	p := &person{generation: generation, male: male, born: born, town: at, culture: c,
		given: w.given(c, male, born, f)}
	var father word
	var mother pair
	if c.men != nil {
		father = w.pick(c.men)
	}
	if c.surnames != nil {
		p.family = c.surnames[w.rand.IntN(len(c.surnames))]
		mother = c.surnames[w.rand.IntN(len(c.surnames))]
	}
	if f != nil {
		father, p.family = f.husband.given, f.husband.family
		mother = f.wife.family
		if mother.male.text != "" {
			mother.male = word{text: mother.male.roman()}
		}
	}

	var parts []surname
	switch c.form {
	case both:
		parts = []surname{{name: p.family.male, origin: "patrilineal", primary: true},
			{name: mother.male, origin: "matrilineal"}}
	case reversed:
		parts = []surname{{name: mother.male, origin: "matrilineal"},
			{name: p.family.male, origin: "patrilineal", primary: true}}
	case fatherly:
		parts = []surname{{name: patronym(father, male), origin: "patronymic"}}
	case russian:
		parts = []surname{{name: patronymic(father, male), origin: "patronymic"},
			{name: p.family.of(male), primary: true}}
	case nasab:
		prefix := word{"بنت", "bint"}
		if male {
			prefix = word{"بن", "ibn"}
		}
		parts = []surname{{name: father, prefix: prefix, origin: "patronymic"},
			{name: p.family.male, origin: "nisba", primary: true}}
	case marathi:
		parts = []surname{{name: father, origin: "patronymic"}, {name: p.family.male, primary: true}}
	default:
		parts = []surname{{name: p.family.of(male)}}
	}
	parts = slices.DeleteFunc(parts, func(s surname) bool { return s.name.text == "" })

	name, roman := names(c, p.given, parts)
	filed := name
	if roman != nil {
		filed = *roman
	}
	base, err := filed.Slug()
	if err != nil {
		panic(err)
	}
	sex := model.Female
	if male {
		sex = model.Male
	}
	p.record = &model.Person{
		Slug: w.namer.Assign(base, []string{fmt.Sprint(born.Year()), w.towns[at].name.roman()}, filed.String()),
		Name: name,
		Sex:  model.StatedSex{Value: sex},
		Events: []model.Event{{Type: "birth", Date: day(born),
			Place: model.Slug(w.towns[at].slug)}},
	}
	if roman != nil {
		p.record.AlsoKnownAs = []model.PersonName{*roman}
	}
	w.corpus.People[p.record.Slug] = p.record
	w.people = append(w.people, p)
	return p
}

// marry finds somebody a spouse from outside, near enough in age and from the same
// culture more often than not, and makes them a family, living where the husband does.
func (w *world) marry(p *person) *family {
	age := 18 + w.rand.IntN(10)
	if p.male {
		age += 2 + w.rand.IntN(5)
	}
	married := p.born.AddDate(age, 0, w.rand.IntN(200))
	other := w.rand.IntN(len(w.towns))
	if w.rand.Float64() < 0.8 {
		other = p.culture.first + w.rand.IntN(len(p.culture.towns))
	}
	gap := w.rand.IntN(9) - 1 // how much older the husband is
	if p.male {
		gap = -gap
	}
	spouse := w.newPerson(p.generation, !p.male, p.born.AddDate(gap, 0, w.rand.IntN(300)-150), other, nil)
	if born := spouse.born.AddDate(18, 0, 0); married.Before(born) {
		married = born.AddDate(0, 0, 30+w.rand.IntN(300))
	}
	f := &family{married: married, husband: p, wife: spouse, town: p.town}
	if !p.male {
		f.husband, f.wife, f.town = spouse, p, spouse.town
	}
	bride := f.wife.town
	f.firstFree = married.Year() + 1
	f.last = min(f.wife.born.Year()+44, lastYear)

	partners := model.PartnersOf(f.husband.record.Slug, f.wife.record.Slug)
	base, err := model.FamilySlug(partners)
	if err != nil {
		panic(err)
	}
	f.record = &model.Family{
		Slug:     w.namer.Assign(base, []string{fmt.Sprint(married.Year())}, string(base)),
		Partners: partners,
		Type:     model.Married,
		Events: []model.Event{{Type: "marriage", Date: day(married),
			Place: model.Slug(w.towns[bride].slug)}},
	}
	// a wife takes her husband's surname where he is English
	if f.husband.culture.form == middle {
		f.wife.record.AlsoKnownAs = append(f.wife.record.AlsoKnownAs, model.PersonName{
			Given: f.wife.given.roman(), Surname: model.SurnameOf(f.husband.family.male.text),
			Type: "married", From: day(married)})
	}
	w.corpus.Families[f.record.Slug] = f.record
	w.families = append(w.families, f)
	f.husband.families = append(f.husband.families, f)
	f.wife.families = append(f.wife.families, f)
	return f
}

// bear gives the families count children between them, a year or more apart and each
// while the mother could have one, a child going to a family chosen at random among
// those with a year left.
func (w *world) bear(families []*family, count int) []*person {
	var children []*person
	for range count {
		var open []*family
		for _, f := range families {
			if f.firstFree <= f.last {
				open = append(open, f)
			}
		}
		if len(open) == 0 {
			panic(fmt.Sprintf("no family has room for another child, with %d still to be born", count-len(children)))
		}
		f := open[w.rand.IntN(len(open))]
		year := f.firstFree + w.rand.IntN(min(2, f.last-f.firstFree+1))
		f.firstFree = year + 1 + w.rand.IntN(2)
		born := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, w.rand.IntN(365))
		if born.Before(f.married) {
			born = f.married.AddDate(0, 0, 280+w.rand.IntN(60))
		}
		child := w.newPerson(f.husband.generation+1, w.rand.IntN(2) == 0, born, f.town, f)
		f.children = append(f.children, child)
		f.record.Children = append(f.record.Children, model.Child{ID: child.record.Slug})
		children = append(children, child)
	}
	// the record lists them in the order they were born, not the order they were made
	for _, f := range families {
		slices.SortStableFunc(f.children, func(a, b *person) int { return a.born.Compare(b.born) })
		for i, c := range f.children {
			f.record.Children[i] = model.Child{ID: c.record.Slug}
		}
	}
	return children
}

func (w *world) dieYoung(p *person) {
	p.died = p.born.AddDate(w.rand.IntN(8), 0, 1+w.rand.IntN(300))
}

// die gives everybody not already dead a lifespan, outliving their last child by a
// year at least, and leaves alive whoever it would carry past the end of 2024.
func (w *world) die(p *person) {
	if p.died.IsZero() {
		died := p.born.AddDate(48+w.rand.IntN(45), 0, w.rand.IntN(365))
		for _, f := range p.families {
			if after := f.married.AddDate(1, 0, 0); died.Before(after) {
				died = after
			}
			for _, c := range f.children {
				if after := c.born.AddDate(1, 0, 0); died.Before(after) {
					died = after.AddDate(0, 0, w.rand.IntN(3000))
				}
			}
		}
		if died.Year() > lastYear {
			return
		}
		p.died = died
	}
	at := p.town
	if len(p.families) > 0 {
		at = p.families[0].town
	}
	p.record.Events = append(p.record.Events, model.Event{Type: "death", Date: day(p.died),
		Place: model.Slug(w.towns[at].slug)})
}

// cite puts every event into the register of the place it happened in, as an entry of
// its own, in the older register before the state registered anybody there and the
// state's after, in the words of the place.
func (w *world) cite() {
	byPlace := map[model.Slug]*town{}
	for _, t := range w.towns {
		byPlace[model.Slug(t.slug)] = t
	}
	for _, p := range w.people {
		w.citeEvent(byPlace, &p.record.Events[0], p.born, p, nil)
		if len(p.record.Events) > 1 {
			w.citeEvent(byPlace, &p.record.Events[1], p.died, p, nil)
		}
	}
	for _, f := range w.families {
		w.citeEvent(byPlace, &f.record.Events[0], f.married, nil, f)
	}
}

func (w *world) citeEvent(byPlace map[model.Slug]*town, event *model.Event, when time.Time, p *person, f *family) {
	t := byPlace[event.Place]
	c := t.culture
	source, kinds, written := t.register, c.civil.events, when.AddDate(0, 0, 1+w.rand.IntN(3))
	if when.Year() < c.since {
		source, kinds = t.parish, c.older.events
	}
	kind := kinds[map[string]int{"birth": 0, "marriage": 1, "death": 2}[event.Type]]
	key := fmt.Sprintf("%s %d %s", source.Slug, written.Year(), kind)
	t.entries[key]++
	n := t.entries[key]
	anchor := fmt.Sprintf("%d-%s%03d", written.Year(), kind[:1], n)

	var text string
	switch {
	case f != nil:
		text = fmt.Sprintf(c.married, f.husband.record.Name, f.wife.record.Name, c.dated(when))
	case event.Type == "birth":
		text = fmt.Sprintf(c.born[gender(p)], p.record.Name, c.dated(when))
	default:
		text = fmt.Sprintf(c.died[gender(p)], p.record.Name, c.dated(when))
	}
	source.Citations[anchor] = model.Citation{
		Event: kind,
		Page:  fmt.Sprintf("f. %d%s, no. %d", 1+n/4, []string{"r", "v"}[n/2%2], n),
		Date:  day(written),
		Text:  text + "\n",
	}
	event.Sources = []model.CitationRef{{Source: source.Slug, Anchor: anchor}}
}

func gender(p *person) int {
	if p.male {
		return 0
	}
	return 1
}

// roman is somebody's name in Latin letters, as a research note in English writes it.
func roman(p *person) model.PersonName {
	if p.culture.latn != "" {
		return p.record.AlsoKnownAs[0]
	}
	return p.record.Name
}

// notes are a research log for one family in fifty, linking to it and leaving the
// question it asks for somebody to tick.
func (w *world) notes() []model.Slug {
	var made []model.Slug
	for i := 0; i < len(w.families); i += 50 {
		f := w.families[i]
		husband, wife := roman(f.husband), roman(f.wife)
		at := w.towns[f.town].name.roman()
		title := fmt.Sprintf("The %s of %s, %d", husband.PrimarySurname().Name, at, f.married.Year())
		note := &model.Note{Slug: w.slug(title), Title: title, Body: fmt.Sprintf(
			"*Research:* the marriage of [%s and %s](../../families/%s/README.md) is the\n"+
				"earliest record of this branch in %s.\n\n- [ ] Look for the banns in the parish of the bride.\n",
			husband.Given, wife.Given, f.record.Slug, at)}
		w.corpus.Notes[note.Slug] = note
		made = append(made, note.Slug)
	}
	return made
}

// hypotheses are one open, proposing a child the registers do not name, and one
// rejected, moving a birth, both pinned to the commit the records were adopted in, the
// open one first.
func (w *world) hypotheses(base string) []any {
	var made []any
	for i, f := range []*family{w.families[3], w.families[len(w.families)/2]} {
		cited := f.record.Events[0].Sources[0]
		if i == 0 {
			child := model.Slug(string(f.husband.record.Slug) + "-unrecorded")
			born := f.married.Year() + 2
			made = append(made, &model.Hypothesis{
				Slug: w.slug("An unrecorded child of " + string(f.record.Slug)), Status: model.Open,
				Base: base, Confidence: model.Possible, Supports: []model.CitationRef{cited},
				Edits: []model.Edit{
					model.Create{Entity: entity(model.KindPerson, child), Value: map[string]any{
						"name": map[string]any{"surname": f.husband.record.Name.PrimarySurname().Name},
						"events": []any{map[string]any{"type": "birth",
							"date": fmt.Sprintf("ABT %d", born), "place": w.towns[f.town].slug}},
					}},
					model.Add{Entity: entity(model.KindFamily, f.record.Slug), Path: "children",
						Value: string(child)},
				},
				Body: "# Was there a child the registers do not name?\n\n" +
					"The gap after the marriage is longer than any other between the children.\n",
			})
			continue
		}
		child := f.children[0]
		made = append(made, &model.Hypothesis{
			Slug: w.slug("Was " + string(child.record.Slug) + " born a year earlier"), Status: model.Rejected,
			Base: base, Confidence: model.Doubtful, Contradicts: []model.CitationRef{child.record.Events[0].Sources[0]},
			Edits: []model.Edit{model.Set{Entity: entity(model.KindPerson, child.record.Slug),
				Path: "events[0].date", Value: fmt.Sprint(child.born.Year() - 1)}},
			Body: "# Born a year earlier?\n\nThe register is contemporary with the birth, so no.\n",
		})
	}
	return made
}
