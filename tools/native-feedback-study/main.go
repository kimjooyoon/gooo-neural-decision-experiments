// native-feedback-study measures the actual optional Go compiler integration.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var modelPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}
type nativeResult struct {
	Source string `json:"source"`
	Report struct {
		Decision   string `json:"decision"`
		ActivityID string `json:"activity_id"`
		Compiler   string `json:"compiler_source_sha"`
		Types      bool   `json:"typecheck_passed"`
		Replay     bool   `json:"deterministic_replay"`
		Writes     int    `json:"repository_writes"`
		Paths      struct {
			Unfixed     bool                       `json:"feedback_unfixed,omitempty"`
			OriginalSHA string                     `json:"original_source_sha256"`
			DocumentSHA string                     `json:"document_sha256"`
			SuiteSHA    string                     `json:"test_suite_sha256"`
			Bound       bool                       `json:"source_base_matched"`
			Search      pathplan.SearchResult      `json:"search"`
			Progress    []pathplan.SessionProgress `json:"session_progress"`
			Feedback    []pathplan.FeedbackReceipt `json:"feedback_judgments"`
			Cases       []struct {
				Input, Expected, Actual int64
				Passed                  bool
			} `json:"native_case_results"`
			Completeness float64 `json:"finite_functional_completeness_percent"`
			Timing       struct {
				Total  float64 `json:"total_ms"`
				Search float64 `json:"bounded_search_ms"`
				Load   float64 `json:"model_load_ms"`
			} `json:"timing"`
		} `json:"body_paths"`
	} `json:"report"`
}
type metrics struct {
	Wall   int64   `json:"wall_ns"`
	User   int64   `json:"user_cpu_ns"`
	System int64   `json:"system_cpu_ns"`
	RSS    int64   `json:"child_max_rss_bytes"`
	CPU    float64 `json:"process_cpu_percent_of_one_core_over_wall"`
}
type observation struct {
	ID            string  `json:"id"`
	Arm           string  `json:"arm"`
	Language      string  `json:"language"`
	Contract      string  `json:"contract"`
	Feedback      bool    `json:"feedback_enabled"`
	Capture       string  `json:"capture_sha256"`
	GoSHA         string  `json:"generated_go_sha256"`
	DocumentSHA   string  `json:"document_sha256"`
	Calls         int     `json:"actual_local_predictions"`
	FeedbackCalls int     `json:"feedback_predictions"`
	Rounds        int     `json:"feedback_rounds"`
	Attempts      int     `json:"candidate_attempts"`
	Passed        int     `json:"finite_passed"`
	Cases         int     `json:"finite_cases"`
	Completeness  float64 `json:"finite_completeness_percent"`
	PredictionNS  int64   `json:"sum_observed_prediction_ns"`
	NativeMS      float64 `json:"native_total_ms"`
	SearchMS      float64 `json:"native_bounded_search_ms"`
	LoadMS        float64 `json:"native_model_load_ms"`
	Metrics       metrics `json:"process_metrics"`
}
type pair struct {
	Baseline     string `json:"baseline"`
	Feedback     string `json:"feedback"`
	EqualGo      bool   `json:"same_generated_go"`
	EqualPassed  bool   `json:"same_finite_pass_count"`
	AttemptDelta int    `json:"feedback_minus_baseline_attempts"`
}
type report struct {
	Schema        string        `json:"schema"`
	Runner        string        `json:"runner_revision"`
	Native        string        `json:"native_revision"`
	BinarySHA     string        `json:"native_binary_sha256"`
	Observations  []observation `json:"observations"`
	Pairs         []pair        `json:"pairs"`
	Calls         int           `json:"actual_local_predictions"`
	FeedbackCalls int           `json:"feedback_predictions"`
	Attempts      int           `json:"candidate_attempts"`
	FinitePassed  int           `json:"repeated_policy_finite_passes"`
	FiniteCases   int           `json:"repeated_policy_finite_cases"`
	Scope         string        `json:"scope"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(raw []byte) (int, error) {
	if b.Len()+len(raw) > 2<<20 {
		return 0, errors.New("child output exceeded 2 MiB")
	}
	return b.Buffer.Write(raw)
}
func hash(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 8<<20 {
		return nil, errors.New("bounded regular input required")
	}
	return os.ReadFile(path)
}
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func child(ctx context.Context, dir, binary string, args ...string) ([]byte, metrics, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	configure(cmd)
	var out, stderr bounded
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	started := time.Now()
	err := cmd.Run()
	m := metrics{Wall: time.Since(started).Nanoseconds()}
	if cmd.ProcessState != nil {
		m.User = cmd.ProcessState.UserTime().Nanoseconds()
		m.System = cmd.ProcessState.SystemTime().Nanoseconds()
		m.RSS = maxRSS(cmd.ProcessState)
		m.CPU = 100 * float64(m.User+m.System) / float64(m.Wall)
	}
	if err != nil || stderr.Len() != 0 {
		return out.Bytes(), m, errors.New("bounded native child failed; capture retained")
	}
	return out.Bytes(), m, nil
}
func inspect(value nativeResult, feedback bool, modelCalls int, expectedRevision string) error {
	r := value.Report
	p := r.Paths
	if r.Decision != "PASS" || !r.Types || !r.Replay || r.Writes != 0 || !p.Bound || r.Compiler != expectedRevision ||
		len(p.Cases) != 7 || p.Search.TrainingTotal != 7 || p.Search.Evaluated > 64 || p.Search.Selection.ExternalCalls != 0 {
		return errors.New("native source, finite or verification contract mismatch")
	}
	seen := map[uint16]bool{}
	previous := ""
	attempts := 0
	calls := modelCalls
	progresses := map[string]pathplan.SessionProgress{}
	for i, progress := range p.Progress {
		if progress.Sequence != i+1 || progress.PreviousSHA != previous || progress.SHA == "" || progress.Selection.ModelCalls < calls ||
			progress.PredictionsThisAdvance != 0 || len(progress.NewAttempts) > 8 {
			return errors.New("invalid native progress")
		}
		previous = progress.SHA
		calls = progress.Selection.ModelCalls
		progresses[previous] = progress
		for _, attempt := range progress.NewAttempts {
			if seen[attempt.Mask] {
				return errors.New("native path repeated")
			}
			seen[attempt.Mask] = true
			attempts++
		}
		if progress.Attempted != attempts {
			return errors.New("native progress counters changed")
		}
	}
	previous = ""
	feedbackCalls := 0
	for i, judgment := range p.Feedback {
		prior, exists := progresses[judgment.FromProgressSHA]
		if !exists || prior.Attempted != judgment.Attempted || judgment.Round != i+1 || judgment.PreviousSHA != previous ||
			judgment.SHA == "" || !judgment.Applied || judgment.CIIsAuthority || judgment.ModelCalls != 6 || judgment.CI == nil {
			return errors.New("invalid native feedback binding")
		}
		previous = judgment.SHA
		feedbackCalls += judgment.ModelCalls
	}
	passed := 0
	for _, c := range p.Cases {
		if c.Passed != (c.Actual == c.Expected) {
			return errors.New("native finite outcome mismatch")
		}
		if c.Passed {
			passed++
		}
	}
	if attempts != p.Search.Evaluated+p.Search.TypeRejected || p.Search.Selection.ModelCalls != modelCalls+feedbackCalls ||
		passed != p.Search.SelectedTrainingPassed || p.Completeness != 100*float64(passed)/7 || len(p.Feedback) > 2 || (!feedback && len(p.Feedback) != 0) {
		return errors.New("native aggregate does not reconcile")
	}
	return nil
}
func run(binary, nativeRoot, output, revision, nativeRevision string) error {
	validSHA := regexp.MustCompile(`^[a-f0-9]{40}$`)
	if !validSHA.MatchString(revision) || !validSHA.MatchString(nativeRevision) || filepath.IsAbs(output) ||
		filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return errors.New("pinned revisions and fresh relative output required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	info, err := buildinfo.ReadFile(binary)
	if err != nil || info.GoVersion != "go1.27.1" {
		return errors.New("Go 1.27.1 native binary required")
	}
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	sdk := ""
	for _, d := range info.Deps {
		if d.Path == "github.com/kimjooyoon/gooo-decision-runtime" {
			sdk = d.Version
		}
	}
	if settings["vcs.revision"] != nativeRevision || settings["vcs.modified"] != "false" || sdk != "v0.2.3-experimental" {
		return errors.New("clean pinned native source and SDK required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner revision mismatch")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/native-feedback-study", "internal/pathplan").Run(); err != nil {
		return errors.New("runner sources dirty")
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "tools/native-feedback-study").Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("runner source untracked")
	}
	binaryInfo, err := os.Lstat(binary)
	if err != nil || !binaryInfo.Mode().IsRegular() || binaryInfo.Size() <= 0 || binaryInfo.Size() > 64<<20 {
		return errors.New("bounded native executable required")
	}
	binaryRaw, err := os.ReadFile(binary)
	if err != nil {
		return err
	}
	models := map[string]string{}
	weights := map[string]string{}
	for arm, pin := range modelPins {
		path, err := filepath.Abs(filepath.Join("runs/typed-path-positioned-random-20261001/models", arm, "model.json"))
		if err != nil {
			return err
		}
		model, err := decision.LoadPath(path)
		if err != nil || model.MetadataSHA256() != pin {
			return errors.New("frozen model mismatch")
		}
		models[arm] = path
		weights[arm] = model.WeightsSHA256()
	}
	source, err := read(filepath.Join(nativeRoot, "examples/body-codegen/typed-path-conditional-assignment.gooo.fixture"))
	if err != nil {
		return err
	}
	documents := map[string]document{}
	pins := map[string]string{"source": hash(source)}
	for _, lang := range []string{"en", "ko"} {
		name := "typed-path-conditional-assignment-plan.json"
		if lang == "ko" {
			name = "typed-path-conditional-assignment-ko-plan.json"
		}
		raw, err := read(filepath.Join(nativeRoot, "examples/body-codegen", name))
		if err != nil {
			return err
		}
		var d document
		if err = json.Unmarshal(raw, &d); err != nil || d.Schema != "gooo/body-codegen-typed-path-plan/v1" ||
			len(d.Cases) != 7 || len(d.Plan.Decisions) != 6 || d.Max != 64 {
			return errors.New("fixed native fixture required")
		}
		documents[lang] = d
		pins[lang] = hash(raw)
	}
	ciRaw, err := read("publication/incremental-typed-path-main-push-20261001.json")
	if err != nil {
		return err
	}
	var ciProof struct {
		SHA        string `json:"source_sha"`
		Conclusion string `json:"conclusion"`
		Decision   string `json:"proof_decision"`
	}
	if err = json.Unmarshal(ciRaw, &ciProof); err != nil || ciProof.Conclusion != "success" || ciProof.Decision != "PASS" || ciProof.SHA != "307159f041644a3aa56dfd325c345f5325aec902" {
		return errors.New("previous verified CI context required")
	}
	ci := pathplan.CIHint{SourceSHA: ciProof.SHA, Status: "PASS"}
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	value := report{Schema: "gooo/native-feedback-study/v1", Runner: revision, Native: nativeRevision, BinarySHA: hash(binaryRaw),
		Scope: "28 native calls, 12 model pairs and 4 deterministic controls reuse one compound intention, two languages and seven finite cases; not 28 independent tasks. Baseline-first single pass, no causal speed claim, host CPU utilization, new training, GPU or upstream Laya calls. Child max RSS and CPU time cover native processes only. This is a feature-source integration study; merged main verification is recorded separately."}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/native-feedback-preexecution/v1", "runner_revision": revision, "native_revision": nativeRevision, "native_binary_sha256": value.BinarySHA, "sdk": "v0.2.3-experimental", "model_metadata": modelPins, "model_weights": weights, "source_inputs": pins, "ci_hint": ci, "ci_hint_receipt_sha256": hash(ciRaw), "calls": 28, "pairs": 12, "step": 8, "candidate_budget": 64, "feedback_round_budget": 2, "optimizer_steps": 0}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp("", "gooo-native-feedback-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err = os.WriteFile(filepath.Join(dir, "fixture.gooo"), source, 0600); err != nil {
		return err
	}
	if err = save(filepath.Join(dir, "hint.json"), ci); err != nil {
		return err
	}
	for _, lang := range []string{"en", "ko"} {
		for _, contract := range []string{"complete", "inconsistent"} {
			for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
				var baseline observation
				for _, feedback := range []bool{false, true} {
					if arm == "offline" && feedback {
						continue
					}
					d := documents[lang]
					d.Cases = append([]pathplan.TestCase(nil), d.Cases...)
					if contract == "inconsistent" {
						d.Cases[6].Expected = 999
					}
					if err = save(filepath.Join(dir, "plan.json"), d); err != nil {
						return err
					}
					docRaw, err := read(filepath.Join(dir, "plan.json"))
					if err != nil {
						return err
					}
					mode := "baseline"
					if feedback {
						mode = "feedback"
					}
					id := lang + "-" + contract + "-" + arm + "-" + mode
					args := []string{"body-codegen", "--json", "--path-plan", "plan.json", "--path-step-attempts", "8", "--activity", "ConditionalAssign", "fixture.gooo"}
					initialCalls := 0
					if arm != "offline" {
						args = append(args, "--path-model", models[arm])
						initialCalls = 6
					}
					if feedback {
						args = append(args, "--path-feedback-rounds", "2", "--path-feedback-ci", "hint.json")
					}
					raw, m, failure := child(ctx, dir, binary, args...)
					if len(raw) > 0 {
						if err = os.WriteFile(filepath.Join(output, "captures", id+".json"), raw, 0644); err != nil {
							return err
						}
					}
					if failure != nil {
						return failure
					}
					var result nativeResult
					if err = json.Unmarshal(raw, &result); err != nil {
						return err
					}
					if err = inspect(result, feedback, initialCalls, nativeRevision); err != nil {
						return err
					}
					p := result.Report.Paths
					o := observation{ID: id, Arm: arm, Language: lang, Contract: contract, Feedback: feedback, Capture: hash(raw), GoSHA: hash([]byte(result.Source)), DocumentSHA: hash(docRaw), Calls: p.Search.Selection.ModelCalls, Rounds: len(p.Feedback), Attempts: len(p.Search.Attempts), Passed: p.Search.SelectedTrainingPassed, Cases: p.Search.TrainingTotal, Completeness: p.Completeness, NativeMS: p.Timing.Total, SearchMS: p.Timing.Search, LoadMS: p.Timing.Load, Metrics: m}
					for _, r := range p.Search.Selection.Receipts {
						o.PredictionNS += r.PredictNS
					}
					for _, f := range p.Feedback {
						o.FeedbackCalls += f.ModelCalls
						for _, j := range f.Judgments {
							o.PredictionNS += j.Prediction.PredictNS
						}
					}
					value.Observations = append(value.Observations, o)
					value.Calls += o.Calls
					value.FeedbackCalls += o.FeedbackCalls
					value.Attempts += o.Attempts
					value.FinitePassed += o.Passed
					value.FiniteCases += o.Cases
					if !feedback {
						baseline = o
					} else {
						value.Pairs = append(value.Pairs, pair{baseline.ID, o.ID, baseline.GoSHA == o.GoSHA, baseline.Passed == o.Passed, o.Attempts - baseline.Attempts})
					}
					if err = save(filepath.Join(output, "report.json"), value); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
func main() {
	mode := flag.String("mode", "study", "study, execute, audit, input-bound, continued-bound, continued-audit, trained-dogfood, trained-audit, family modes or compound-cohort/compound-study/compound-pilot")
	goBinary := flag.String("go-bin", "", "Go 1.27.1 executable for execute mode")
	auditOutput := flag.String("audit-output", "", "audit receipt output; defaults to study directory")
	binary := flag.String("native", "", "native binary")
	root := flag.String("native-root", "", "native fixtures")
	output := flag.String("out", "", "fresh relative runs directory")
	revision := flag.String("source-revision", "", "committed runner SHA")
	nativeRevision := flag.String("native-revision", "", "clean binary source SHA")
	familyCIStatus := flag.String("family-ci-status", "PASS", "caller family-study CI context: PASS, FAIL or UNKNOWN")
	flag.Parse()
	if *mode == "native-unfixed-audit" {
		value, err := auditNativeUnfixed(*output, *revision, *nativeRevision)
		if err == nil {
			err = save(*auditOutput, value)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "native-unfixed-audit:", err)
			os.Exit(1)
		}
		return
	}
	if *mode == "native-unfixed-pilot" {
		if err := runNativeUnfixedPilot(*binary, *goBinary, *output, *revision, *nativeRevision); err != nil {
			fmt.Fprintln(os.Stderr, "native-unfixed:", err)
			os.Exit(1)
		}
		return
	}
	if *mode == "unfixed-study" || *mode == "unfixed-audit" {
		var err error
		if *mode == "unfixed-study" {
			err = runUnfixed(*goBinary, *output, *revision)
		} else {
			var result map[string]any
			result, err = auditUnfixed(*output, *revision)
			if err == nil {
				err = save(*auditOutput, result)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "unfixed:", err)
			os.Exit(1)
		}
		return
	}
	if *mode == "compound-audit" {
		value, err := auditCompound(*output)
		if err == nil {
			path := *auditOutput
			if path == "" {
				path = filepath.Join(*output, "audit.json")
			}
			err = save(path, value)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "compound-cohort" {
		if err := writeCompoundCohort(*output); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "compound-study" || *mode == "compound-pilot" {
		if err := runCompound(*binary, *goBinary, *output, *revision, *mode == "compound-pilot"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "family-optimized" {
		if err := runFamilyFor(*binary, *goBinary, *output, *revision, false, familySpec{Revision: *nativeRevision, SDK: "v0.2.5-experimental", NoChoice: true, CIStatus: *familyCIStatus}); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "family-pilot" || *mode == "family-study" {
		if err := runFamily(*binary, *goBinary, *output, *revision, *mode == "family-pilot"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "family-cohort" {
		if err := writeFamilyCohort(*output); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "trained-dogfood" {
		if err := trainedDogfood(*binary, *goBinary, *output, *revision, *nativeRevision); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "continued-audit" || *mode == "trained-audit" || *mode == "family-audit" {
		var value map[string]any
		var err error
		if *mode == "family-audit" {
			value, err = auditFamily(*output)
		} else if *mode == "trained-audit" {
			value, err = auditTrainedDogfood(*output)
		} else {
			value, err = auditContinuedBound(*output)
		}
		if err == nil {
			path := *auditOutput
			if path == "" {
				path = filepath.Join(*output, "audit.json")
			}
			err = save(path, value)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "continued-bound" {
		if err := probeContinuedBound(*binary, *goBinary, *output, *revision, *nativeRevision); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "input-bound" {
		if err := probeInputBound(*binary, *root, *output, *revision, *nativeRevision); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode == "audit" || *mode == "execute" {
		value, err := audit(*output, *root, *goBinary, *mode == "execute")
		if err == nil {
			path := *auditOutput
			if path == "" {
				path = filepath.Join(*output, "audit.json")
			}
			err = save(path, value)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *mode != "study" {
		fmt.Fprintln(os.Stderr, "unknown mode")
		os.Exit(1)
	}
	if err := run(*binary, *root, *output, *revision, *nativeRevision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
