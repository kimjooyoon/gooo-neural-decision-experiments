package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type captureRow struct {
	Profile     string      `json:"profile"`
	Trial       int         `json:"trial"`
	Workflow    string      `json:"workflow"`
	Measurement measurement `json:"measurement"`
	Resources   resources   `json:"resources"`
}

func capture(compiler, sha, goBin, source, series, models, out, private string) error {
	if runtime.GOOS != "darwin" || len(sha) != 40 || !filepath.IsAbs(compiler) || !filepath.IsAbs(goBin) || !filepath.IsAbs(models) || out == "" || private == "" {
		return fmt.Errorf("macOS, exact revision, absolute compiler/Go/model paths and fresh directories required")
	}
	for _, dir := range []string{out, private} {
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	for _, file := range []struct{ from, to string }{{source, "source.gooo.fixture"}, {series, "case-series.json"}} {
		raw, err := os.ReadFile(file.from)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(out, file.to), raw, 0644); err != nil {
			return err
		}
	}
	var rows []captureRow
	var program string
	for trial := range 3 {
		profiles := []string{"deterministic", "fp32"}
		if trial%2 == 1 {
			profiles[0], profiles[1] = profiles[1], profiles[0]
		}
		if trial == 0 {
			profiles = append(profiles, "ptq_ternary", "qat_ternary")
		}
		for _, profile := range profiles {
			stem := fmt.Sprintf("%s-%d", profile, trial)
			args := []string{"body-compose", "--source", "source.gooo.fixture", "--case-series", "case-series.json", "--repeat", "2", "--go-bin", goBin}
			if profile != "deterministic" {
				args = append(args, "--model", filepath.Join(models, profile, "model.json"))
			}
			raw, cost, err := invoke(compiler, out, private, stem+"-fresh", args...)
			if err != nil {
				return err
			}
			m, err := measure(raw, sha, false)
			if err != nil {
				return fmt.Errorf("%s: %w", stem, err)
			}
			if m.Profile != profile {
				return fmt.Errorf("captured model profile differs")
			}
			if program != "" && program != m.GoSHA {
				return fmt.Errorf("full-budget program differs across profiles")
			}
			program = m.GoSHA
			if err = os.WriteFile(filepath.Join(out, stem+"-fresh.json"), raw, 0644); err != nil {
				return err
			}
			rows = append(rows, captureRow{profile, trial, "fresh-retained", m, cost})
			var original envelope
			if err = json.Unmarshal(raw, &original); err != nil {
				return err
			}
			if profile == "ptq_ternary" || profile == "qat_ternary" {
				continue
			}
			var saved struct {
				Composition json.RawMessage `json:"composition"`
			}
			if err = json.Unmarshal(raw, &saved); err != nil {
				return err
			}
			checkpoint := stem + "-composition.json"
			if err = os.WriteFile(filepath.Join(out, checkpoint), saved.Composition, 0644); err != nil {
				return err
			}
			for i, suite := range original.Series.Suites {
				data, _ := json.Marshal(suite)
				if err = os.WriteFile(filepath.Join(out, fmt.Sprintf("suite-%d.json", i)), data, 0644); err != nil {
					return err
				}
			}
			workflows := []string{"stateless", "saved-retained"}
			if trial%2 == 1 {
				workflows[0], workflows[1] = workflows[1], workflows[0]
			}
			for _, workflow := range workflows {
				current := captureRow{profile, trial, workflow, m, resources{}}
				current.Measurement.Saved, current.Measurement.Calls = true, 0
				current.Measurement.Frames = nil
				if workflow == "saved-retained" {
					args = []string{"body-compose", "--source", "source.gooo.fixture", "--case-series", "case-series.json", "--repeat", "2", "--composition", checkpoint, "--go-bin", goBin}
					raw, cost, err = invoke(compiler, out, private, stem+"-saved", args...)
					if err != nil {
						return err
					}
					current.Measurement, err = measure(raw, sha, true)
					if err != nil {
						return err
					}
					current.Resources = cost
					var replay envelope
					if err = json.Unmarshal(raw, &replay); err != nil {
						return err
					}
					for i := range replay.History {
						if err = sameValues(original.History[i], replay.History[i]); err != nil {
							return err
						}
					}
					if err = os.WriteFile(filepath.Join(out, stem+"-saved.json"), raw, 0644); err != nil {
						return err
					}
				} else {
					for i := range 6 {
						name := fmt.Sprintf("%s-stateless-%d", stem, i)
						args = []string{"body-compose", "--source", "source.gooo.fixture", "--cases", fmt.Sprintf("suite-%d.json", i%3), "--composition", checkpoint, "--go-bin", goBin}
						raw, cost, err = invoke(compiler, out, private, name, args...)
						if err != nil {
							return err
						}
						var single envelope
						if err = json.Unmarshal(raw, &single); err != nil {
							return err
						}
						if single.Generated || len(single.History) != 0 || single.Runtime.Artifact != nil || !complete(single.Runtime.Build) {
							return fmt.Errorf("stateless build observation differs")
						}
						if err = sameValues(original.History[i], single.Runtime); err != nil {
							return err
						}
						frame, err := measureFrame(single.Runtime, original.Series.Suites[i%3], sha)
						if err != nil {
							return err
						}
						current.Measurement.Frames = append(current.Measurement.Frames, frame)
						current.Resources.WallMS += cost.WallMS
						current.Resources.UserS += cost.UserS
						current.Resources.SystemS += cost.SystemS
						if cost.MaxRSS > current.Resources.MaxRSS {
							current.Resources.MaxRSS = cost.MaxRSS
						}
						if err = os.WriteFile(filepath.Join(out, name+".json"), raw, 0644); err != nil {
							return err
						}
					}
				}
				rows = append(rows, current)
			}
		}
	}
	result := struct {
		Schema   string       `json:"schema"`
		Compiler string       `json:"compiler_source"`
		Scope    string       `json:"scope"`
		Rows     []captureRow `json:"rows"`
	}{"gooo/composition-series-observation/v1", sha,
		"One authored two-node/three-field source; three ordered suites repeated twice per workload. Fresh construction calls the frozen model at most once. Paired saved workloads make zero new predictions: one retained CLI command versus six sequential stateless CLI commands (startup count differs); three paired trials per deterministic/FP32 mode, alternating workflow order. One fresh ternary pilot per profile. Current inputs, actual values and deliberate partial expectations are independently checked. CPU/RSS include builds and children; whole-host utilization, new training and broad model quality are unmeasured.", rows}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(data, '\n'), 0644)
}

func sameValues(a, b frame) error {
	x, _ := json.Marshal(a.Traces)
	y, _ := json.Marshal(b.Traces)
	if !equalJSON(x, y) || a.GoSHA != b.GoSHA || a.DriverSHA != b.DriverSHA || a.SuiteSHA != b.SuiteSHA || a.ExecutableSHA != b.ExecutableSHA {
		return fmt.Errorf("fresh current values or program identities differ")
	}
	return nil
}
