package main

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func fmtInt(value int) string { return strconv.Itoa(value) }

type retainedRequest struct {
	Schema   string                 `json:"schema"`
	ID       string                 `json:"correlation_id"`
	Source   string                 `json:"source"`
	Activity string                 `json:"activity"`
	Document compoundstudy.Document `json:"document"`
	Options  struct {
		Step    int              `json:"step_attempts"`
		Rounds  int              `json:"feedback_rounds,omitempty"`
		Unfixed bool             `json:"feedback_unfixed,omitempty"`
		CI      *pathplan.CIHint `json:"ci,omitempty"`
	} `json:"options"`
}

type retainedPre struct {
	Schema string          `json:"schema"`
	Runner string          `json:"runner_revision"`
	Native string          `json:"native_revision"`
	CLI    string          `json:"cli_binary_sha256"`
	Worker string          `json:"worker_binary_sha256"`
	Go     string          `json:"go_binary_sha256"`
	Hint   pathplan.CIHint `json:"caller_ci_hint"`
	Scope  string          `json:"scope"`
}

func retainedArms() ([]familyArm, error) {
	all, err := compoundArms()
	if err != nil {
		return nil, err
	}
	var arms []familyArm
	for _, arm := range all {
		if arm.Path == "" || arm.Feedback {
			arms = append(arms, arm)
		}
	}
	if len(arms) != 5 {
		return nil, errors.New("five frozen arms required")
	}
	return arms, nil
}

func retainedRows() ([]compoundstudy.Case, error) {
	rows, err := nativeUnfixedRows()
	if err != nil {
		return nil, err
	}
	for i := len(rows) - 1; i >= 0; i-- {
		rows = append(rows, rows[i])
	}
	return rows, nil
}

func retainedWire(row compoundstudy.Case, arm familyArm, ci pathplan.CIHint, id string, bad bool) ([]byte, error) {
	r := retainedRequest{Schema: "gooo/native-body-stream-request/v1", ID: id, Source: row.Source,
		Activity: "ComposePaths", Document: row.Document}
	r.Options.Step = 1
	if arm.Feedback {
		r.Options.Rounds = 3
		r.Options.Unfixed = true
		r.Options.CI = &ci
	}
	if bad {
		r.Source = "invalid source"
	}
	return json.Marshal(r)
}

func pinnedRetainedBinary(binary, native string) error {
	i, err := buildinfo.ReadFile(binary)
	if err != nil || i.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 native worker required")
	}
	settings := map[string]string{}
	for _, s := range i.Settings {
		settings[s.Key] = s.Value
	}
	if settings["vcs.revision"] != native || settings["vcs.modified"] != "false" {
		return errors.New("clean worker source required")
	}
	for _, dep := range i.Deps {
		if dep.Path == "github.com/kimjooyoon/gooo-decision-runtime" && dep.Version == "v0.2.7-experimental" {
			return nil
		}
	}
	return errors.New("worker SDK differs")
}

func runRetainedPilot(cli, worker, goBinary, output, revision, native string) error {
	if err := pinnedRetainedBinary(worker, native); err != nil {
		return err
	}
	_, _, err := familyPreflightFor(cli, goBinary, output, revision,
		familySpec{Revision: native, SDK: "v0.2.7-experimental", CIStatus: "UNKNOWN"})
	if err != nil {
		return err
	}
	arms, err := retainedArms()
	if err != nil {
		return err
	}
	rows, err := retainedRows()
	if err != nil {
		return err
	}
	cliSHA, err := executableHash(cli)
	if err != nil {
		return err
	}
	workerSHA, err := executableHash(worker)
	if err != nil {
		return err
	}
	goSHA, err := executableHash(goBinary)
	if err != nil {
		return err
	}
	pre := retainedPre{Schema: "gooo/retained-native-preexecution/v1", Runner: revision, Native: native,
		CLI: cliSHA, Worker: workerSHA, Go: goSHA, Hint: pathplan.CIHint{SourceSHA: native, Status: "UNKNOWN"},
		Scope: "Six reused bilingual contradictory views, forward then reverse; five frozen own-model/disconnected arms. Per arm: 12 fresh CLI processes, 12 sequential retained requests and 12 parallel-4 retained requests; each worker first rejects one invalid source. 180 valid constructions, 10 source rejections, 70 native processes. Order fresh/retained1/retained4 reverses for odd arm indices. Setup excluded from retained round-trip, included in whole-worker metrics. Parallel arrival latency includes queueing; not a one-shot language benchmark. Zero training/GPU/upstream Laya. CI UNKNOWN is unauthenticated caller context, not authority."}
	for _, dir := range []string{"captures", "metrics", "executions"} {
		if err = os.MkdirAll(filepath.Join(output, dir), 0755); err != nil {
			return err
		}
	}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	temp, err := os.MkdirTemp("", "gooo-retained-native-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	if err = save(filepath.Join(temp, "hint.json"), pre.Hint); err != nil {
		return err
	}
	for a, arm := range arms {
		modes := []int{0, 1, 4}
		if a%2 == 1 {
			modes = []int{4, 1, 0}
		}
		model := ""
		if arm.Path != "" {
			model, err = filepath.Abs(arm.Path)
			if err != nil {
				return err
			}
		}
		for _, workers := range modes {
			if workers == 0 {
				for i, row := range rows {
					id := fmt.Sprintf("%s-fresh-%02d", arm.Name, i)
					if err = os.WriteFile(filepath.Join(temp, "source.gooo"), []byte(row.Source), 0600); err != nil {
						return err
					}
					if err = save(filepath.Join(temp, "plan.json"), row.Document); err != nil {
						return err
					}
					args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "1", "--activity", "ComposePaths"}
					if arm.Feedback {
						args = append(args, "--path-model", model, "--path-feedback-rounds", "3", "--path-feedback-unfixed", "--path-feedback-ci", "hint.json")
					}
					args = append(args, "source.gooo")
					raw, m, failure := child(ctx, temp, cli, args...)
					if err = os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); err != nil {
						return err
					}
					if err = save(filepath.Join(output, "metrics", id+".json"), retainedProcess{Metrics: m}); err != nil {
						return err
					}
					if failure != nil {
						return failure
					}
				}
				continue
			}
			id := fmt.Sprintf("%s-retained-%d", arm.Name, workers)
			bad, err := retainedWire(rows[0], arm, pre.Hint, "bad-source", true)
			if err != nil {
				return err
			}
			lines := [][]byte{bad}
			for i, row := range rows {
				raw, err := retainedWire(row, arm, pre.Hint, fmt.Sprintf("request-%02d", i), false)
				if err != nil {
					return err
				}
				lines = append(lines, raw)
			}
			raw, m, failure := retainedChild(ctx, worker, model, workers, lines)
			if err = os.WriteFile(filepath.Join(output, "captures", id+".jsonl"), raw, 0644); err != nil {
				return err
			}
			if err = save(filepath.Join(output, "metrics", id+".json"), m); err != nil {
				return err
			}
			if failure != nil {
				return failure
			}
		}
	}
	collected, err := collectRetained(output, revision, native)
	if err != nil {
		return err
	}
	for sha, source := range collected.sources {
		raw, err := executeFunction(ctx, goBinary, source, "ComposePaths", collected.inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
	}
	value, err := auditRetained(output, revision, native)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "report.json"), value); err != nil {
		return err
	}
	return save(filepath.Join(output, "audit.json"), value)
}
