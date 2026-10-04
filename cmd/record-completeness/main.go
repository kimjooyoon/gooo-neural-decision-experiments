// record-completeness reads a captured Gooo body-compose response and reports
// finite named-output and record-field matches with explicit missing observations.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	path := flag.String("input", "", "captured body-compose JSON response")
	asJSON := flag.Bool("json", false, "write machine-readable finite completeness")
	flag.Parse()
	if err := run(*path, *asJSON); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string, asJSON bool) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	report, err := measure(raw)
	if err != nil {
		return err
	}
	if asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	fmt.Printf("Named outputs: %d/%d (%s); unobserved: %d\n", report.Outputs.Passed, report.Outputs.Total, percentage(report.Outputs.Percent), report.Outputs.Unobserved)
	fmt.Printf("Record output fields: %d/%d (%s); unobserved: %d\n", report.Fields.Passed, report.Fields.Total, percentage(report.Fields.Percent), report.Fields.Unobserved)
	for _, gap := range report.Gaps {
		fmt.Printf("Case %d %s%s: %s; actual=%s expected=%s\n", gap.Case, gap.Activity, suffix(gap.Field), gap.Status, printable(gap.Actual), printable(gap.Expected))
	}
	return nil
}

func percentage(value *float64) string {
	if value == nil {
		return "unobserved"
	}
	return fmt.Sprintf("%.2f%%", *value)
}
func suffix(field string) string {
	if field == "" {
		return ""
	}
	return "." + field
}
func printable(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "unobserved"
	}
	return string(raw)
}
