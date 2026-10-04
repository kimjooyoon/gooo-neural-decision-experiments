// record-field-observe compares finite field completion under explicit attempt
// budgets using the actual Gooo compiler and the existing frozen own model.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	compiler := flag.String("gooo", "gooo", "clean compiler executable")
	sha := flag.String("compiler-source", "", "exact compiler revision")
	source := flag.String("source", "", "record field assembly fixture")
	cases := flag.String("cases", "", "native composition case fixture")
	model := flag.String("model", "", "existing frozen own-model metadata")
	out := flag.String("out", "", "fresh public observation directory")
	private := flag.String("private-out", "", "fresh private resource log directory")
	verify := flag.String("verify-json", "", "read one captured composition response")
	saved := flag.Bool("saved", false, "captured response reuses a saved composition")
	flag.Parse()
	var err error
	if *verify != "" {
		var raw []byte
		raw, err = os.ReadFile(*verify)
		if err == nil {
			var row measurement
			row, _, err = measure(raw, *sha, *saved)
			if err == nil {
				err = json.NewEncoder(os.Stdout).Encode(row)
			}
		}
	} else {
		err = capture(*compiler, *sha, *source, *cases, *model, *out, *private)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
