package store

import (
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/origins-project/origins/internal/model"
)

// Combine folds one entity into another of the same kind, which is what deciding that
// two records describe one thing amounts to inside a record.
//
// Whatever can hold more than one thing takes both. Lists are joined, leaving out what
// the surviving record already says word for word; a source's citations take the
// anchors only the other transcribed, and two transcriptions of one anchor are
// reconciled the way the records themselves are. The two bodies become two paragraphs,
// and so do two notes on one thing, prose being the one text where having both is
// having more rather than having a muddle.
//
// Whatever holds one thing is never made to choose. Where one record is silent the
// other's value stands, and where both speak and disagree the fold stops and says which
// field it was: which of two spellings is the surname is a judgement about the records,
// and origins has nothing to decide it with. Unsetting one of them is how the
// researcher says which.
//
// A refusal leaves the surviving record as it was. The two are walked once to find
// whatever they disagree about and again to take one into the other, so a fold that
// stops does not stop halfway and leave a record that is partly both.
//
// What is attached to the two records is folded with them, a photograph being part of
// what a record has to say. Saying so is all that happens here: putting the files
// where the surviving record can reach them is the business of whatever writes the
// corpus back.
//
// Nothing here looks beyond the two entities. What pointed at the record that goes, and
// what its name did to the name of a family, are questions about a corpus and not about
// a record.
func Combine(into, from any) error {
	fields := addressable(into)
	if fields == nil || reflect.TypeOf(into) != reflect.TypeOf(from) {
		return fmt.Errorf("origins cannot combine a %T with a %T", from, into)
	}
	for _, taking := range []bool{false, true} {
		if err := reconcile("", fields, reflect.ValueOf(from).Elem(), taking); err != nil {
			return err
		}
		if err := beside(model.Files(into), *model.Files(from), taking); err != nil {
			return err
		}
	}
	return nil
}

// beside takes the files of one record's directory in among another's.
//
// A name both of them use stops the fold like any other disagreement. Whether two
// files called photos/portrait.jpg are one photograph is not a question that can be
// answered from the names, and keeping either one of them would be answering it;
// renaming one is how the researcher says. The rest are taken whether or not anything
// points at them, an untranscribed scan being evidence that has not been read yet
// rather than evidence nobody wanted.
func beside(into *[]string, from []string, taking bool) error {
	for _, file := range from {
		if slices.Contains(*into, file) {
			return fmt.Errorf("files: both records have one called %q", file)
		}
	}
	if taking && len(from) > 0 {
		*into = append(*into, from...)
		slices.Sort(*into)
	}
	return nil
}

// reconcile takes each field of one record into the corresponding field of another.
// The fields are taken in name order, so a pair of records disagreeing about two of
// them reports the same one every time.
func reconcile(where string, fields map[string]any, from reflect.Value, taking bool) error {
	copied := reflect.New(from.Type()).Elem()
	copied.Set(from)
	other := addressable(copied.Addr().Interface())
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		a := reflect.ValueOf(fields[name]).Elem()
		b := reflect.ValueOf(other[name]).Elem()
		if err := absorb(under(where, name), a, b, taking); err != nil {
			return err
		}
	}
	return nil
}

// absorb puts what b holds into a, or says why the two cannot both stand. Asked without
// taking, it says the same thing and puts nothing anywhere.
func absorb(where string, a, b reflect.Value, taking bool) error {
	switch {
	case b.IsZero():
		return nil
	case a.IsZero():
		if taking {
			a.Set(b)
		}
		return nil
	case reflect.DeepEqual(a.Interface(), b.Interface()):
		return nil
	}

	// A name's surnames are one value, which of two lists is the name being a judgement.
	if _, whole := a.Interface().(model.Surnames); whole {
		return fmt.Errorf("%s: one record says %s and the other %s", where, stated(a), stated(b))
	}
	switch a.Kind() {
	case reflect.Slice:
		for i := 0; taking && i < b.Len(); i++ {
			if !holds(a, b.Index(i)) {
				a.Set(reflect.Append(a, b.Index(i)))
			}
		}
		return nil
	case reflect.Map:
		return anchors(where, a, b, taking)
	}
	if fields := addressable(a.Addr().Interface()); fields != nil {
		return reconcile(where, fields, b, taking)
	}
	if where == "body" || strings.HasSuffix(where, ".note") {
		if taking {
			a.SetString(strings.TrimRight(a.String(), "\n") + "\n\n" + strings.TrimLeft(b.String(), "\n"))
		}
		return nil
	}
	return fmt.Errorf("%s: one record says %s and the other %s", where, stated(a), stated(b))
}

// anchors takes the entries of one mapping into another. A mapping hands out a copy of
// what it holds rather than a place to change it, so an entry both of them have is
// reconciled in that copy and put back.
func anchors(where string, a, b reflect.Value, taking bool) error {
	for _, key := range b.MapKeys() {
		standing := a.MapIndex(key)
		if !standing.IsValid() {
			if taking {
				a.SetMapIndex(key, b.MapIndex(key))
			}
			continue
		}
		held := reflect.New(a.Type().Elem()).Elem()
		held.Set(standing)
		if err := absorb(under(where, key.String()), held, b.MapIndex(key), taking); err != nil {
			return err
		}
		if taking {
			a.SetMapIndex(key, held)
		}
	}
	return nil
}

func holds(list, entry reflect.Value) bool {
	for i := range list.Len() {
		if reflect.DeepEqual(list.Index(i).Interface(), entry.Interface()) {
			return true
		}
	}
	return false
}

// stated is a value as a message about it should read. A field holding a pointer is
// shown as what it points at, an address being no help to anybody deciding which of two
// records is right.
func stated(v reflect.Value) string {
	if v.Kind() == reflect.Pointer && !v.IsNil() {
		v = v.Elem()
	}
	return fmt.Sprintf("%q", fmt.Sprint(v.Interface()))
}

func under(where, name string) string {
	if where == "" {
		return name
	}
	return where + "." + name
}
