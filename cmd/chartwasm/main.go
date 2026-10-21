//go:build js && wasm

// Command chartwasm is internal/chart for the pages, built as WebAssembly into the
// application by its build. It sets originsChart on the page: load takes the index as
// JSON, and draw lays out the chart a chart.Ask in JSON asks of it, answering with the
// JSON of a chart.Chart, or null for a shape there is not or before an index was
// loaded. A timeline runs a life still going to this year.
package main

import (
	"encoding/json"
	"syscall/js"
	"time"

	"github.com/origins-project/origins/internal/chart"
)

func main() {
	var kin *chart.Kin
	load := js.FuncOf(func(_ js.Value, args []js.Value) any {
		var index chart.Index
		if err := json.Unmarshal([]byte(args[0].String()), &index); err != nil {
			return err.Error()
		}
		kin = chart.KinOf(index)
		return nil
	})
	draw := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if kin == nil {
			return nil
		}
		var ask chart.Ask
		if err := json.Unmarshal([]byte(args[0].String()), &ask); err != nil {
			return nil
		}
		if ask.Year == 0 {
			ask.Year = time.Now().Year()
		}
		c, ok := chart.Draw(kin, ask)
		if !ok {
			return nil
		}
		data, err := json.Marshal(c)
		if err != nil {
			return nil
		}
		return string(data)
	})
	js.Global().Set("originsChart", map[string]any{"load": load, "draw": draw})
	select {}
}
