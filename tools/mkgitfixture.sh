#!/usr/bin/env bash
set -euo pipefail
R="$(cd "$(dirname "$0")/.." && pwd)"
T="$R/test"
G="$R/testgit"

# The same corpus as test/, built from nothing by running origins, into a repository
# whose history is the one a researcher would have made getting there. The facts go in
# through the commands. What no command reaches arrives the way it would by hand —
# written into the files, then adopted — and is read out of test/, so the prose, the
# photographs, the manifest and the hypotheses are stated once, in mkfixture.sh.
"$R/tools/mkfixture.sh" > /dev/null

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
ORIGINS="$tmp/origins$(go env GOEXE)"
# Every commit names the version of origins that made it, so the version is fixed too,
# or a checkout standing on a release tag would make other shas.
(cd "$R" && go build -ldflags "-X github.com/origins-project/origins/internal/api.release=dev" -o "$ORIGINS" ./cmd/origins)

# Nothing of the machine's git configuration reaches the fixture, and the repository
# this script lives in is not one git may find from inside testgit/. A fixed identity
# and a clock that advances one hour per commit make every run the same history, down
# to the shas.
cat > "$tmp/git-config" <<'XX'
[user]
	name = Rosa Bianchi
	email = rosa@example.invalid
[core]
	autocrlf = false
[init]
	defaultBranch = main
XX
export GIT_CONFIG_GLOBAL="$tmp/git-config" GIT_CONFIG_SYSTEM="$tmp/no-system-config"
export GIT_CEILING_DIRECTORIES="$R"
clock=1551398400

o() {
  clock=$((clock + 3600))
  GIT_AUTHOR_DATE="@$clock +0000" GIT_COMMITTER_DATE="@$clock +0000" \
    "$ORIGINS" "$@" > /dev/null
}

# A citation's text is folded prose, and the newline it ends with is part of it.
cite() { o set "$1" "citations.$2.text" "$3"$'\n'; }

# Everything below an entity's frontmatter.
prose() { awk 'body { print } /^---$/ && ++fences == 2 { body = 1; getline }' "$1"; }

rm -rf "$G"
o init --name "Marchena Sólveigarson family" "$G"
cd "$G"

# ---------------------------------------------------------------------- places

o create place/granducato-di-toscana "name=Granducato di Toscana" "also-known-as=[Grand Duchy of Tuscany]" type=state until=1860
o create place/toscana name=Toscana "also-known-as=[Tuscany]" type=region from=1861
o create place/firenze name=Firenze "also-known-as=[{name: Florence, lang: en}, {name: Florenz, lang: de, note: As the German guidebooks of the period have it.}]" type=city "within=[{place: granducato-di-toscana, until: 1860}, {place: toscana, from: 1861}]" "coordinates=43.7696, 11.2558"
o set place/firenze ids.geonames 3176959
o set place/firenze ids.wikidata Q2044
o create place/pian-del-mugnone "name=Pian del Mugnone" type=hamlet "within=[{place: granducato-di-toscana, until: 1860}, {place: toscana, from: 1861}]" "coordinates=(43°48'N 11°17'E)" until=1897
o add place/pian-del-mugnone addresses "{parish: San Michele a Pian del Mugnone, city: Fiesole, note: 'Where the parish registers put it, and the comune that absorbed it.'}"
o create place/stade name=Stade type=town "coordinates=53.5997, 9.4767"
o create place/hamburg name=Hamburg type=city "coordinates=53.5511, 9.9937"
o create place/london name=London type=city "coordinates=51.5074, -0.1278"
o create place/lyon name=Lyon type=city "coordinates=45.764, 4.8357"
o create place/trieste name=Trieste "also-known-as=[{name: Triest, lang: de}, {name: Τεργέστη, lang: el}]" type=city "coordinates=45.6495, 13.7768"
o create place/nizhny-tagil "name={name: Нижний Тагил, lang: ru}" "also-known-as=[{name: Nizhny Tagil, lang: ru-Latn, type: romanized}]" type=town "coordinates=57.9194, 59.965"
o create place/thessaloniki "name={name: Θεσσαλονίκη, lang: el, from: 1912}" "also-known-as=[{name: Thessaloniki, lang: el-Latn, type: romanized}, {name: Selanik, lang: ota, until: 1912}, {name: Salonicco, lang: it}]" type=city "coordinates=40.6401, 22.9444"
o create place/alexandria "name={name: الإسكندرية, lang: ar}" "also-known-as=[{name: al-Iskandariyya, lang: ar-Latn, type: romanized}, {name: Alexandria, lang: en}]" type=city "coordinates=31.2001, 29.9187"
o create place/lisboa name=Lisboa "also-known-as=[{name: Lisbon, lang: en}]" type=city "coordinates=38.7223, -9.1393"
o create place/amsterdam name=Amsterdam type=city "coordinates=52.3676, 4.9041"
o create place/kinkerstraat-41 "name=Kinkerstraat 41" type=address "within=[amsterdam]"
o set place/kinkerstraat-41 private true
o create place/buenos-aires "name=Buenos Aires" type=city "coordinates=-34.6037, -58.3816"
o create place/paris name=Paris type=city "coordinates=48.8566, 2.3522"
o create place/wenzhou "name={name: 溫州, lang: zh-Hant}" "also-known-as=[{name: Wēnzhōu, lang: zh-Latn-pinyin, type: romanized}, {name: Wenchow, lang: en, note: \"As the Treaty Port's customs returns spell it.\"}]" type=city "coordinates=27.9938, 120.6994"
o create place/madrid name=Madrid type=city "coordinates=40.4168, -3.7038"
o create place/kumasi name=Kumasi type=city "coordinates=6.6885, -1.6244"
o create place/edinburgh name=Edinburgh type=city "coordinates=55.9533, -3.1883"
o create place/reykjavik name=Reykjavík type=city "coordinates=64.1466, -21.9426"
o create place/tehran "name={name: تهران, lang: fa}" "also-known-as=[{name: Tehran, lang: fa-Latn, type: romanized}]" type=city "coordinates=35.6892, 51.389"
o create place/kasegaon "name={name: कासेगाव, lang: mr}" "also-known-as=[{name: Kasegaon, lang: mr-Latn, type: romanized}]" type=village "coordinates=17.1167, 74.2"
o create place/busan "name={name: 부산, lang: ko}" "also-known-as=[{name: Busan, lang: ko-Latn, type: romanized}, {name: Pusan, lang: ko-Latn, note: 'The McCune–Reischauer spelling, before 2000.'}]" type=city "coordinates=35.1796, 129.0756"
o create place/iwate "name={name: 岩手県, lang: ja}" "also-known-as=[{name: Iwate, lang: ja-Latn-hepburn, type: romanized}]" type=prefecture
o create place/tono "name={name: 遠野市, lang: ja}" "also-known-as=[{name: Tōno, lang: ja-Latn-hepburn, type: romanized}]" type=city "within=[iwate]" "coordinates=39.3275, 141.5333"

# ---------------------------------------------------------------- repositories

o create repository/archivio-diocesano-di-firenze "name=Archivio Diocesano di Firenze" type=archive "addresses=[{lines: \"Piazza San Giovanni 3\n50129 Firenze\n\", street: Piazza San Giovanni 3, city: Firenze, postal: 50129, country: Italia}]" "links=[https://www.example.org/archivio-diocesano]"
o add repository/archivio-diocesano-di-firenze "addresses[0].links" "tel:055 213456"
o add repository/archivio-diocesano-di-firenze "addresses[0].links" "{url: tel:055 213457, type: fax}"
o add repository/archivio-diocesano-di-firenze links mailto:archivio@example.org
o add repository/archivio-diocesano-di-firenze addresses "{lines: \"Via dei Servi 15\n50122 Firenze\n\", date: TO 1998}"
o create repository/archivio-di-stato-di-firenze "name=Archivio di Stato di Firenze" type=archive "addresses=[\"Viale Giovine Italia 6\n50122 Firenze\n\"]" "links=[https://www.example.org/archivio-di-stato]"
o create repository/papers-of-ingrid-de-vries "name=Ingrid de Vries's papers" type=private-collection "addresses=[\"Lindengracht 112\n1015 KM Amsterdam\n\"]"
o set repository/papers-of-ingrid-de-vries private true
o create repository/the-national-archives "name=The National Archives" type=archive "addresses=[{city: Kew, country: United Kingdom}]"
o create repository/staatsarchiv-hamburg "name=Staatsarchiv Hamburg" type=archive "addresses=[{city: Hamburg, country: Deutschland}]"
o create repository/stadsarchief-amsterdam "name=Stadsarchief Amsterdam" type=archive "addresses=[{city: Amsterdam, country: Nederland}]"
o create repository/arquivo-nacional-torre-do-tombo "name=Arquivo Nacional Torre do Tombo" type=archive "addresses=[{city: Lisboa, country: Portugal}]"
o create repository/cahjp "name=Central Archives for the History of the Jewish People" type=archive "addresses=[{city: Jerusalem}]"
o create repository/dar-al-wathaiq "name=دار الوثائق القومية" type=archive "addresses=[{city: القاهرة, country: مصر}]"
o create repository/gaso "name=Государственный архив Свердловской области" type=archive "addresses=[{city: Екатеринбург, country: Россия}]"
o create repository/krigsarkivet name=Krigsarkivet type=archive "addresses=[{city: Stockholm, country: Sverige}]"

# --------------------------------------------------------------------- sources

o create source/parish-san-lorenzo-baptisms "title=Basilica di San Lorenzo, Firenze — Baptismal Register 1770-1790" type=parish-register "repositories=[{repository: archivio-diocesano-di-firenze, call-number: ADF/SL/B/12}]"
o set source/parish-san-lorenzo-baptisms short-title "S. Lorenzo, Battesimi 12"
o set source/parish-san-lorenzo-baptisms covers "[{events: [baptism], date: FROM 1770 TO 1790, place: firenze}]"
o add source/parish-san-lorenzo-baptisms repositories "{repository: archivio-diocesano-di-firenze, call-number: 'S. Lorenzo, Battesimi XII', note: 'The shelf mark before the archive was reordered, still on the spine.'}"
o add source/parish-san-lorenzo-baptisms repositories "{repository: archivio-di-stato-di-firenze, call-number: Film 1903581, medium: microfilm}"
cite source/parish-san-lorenzo-baptisms 1779-042 "Caterina, figlia di Bartolomeo Rossi e Angela Sarti, del popolo di San Michele a Pian del Mugnone, nata il 3 aprile 1779 e battezzata il 5 detto."
o set source/parish-san-lorenzo-baptisms citations.1779-042.event baptism
o set source/parish-san-lorenzo-baptisms citations.1779-042.page "c. 88r, n. 112"
o set source/parish-san-lorenzo-baptisms citations.1779-042.date "5 APR 1779"
o set source/parish-san-lorenzo-baptisms citations.1779-042.note "*Translation, en:* Caterina, daughter of Bartolomeo Rossi and Angela Sarti, of the parish of San Michele at Pian del Mugnone, born on 3 April 1779 and baptised on the 5th."$'\n'
o add --source parish-san-lorenzo-baptisms#1779-042 place/firenze "within[0].sources" parish-san-lorenzo-baptisms#1779-042
o set place/firenze "within[0].note" "The register heads each volume Granducato di Toscana."

o create source/hamburg-st-michaelis-baptisms "title=Hauptkirche St. Michaelis, Hamburg — Taufregister 1805-1815" type=parish-register "repositories=[staatsarchiv-hamburg]"
cite source/hamburg-st-michaelis-baptisms 1810-003 "Joseph, Sohn des Kaufmanns Georg Lang und der Katharina geb. Rossi, geboren den 2. Januar 1810 und am selben Tage nothgetauft."
cite source/hamburg-st-michaelis-baptisms 1812-017 "Anna Maria, Tochter des Kaufmanns Georg Lang."

o create source/french-church-london-baptisms "title=French Protestant Church of London, Threadneedle Street — Baptisms 1830-1849" type=parish-register "repositories=[{repository: the-national-archives, call-number: RG 4}]"
cite source/french-church-london-baptisms 1834-042 "Jean Baptiste, fils de Louis Verdier et d'Anna Maria Lang, né le 12 mars 1834 et baptisé le 14. Parrain Georg Lang, aïeul maternel; marraine Mary Ann Collins."
cite source/french-church-london-baptisms 1842-007 "Louisa, fille des mêmes, baptisée le 19 mars 1842."

o create source/nizhny-tagil-revision-1834 "title=Ревизская сказка, Нижнетагильский завод, 1834" type=census "repositories=[gaso]"
cite source/nizhny-tagil-revision-1834 ershov-ivan "Иван Петров сын Ершов, 16 лет."

o create source/trieste-greek-baptisms "title=San Nicolò dei Greci, Trieste — Baptismal Register 1835-1845" type=parish-register
cite source/trieste-greek-baptisms 1840-018 "Μαρία, θυγάτηρ του Γεωργίου Παπαδόπουλου, εβαπτίσθη τον Φεβρουάριον 1840."

o create source/london-marriages-1858 "title=General Register Office, Marriage Certificate, Whitechapel, 1858" type=civil-registration "links=[https://www.example.org/gro/marriages/1858/whitechapel]" "repositories=[the-national-archives]"
cite source/london-marriages-1858 p112 "21 April 1858 — John Verdia, 24, bachelor, silk weaver, father Louis Verdier; Maria Papadopulo, 18, spinster, father George Papadopulo. Witnesses Anna Maria Ershov, Πέτρος Παπαδόπουλος."
o set source/london-marriages-1858 citations.p112.links "[https://www.example.org/gro/marriages/1858/whitechapel/112]"

o create source/cimitero-san-lorenzo-gravestones "title=Cimitero di San Lorenzo — surviving gravestones, photographed 2018" type=gravestone "repositories=[papers-of-ingrid-de-vries]"
cite source/cimitero-san-lorenzo-gravestones sarti-angela "ANGELA SARTI / MORI IL 3 MARZO 1... / DI ANNI [illegible]"

o create source/census-1861-england "title=Census of England and Wales, 1861 — Christ Church, Spitalfields" type=census "repositories=[the-national-archives]"
cite source/census-1861-england f088 "14 Fournier Street. J. B. Verdier, head, married, 27, silk weaver, born London; Maria, wife, 20, born Trieste, Austria; Anna, daughter, 7 months, born Spitalfields."
cite source/census-1861-england f214 "Louisa Verdier, daughter, 17, dressmaker, born London."

o create source/salonica-community-births "title=Jewish Community of Salonica — Register of Births 5655-5665" type=parish-register "repositories=[cahjp]"
cite source/salonica-community-births 5661-073 'רחל בת אברהם מודיאנו, י"ז אייר תרס"א.'

o create source/alexandria-sharia-court-1302 "title=محكمة الإسكندرية الشرعية — سجل عقود الزواج، ١٣٠٢" type=court-record "repositories=[dar-al-wathaiq]"
cite source/alexandria-sharia-court-1302 marriage-al-halabi "محمود بن علي بن حسن الحلبي و آنا بنت جان بابتيست فردييه."

o create source/lisboa-santa-isabel-baptisms "title=Paróquia de Santa Isabel, Lisboa — Registo de baptismos 1875-1880" type=parish-register "repositories=[arquivo-nacional-torre-do-tombo]"
cite source/lisboa-santa-isabel-baptisms 1878-091 "Henrique, filho legítimo de Dom Manuel de Cerqueira Galvão e de Dona Maria do Carmo de Cerqueira."

o create source/amsterdam-population-register "title=Bevolkingsregister Amsterdam, 1893-1939" type=census "links=[https://www.example.org/stadsarchief/bevolkingsregister]" "repositories=[stadsarchief-amsterdam]"
cite source/amsterdam-population-register de-wit-1895 "Wit, Geertruida Maria de, geb. Amsterdam 1895; vertrokken naar Lissabon, gouvernante."

o create source/claudia-interview-2019 "title=Recorded interview with Cláudia Chen Galvão, March 2019" type=interview "author=Ingrid de Vries" "repositories=[papers-of-ingrid-de-vries]"
o set source/claudia-interview-2019 private true
cite source/claudia-interview-2019 adoption-helena '"My aunt always knew she was adopted. It was 1923, she was three, and the papers went through in Lisbon — her birth mother was a Dutch governess."'
cite source/claudia-interview-2019 death-michael '"The letter came in November. My grandmother always said November, because it was just before the feast. But what year, I could not tell you."'

o create source/ofori-family-account "title=The Ofori family's account, as told at Kumasi, 2012" type=interview "author=Ingrid de Vries"
cite source/ofori-family-account kwame '"He was born on the Saturday after the New Year, in 1950, and so he is Kwame."'

o create source/tono-koseki "title=遠野市 戸籍謄本, 佐藤家" type=civil-registration
cite source/tono-koseki sato-yumiko "昭和三十年六月二十一日 岩手県遠野市で出生。"

o create source/upplands-regemente-rolls "title=Upplands regemente, generalmönsterrullor 1700-1712" type=military "links=[https://www.example.org/krigsarkivet/rullor/uppland]" "repositories=[krigsarkivet]"
cite source/upplands-regemente-rolls 1704-rapp "Per Rapp, från Väsby, 22 år."

o create source/amsterdam-births-2018 "title=Gemeente Amsterdam, Akte van geboorte, 2018" type=civil-registration
cite source/amsterdam-births-2018 2018-021733 "Sasha Marchena Sólveigarson, geboren 3 april 2018 te Amsterdam. Geslacht: X. Ouder: Marco Marchena Chen, geboren te Madrid. Ouder: Stefán Sólveigarson, geboren te Reykjavík."

# ------------------------------------------------------------ living memory

# Research starts from the people somebody remembers, so the grandson is entered before
# anybody knows there was an older Ivan and takes the plain name. He gives it up once
# the grandfather turns up, and the family is renamed with him.
o create person/ershov-ivan "name={given: Iván, nickname: Vanya, surname: Ershov}" sex=M
o add person/ershov-ivan events "{type: birth, date: 1885, place: buenos-aires}"
o add person/ershov-ivan events "{type: title, value: Doctor}"
o add person/ershov-ivan events "{type: death, date: 1960, place: buenos-aires}"

o create person/modiano-rachel "name={given: Rachel, surname: Modiano}" sex=F
o add --source salonica-community-births#5661-073 person/modiano-rachel events "{type: birth, date: HEBREW 17 IYR 5661, place: thessaloniki, sources: [salonica-community-births#5661-073]}"
o add person/modiano-rachel events "{type: registration, date: ISLAMIC 3 SAFAR 1319, place: thessaloniki}"
o add person/modiano-rachel events "{type: Μετανάστευση, date: JULIAN 20 AUG 1917, place: thessaloniki}"
o add person/modiano-rachel events "{type: death, date: 1969, place: buenos-aires}"

o create family/ershov-ivan--modiano-rachel "partners=[ershov-ivan, modiano-rachel]" type=married
o add family/ershov-ivan--modiano-rachel events "{type: marriage, date: 14 OCT 1934, place: buenos-aires}"
o add person/modiano-rachel also-known-as "{given: Raquel, surname: Ershov, type: married, from: 1934, note: 'Taken on marrying [Iván](../../people/ershov-ivan/README.md), and kept as a widow.'}"
o add person/modiano-rachel also-known-as "{given: רחל, surname: מודיאנו, lang: he}"

# ---------------------------------------------------------------- generation -4

o create person/andersson-per "name={given: Per, surname: [{name: Andersson, origin: patronymic}]}" sex=M
o add person/andersson-per events "{type: birth, date: EST BET 1670 AND 1685}"
o add person/andersson-per events "{type: residence, sort-date: 1700, note: 'On the farm at Väsby, which the rolls give as where he came from'}"
o add --source upplands-regemente-rolls#1704-rapp person/andersson-per events "{type: military-service, date: FROM SWEDISH 1704 TO SWEDISH 30 FEB 1712, note: 'Upplands regemente, in the Great Northern War', sources: [upplands-regemente-rolls#1704-rapp]}"
o add --source upplands-regemente-rolls#1704-rapp person/andersson-per also-known-as "{given: Per, surname: Rapp, type: soldier, from: 1704, sources: [upplands-regemente-rolls#1704-rapp]}"

# ---------------------------------------------------------------- generation -3

o create person/persdotter-karin "name={given: Karin, surname: [{name: Persdotter, origin: patronymic}]}" sex=F
o add person/persdotter-karin events "{type: birth, date: ABT 1735, place: stade}"
o add person/persdotter-karin events "{type: death, date: 1791, place: stade}"

o create person/rossi-bartolomeo "name={given: Bartolomeo, surname: Rossi}" sex=M
o add person/rossi-bartolomeo events "{type: birth, date: 12 FEB 1750/51, place: pian-del-mugnone}"
o add person/rossi-bartolomeo events "{type: will, date: 20 APR 1799 NY 25 MAR}"
o add person/rossi-bartolomeo events "{type: death, date: 1799, place: pian-del-mugnone}"

o create person/sarti-angela "name={given: Angela, surname: Sarti}" sex=F
o add person/sarti-angela events "{type: birth, date: EST 1755}"
o add --source 'cimitero-san-lorenzo-gravestones#sarti-angela O--' person/sarti-angela events "{type: death, date: 3 MAR, place: pian-del-mugnone, sources: [cimitero-san-lorenzo-gravestones#sarti-angela O--]}"

# ---------------------------------------------------------------- generation -2

o create person/lang-georg "name={given: Johann Georg, call: Georg, surname: Lang}" sex=M
o add person/lang-georg events "{type: birth, date: CAL 1772, place: stade}"
o add person/lang-georg events "{type: death, date: 19 OCT 1841, place: hamburg}"

o create person/rossi-caterina "name={given: Caterina, surname: Rossi}" "also-known-as=[{given: Katharina, surname: Lang, type: married, from: 1802}]" sex=F
o add --source parish-san-lorenzo-baptisms#1779-042 person/rossi-caterina events "{type: birth, date: 3 APR 1779, place: pian-del-mugnone, sources: [parish-san-lorenzo-baptisms#1779-042]}"
o add --source 'parish-san-lorenzo-baptisms#1779-042 OPD' person/rossi-caterina events "{type: baptism, date: 5 APR 1779, place: firenze, sources: [parish-san-lorenzo-baptisms#1779-042 OPD]}"
o set source/parish-san-lorenzo-baptisms citations.1779-042.with "[{person: rossi-caterina, role: child}, {person: rossi-bartolomeo, role: father}]"
o add person/rossi-caterina events "{type: death, date: 1840, place: hamburg}"
o add person/rossi-caterina events "{type: burial, date: '22 ? 1840', place: hamburg}"

# ---------------------------------------------------------------- generation -1

o create person/lang-joseph "name={given: Joseph, surname: Lang}" sex=M
o add --source hamburg-st-michaelis-baptisms#1810-003 person/lang-joseph events "{type: birth, date: 2 JAN 1810, place: hamburg, sources: [hamburg-st-michaelis-baptisms#1810-003]}"
o add person/lang-joseph events "{type: death, date: 5 JAN 1810, place: hamburg}"

o create person/lang-anna-maria "name={given: Anna Maria, surname: Lang}" "also-known-as=[{given: Anna Maria, surname: Verdier, type: married, from: 1832}]" sex=F
o add --source hamburg-st-michaelis-baptisms#1812-017 person/lang-anna-maria events "{type: birth, date: ABT 1812, place: hamburg, sources: [hamburg-st-michaelis-baptisms#1812-017]}"
o add person/lang-anna-maria events "{type: occupation, value: dressmaker, date: FROM 1830 TO 1860, place: london, note: 'In Spitalfields, for a court dressmaker'}"
o add person/lang-anna-maria events "{type: death, date: 14 JUL 1878, place: london, age: {value: 66, note: 'Given by Michael, who registered the death.'}}"

o create person/verdier-louis "name={given: Louis, surname: Verdier}" sex=M
o add person/verdier-louis events "{type: birth, date: FRENCH_R 12 FLOR 7, place: lyon}"
o add person/verdier-louis events "{type: occupation, value: silk weaver, place: london}"
o add person/verdier-louis events "{type: death, date: BET 1845 AND 1851, place: london}"

o rename person/ershov-ivan ershov-ivan-1885

o create person/ershov-ivan "name={given: Иван, surname: [{name: Петрович, origin: patronymic}, {name: Ершов, primary: true}], lang: ru}" "also-known-as=[{given: Ivan, surname: [{name: Petrovich, origin: patronymic}, {name: Ershov, primary: true}], lang: ru-Latn}]" sex=M
o add person/ershov-ivan events "{type: birth, date: JULIAN 3 MAR 1818 (3/15 марта 1818), place: nizhny-tagil}"
o add --source nizhny-tagil-revision-1834#ershov-ivan person/ershov-ivan events "{type: census, date: 1834, place: nizhny-tagil, sources: [nizhny-tagil-revision-1834#ershov-ivan]}"
o add person/ershov-ivan events "{type: residence, date: FROM 1851, place: london}"
o add person/ershov-ivan events "{type: death, date: 1870, place: london}"
o add person/lang-anna-maria also-known-as "{given: Anna Maria, surname: Ershov, type: married, from: 1853}"

# ---------------------------------------------------------------- generation 0

o create --source 'french-church-london-baptisms#1834-042 OPD' person/verdier-jean-baptiste "name={text: 'Jean Baptiste Verdier, fils de Louis', given: Jean Baptiste, call: Jean, surname: Verdier, sources: [french-church-london-baptisms#1834-042 OPD]}" sex=M
o add --source 'french-church-london-baptisms#1834-042 OPD3' person/verdier-jean-baptiste events "{type: birth, date: 12 MAR 1834, place: london, sources: [french-church-london-baptisms#1834-042 OPD3]}"
o add --source french-church-london-baptisms#1834-042 person/verdier-jean-baptiste events "{type: baptism, date: 14 MAR 1834, place: london, sources: [french-church-london-baptisms#1834-042]}"
o add --source 'french-church-london-baptisms#1834-042 OPD' person/verdier-jean-baptiste "events[1].with" "{person: lang-georg, role: godfather, sources: [french-church-london-baptisms#1834-042 OPD]}"
o add --source french-church-london-baptisms#1834-042 person/verdier-jean-baptiste "events[1].with" "{name: Mary Ann Collins, role: godmother, note: No other entry in the register names her.}"
o add --source 'census-1861-england#f088 OPD' person/verdier-jean-baptiste events "{type: residence, date: 1861, place: london, address: '14 Fournier Street, Spitalfields', sources: [census-1861-england#f088 OPD], ids: {gramps: E0007}}"
o add --source census-1861-england#f088 person/verdier-jean-baptiste also-known-as "J. B. Verdier"
o add source/census-1861-england citations.f088.ids.gramps "{id: C0012, note: Its ID in the Gramps tree the census was first entered in.}"
o add --source 'london-marriages-1858#p112 ---1' person/verdier-jean-baptiste also-known-as "{given: John, surname: Verdia, type: aka, sources: [london-marriages-1858#p112 ---1]}"
o add person/verdier-jean-baptiste events "{type: death, date: 3 NOV 1901, place: london}"
o set person/verdier-jean-baptiste ids.familysearch XXXX-YYY
o add person/verdier-jean-baptiste ids.familysearch "{id: XXXX-XXX, note: \"A duplicate of him somebody else entered, which FamilySearch merged into this one in 2016; links to the old one still lead here.\n\"}"

o create person/verdier-louisa "name={given: Louisa, surname: Verdier}" sex=F
o add --source french-church-london-baptisms#1842-007 person/verdier-louisa events "{type: birth, date: 1842, place: london, sources: [french-church-london-baptisms#1842-007]}"
o add --source census-1861-england#f214 person/verdier-louisa "events[0].sources" census-1861-england#f214

o create person/ershov-michael "name={given: Michael, surname: Ershov}" sex=M
o add person/ershov-michael events "{type: birth, date: 1856, place: london}"
o add person/ershov-michael events "{type: emigration, date: AFT 1880, note: 'To Buenos Aires, where he married in 1884; no departure record found.'}"
o add --source claudia-interview-2019#death-michael person/ershov-michael events "{type: death, date: NOV (just before the feast), sources: [claudia-interview-2019#death-michael]}"
o add person/ershov-michael not "{type: naturalisation, date: FROM 1880 TO 1910, note: Searched the Argentine naturalisation index for those years.}"
o add person/ershov-michael tags Emigrants
o add person/ershov-michael "events[1].tags" "To do"

o create person/papadopoulou-maria "name={given: Μαρία, surname: Παπαδοπούλου, lang: el}" "also-known-as=[{given: Maria, surname: Papadopoulou, lang: el-Latn}, Maria Papadopulo, {given: Maria, surname: Verdier, type: married, from: 1858}]" sex=F
o add --source trieste-greek-baptisms#1840-018 person/papadopoulou-maria events "{type: birth, date: FEB 1840, place: trieste, sources: [trieste-greek-baptisms#1840-018]}"
o add person/papadopoulou-maria events "{type: death, date: AFT 1901}"
o add person/verdier-jean-baptiste "events[3].with" "{person: papadopoulou-maria, role: informant}"
o add person/ershov-ivan-1885 same-as "{id: ershov-ivan, note: 'A tree on FamilySearch has the two as one man, living from 1818 to 1960.'}"

o create person/ershov-lawrence "name={given: Lawrence Henry, surname: Ershov}" sex=M
o add person/ershov-lawrence events "{type: birth, date: 17 MAR 1866, place: london}"

o create person/benitez-rosa "name={given: Rosa, surname: Benítez}" sex=F
o add person/benitez-rosa events "{type: birth, date: 1862, place: buenos-aires}"
o add person/benitez-rosa events "{type: death, date: 1931, place: buenos-aires}"

# ---------------------------------------------------------------- generation 1

o create person/verdier-anna "name={given: Anna, surname: Verdier}" sex=F
o add --source 'census-1861-england#f088 OSI' person/verdier-anna events "{type: birth, date: 9 SEP 1860, place: london, sources: [census-1861-england#f088 OSI]}"

o create person/verdier-pauline "name={given: Pauline, surname: Verdier}" sex=F
o add person/verdier-pauline events "{type: birth, date: 4 JUN 1863, place: london, attributes: [{type: fathers-age, value: 29}]}"
o add person/verdier-pauline events "{type: death, date: 1941, place: lisboa}"

o create person/verdier-pierre "name={given: Pierre, surname: Verdier}" sex=M
o add person/verdier-pierre events "{type: birth, date: 4 JUN 1863, place: london}"
o add person/verdier-pierre events "{type: death, date: 1863, place: london}"

o create person/al-halabi-mahmud "name={prefix: أبو يوسف, given: محمود, surname: [{name: علي, prefix: بن, origin: patronymic}, {name: حسن, prefix: بن, origin: patronymic}, {name: الحلبي, origin: nisba, primary: true}], lang: ar}" "also-known-as=[{prefix: Abū Yūsuf, given: Maḥmūd, surname: [{name: ʿAlī, prefix: ibn, origin: patronymic}, {name: Ḥasan, prefix: ibn, origin: patronymic}, {name: al-Ḥalabī, origin: nisba, primary: true}], lang: ar-Latn}]" sex=M
o add person/al-halabi-mahmud events "{type: birth, date: ABT 1850, note: 'At Aleppo, which his nisba names'}"
o add person/al-halabi-mahmud events "{type: occupation, value: cotton merchant, place: alexandria}"
o add person/al-halabi-mahmud events "{type: death, date: 1915, place: alexandria}"


o create person/galvao-henrique "name={given: Henrique, surname: [{name: Cerqueira, prefix: de, origin: matrilineal}, {name: Galvão, origin: patrilineal, primary: true}]}" sex=M
o add --source lisboa-santa-isabel-baptisms#1878-091 person/galvao-henrique events "{type: birth, date: 1878, place: lisboa, sources: [lisboa-santa-isabel-baptisms#1878-091]}"
o add person/galvao-henrique events "{type: title, value: Dom, date: FROM 1878}"
o add person/galvao-henrique events "{type: death, date: 1949, place: lisboa}"

o create person/chen-yuying "name={given: 玉英, surname: 陳, lang: zh-Hant, display: '{surname}{given}'}" "also-known-as=[{given: Yùyīng, surname: Chén, lang: zh-Latn-pinyin, display: '{surname} {given}'}, {given: Teresa, type: religious}]" sex=F
o add person/chen-yuying events "{type: birth, date: 1865, place: wenzhou}"
o add person/chen-yuying events "{type: death, date: 1932, place: wenzhou}"
o add person/chen-yuying not "{type: marriage, note: 'Unmarried on her death record, at sixty-seven.'}"

# ---------------------------------------------------------------- generation 2

o create person/al-halabi-zaynab "name={given: زينب, surname: [{name: محمود, prefix: بنت, origin: patronymic}, {name: الحلبي, origin: nisba, primary: true}], lang: ar}" "also-known-as=[{given: Zaynab, surname: [{name: Maḥmūd, prefix: bint, origin: patronymic}, {name: al-Ḥalabī, origin: nisba, primary: true}], lang: ar-Latn}]" sex=F
o add person/al-halabi-zaynab events "{type: birth, date: ISLAMIC 3 SHAWW 1304, place: alexandria}"
o add person/al-halabi-zaynab events "{type: death, date: 1960, place: alexandria}"

o create person/al-halabi-1889 "name={surname: الحلبي, lang: ar}" "also-known-as=[{surname: al-Ḥalabī, lang: ar-Latn}]"
o add person/al-halabi-1889 events "{type: birth, date: 12 MAY 1889, place: alexandria, attributes: [{type: classification, value: Stillborn}]}"
o add person/al-halabi-1889 events "{type: death, date: 12 MAY 1889, place: alexandria}"

o create person/galvao-joao "name={given: João, surname: [{name: Verdier, origin: matrilineal}, {name: Galvão, origin: patrilineal, primary: true}]}" sex=M
o add person/galvao-joao events "{type: birth, date: 1902, place: lisboa}"
o add person/galvao-joao events "{type: death, date: 1975, place: paris}"

o create person/chen-yongfu "name={given: 永福, surname: 陳, lang: zh-Hant, display: '{surname}{given}'}" "also-known-as=[{given: Yǒngfú, surname: Chén, lang: zh-Latn-pinyin, display: '{surname} {given}'}, {given: Louis, type: religious}]" sex=M
o add person/chen-yongfu also-known-as "{given: Yung-fu, surname: \"Ch'en\", lang: zh-Latn-wadegile, display: '{SURNAME} {given}', note: 'As the Chinese Labour Corps roll writes him, surname first and in capitals.'}"
o add person/chen-yongfu events "{type: birth, date: 3 AUG 1888, place: wenzhou}"
o add person/chen-yongfu events "{type: military-service, date: FROM 1917 TO 1920, note: 'The Chinese Labour Corps, in France'}"
o add person/chen-yongfu events "{type: residence, date: FROM 1920, place: paris}"
o add person/chen-yongfu events "{type: death, date: 1954, place: paris}"

# ---------------------------------------------------------------- generation 3

o create person/wit-geertruida "name={given: Geertruida Maria, surname-prefix: de, surname: Wit}" sex=F
o add --source amsterdam-population-register#de-wit-1895 person/wit-geertruida events "{type: birth, date: 1895, place: amsterdam, sources: [amsterdam-population-register#de-wit-1895]}"
o add person/wit-geertruida events "{type: occupation, value: governess, place: lisboa}"
o add person/wit-geertruida events "{type: Burial, place: lisboa}"

o create person/galvao-helena "name={given: Helena, surname: [{name: Verdier, origin: matrilineal}, {name: Galvão, origin: patrilineal, primary: true}]}" sex=F
o add person/galvao-helena events "{type: birth, date: 2 FEB 1920, place: lisboa}"
o add --source claudia-interview-2019#adoption-helena person/galvao-helena events "{type: adoption, date: 1923, place: lisboa, sources: [claudia-interview-2019#adoption-helena]}"
o add person/galvao-helena events "{type: death, date: 2001, place: lisboa}"

o create person/chen-jiaming "name={given: 家明, surname: 陳, lang: zh-Hant, display: '{surname}{given}'}" "also-known-as=[{given: Jiāmíng, surname: Chén, lang: zh-Latn-pinyin, display: '{surname} {given}'}]" sex=M
o add person/chen-jiaming events "{type: birth, date: 1912, place: wenzhou}"
o add person/chen-jiaming events "{type: death, date: 1980, place: wenzhou}"

o create person/chen-jialan "name={given: 家蘭, surname: 陳, lang: zh-Hant, display: '{surname}{given}'}" "also-known-as=[{given: Jiālán, surname: Chén, lang: zh-Latn-pinyin, display: '{surname} {given}'}]" sex=F
o add person/chen-jialan events "{type: birth, date: 1914, place: wenzhou}"
o add person/chen-jialan events "{type: residence, date: FROM 1930, place: paris}"
o add person/chen-jialan events "{type: death, date: 1990, place: paris}"

o create person/rahimi-parvin "name={given: پروین, surname: رحیمی, lang: fa}" "also-known-as=[{given: Parvin, surname: Rahimi, lang: fa-Latn}]" sex=F
o add person/rahimi-parvin events "{type: birth, date: PERSIAN 1308, place: tehran}"
o add person/rahimi-parvin events "{type: education, date: FROM PERSIAN 1 MEHR 1326 TO PERSIAN KHORD 1330, place: tehran, note: The University of Tehran}"
o add person/rahimi-parvin events "{type: death, date: 2010, place: reykjavik}"

o create person/jonsson-gunnar "name={given: Gunnar, surname: [{name: Jónsson, origin: patronymic}]}" sex=M
o add person/jonsson-gunnar events "{type: birth, date: 1925, place: reykjavik}"
o add person/jonsson-gunnar events "{type: education, date: FROM 1948 TO 1952, place: edinburgh}"
o add person/jonsson-gunnar events "{type: death, date: 1999, place: reykjavik}"

# ---------------------------------------------------------------- generation 4

o create person/galvao-claudia "name={given: Cláudia, surname: [{name: Chen, origin: matrilineal}, {name: Galvão, origin: patrilineal, primary: true}]}" sex=F
o add person/galvao-claudia events "{type: birth, date: 1946, place: paris}"
o add person/galvao-claudia also-known-as "{given: Cláudia, surname: [Galvão, {name: Marchena, prefix: de}], type: married, from: 4 SEP 1971, until: 16 FEB 1983}"
o set person/galvao-claudia "also-known-as[0].private" true

o create person/marchena-pablo "name={given: Pablo, surname: [{name: Marchena, origin: patrilineal}, {name: Ruiz, origin: matrilineal}]}" sex=M
o add person/marchena-pablo events "{type: birth, date: 1945, place: madrid}"
o add person/marchena-pablo events "{type: illness, date: 2009, place: madrid, note: a heart operation at La Paz}"
o set person/marchena-pablo "events[1].private" true

o create person/ofori-kwame "name={given: Kwame, surname: Ofori}" sex=M
o add --source ofori-family-account#kwame person/ofori-kwame events "{type: birth, date: 7 JAN 1950, place: kumasi, sources: [ofori-family-account#kwame]}"
o set person/ofori-kwame living false

o create person/sato-yumiko "name={given: 由美子, surname: 佐藤, lang: ja, display: '{surname}{given}'}" "also-known-as=[{given: Yumiko, surname: Satō, lang: ja-Latn-hepburn, display: '{surname} {given}'}, {given: ゆみこ, surname: さとう, lang: ja-Hira, display: '{surname} {given}'}]" sex=F
o add --source 'tono-koseki#sato-yumiko OPD' person/sato-yumiko events "{type: birth, date: 21 JUN 1955, place: tono, sources: [tono-koseki#sato-yumiko OPD]}"
o set person/sato-yumiko private true

o create person/gunnarsdottir-solveig "name={given: Sólveig, surname: [{name: Gunnarsdóttir, origin: patronymic}]}" sex=F
o add person/gunnarsdottir-solveig events "{type: birth, date: 1953, place: reykjavik}"

o create person/vitthal-ramesh "name={given: रमेश, surname: [{name: विठ्ठल, origin: patronymic}], lang: mr}" "also-known-as=[{given: Ramesh, surname: [{name: Vitthal, origin: patronymic}], lang: mr-Latn}, {given: रमेश, surname: [{name: विठ्ठल, origin: patronymic}, {name: कासेगावकर, origin: locative, primary: true}], lang: mr, type: birth, until: 1975}]" sex=M
o add person/vitthal-ramesh events "{type: birth, date: 1950, place: kasegaon}"

# ---------------------------------------------------------------- generation 5

o create person/marchena-marco "name={given: Marco, surname: [{name: Marchena, origin: patrilineal}, {name: Chen, origin: matrilineal}]}" sex=M
o add person/marchena-marco events "{type: birth, date: 14 MAR 1975 09:20, place: madrid}"
o add person/marchena-marco events "{type: residence, date: FROM 2010, place: kinkerstraat-41}"

o create person/solveigarson-stefan "name={given: Stefán, surname: [{name: Sólveigarson, origin: matronymic}], sort: '{given} {patronymic}'}" sex=M
o add person/solveigarson-stefan events "{type: birth, date: 8 NOV 1978, place: reykjavik}"

o create person/marchena-lucas "name={given: Lucas, surname: [{name: Marchena, origin: patrilineal}, {name: Sato, origin: matrilineal}]}" sex=M
o add person/marchena-lucas events "{type: birth, date: 1988, place: madrid}"

o create person/bak-seoyeon "name={given: 서연, surname: 박, lang: ko, display: '{surname}{given}'}" "also-known-as=[{given: Seo-yeon, surname: Bak, lang: ko-Latn, display: '{surname} {given}'}]" sex=F
o add person/bak-seoyeon events "{type: birth, date: 1990, place: busan}"

# ---------------------------------------------------------------- generation 6

o create --source 'amsterdam-births-2018#2018-021733 OPD' person/marchena-solveigarson-sasha "name={given: Sasha, surname: [Marchena, Sólveigarson]}" "sex={value: X, sources: [amsterdam-births-2018#2018-021733 OPD]}"
o add --source 'amsterdam-births-2018#2018-021733 OPD' person/marchena-solveigarson-sasha events "{type: birth, date: 3 APR 2018, place: amsterdam, sources: [amsterdam-births-2018#2018-021733 OPD]}"
o add --source amsterdam-births-2018#2018-021733 person/marchena-solveigarson-sasha sources amsterdam-births-2018#2018-021733

# ------------------------------------------------------------- not yet placed

o create person/doe-john "name={given: John, surname: Doe}" sex=M

# ---------------------------------------------------------------------- families

o create family/andersson-per--unknown "partners=[andersson-per, unknown]" "children=[persdotter-karin]"
o create family/persdotter-karin--unknown "partners=[persdotter-karin, {name: Hinrich Lang}]" "children=[lang-georg]"

o create family/rossi-bartolomeo--sarti-angela "partners=[rossi-bartolomeo, sarti-angela]" type=clandestine
o add family/rossi-bartolomeo--sarti-angela events "{type: marriage, date: 1775}"
o add family/rossi-bartolomeo--sarti-angela events "{type: annulment, date: 1776}"
o create family/rossi-bartolomeo--sarti-angela-1778 "partners=[rossi-bartolomeo, sarti-angela]" "children=[rossi-caterina]"
o add family/rossi-bartolomeo--sarti-angela-1778 events "{type: marriage, date: BEF 1778}"

o create family/lang-georg--rossi-caterina "partners=[lang-georg, rossi-caterina]" type=married "children=[lang-joseph, lang-anna-maria]"
o add family/lang-georg--rossi-caterina events "{type: marriage, date: 1802, place: firenze}"

o create family/lang-anna-maria--verdier-louis "partners=[lang-anna-maria, verdier-louis]" type=married "children=[verdier-jean-baptiste, verdier-louisa]"
o add --source 'french-church-london-baptisms#1834-042 OPD' family/lang-anna-maria--verdier-louis "children[0].sources" 'french-church-london-baptisms#1834-042 OPD'
o add family/lang-anna-maria--verdier-louis events "{type: marriage, date: ABT 1832, place: london}"

o create family/ershov-ivan--lang-anna-maria "partners=[ershov-ivan, lang-anna-maria]" type=married "children=[ershov-michael]"
o add family/ershov-ivan--lang-anna-maria children "{id: ershov-lawrence, status: disproven, note: \"An index of births read the household's surname for his parentage.\"}"
o add family/ershov-ivan--lang-anna-maria events "{type: marriage, date: 7 MAY 1853, place: london}"

o create family/verdier-louisa--unknown "partners=[verdier-louisa, unknown]" "children=[ershov-lawrence]"

o create family/papadopoulou-maria--verdier-jean-baptiste "partners=[papadopoulou-maria, verdier-jean-baptiste]" type=married "children=[verdier-anna, verdier-pauline, verdier-pierre]"
o add family/papadopoulou-maria--verdier-jean-baptiste children "{id: unknown, note: \"Jean Baptiste's death record counts four children of the marriage, and three are known.\"}"
o add --source london-marriages-1858#p112 family/papadopoulou-maria--verdier-jean-baptiste events "{type: marriage, date: 21 APR 1858, place: london, sources: [london-marriages-1858#p112]}"
o set --source london-marriages-1858#p112 family/papadopoulou-maria--verdier-jean-baptiste "events[0].with" "[{person: verdier-jean-baptiste, role: groom, age: 24}, {person: papadopoulou-maria, role: bride, age: 18}]"
o add --source london-marriages-1858#p112 family/papadopoulou-maria--verdier-jean-baptiste "events[0].with" "{person: lang-anna-maria, role: witness}"
o add --source london-marriages-1858#p112 family/papadopoulou-maria--verdier-jean-baptiste "events[0].with" "{name: {given: Πέτρος, surname: Παπαδόπουλος, lang: el}, role: witness}"
o add --source 'census-1861-england#f088 OSI' family/papadopoulou-maria--verdier-jean-baptiste sources 'census-1861-england#f088 OSI'
o add person/verdier-pauline events "{type: lds-sealing-to-parents, date: 14 MAR 1996, family: papadopoulou-maria--verdier-jean-baptiste, attributes: [{type: temple, value: SLAKE}, {type: status, value: completed, date: 20 MAR 1996}]}"

o create family/benitez-rosa--ershov-michael "partners=[benitez-rosa, ershov-michael]" type=married "children=[ershov-ivan-1885]"
o add family/benitez-rosa--ershov-michael events "{type: marriage, date: 1884, place: buenos-aires}"

o create family/al-halabi-mahmud--verdier-anna "partners=[al-halabi-mahmud, verdier-anna]" type=married "children=[al-halabi-zaynab, al-halabi-1889]"
o add --source alexandria-sharia-court-1302#marriage-al-halabi family/al-halabi-mahmud--verdier-anna events "{type: marriage, date: ISLAMIC 12 RAJAB 1302, place: alexandria, sources: [alexandria-sharia-court-1302#marriage-al-halabi]}"

o create family/galvao-henrique--verdier-pauline "partners=[galvao-henrique, verdier-pauline]" type=married "children=[galvao-joao]"
o add --source claudia-interview-2019#adoption-helena family/galvao-henrique--verdier-pauline children "{id: galvao-helena, relation: {galvao-henrique: step, verdier-pauline: adoptive}}"
o add family/galvao-henrique--verdier-pauline events "{type: marriage, date: 11 JUN 1900, place: lisboa}"

o create family/chen-jialan--galvao-joao "partners=[chen-jialan, galvao-joao]" "children=[galvao-claudia]"
o add family/chen-jialan--galvao-joao events "{type: residence, date: FROM 1940, place: paris}"

o create family/galvao-claudia--marchena-pablo "partners=[galvao-claudia, marchena-pablo]" type=married "children=[marchena-marco]"
o add family/galvao-claudia--marchena-pablo events "{type: marriage, date: 4 SEP 1971, place: madrid}"
o add family/galvao-claudia--marchena-pablo events "{type: divorce, date: 16 FEB 1983, place: madrid}"

o create family/galvao-claudia--ofori-kwame "partners=[galvao-claudia, ofori-kwame]" type=unmarried
o add family/galvao-claudia--ofori-kwame events "{type: residence, date: FROM 1991, place: madrid}"
o set family/galvao-claudia--ofori-kwame private true

o create family/marchena-pablo--sato-yumiko "partners=[marchena-pablo, sato-yumiko]" type=married "children=[marchena-lucas]"
o add family/marchena-pablo--sato-yumiko events "{type: marriage, date: 1986, place: madrid}"

o create family/bak-seoyeon--marchena-lucas "partners=[bak-seoyeon, marchena-lucas]" type=married
o add family/bak-seoyeon--marchena-lucas events "{type: marriage, date: 2016, place: madrid}"

o create family/jonsson-gunnar--rahimi-parvin "partners=[jonsson-gunnar, rahimi-parvin]" type=married "children=[gunnarsdottir-solveig]"
o add family/jonsson-gunnar--rahimi-parvin events "{type: marriage, date: 1952, place: edinburgh}"

o create family/gunnarsdottir-solveig--vitthal-ramesh "partners=[gunnarsdottir-solveig, vitthal-ramesh]" "children=[solveigarson-stefan]"

o create family/marchena-marco--solveigarson-stefan "partners=[marchena-marco, solveigarson-stefan]" type=civil-union
o add --source 'amsterdam-births-2018#2018-021733 OPD' family/marchena-marco--solveigarson-stefan children "{id: marchena-solveigarson-sasha, relation: unknown, sources: [amsterdam-births-2018#2018-021733 OPD]}"
o add family/marchena-marco--solveigarson-stefan events "{type: civil-union, date: 19 MAY 2018, place: amsterdam}"
o add family/marchena-marco--solveigarson-stefan "events[0].with" "{person: sato-yumiko, role: witness}"

# a parent with nobody beside them, or somebody nobody has named
o create family/wit-geertruida--unknown "partners=[wit-geertruida, unknown]" "children=[galvao-helena]"
o create family/chen-yuying--unknown "partners=[chen-yuying, unknown]" "children=[{id: chen-yongfu, relation: natural, note: \"私生子, a natural son, as the mission's register has it.\"}]"
o create family/chen-yuying--none "partners=[chen-yuying, none]" "children=[{name: 陳玉珍, relation: adoptive}]"
o create family/chen-yongfu--unknown "partners=[chen-yongfu, unknown]" "children=[chen-jiaming, chen-jialan]"

# ------------------------------------------------------------------------ notes

o create note/a-london-wedding-of-1859 "title=A London wedding of 1859"
o create note/the-boatman-of-wenzhou "title=The boatman of Wenzhou"
o set note/the-boatman-of-wenzhou private true
o add note/a-london-wedding-of-1859 events "{type: marriage, date: 3 OCT 1859, place: london, with: [{person: lang-anna-maria, role: witness}, {name: {given: Jan, surname: Kowalski}, role: groom}, {name: {given: Bridget, surname: Murphy}, role: bride}]}"

# ------------------------------------------------------------------- by hand

cp "$T/origins.yaml" origins.yaml
o adopt -m "Say what the project is about"

while read -r file; do
  mkdir -p "$(dirname "$file")"
  cp "$T/$file" "$file"
done < <(cd "$T" && find . -type f ! -name '*.md' ! -name origins.yaml ! -name .gitattributes | sort)
o adopt -m "Add the photographs and the scans"
o set person/verdier-jean-baptiste portrait photos/1890-portrait.jpg
o add person/verdier-jean-baptiste media "{file: photos/1890-portrait.jpg, title: Jean Baptiste in his fifties, date: ABT 1890, place: london, depicts: [verdier-jean-baptiste], note: 'A cabinet card, the studio''s stamp on the back cut away.'}"
o add family/papadopoulou-maria--verdier-jean-baptiste media "{file: photos/1858-wedding-group.jpg, title: The wedding party, date: 21 APR 1858, place: london}"
o add family/papadopoulou-maria--verdier-jean-baptiste "media[0].depicts" "{person: papadopoulou-maria, region: [0.41, 0.12, 0.58, 0.47], note: 'Named in pencil on the back, as the bride.'}"
o add family/papadopoulou-maria--verdier-jean-baptiste "media[0].depicts" "{person: verdier-jean-baptiste, region: [0.59, 0.1, 0.76, 0.49]}"
o add person/verdier-jean-baptiste media "{file: photos/1875-carte-de-visite.jpg, type: photo, title: A carte de visite, date: ABT 1875}"
o add family/papadopoulou-maria--verdier-jean-baptiste media "{file: ../../people/verdier-jean-baptiste/photos/1875-carte-de-visite.jpg, note: 'Kept in the album beside the wedding group.'}"
o add person/verdier-jean-baptiste "events[3].media" "{file: documents/1901-death-notice.jpg, type: newspaper, title: A death notice, note: 'Cut from a London daily and kept loose in the album.'}"
o add person/wit-geertruida "events[2].media" "{file: photos/gravestone.jpg, type: tombstone, note: 'Her name alone, the stone too worn to show any dates.'}"
o add person/galvao-helena media "{file: 'D:/Fotos/helena-1940.jpg', type: photo, title: Helena at twenty, note: 'Scanned by a cousin, on whose computer it was, and not yet found.'}"
o add person/galvao-helena "media[0].tags" "To do"
o add person/marchena-marco media "{file: photos/2019-at-home.jpg, title: 'At home, with Stefán and Sasha', date: 2019, depicts: [marchena-marco, solveigarson-stefan, marchena-solveigarson-sasha]}"
o set person/marchena-marco "media[0].private" true
o add person/marchena-lucas media "{file: photos/1988-christening.jpg, title: \"Lucas's christening, in his mother's arms\", date: 1988, place: madrid, depicts: [marchena-lucas, sato-yumiko]}"
o add source/parish-san-lorenzo-baptisms citations.1779-042.images scans/1779-042.jpg
o set source/parish-san-lorenzo-baptisms "citations.1779-042.images[0].region" "[0.06, 0.38, 0.94, 0.61]"
o add source/parish-san-lorenzo-baptisms citations.1779-042.images scans/1779-042-continued.jpg
o add source/cimitero-san-lorenzo-gravestones citations.sarti-angela.images scans/sarti-angela.jpg

# the state of the corpus the hypotheses below were written against, which everything
# after it has moved on from
base="$(git rev-parse --short=7 HEAD)"

# ------------------------------------------------------------------ arguments

while read -r file; do
  body="$(prose "$T/$file")"
  [ -n "$body" ] || continue
  case "$file" in
    people/*) kind=person ;;
    families/*) kind=family ;;
    places/*) kind=place ;;
    sources/*) kind=source ;;
    repositories/*) kind=repository ;;
    notes/*) kind=note ;;
  esac
  slug="$(basename "${file%/README.md}")"
  o set "$kind/$slug" body "$body"
done < <(cd "$T" && find people families places sources repositories notes -mindepth 2 -maxdepth 2 -name README.md | sort)

# ------------------------------------------------------------------ hypotheses

for file in "$T"/hypotheses/*/README.md; do
  slug="$(basename "${file%/README.md}")"
  status="$(sed -n 's/^status: //p' "$file")"
  mkdir -p "hypotheses/$slug"
  sed -e 's/^status: .*/status: open/' -e "s/^base: .*/base: $base/" "$file" > "hypotheses/$slug/README.md"
  o adopt -m "$(sed -n 's/^# //p' "$file" | head -n 1)"
  case "$status" in
    open) ;;
    rejected) o hypothesis reject "$slug" ;;
    *) echo "nothing takes a hypothesis to $status" >&2; exit 1 ;;
  esac
done

# One pack in place of an object loose for everything each command wrote, so that a
# test taking a copy of the fixture copies a hundred files rather than well over a
# thousand. Packing changes no sha.
git repack -a -d -q

echo "done: $(git rev-list --count HEAD) commits"
