// compare-feedback-paths is a local bounded dogfood study, not a training loop.
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
	"strconv"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

var modelPins = map[string]string{
	"fp32":        "1ea3bada068f2487418f17270db4ba6785a98c3ad60e0ad7eb92359400a01bb5",
	"ptq_ternary": "7c4eb4068d76e83620a9b8e7ce7f8d948b5e2436d44c7b79cf241da609dab894",
	"qat_ternary": "368e37e7899cecba03874b69e933525a2f8c90ecc53d787d8e698de6aa21c087",
}

type document struct {
	Plan  pathplan.Plan       `json:"path_plan"`
	Cases []pathplan.TestCase `json:"test_cases"`
}
type policy struct {
	ID              string                     `json:"id"`
	Arm             string                     `json:"arm"`
	Language        string                     `json:"language"`
	Contract        string                     `json:"contract"`
	FeedbackEnabled bool                       `json:"feedback_enabled"`
	WallNS          int64                      `json:"search_wall_ns"`
	Progress        []pathplan.SessionProgress `json:"progress"`
	Feedback        []pathplan.FeedbackReceipt `json:"feedback"`
	NativeSHA       string                     `json:"native_capture_sha256"`
	GoSHA           string                     `json:"generated_go_sha256"`
	ReplaySHA       string                     `json:"executed_go_capture_sha256"`
	FinitePassed    int                        `json:"executed_finite_passed"`
	FiniteTotal     int                        `json:"executed_finite_cases"`
	EvalPassed      int                        `json:"executed_disjoint_input_passed"`
	EvalTotal       int                        `json:"executed_disjoint_input_cases"`
}
type bounded struct{ bytes.Buffer }

func (b *bounded) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 2<<20 {
		return 0, errors.New("bounded child output exceeded")
	}
	return b.Buffer.Write(p)
}
func hash(raw []byte) string { value := sha256.Sum256(raw); return hex.EncodeToString(value[:]) }
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func read(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4<<20 {
		return nil, errors.New("bounded regular input required")
	}
	return os.ReadFile(path)
}
func child(ctx context.Context, dir, binary string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=")
	var out, errout bounded
	cmd.Stdout = &out
	cmd.Stderr = &errout
	err := cmd.Run()
	if err != nil || errout.Len() != 0 {
		return out.Bytes(), fmt.Errorf("bounded child failed: %v", err)
	}
	return out.Bytes(), nil
}
func search(ctx context.Context, prepared *pathplan.PreparedPlan, model *decision.Model, cases []pathplan.TestCase, feedback bool, ci *pathplan.CIHint) (policy, *bodyplan.Program, error) {
	started := time.Now()
	value := policy{FeedbackEnabled: feedback}
	session, err := prepared.NewSession(ctx, model, cases, "")
	if err != nil {
		return value, nil, err
	}
	initial, err := session.Observe()
	if err != nil {
		return value, nil, err
	}
	value.Progress = append(value.Progress, initial)
	seen := [64]bool{}
	var body *bodyplan.Program
	for {
		progress, next, err := session.Advance(ctx, 8)
		if err != nil {
			return value, nil, err
		}
		body = next
		for _, attempt := range progress.NewAttempts {
			if int(attempt.Mask) >= len(seen) || seen[attempt.Mask] {
				return value, nil, errors.New("repeated or out-of-range mask")
			}
			seen[attempt.Mask] = true
		}
		value.Progress = append(value.Progress, progress)
		if progress.Exhausted || progress.Status == "TRAINING_COMPLETE" {
			break
		}
		if len(progress.NewAttempts) == 0 {
			return value, nil, errors.New("no candidate progress")
		}
		if feedback && model != nil && len(value.Feedback) < 2 {
			receipt, err := session.Reconsider(ctx, model, ci)
			value.Feedback = append(value.Feedback, receipt)
			if err != nil {
				return value, nil, err
			}
		}
	}
	value.WallNS = time.Since(started).Nanoseconds()
	return value, body, nil
}
func executeGo(ctx context.Context, goBinary, source string, inputs []int64) ([]byte, error) {
	dir, err := os.MkdirTemp("", "gooo-feedback-replay-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err = os.Mkdir(filepath.Join(dir, "projection"), 0700); err != nil {
		return nil, err
	}
	var harness strings.Builder
	harness.WriteString("package main\nimport (\"encoding/json\";\"os\";p \"gooo.feedback.replay/projection\")\nfunc main(){json.NewEncoder(os.Stdout).Encode([]int64{")
	for _, input := range inputs {
		fmt.Fprintf(&harness, "p.ConditionalAssign(%d),", input)
	}
	harness.WriteString("})}\n")
	for name, content := range map[string]string{"go.mod": "module gooo.feedback.replay\n\ngo 1.27.1\n", "projection/generated.go": source, "main.go": harness.String()} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			return nil, err
		}
	}
	return child(ctx, dir, goBinary, "run", ".")
}
func run(native, goBinary, output, revision string) error {
	if output == "" || filepath.IsAbs(output) || filepath.Clean(output) != output || !strings.HasPrefix(output, "runs/") {
		return errors.New("fresh relative runs output required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("output must be fresh")
	}
	for _, binary := range []string{native, goBinary} {
		info, err := buildinfo.ReadFile(binary)
		if err != nil || info.GoVersion != "go1.27.1" {
			return errors.New("exact Go 1.27.1 binaries required")
		}
	}
	info, _ := buildinfo.ReadFile(native)
	settings := map[string]string{}
	for _, s := range info.Settings {
		settings[s.Key] = s.Value
	}
	if settings["vcs.revision"] != "307159f041644a3aa56dfd325c345f5325aec902" || settings["vcs.modified"] != "false" {
		return errors.New("clean current native main required")
	}
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("runner revision mismatch")
	}
	if err = exec.Command("git", "diff", "--quiet", "HEAD", "--", "tools/compare-feedback-paths", "internal", "cmd/gooo-path-compose").Run(); err != nil {
		return errors.New("dirty runtime or study sources")
	}
	untracked, err := exec.Command("git", "ls-files", "--others", "--exclude-standard", "--", "tools/compare-feedback-paths", "internal", "cmd/gooo-path-compose").Output()
	if err != nil || len(untracked) != 0 {
		return errors.New("untracked runtime or study sources")
	}
	proof, err := read("publication/incremental-typed-path-main-push-20261001.json")
	if err != nil {
		return err
	}
	var publication struct {
		Head       string `json:"source_sha"`
		Conclusion string `json:"conclusion"`
		Decision   string `json:"proof_decision"`
	}
	if err = json.Unmarshal(proof, &publication); err != nil || publication.Conclusion != "success" || publication.Decision != "PASS" || publication.Head != settings["vcs.revision"] {
		return errors.New("actual prior CI publication missing")
	}
	ci := &pathplan.CIHint{SourceSHA: publication.Head, Status: "PASS"}
	if err = ci.Validate(); err != nil {
		return err
	}
	models := map[string]*decision.Model{}
	pins := map[string]string{}
	for arm, pin := range modelPins {
		model, err := decision.LoadPath(filepath.Join("runs/typed-path-positioned-random-20261001/models", arm, "model.json"))
		if err != nil || model.MetadataSHA256() != pin {
			return errors.New("frozen model mismatch")
		}
		models[arm] = model
		pins[arm] = model.WeightsSHA256()
	}
	evalRaw, err := read("studies/conditional-paths-v1/cohort/holdout-cases.json")
	if err != nil {
		return err
	}
	var evaluation []pathplan.TestCase
	if err = json.Unmarshal(evalRaw, &evaluation); err != nil || len(evaluation) != 9 {
		return errors.New("fixed disjoint evaluation missing")
	}
	if err = os.MkdirAll(filepath.Join(output, "captures"), 0755); err != nil {
		return err
	}
	fixtureHashes := map[string]string{"evaluation": hash(evalRaw)}
	for _, language := range []string{"en", "ko"} {
		raw, err := read("studies/conditional-paths-v1/cohort/" + language + "-budget-64.json")
		if err != nil {
			return err
		}
		fixtureHashes[language] = hash(raw)
	}
	binaryHashes := map[string]string{}
	for name, path := range map[string]string{"native": native, "go": goBinary} {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Size() > 64<<20 {
			return errors.New("bounded physical binary required")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		binaryHashes[name] = hash(raw)
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/feedback-path-pilot-preexecution/v1", "runner_revision": revision, "native_revision": settings["vcs.revision"], "model_metadata": modelPins, "model_weights": pins, "ci_hint": ci, "ci_hint_publication_sha256": hash(proof), "policies": 32, "paired_comparisons": 16, "compound_intents": 1, "attempt_step": 8, "maximum_candidates": 64, "maximum_feedback_rounds": 2, "evaluation_inputs": 9, "optimizer_steps": 0, "scope": "Same frozen compound intent, two languages, complete and inconsistent contracts, three models and offline controls; neither new independent ideas nor a feedback-trained model. CI hint is previously verified snapshot context, not current study authorization."}); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "source-input-pins.json"), map[string]any{"schema": "gooo/feedback-path-pilot-input-pins/v1", "fixtures": fixtureHashes, "binaries": binaryHashes}); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	var policies []policy
	replays := map[string][]byte{}
	nativeCalls := 0
	goCalls := 0
	for _, language := range []string{"en", "ko"} {
		raw, err := read("studies/conditional-paths-v1/cohort/" + language + "-budget-64.json")
		if err != nil {
			return err
		}
		var doc document
		if err = json.Unmarshal(raw, &doc); err != nil {
			return err
		}
		for _, contract := range []string{"complete", "inconsistent"} {
			cases := append([]pathplan.TestCase(nil), doc.Cases...)
			if len(cases) != 7 {
				return errors.New("finite fixture differs")
			}
			if contract == "inconsistent" {
				cases[6].Expected = 999
			}
			prepared, err := pathplan.Prepare(doc.Plan)
			if err != nil {
				return err
			}
			if len(doc.Plan.Decisions) != 6 {
				return errors.New("fixed six decisions required")
			}
			for _, arm := range []string{"offline", "fp32", "ptq_ternary", "qat_ternary"} {
				for _, feedback := range []bool{false, true} {
					value, body, err := search(ctx, prepared, models[arm], cases, feedback, ci)
					if err != nil {
						return err
					}
					value.ID = language + "-" + contract + "-" + arm + "-" + strconv.FormatBool(feedback)
					value.Arm, value.Language, value.Contract = arm, language, contract
					fixture := filepath.Join(output, "captures", value.ID+".gooo.fixture")
					text := "package pathstudy\nnamespace pathstudy\nentity Integer id \"pathstudy://entity/integer\"\nactivity ConditionalAssign(Integer) -> Integer computes " + strconv.Quote(body.GoooSource()) + "\n"
					if err = os.WriteFile(fixture, []byte(text), 0644); err != nil {
						return err
					}
					nativeRaw, err := child(ctx, "", native, "body-codegen", "--json", "--activity", "ConditionalAssign", fixture)
					nativeCalls++
					if writeErr := os.WriteFile(filepath.Join(output, "captures", value.ID+".native.json"), nativeRaw, 0644); writeErr != nil {
						return writeErr
					}
					if err != nil {
						return err
					}
					var generated struct {
						Source string `json:"source"`
						Report struct {
							Decision  string `json:"decision"`
							Typecheck bool   `json:"typecheck_passed"`
							Replay    bool   `json:"deterministic_replay"`
							Writes    int    `json:"repository_writes"`
						} `json:"report"`
					}
					if err = json.Unmarshal(nativeRaw, &generated); err != nil || generated.Report.Decision != "PASS" || !generated.Report.Typecheck || !generated.Report.Replay || generated.Report.Writes != 0 {
						return errors.New("native candidate generation failed type/replay boundary")
					}
					value.NativeSHA, value.GoSHA = hash(nativeRaw), hash([]byte(generated.Source))
					inputs := make([]int64, 0, len(cases)+len(evaluation))
					for _, test := range append(append([]pathplan.TestCase(nil), cases...), evaluation...) {
						inputs = append(inputs, test.Input)
					}
					inputRaw, _ := json.Marshal(inputs)
					replayKey := value.GoSHA + ":" + hash(inputRaw)
					replay := replays[replayKey]
					if replay == nil {
						replay, err = executeGo(ctx, goBinary, generated.Source, inputs)
						goCalls++
						if err != nil {
							return err
						}
						replays[replayKey] = replay
					}
					if err = os.WriteFile(filepath.Join(output, "captures", value.ID+".executed.json"), replay, 0644); err != nil {
						return err
					}
					value.ReplaySHA = hash(replay)
					var actual []int64
					if err = json.Unmarshal(replay, &actual); err != nil || len(actual) != 16 {
						return errors.New("executed output denominator differs")
					}
					value.FiniteTotal, value.EvalTotal = 7, 9
					for i, test := range cases {
						if actual[i] == test.Expected {
							value.FinitePassed++
						}
					}
					for i, test := range evaluation {
						if actual[7+i] == test.Expected {
							value.EvalPassed++
						}
					}
					last := value.Progress[len(value.Progress)-1]
					if value.FinitePassed != last.SelectedPassed {
						return errors.New("actual generated Go disagrees with finite arena results")
					}
					policies = append(policies, value)
				}
			}
		}
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/feedback-path-pilot/v1", "runner_revision": revision, "native_calls": nativeCalls, "unique_generated_go_input_executions": goCalls, "external_model_calls": 0, "optimizer_steps": 0, "gpu_calls": 0, "ci_hint": ci, "policies": policies, "scope": "One compound intent and existing disjoint function inputs; repeated language/model/policy views are not new independent tasks. Search wall time excludes load, native generation, generated-Go compilation and process startup. Reused Go executions are explicitly deduplicated by identical emitted bytes and the hash of the input vector."})
}
func main() {
	native := flag.String("native", "", "clean main compiler binary")
	goBinary := flag.String("go", "", "physical Go 1.27.1 binary")
	output := flag.String("output", "", "fresh relative runs directory")
	revision := flag.String("revision", "", "exact runner source SHA")
	flag.Parse()
	if err := run(*native, *goBinary, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
