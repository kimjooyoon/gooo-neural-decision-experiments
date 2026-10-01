package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/compoundstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const budgetNative = "37fb287a9c888bd0262194f524dde16426eca3a9"
const budgetPreregistration = "docs/native-budget-preregistration.md"
const budgetChunk = 12

type budgetPre struct {
	Schema          string          `json:"schema"`
	Runner          string          `json:"runner_revision"`
	Native          string          `json:"native_revision"`
	Worker          string          `json:"worker_binary_sha256"`
	Go              string          `json:"go_binary_sha256"`
	Preregistration string          `json:"preregistration_sha256"`
	Cohort          string          `json:"cohort_sha256"`
	Hint            pathplan.CIHint `json:"caller_ci_hint"`
	Scope           string          `json:"scope"`
}

func budgetArmID(arm familyArm) string {
	if arm.Path == "" {
		return "offline"
	}
	mode := "initial"
	if arm.Feedback {
		mode = "feedback"
	}
	return arm.Name + "-" + mode
}

func budgetOrder(index int) []int {
	if index%2 == 0 {
		return []int{1, 2, 4}
	}
	return []int{4, 2, 1}
}

func budgetRows(index int) ([]compoundstudy.Case, error) {
	rows, err := compoundRows()
	if err != nil || len(rows) != 72 {
		return nil, errors.New("frozen 72-view cohort required")
	}
	if index%2 != 0 {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	return rows, nil
}

func budgetWire(row compoundstudy.Case, arm familyArm, budget int, ci pathplan.CIHint, id string, bad bool) ([]byte, error) {
	if budget != 1 && budget != 2 && budget != 4 {
		return nil, errors.New("budget must be 1, 2 or 4")
	}
	row.Document.Max = budget
	return retainedWire(row, arm, ci, id, bad)
}

func budgetBlockID(arm familyArm, budget, chunk int) string {
	return fmt.Sprintf("%s-budget-%d-chunk-%d", budgetArmID(arm), budget, chunk)
}

func runBudgetStudy(worker, goBinary, output, revision, native string) error {
	if native != budgetNative {
		return errors.New("declared main source required")
	}
	if err := pinnedRetainedBinary(worker, native); err != nil {
		return err
	}
	if _, _, err := familyPreflightFor(worker, goBinary, output, revision,
		familySpec{Revision: native, SDK: "v0.2.7-experimental", CIStatus: "UNKNOWN"}); err != nil {
		return err
	}
	inputs := []string{"internal/compoundstudy", "internal/pathplan", "internal/bodyplan", "internal/decision", "studies/compound-path-v1", budgetPreregistration}
	if err := exec.Command("git", append([]string{"diff", "--quiet", "HEAD", "--"}, inputs...)...).Run(); err != nil {
		return errors.New("budget input or oracle source dirty")
	}
	untracked, err := exec.Command("git", append([]string{"ls-files", "--others", "--exclude-standard", "--"}, inputs...)...).Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("budget input or oracle source untracked")
	}
	arms, err := compoundArms()
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
	prereg, err := read(budgetPreregistration)
	if err != nil {
		return err
	}
	cohort, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil {
		return err
	}
	pre := budgetPre{Schema: "gooo/native-budget-preexecution/v1", Runner: revision, Native: native, Worker: workerSHA, Go: goSHA,
		Preregistration: hash(prereg), Cohort: hash(cohort), Hint: pathplan.CIHint{SourceSHA: native, Status: "UNKNOWN"},
		Scope: "72 reused contract/language views over 12 existing intention groups and three templates; nine disconnected/initial/feedback arms, budgets 1/2/4, one worker, 12 valid requests plus one zero-prediction source rejection per process. 1944 valid constructions, 162 rejections, 162 native processes. Budget and row order reverse for odd arms. No separate-input or intended-mask information is sent to selection. One-time setup excluded from request round-trip, included in whole-child CPU/RSS/wall. Original unauthenticated UNKNOWN CI context is retained. No new training, GPU or upstream Laya."}
	for _, dir := range []string{"captures", "metrics", "executions"} {
		if err = os.MkdirAll(filepath.Join(output, dir), 0755); err != nil {
			return err
		}
	}
	if err = save(filepath.Join(output, "preexecution.json"), pre); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	for a, arm := range arms {
		rows, err := budgetRows(a)
		if err != nil {
			return err
		}
		model := ""
		if arm.Path != "" {
			model, err = filepath.Abs(arm.Path)
			if err != nil {
				return err
			}
		}
		for _, budget := range budgetOrder(a) {
			for chunk := 0; chunk < len(rows)/budgetChunk; chunk++ {
				group := rows[chunk*budgetChunk : (chunk+1)*budgetChunk]
				bad, err := budgetWire(group[0], arm, budget, pre.Hint, "bad-source", true)
				if err != nil {
					return err
				}
				lines := [][]byte{bad}
				for _, row := range group {
					raw, err := budgetWire(row, arm, budget, pre.Hint, row.ID, false)
					if err != nil {
						return err
					}
					lines = append(lines, raw)
				}
				id := budgetBlockID(arm, budget, chunk)
				raw, m, failure := retainedChild(ctx, worker, model, 1, lines)
				if err = os.WriteFile(filepath.Join(output, "captures", id+".jsonl"), raw, 0644); err != nil {
					return err
				}
				if err = save(filepath.Join(output, "metrics", id+".json"), m); err != nil {
					return err
				}
				if failure != nil {
					return fmt.Errorf("%s: %w", id, failure)
				}
			}
		}
	}
	collection, err := collectBudget(output, revision, native)
	if err != nil {
		return err
	}
	var keys []string
	for sha := range collection.sources {
		keys = append(keys, sha)
	}
	sort.Strings(keys)
	for _, sha := range keys {
		raw, err := executeFunction(ctx, goBinary, collection.sources[sha], "ComposePaths", collection.inputs)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(output, "executions", sha+".json"), raw, 0644); err != nil {
			return err
		}
	}
	report, err := auditBudget(output, revision, native)
	if err != nil {
		return err
	}
	if err = save(filepath.Join(output, "report.json"), report); err != nil {
		return err
	}
	return save(filepath.Join(output, "audit.json"), report)
}

func decodeBudgetPre(raw []byte, revision, native string) (budgetPre, error) {
	var pre budgetPre
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return pre, errors.New("explicit runner revision required")
	}
	if json.Unmarshal(raw, &pre) != nil || pre.Schema != "gooo/native-budget-preexecution/v1" || pre.Runner != revision ||
		pre.Native != native || native != budgetNative || pre.Hint != (pathplan.CIHint{SourceSHA: native, Status: "UNKNOWN"}) {
		return pre, errors.New("budget preexecution source tuple differs")
	}
	sha64 := regexp.MustCompile(`^[a-f0-9]{64}$`)
	if !sha64.MatchString(pre.Worker) || !sha64.MatchString(pre.Go) {
		return pre, errors.New("binary digests required")
	}
	prereg, err := read(budgetPreregistration)
	if err != nil || hash(prereg) != pre.Preregistration {
		return pre, errors.New("budget preregistration differs")
	}
	cohort, err := read(compoundRoot + "/cohort.jsonl")
	if err != nil || hash(cohort) != pre.Cohort {
		return pre, errors.New("budget cohort differs")
	}
	return pre, nil
}
