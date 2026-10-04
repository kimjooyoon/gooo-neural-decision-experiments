package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func capture(compiler, sha, source, cases, model, out, private string) error {
	if runtime.GOOS != "darwin" || len(sha) != 40 || out == "" || private == "" || !filepath.IsAbs(model) {
		return fmt.Errorf("macOS resources, exact compiler SHA, absolute model, fresh public/private directories required")
	}
	for _, dir := range []string{out, private} {
		if err := os.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	body, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	caseBytes, err := os.ReadFile(cases)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(out, "cases.json"), caseBytes, 0644); err != nil {
		return err
	}
	var rows []measurement
	var fullGo string
	for _, budget := range []int{1, 2, 4, 8} {
		if strings.Count(string(body), `attempts "8"`) != 1 {
			return fmt.Errorf("one attempts 8 baseline required")
		}
		input := fmt.Sprintf("source-budget-%d.gooo.fixture", budget)
		current := strings.Replace(string(body), `attempts "8"`, fmt.Sprintf(`attempts "%d"`, budget), 1)
		if err = os.WriteFile(filepath.Join(out, input), []byte(current), 0644); err != nil {
			return err
		}
		for trial := range 3 {
			modes := []string{"deterministic", "model"}
			if trial%2 == 1 {
				modes[0], modes[1] = modes[1], modes[0]
			}
			for _, mode := range modes {
				stem := fmt.Sprintf("budget-%d-%s-%d", budget, mode, trial)
				args := []string{"body-compose", "--source", input, "--cases", "cases.json"}
				if mode == "model" {
					args = append(args, "--model", model)
				}
				raw, resources, err := invoke(compiler, out, private, stem, args...)
				if err != nil {
					return err
				}
				row, observation, err := measure(raw, sha, false)
				if err != nil {
					return fmt.Errorf("%s: %w", stem, err)
				}
				if row.Mode != mode || row.Attempts > budget {
					return fmt.Errorf("mode or source budget differs")
				}
				trialIndex := trial
				row.Budget, row.Trial, row.Resources = budget, &trialIndex, &resources
				if err = os.WriteFile(filepath.Join(out, stem+".json"), raw, 0644); err != nil {
					return err
				}
				rows = append(rows, row)
				if budget != 8 {
					continue
				}
				if fullGo != "" && fullGo != row.GoSHA {
					return fmt.Errorf("complete generated program differs across fresh controls")
				}
				fullGo = row.GoSHA
				var envelope struct {
					Composition json.RawMessage `json:"composition"`
				}
				if err = json.Unmarshal(raw, &envelope); err != nil {
					return err
				}
				saved := stem + "-composition.json"
				if err = os.WriteFile(filepath.Join(out, saved), envelope.Composition, 0644); err != nil {
					return err
				}
				replayed, cost, err := invoke(compiler, out, private, stem+"-replay", "body-compose", "--source", input, "--cases", "cases.json", "--composition", saved)
				if err != nil {
					return err
				}
				replay, _, err := measure(replayed, sha, true)
				if err != nil || replay.GoSHA != row.GoSHA || replay.SourceSHA != row.SourceSHA {
					return fmt.Errorf("saved replay differs: %v", err)
				}
				replay.Budget, replay.Trial, replay.Resources = budget, &trialIndex, &cost
				rows = append(rows, replay)
				if err = os.WriteFile(filepath.Join(out, stem+"-replay.json"), replayed, 0644); err != nil {
					return err
				}
				if err = os.WriteFile(filepath.Join(out, stem+"-checkpoint.gooo.fixture"), []byte(observation.Composition.Gooo), 0644); err != nil {
					return err
				}
			}
		}
	}
	result := struct {
		Schema   string        `json:"schema"`
		Compiler string        `json:"compiler_source"`
		Scope    string        `json:"scope"`
		Rows     []measurement `json:"rows"`
	}{
		"gooo/record-field-budget-observation/v1", sha,
		"One three-field/two-node fixture; five selection cases and seven runtime cases with five overlapping inputs; budgets 1/2/4/8, three paired fresh trials per mode, six saved full-budget controls; alternating mode order after warm cache; frozen integer-model ordinal transfer, no new training; resources include native builds/children; repeated observations are controls rather than independent experiments; host CPU change and broad model quality remain unmeasured", rows,
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(out, "metrics.json"), append(raw, '\n'), 0644)
}
