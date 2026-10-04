// native-record-observe measures real record assembly, actual field delivery,
// finite expectations and saved zero-inference controls using the Gooo CLI.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"time"
)

func main() {
	compiler := flag.String("gooo", "gooo", "clean compiler executable")
	sha := flag.String("compiler-source", "", "exact compiler source SHA")
	source := flag.String("source", "", "public record graph fixture")
	cases := flag.String("cases", "", "public finite case fixture")
	model := flag.String("model", "", "absolute unchanged own-model path")
	private := flag.String("private-out", "", "fresh resource log directory")
	public := flag.String("out", "", "fresh public evidence directory")
	controls := flag.Bool("controls", false, "add parallel, separate runtime-case and partial-expectation controls to an existing cohort")
	verify := flag.String("verify-json", "", "verify a captured six-case observation on any host")
	mode := flag.String("mode", "model", "model or deterministic capture")
	saved := flag.Bool("saved", false, "capture used saved composition")
	flag.Parse()
	if *verify != "" {
		raw, err := os.ReadFile(*verify)
		if err == nil {
			var row summary
			row, _, err = readObservation(raw, *sha, *mode, 0, *saved, resources{})
			if err == nil {
				err = json.NewEncoder(os.Stdout).Encode(row)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *controls {
		if err := additionalControls(*compiler, *sha, *model, *private, *public); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := observe(*compiler, *sha, *source, *cases, *model, *private, *public); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func observe(compiler, sha, source, cases, model, private, public string) error {
	if runtime.GOOS != "darwin" || len(sha) != 40 || !filepath.IsAbs(model) || private == "" || public == "" {
		return fmt.Errorf("macOS resources, exact source SHA, absolute model and fresh output paths required")
	}
	for _, directory := range []string{private, public} {
		if err := os.Mkdir(directory, 0755); err != nil {
			return err
		}
	}
	for name, path := range map[string]string{"source.gooo.fixture": source, "cases.json": cases} {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(public, name), raw, 0644); err != nil {
			return err
		}
	}
	var rows []summary
	var common envelope
	for _, mode := range []string{"model", "deterministic"} {
		input := "source.gooo.fixture"
		for stage := range 3 {
			stem := fmt.Sprintf("%s-%d", mode, stage)
			args := []string{"body-compose", "--source", input, "--cases", "cases.json"}
			if mode == "model" {
				args = append(args, "--model", model)
			}
			raw, resource, err := invoke(compiler, public, private, stem, args...)
			if err != nil {
				return err
			}
			row, current, err := readObservation(raw, sha, mode, stage, false, resource)
			if err != nil {
				return fmt.Errorf("%s: %w", stem, err)
			}
			if len(rows) != 0 && !sameProgram(common, current) {
				return fmt.Errorf("%s source/code/driver fixed point differs", stem)
			}
			common = current
			if err := os.WriteFile(filepath.Join(public, stem+".json"), raw, 0644); err != nil {
				return err
			}
			var original struct {
				Composition json.RawMessage `json:"composition"`
			}
			if err := json.Unmarshal(raw, &original); err != nil {
				return err
			}
			compositionFile := stem + "-composition.json"
			if err := os.WriteFile(filepath.Join(public, compositionFile), original.Composition, 0644); err != nil {
				return err
			}
			rows = append(rows, row)
			replayRaw, replayResources, err := invoke(compiler, public, private, stem+"-replay", "body-compose", "--source", input, "--cases", "cases.json", "--composition", compositionFile)
			if err != nil {
				return err
			}
			replayed, replay, err := readObservation(replayRaw, sha, mode, stage, true, replayResources)
			if err != nil {
				return fmt.Errorf("%s replay: %w", stem, err)
			}
			if !sameProgram(current, replay) {
				return fmt.Errorf("saved program differs")
			}
			if err := os.WriteFile(filepath.Join(public, stem+"-replay.json"), replayRaw, 0644); err != nil {
				return err
			}
			rows = append(rows, replayed)
			input = stem + "-checkpoint.gooo.fixture"
			if err := os.WriteFile(filepath.Join(public, input), []byte(current.Composition.Gooo), 0644); err != nil {
				return err
			}
		}
	}
	metrics := struct {
		Schema   string    `json:"schema"`
		Compiler string    `json:"compiler_source"`
		Scope    string    `json:"scope"`
		Rows     []summary `json:"rows"`
	}{
		"gooo/native-record-observation/v1", sha,
		"One six-activity graph with two nominal records, two required string fields each; six cases/36 named expectations; two modes, three fresh generations each, six saved replays; record bodies come from source, own model ranks Score only; fixed order/warm build cache; process resources include native build/children; repeated controls share the same finite fixture; no host CPU increase or accuracy gain estimate", rows,
	}
	raw, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(public, "metrics.json"), append(raw, '\n'), 0644)
}

func sameProgram(a, b envelope) bool {
	return a.Composition.Gooo == b.Composition.Gooo && a.Composition.GoSHA == b.Composition.GoSHA && a.Composition.DriverSHA == b.Composition.DriverSHA
}

var cpuLine = regexp.MustCompile(`([0-9.]+) real\s+([0-9.]+) user\s+([0-9.]+) sys`)
var rssLine = regexp.MustCompile(`([0-9]+)\s+maximum resident set size`)

func invoke(compiler, public, private, stem string, args ...string) ([]byte, resources, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/usr/bin/time", append([]string{"-l", compiler}, args...)...)
	cmd.Dir = public
	var output, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	start := time.Now()
	err := cmd.Run()
	r := resources{WallMS: float64(time.Since(start).Nanoseconds()) / 1e6}
	if writeErr := os.WriteFile(filepath.Join(private, stem+".log"), diagnostics.Bytes(), 0600); writeErr != nil {
		return nil, r, writeErr
	}
	if err != nil {
		return nil, r, fmt.Errorf("%s: %w", stem, err)
	}
	clock, rss := cpuLine.FindStringSubmatch(diagnostics.String()), rssLine.FindStringSubmatch(diagnostics.String())
	if len(clock) != 4 || len(rss) != 2 {
		return nil, r, fmt.Errorf("resource observations missing")
	}
	r.UserS, err = strconv.ParseFloat(clock[2], 64)
	if err != nil {
		return nil, r, err
	}
	r.SystemS, err = strconv.ParseFloat(clock[3], 64)
	if err != nil {
		return nil, r, err
	}
	r.MaxRSS, err = strconv.ParseInt(rss[1], 10, 64)
	return output.Bytes(), r, err
}
