package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const incrementalRun = "runs/incremental-native-20261001"

var incrementalPins = map[string]string{
	"report.json":       "7dc697b7036cc510e30db1528b29410a4bd8cf889a8aec12b0cf67898d10a144",
	"preexecution.json": "f7ed0704a1254a68b5ad42ceaf3d93751909c090b2d42ddc898e8931dedb67e2",
	"summary.json":      "2ad0aa31f0644d0c6666b3b10319d1dce75cd238f4e2b1472cd95c599808dea9",
}

func incrementalSources() map[string]string {
	files := preparedSources()
	for name := range incrementalPins {
		files["native-incremental/"+name] = incrementalRun + "/" + name
	}
	return files
}

type incrementalObservation struct {
	Budget    int    `json:"total_budget"`
	ReplySHA  string `json:"stdout_sha256"`
	SourceSHA string `json:"go_source_sha256"`
	Calls     int    `json:"local_model_predictions"`
	Attempts  int    `json:"attempted_candidates"`
	Passed    int    `json:"selected_finite_passed"`
	Cases     int    `json:"finite_cases"`
}
type incrementalReport struct {
	Status      string `json:"status"`
	Calls       int    `json:"native_calls"`
	Predictions int    `json:"fresh_local_model_predictions"`
	Cells       []struct {
		ID       string                     `json:"id"`
		Arm      string                     `json:"arm"`
		Language string                     `json:"language"`
		Contract string                     `json:"contract"`
		Same     bool                       `json:"same_final_body_and_search"`
		Prefixes int                        `json:"matched_progress_prefixes"`
		Restart  []incrementalObservation   `json:"restart_observations"`
		Batch    incrementalObservation     `json:"batched_observation"`
		Progress []pathplan.SessionProgress `json:"session_progress"`
	} `json:"cells"`
}

func auditIncrementalDirectory(root, cohort, captures string) error {
	for name, pin := range incrementalPins {
		raw, err := read(filepath.Join(root, name))
		if err != nil || digest(raw) != pin {
			return errors.New("frozen incremental evidence digest mismatch")
		}
	}
	raw, err := read(filepath.Join(root, "report.json"))
	if err != nil {
		return err
	}
	var report incrementalReport
	if json.Unmarshal(raw, &report) != nil || report.Status != "PASS" || len(report.Cells) != 32 || report.Calls != 288 || report.Predictions != 1296 {
		return errors.New("incomplete incremental report")
	}
	seen := map[string]bool{}
	calls, predictions, attempts, passed, prefixes := 0, 0, 0, 0, 0
	for _, c := range report.Cells {
		if seen[c.ID] || !c.Same || len(c.Restart) != 8 || len(c.Progress) < 2 {
			return errors.New("invalid incremental cell")
		}
		seen[c.ID] = true
		raw, err := read(filepath.Join(cohort, c.Language+"-budget-64.json"))
		if err != nil {
			return err
		}
		var document struct {
			Plan  pathplan.Plan       `json:"path_plan"`
			Cases []pathplan.TestCase `json:"test_cases"`
		}
		if json.Unmarshal(raw, &document) != nil || len(document.Cases) != 7 {
			return errors.New("finite cohort mismatch")
		}
		if c.Contract == "inconsistent" {
			document.Cases[6].Expected = 999
		} else if c.Contract != "complete" {
			return errors.New("undeclared contract")
		}
		prepared, err := pathplan.Prepare(document.Plan)
		if err != nil {
			return err
		}
		targetCalls := 6
		if c.Arm == "offline" {
			targetCalls = 0
		}
		previous := ""
		var visited [64]bool
		cumulative, evaluated, rejected, best := 0, 0, 0, -1
		for i, p := range c.Progress {
			if p.Sequence != i+1 || p.PreviousSHA != previous || p.PredictionsThisAdvance != 0 ||
				p.Selection.ModelCalls != targetCalls || p.Selection.ExternalCalls != 0 || !p.Selection.ExternalCallsKnown ||
				p.Declared != 64 || p.Cases != 7 || len(p.NewAttempts) > 8 || p.ScheduledBytes != 8 || !p.Initialized {
				return errors.New("incremental progress accounting mismatch")
			}
			stored := p.SHA
			p.SHA = ""
			raw, err := json.Marshal(p)
			if err != nil || digest(raw) != stored {
				return errors.New("incremental observation hash mismatch")
			}
			previous = stored
			for _, a := range p.NewAttempts {
				if a.Mask >= 64 || visited[a.Mask] || a.Total != 7 {
					return errors.New("repeated or invalid finite mask")
				}
				visited[a.Mask] = true
				cumulative++
				for j, d := range document.Plan.Decisions {
					if a.Choices[d.ID] != d.Options[int(a.Mask>>j&1)].Label {
						return errors.New("mask differs from choices")
					}
				}
				body, compileErr := prepared.Compile(a.Choices)
				if compileErr != nil {
					if a.Status != "TYPE_REJECTED" || a.Passed != 0 || len(a.Results) != 0 {
						return errors.New("invalid type rejection")
					}
					rejected++
					continue
				}
				if a.Status != "EVALUATED" || len(a.Results) != 7 || a.GoooSHA != digest([]byte(body.GoooSource())) {
					return errors.New("typed body digest mismatch")
				}
				evaluated++
				matched := 0
				for j, test := range document.Cases {
					value, err := body.Evaluate(test.Input)
					if err != nil {
						return err
					}
					ok := value.Int == test.Expected
					if ok {
						matched++
					}
					expected := pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: value.Int, Passed: ok}
					if a.Results[j] != expected {
						return errors.New("finite arena observation mismatch")
					}
				}
				if a.Passed != matched {
					return errors.New("inflated finite pass count")
				}
				best = max(best, matched)
			}
			if p.Attempted != cumulative || p.Unattempted != 64-cumulative || p.Evaluated != evaluated || p.TypeRejected != rejected || p.SelectedPassed != max(best, 0) {
				return errors.New("cumulative finite denominator mismatch")
			}
			if i == 0 && cumulative != 0 {
				return errors.New("tests preceded initial observation")
			}
		}
		last := c.Progress[len(c.Progress)-1]
		if c.Batch.Calls != targetCalls || c.Batch.Attempts != cumulative || c.Batch.Passed != last.SelectedPassed || c.Batch.Cases != 7 || c.Prefixes != len(c.Progress)-1 {
			return errors.New("final native observation mismatch")
		}
		if c.Contract == "inconsistent" && (last.Status != "PARTIAL" || last.SelectedPassed != 6 || cumulative != 64) {
			return errors.New("partial contract was relabeled")
		}
		if c.Contract == "complete" && (last.Status != "TRAINING_COMPLETE" || last.SelectedPassed != 7) {
			return errors.New("complete finite contract mismatch")
		}
		for i, o := range c.Restart {
			if o.Budget != (i+1)*8 || o.Calls != targetCalls || o.Attempts > o.Budget || o.Cases != 7 {
				return errors.New("invalid restart accounting")
			}
			if captures != "" {
				if err := auditIncrementalCapture(filepath.Join(captures, fmt.Sprintf("%s-restart-%d.json", c.ID, o.Budget)), o, nil); err != nil {
					return err
				}
			}
			calls++
			predictions += o.Calls
		}
		if captures != "" {
			if err := auditIncrementalCapture(filepath.Join(captures, c.ID+"-batch-64.json"), c.Batch, c.Progress); err != nil {
				return err
			}
		}
		calls++
		predictions += c.Batch.Calls
		attempts += cumulative
		passed += last.SelectedPassed
		prefixes += c.Prefixes
	}
	if calls != 288 || predictions != 1296 || attempts != 1268 || passed != 208 || prefixes != 166 {
		return errors.New("incremental totals mismatch")
	}
	return nil
}
func auditIncrementalCapture(path string, o incrementalObservation, progress []pathplan.SessionProgress) error {
	raw, err := read(path)
	if err != nil || digest(raw) != o.ReplySHA {
		return errors.New("native capture digest mismatch")
	}
	var capture struct {
		Source string `json:"source"`
		Report struct {
			Decision  string `json:"decision"`
			Compiler  string `json:"compiler_source_sha"`
			Typecheck bool   `json:"typecheck_passed"`
			Replay    bool   `json:"deterministic_replay"`
			Writes    int    `json:"repository_writes"`
			Paths     struct {
				Matched  bool                       `json:"source_base_matched"`
				Search   pathplan.SearchResult      `json:"search"`
				Progress []pathplan.SessionProgress `json:"session_progress"`
			} `json:"body_paths"`
		} `json:"report"`
	}
	if json.Unmarshal(raw, &capture) != nil || digest([]byte(capture.Source)) != o.SourceSHA || capture.Report.Decision != "PASS" ||
		capture.Report.Compiler != "f6323846f18602c2cf3cef5a40692340cae9fe3d" || !capture.Report.Typecheck ||
		!capture.Report.Replay || capture.Report.Writes != 0 || !capture.Report.Paths.Matched {
		return errors.New("native verified boundary mismatch")
	}
	s := capture.Report.Paths.Search
	if len(s.Attempts) != o.Attempts || s.SelectedTrainingPassed != o.Passed || s.TrainingTotal != o.Cases || s.Selection.ModelCalls != o.Calls || !reflect.DeepEqual(progress, capture.Report.Paths.Progress) {
		return errors.New("raw native receipt mismatch")
	}
	return nil
}
