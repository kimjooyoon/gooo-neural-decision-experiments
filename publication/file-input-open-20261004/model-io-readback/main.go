// Inspect retained model-file errors; never invoke a model or compiler.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
)

type process struct {
	Condition string `json:"condition"`
	Elapsed   int64  `json:"elapsed_ns"`
	Exit      *int   `json:"exit_code"`
	Joined    *bool  `json:"process_joined"`
	Timeout   *bool  `json:"timeout"`
	Released  *bool  `json:"writer_released_fifo_reader"`
	Absent    *bool  `json:"output_directory_absent"`
	Bytes     *int   `json:"stdout_bytes"`
	Generated *bool  `json:"source_generation_observed"`
	Stderr    string `json:"stderr"`
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func check(ok bool, why string) {
	if !ok {
		panic(why)
	}
}
func raw(path string) []byte { b, err := os.ReadFile(path); must(err); return b }

func inspect(root, scope string, released bool) []map[string]any {
	var summary struct{ Rows []process }
	must(json.Unmarshal(raw(filepath.Join(root, scope, "summary.json")), &summary))
	check(len(summary.Rows) == 2, "model error scope changed")
	var rows []map[string]any
	for i, mode := range []string{"metadata", "weights"} {
		p := summary.Rows[i]
		var saved process
		must(json.Unmarshal(raw(filepath.Join(root, scope, mode+"-process.json")), &saved))
		check(reflect.DeepEqual(p, saved), "process differs from summary")
		check(p.Condition == mode+"-fifo" && p.Elapsed > 0 && p.Exit != nil && *p.Exit == 1 &&
			p.Joined != nil && *p.Joined && p.Timeout != nil && !*p.Timeout &&
			p.Released != nil && *p.Released == released && p.Absent != nil && *p.Absent &&
			p.Bytes != nil && *p.Bytes == 0 && p.Generated != nil && !*p.Generated, "missing/error facts changed")
		check(len(raw(filepath.Join(root, scope, mode+".stdout"))) == 0 &&
			bytes.Equal(raw(filepath.Join(root, scope, mode+".stderr")), []byte(p.Stderr)), "original output differs")
		_, err := os.Stat(filepath.Join(root, scope, mode+"-results"))
		check(os.IsNotExist(err), "early error created a result directory")
		rows = append(rows, map[string]any{"scope": scope, "condition": p.Condition, "elapsed_ns": p.Elapsed,
			"writer_release": released, "joined": true, "timeout": false, "exit_code": 1, "output_directory_absent": true})
	}
	return rows
}

func main() {
	check(len(os.Args) == 2, "usage: model-io-readback saved-root")
	rows := append(inspect(os.Args[1], "model-io-baseline", true), inspect(os.Args[1], "model-io-revision2", false)...)
	must(json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "PASS", "rows": rows,
		"model_predictions_added": 0, "native_runs_added": 0, "scope": "saved model I/O errors only"}))
	fmt.Fprintln(os.Stderr, "Original latencies retained; controlled writer release is not natural completion time.")
}
