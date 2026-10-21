package api

import (
	"encoding/json"

	"github.com/origins-project/origins/internal/api/schema"
)

// Schema is the JSON Schema of one kind of record, of the manifest as project, or of
// what they share as common; asked of nothing, it is every one of them by name.
func Schema(name string) (json.RawMessage, error) {
	if name != "" {
		return schema.Of(name)
	}
	all := map[string]json.RawMessage{}
	for _, one := range schema.Names() {
		read, err := schema.Of(one)
		if err != nil {
			return nil, err
		}
		all[one] = read
	}
	return json.Marshal(all)
}
