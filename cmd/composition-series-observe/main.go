// composition-series-observe checks current values in retained graph histories.
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
	goBin := flag.String("go-bin", "", "actual Go 1.27.1 executable")
	source := flag.String("source", "", "record assembly source")
	series := flag.String("series", "", "ordered case suites")
	models := flag.String("models", "", "frozen own-three model directory")
	out := flag.String("out", "", "fresh public capture directory")
	private := flag.String("private-out", "", "fresh private resource log directory")
	input := flag.String("verify-json", "", "captured body-compose response")
	saved := flag.Bool("saved", false, "response uses a saved construction")
	text := flag.Bool("text", false, "show a short current-input summary")
	flag.Parse()
	var err error
	if *input != "" {
		var raw []byte
		raw, err = os.ReadFile(*input)
		if err == nil {
			var report measurement
			report, err = measure(raw, *sha, *saved)
			if err == nil {
				if *text {
					err = writeText(os.Stdout, report, raw)
				} else {
					err = json.NewEncoder(os.Stdout).Encode(report)
				}
			}
		}
	} else {
		err = capture(*compiler, *sha, *goBin, *source, *series, *models, *out, *private)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
