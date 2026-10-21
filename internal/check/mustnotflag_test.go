package check

import (
	"testing"

	"github.com/origins-project/origins/internal/model"
)

// These are the cases the fixture corpus carries on purpose. Every one of them is a
// correct corpus that a careless validator would complain about, and a complaint here
// is a bug in the validator rather than in the data. Each stands for the fixture entity
// named in guards.
func TestMustNotFlag(t *testing.T) {
	cases := []struct {
		guards string
		why    string
		corpus *model.Corpus
	}{
		{
			guards: "rossi-domenico",
			why:    "a death remembered as a month with no year",
			corpus: build(&model.Person{
				Slug: "rossi-domenico", Name: model.PersonName{Given: "Domenico", Surname: model.SurnameOf("Rossi")},
				Events: []model.Event{born("1856"), died("NOV")},
			}),
		},
		{
			guards: "sarti-angela",
			why:    "a worn gravestone giving a day and a month",
			corpus: build(&model.Person{
				Slug: "sarti-angela", Name: model.PersonName{Given: "Angela", Surname: model.SurnameOf("Sarti")},
				Events: []model.Event{born("ABT 1805"), died("3 MAR")},
			}),
		},
		{
			guards: "rossi-infante-1866",
			why:    "an infant who died on the day of birth, with no recorded sex",
			corpus: build(
				&model.Person{
					Slug: "rossi-infante-1866", Name: model.PersonName{Given: "Infant", Surname: model.SurnameOf("Rossi")},
					Sex:    model.StatedSex{Value: model.SexUnknown},
					Events: []model.Event{born("12 MAY 1866"), died("12 MAY 1866")},
				},
				childOf("rossi-giovanni", "rossi-infante-1866"),
				&model.Person{
					Slug: "rossi-giovanni", Name: model.PersonName{Given: "Giovanni Battista", Surname: model.SurnameOf("Rossi")},
					Sex: model.StatedSex{Value: model.Male}, Events: []model.Event{born("12 MAR 1834")},
				},
			),
		},
		{
			guards: "conti-luigi",
			why:    "an unmarried mother, the father unknown, and the child called a word origins does not know",
			corpus: build(
				&model.Person{
					Slug: "conti-luigi", Name: model.PersonName{Given: "Luigi", Surname: model.SurnameOf("Conti")}, Sex: model.StatedSex{Value: model.Male},
					Events: []model.Event{born("3 AUG 1888")},
				},
				&model.Person{
					Slug: "conti-teresa", Name: model.PersonName{Given: "Teresa", Surname: model.SurnameOf("Conti")}, Sex: model.StatedSex{Value: model.Female},
					Events: []model.Event{born("1865")},
				},
				&model.Family{
					Slug:     "conti-teresa--unknown",
					Partners: model.PartnersOf("conti-teresa", model.Unknown),
					Children: []model.Child{{ID: "conti-luigi", Relation: model.Relationship{Word: "natural"}}},
				},
			),
		},
		{
			guards: "conti-teresa--none",
			why:    "a woman who adopted alone, a child known only by name",
			corpus: build(
				&model.Person{Slug: "conti-teresa", Name: model.PersonName{Given: "Teresa", Surname: model.SurnameOf("Conti")}},
				&model.Family{
					Slug:     "conti-teresa--none",
					Partners: model.PartnersOf("conti-teresa", model.None),
					Children: []model.Child{{Name: model.PersonName{Text: "Gemma Nocentini"}, Relation: model.Relationship{Word: model.Adoptive}}},
				},
			),
		},
		{
			guards: "rossi-lorenzo",
			why:    "a link disproven beside the true one, and an unknown child, are no parentage",
			corpus: build(
				&model.Person{Slug: "rossi-lorenzo", Name: model.PersonName{Given: "Lorenzo", Surname: model.SurnameOf("Rossi")}, Events: []model.Event{born("1890")}},
				&model.Person{Slug: "rossi-annunziata", Name: model.PersonName{Given: "Annunziata", Surname: model.SurnameOf("Rossi")}, Events: []model.Event{born("1860")}},
				&model.Person{Slug: "rossi-antonio", Name: model.PersonName{Given: "Antonio", Surname: model.SurnameOf("Rossi")}, Events: []model.Event{born("1800"), died("1850")}},
				&model.Person{Slug: "neri-caterina", Name: model.PersonName{Given: "Caterina", Surname: model.SurnameOf("Neri")}, Events: []model.Event{born("1820")}},
				childOf("rossi-annunziata", "rossi-lorenzo"),
				&model.Family{
					Slug:     "neri-caterina--rossi-antonio",
					Partners: model.PartnersOf("neri-caterina", "rossi-antonio"),
					Children: []model.Child{{ID: "rossi-lorenzo", Status: model.Disproven}, {ID: model.Unknown}},
				},
			),
		},
		{
			guards: "modiano-rachele--rossi-giovanni-1902",
			why:    "a family with no children",
			corpus: build(
				&model.Person{Slug: "modiano-rachele", Name: model.PersonName{Given: "Rachele", Surname: model.SurnameOf("Modiano")}},
				&model.Person{Slug: "rossi-giovanni-1902", Name: model.PersonName{Given: "Giovanni", Surname: model.SurnameOf("Rossi")}},
				&model.Family{
					Slug:     "modiano-rachele--rossi-giovanni-1902",
					Partners: model.PartnersOf("modiano-rachele", "rossi-giovanni-1902"),
					Events:   []model.Event{{Type: "marriage", Date: on("14 OCT 1934")}},
				},
			),
		},
		{
			guards: "rossi-bartolomeo--sarti-angela",
			why:    "a couple who married twice, the first time in a type origins does not know",
			corpus: build(
				&model.Person{Slug: "rossi-bartolomeo", Name: model.PersonName{Given: "Bartolomeo", Surname: model.SurnameOf("Rossi")},
					Events: []model.Event{born("12 FEB 1750/51"), died("1799")}},
				&model.Person{Slug: "sarti-angela", Name: model.PersonName{Given: "Angela", Surname: model.SurnameOf("Sarti")},
					Events: []model.Event{born("EST 1755")}},
				&model.Family{
					Slug:     "rossi-bartolomeo--sarti-angela",
					Partners: model.PartnersOf("rossi-bartolomeo", "sarti-angela"),
					Type:     "clandestine",
					Events:   []model.Event{{Type: "marriage", Date: on("1775")}, {Type: "annulment", Date: on("1776")}},
				},
				&model.Family{
					Slug:     "rossi-bartolomeo--sarti-angela-1778",
					Partners: model.PartnersOf("rossi-bartolomeo", "sarti-angela"),
					Events:   []model.Event{{Type: "marriage", Date: on("BEF 1778")}},
				},
			),
		},
		{
			guards: "unrecorded-child-1845",
			why:    "an open hypothesis is research to do, never a fault",
			corpus: build(&model.Hypothesis{
				Slug:       "unrecorded-child-1845",
				Status:     model.Open,
				Base:       "8f78553",
				Confidence: model.Possible,
				Supports:   []model.CitationRef{{Source: "parish-san-lorenzo-baptisms", Anchor: "1842-007"}},
				Edits: []model.Edit{model.Create{
					Entity: model.EntityRef{Kind: model.KindPerson, Slug: "rossi-unknown-1845"},
				}},
			}, &model.Source{
				Slug:      "parish-san-lorenzo-baptisms",
				Citations: map[string]model.Citation{"1842-007": {Text: "..."}},
			}),
		},
		{
			guards: "rossi-elena",
			why:    "age rules are for birth parents; an adoptive mother may be younger",
			corpus: build(
				&model.Person{
					Slug: "rossi-elena", Name: model.PersonName{Given: "Elena", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Female},
					Events: []model.Event{born("2 FEB 1920"), died("2001")},
				},
				childOf("moretti-giulia", "rossi-elena"),
				&model.Family{
					Slug:     "galli-ines--rossi-pietro",
					Partners: model.PartnersOf("galli-ines", "rossi-pietro"),
					Children: []model.Child{{ID: "rossi-elena", Relation: model.Relationship{Partners: []model.PartnerRelation{
						{Partner: "galli-ines", Relation: model.Step}, {Partner: "rossi-pietro", Relation: model.Adoptive},
					}}}},
				},
				&model.Person{
					Slug: "galli-ines", Name: model.PersonName{Given: "Ines", Surname: model.SurnameOf("Galli")}, Sex: model.StatedSex{Value: model.Female},
					Events: []model.Event{born("1930")},
				},
				&model.Person{
					Slug: "moretti-giulia", Name: model.PersonName{Given: "Giulia", Surname: model.SurnameOf("Moretti")},
					Sex: model.StatedSex{Value: model.Female}, Events: []model.Event{born("1895")},
				},
				// born after the child and dead before her, which would be three
				// separate errors if the relation were biological
				&model.Person{
					Slug: "rossi-pietro", Name: model.PersonName{Given: "Pietro", Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
					Events: []model.Event{born("1925"), died("1930")},
				},
			),
		},
	}

	for _, c := range cases {
		t.Run(c.guards, func(t *testing.T) {
			if got := ruleIDs(Run(c.corpus), Error); got != nil {
				t.Errorf("%s (%s) is correct data, but check reported %v", c.guards, c.why, got)
			}
		})
	}
}

// The adoptive case above passes if the age rules are skipped, and it also passes if
// they were never written. This is the control.
func TestAgeRulesApplyToBirthParents(t *testing.T) {
	corpus := func(relation model.Relation) *model.Corpus {
		return build(
			&model.Person{
				Slug: "rossi-elena", Name: model.PersonName{Surname: model.SurnameOf("Rossi")},
				Events: []model.Event{born("2 FEB 1920")},
			},
			&model.Family{
				Slug:     "rossi-pietro--unknown",
				Partners: model.PartnersOf("rossi-pietro", model.Unknown),
				Children: []model.Child{{ID: "rossi-elena", Relation: model.Relationship{Word: relation}}},
			},
			&model.Person{
				Slug: "rossi-pietro", Name: model.PersonName{Surname: model.SurnameOf("Rossi")}, Sex: model.StatedSex{Value: model.Male},
				Events: []model.Event{born("1925"), died("1930")},
			},
		)
	}
	fires(t, "child-before-parent", corpus(model.Birth))
	silent(t, "child-before-parent", corpus(model.Adoptive))
	silent(t, "child-before-parent", corpus(model.Guardian))
	silent(t, "child-before-parent", corpus(model.RelationUnknown))
	silent(t, "child-before-parent", corpus("godchild"))
}
