package bodydecision

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func TestChooseUsesClosedLabelsAndKeepsHoleTextOutOfReceipt(t *testing.T) {
	model := loadTestModel(t, [decision.LabelCount]float32{10}, 0)
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "private instruction 9d6d5f")
	selection, err := Choose(plan, model, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := selection.Choices["op"]; got != "add" {
		t.Fatalf("model choice = %q, want the closed label add", got)
	}
	if len(selection.Holes) != 1 || selection.Holes[0].Proposed != "add" || selection.Holes[0].Selected != "add" {
		t.Fatalf("unexpected hole receipt: %+v", selection.Holes)
	}
	if selection.ModelCalls != 1 || selection.ProviderCalls != 0 {
		t.Fatalf("unexpected call counts: model=%d provider=%d", selection.ModelCalls, selection.ProviderCalls)
	}
	encoded, err := json.Marshal(selection)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(plan.Expressions[2].Text)) {
		t.Fatalf("selection receipt leaked raw hole text: %s", encoded)
	}
	if !strings.Contains(string(encoded), "text_sha256") || !strings.Contains(string(encoded), "weights_sha256") {
		t.Fatalf("selection receipt omitted text/model binding: %s", encoded)
	}
	if !isOperationLabel(selection.Holes[0].Proposed) {
		t.Fatalf("model emitted a label outside the fixed enum: %q", selection.Holes[0].Proposed)
	}
}

func TestChooseFallbackReplayAndSeededSamplingAreDeterministic(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "multiply", "add one to input")
	first, err := Choose(plan, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Choose(plan, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Choices, second.Choices) || first.Choices["op"] != "multiply" {
		t.Fatalf("fallback replay changed: first=%v second=%v", first.Choices, second.Choices)
	}
	if first.ModelCalls != 0 || first.ProviderCalls != 0 || first.Holes[0].Mode != "declared_fallback" {
		t.Fatalf("fallback replay claims calls or wrong mode: %+v", first)
	}
	if _, err := Choose(plan, nil, "seed-needs-model"); err == nil {
		t.Fatal("seeded sampling without an explicit model was accepted")
	}

	model := loadTestModel(t, [decision.LabelCount]float32{}, 0)
	a, err := Choose(plan, model, "repeatable-seed")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Choose(plan, model, "repeatable-seed")
	if err != nil {
		t.Fatal(err)
	}
	if a.Choices["op"] != b.Choices["op"] || a.Holes[0].Mode != "seeded_allowed_probability_sample" {
		t.Fatalf("sampling was not replayable: a=%+v b=%+v", a.Holes[0], b.Holes[0])
	}
	if !contains(plan.Expressions[2].Allowed, a.Choices["op"]) {
		t.Fatalf("sampler selected an operation outside the allowed set: %q", a.Choices["op"])
	}
	if a.SeedSHA256 == "" || a.SeedSHA256 != b.SeedSHA256 {
		t.Fatalf("seed binding is missing or unstable: %q %q", a.SeedSHA256, b.SeedSHA256)
	}
}

func TestChooseFallsBackWhenModelArgmaxIsOutsideAllowedSet(t *testing.T) {
	logits := [decision.LabelCount]float32{}
	logits[6] = 12 // and is valid globally, but not for Int × Int.
	model := loadTestModel(t, logits, 0)
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "choose an integer operation")
	selection, err := Choose(plan, model, "")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Holes[0].Proposed != "and" || selection.Holes[0].Selected != "subtract" || selection.Holes[0].Mode != "fallback_global_label_outside_allowed" {
		t.Fatalf("out-of-context model label was not safely rejected: %+v", selection.Holes[0])
	}
}

func TestChooseFailsClosedForUninitializedNonNilModel(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "uninitialized model")
	selection, err := Choose(plan, &decision.Model{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Choices["op"] != "subtract" || selection.ModelCalls != 1 || selection.ProviderCalls != 0 || len(selection.Holes) != 1 || selection.Holes[0].Mode != "fallback_prediction_error" {
		t.Fatalf("uninitialized model did not retain the declared fallback: %+v", selection)
	}
}

func TestChooseRunsAgainstAllFrozenPilotModelVariants(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "small integer expression")
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		t.Run(variant, func(t *testing.T) {
			path := filepath.Join("..", "..", "runs", "pilot-mps-20260930-v1", "models", variant, "model.json")
			model, err := decision.Load(path)
			if err != nil {
				t.Fatalf("load frozen %s model: %v", variant, err)
			}
			selection, err := Choose(plan, model, "")
			if err != nil {
				t.Fatal(err)
			}
			if selection.ModelVariant != variant || selection.WeightsSHA256 != model.WeightsSHA256() || selection.ModelCalls != 1 || selection.ProviderCalls != 0 {
				t.Fatalf("model binding/counters differ: %+v", selection)
			}
			if !contains(plan.Expressions[2].Allowed, selection.Choices["op"]) || !isOperationLabel(selection.Holes[0].Proposed) {
				t.Fatalf("model result escaped the closed operation set: %+v", selection.Holes[0])
			}
		})
	}
}

func TestValidateRejectsEveryIllTypedCandidateBeforeSelection(t *testing.T) {
	plan := intHolePlan([]string{"add", "and"}, "add", "invalid bool operator")
	if _, err := Validate(plan); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("Validate accepted an ill-typed candidate set: %v", err)
	}
	if _, err := Choose(plan, &decision.Model{}, ""); err == nil || !strings.Contains(err.Error(), "candidate") {
		t.Fatalf("Choose did not reject every candidate before attempting inference: %v", err)
	}
}

func TestValidateAndChooseSupportSameTypedBooleanOperationSet(t *testing.T) {
	plan := bodyplan.Plan{
		Schema: bodyplan.Schema, ID: "boolean-hole-plan", Name: "Compare", ResultType: decision.TypeBool,
		Expressions: []bodyplan.Expr{
			{Kind: bodyplan.ExprInput, Name: "input"},
			{Kind: bodyplan.ExprInt, Int: 5},
			{Kind: bodyplan.ExprHole, Left: 0, Right: 1, HoleID: "cmp", Text: "compare the two integers", Allowed: []string{"less_than", "less_equal", "equal"}, Fallback: "less_than"},
		},
		Statements: []bodyplan.Stmt{{Kind: bodyplan.StmtReturn, Expr: 2}}, Root: []int{0},
	}
	logits := [decision.LabelCount]float32{}
	logits[3] = 10
	selection, err := Choose(plan, loadTestModel(t, logits, 0), "")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Choices["cmp"] != "less_than" {
		t.Fatalf("Boolean-result hole selected %q", selection.Choices["cmp"])
	}
	program, err := bodyplan.Compile(plan, selection.Choices)
	if err != nil {
		t.Fatal(err)
	}
	value, err := program.Evaluate(4)
	if err != nil || value.Type != decision.TypeBool || !value.Bool {
		t.Fatalf("typed Boolean composition got %+v, %v; want true Bool", value, err)
	}
}

func TestSearchIsTrainingOnlyBoundedAndKeepsInitialOnTies(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "multiply", "finite search")
	initial := map[string]string{"op": "multiply"}
	training := []Case{
		{Input: 1, Expected: bodyplan.Value{Type: decision.TypeInt, Int: 1}},
		{Input: 2, Expected: bodyplan.Value{Type: decision.TypeInt, Int: 0}},
	}
	result, err := Search(plan, initial, training, 8)
	if err != nil {
		t.Fatal(err)
	}
	if result.Initial.Total != len(training) || result.Final.Total != len(training) {
		t.Fatalf("search denominator is not training-only: initial=%+v final=%+v", result.Initial, result.Final)
	}
	if result.Choices["op"] != "multiply" || result.Initial.Correct != 1 || result.Final.Correct != 1 {
		t.Fatalf("equal-score candidates replaced the initial proposal: %+v", result)
	}
	seen := map[string]bool{}
	for _, attempt := range result.Attempts {
		key := attempt.Choices["op"]
		if seen[key] {
			t.Fatalf("search-space candidate %q was tested twice: %+v", key, result.Attempts)
		}
		seen[key] = true
		if attempt.Score.Total != len(training) {
			t.Fatalf("attempt denominator differs from training cases: %+v", attempt)
		}
	}
	if len(result.Attempts) != len(plan.Expressions[2].Allowed) {
		t.Fatalf("search did not visit each unique operation once, including the initial: %+v", result.Attempts)
	}
	if result.StopReason != "candidate_space_exhausted" {
		t.Fatalf("unexpected stop reason: %q", result.StopReason)
	}

	for _, limit := range []int{0, 129} {
		if _, err := Search(plan, initial, training, limit); err == nil {
			t.Errorf("Search accepted out-of-range attempt limit %d", limit)
		}
	}
	if _, err := Search(plan, initial, nil, 1); err == nil {
		t.Error("Search accepted an empty training set")
	}
	if one, err := Search(plan, initial, training, 1); err != nil || len(one.Attempts) != 1 || one.StopReason != "attempt_budget_reached" {
		t.Fatalf("one-candidate attempt budget was not enforced: result=%+v err=%v", one, err)
	}
	if result, err := Search(plan, initial, training, 128); err != nil || result.Limit != 128 {
		t.Fatalf("maximum valid attempt budget rejected: result=%+v err=%v", result, err)
	}
}

func TestScoreProgramRejectsNilProgram(t *testing.T) {
	if _, err := ScoreProgram(nil, []Case{{Input: 1, Expected: bodyplan.Value{Type: decision.TypeInt}}}); err == nil {
		t.Fatal("ScoreProgram panicked or accepted a nil program; want a returned error")
	}
}

func TestSearchExhaustiveMultiHoleSpaceHasNoDuplicateRowsAndRetainsInitialTie(t *testing.T) {
	plan := twoArithmeticHolePlan()
	initial := map[string]string{"outer": "add", "inner": "add"}
	training := []Case{
		{Input: 0, Expected: bodyplan.Value{Type: decision.TypeInt, Int: 1}},
		{Input: 0, Expected: bodyplan.Value{Type: decision.TypeInt, Int: -1}},
	}
	result, err := Search(plan, initial, training, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Attempts) != 4 {
		t.Fatalf("enumeration should include four unique combinations, got %d: %+v", len(result.Attempts), result.Attempts)
	}
	seen := make(map[string]bool)
	for _, attempt := range result.Attempts {
		key := fmt.Sprintf("%s/%s", attempt.Choices["inner"], attempt.Choices["outer"])
		if seen[key] {
			t.Fatalf("duplicate combination %s in %+v", key, result.Attempts)
		}
		seen[key] = true
		if attempt.Score.Total != 2 || attempt.Score.Correct != 1 {
			t.Fatalf("synthetic contradictory training denominator/score changed: %+v", attempt)
		}
	}
	if !reflect.DeepEqual(result.Choices, initial) {
		t.Fatalf("search replaced the initial proposal on a tie: got=%v initial=%v", result.Choices, initial)
	}
}

func TestProposalAndTrainingSearchFinalRemainSeparate(t *testing.T) {
	logits := [decision.LabelCount]float32{}
	logits[0] = 12
	model := loadTestModel(t, logits, 0)
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "multiply", "proposal is not the final score winner")
	proposal, err := Choose(plan, model, "")
	if err != nil {
		t.Fatal(err)
	}
	training := []Case{
		{Input: 1, Expected: bodyplan.Value{Type: decision.TypeInt, Int: 0}},
		{Input: 2, Expected: bodyplan.Value{Type: decision.TypeInt, Int: 1}},
	}
	search, err := Search(plan, proposal.Choices, training, 8)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.Choices["op"] != "add" || proposal.Holes[0].Proposed != "add" {
		t.Fatalf("unexpected model proposal: %+v", proposal)
	}
	if search.Choices["op"] != "subtract" || search.Final.Correct != 2 || search.Final.Total != 2 {
		t.Fatalf("training-tested final choice/denominator incorrect: %+v", search)
	}
	if proposal.ModelCalls != 1 || search.ModelCalls != 0 {
		t.Fatalf("proposal and deterministic local search call counts were conflated: proposal=%d search=%d", proposal.ModelCalls, search.ModelCalls)
	}
}

func TestChooseLayaSendsOnlyExactClosedChoiceAndRequiresPinnedRoute(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "add one")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected provider request: %s %s", request.Method, request.URL.String())
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var payload struct {
			Model     string            `json:"model"`
			State     map[string]string `json:"state"`
			Questions map[string]struct {
				Type         string            `json:"type"`
				Instructions string            `json:"instructions"`
				Criteria     map[string]string `json:"criteria"`
			} `json:"questions"`
		}
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		if payload.Model != "multilingual" || payload.State["request"] != "add one" {
			t.Errorf("unexpected model/state: %+v", payload)
		}
		question, ok := payload.Questions["operation"]
		if !ok || question.Type != "choice" || question.Instructions == "" {
			t.Errorf("unexpected question: %+v", payload.Questions)
		}
		if len(question.Criteria) != 3 || question.Criteria["add"] == "" || question.Criteria["subtract"] == "" || question.Criteria["multiply"] == "" || question.Criteria["and"] != "" {
			t.Errorf("criteria were not restricted to the declared operations: %+v", question.Criteria)
		}
		io.WriteString(writer, `{"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"add","probabilities":{"add":0.8,"subtract":0.1,"multiply":0.1}}}}`)
	}))
	defer server.Close()

	selection, exchanges, err := ChooseLaya(context.Background(), plan, server.URL+"/v1/systemone")
	if err != nil {
		t.Fatal(err)
	}
	if selection.Choices["op"] != "add" || selection.ProviderCalls != 1 || selection.ModelCalls != 0 {
		t.Fatalf("unexpected provider selection/counters: %+v", selection)
	}
	if len(exchanges) != 1 || exchanges[0].Status != "completed_validated" || len(exchanges[0].Response) == 0 {
		t.Fatalf("successful exchange was not completely captured: %+v", exchanges)
	}
	for _, endpoint := range []string{"https://127.0.0.1:1/v1/systemone", "http://example.invalid/v1/systemone", "http://127.0.0.1:1/wrong"} {
		if _, _, err := ChooseLaya(context.Background(), plan, endpoint); err == nil {
			t.Errorf("accepted non-loopback or non-exact endpoint %q", endpoint)
		}
	}
}

func TestChooseLayaRetainsInvalidReplyBytesAndRejectsRouteProbabilityErrors(t *testing.T) {
	tests := []struct {
		name  string
		body  string
		count int
	}{
		{
			name: "wrong route",
			body: `{"routing":{"model":"english"},"answers":{"operation":{"choice":"add","probabilities":{"add":1,"subtract":0,"multiply":0}}}}`,
		},
		{
			name: "nonfinite probability",
			body: `{"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"add","probabilities":{"add":1e10000,"subtract":0,"multiply":0}}}}`,
		},
		{
			name: "extra probability label",
			body: `{"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"add","probabilities":{"add":0.5,"subtract":0.2,"multiply":0.2,"and":0.1}}}}`,
		},
		{
			name: "duplicate reply field",
			body: `{"routing":{"model":"multilingual"},"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"add","probabilities":{"add":1,"subtract":0,"multiply":0}}}}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "bad response")
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				io.WriteString(writer, test.body)
			}))
			defer server.Close()
			selection, exchanges, err := ChooseLaya(context.Background(), plan, server.URL+"/v1/systemone")
			if err == nil {
				t.Fatal("invalid provider reply was accepted")
			}
			if selection.Choices["op"] != "subtract" || selection.ProviderCalls != 1 {
				t.Fatalf("failed call lost fallback or count: selection=%+v err=%v", selection, err)
			}
			if len(exchanges) != 1 || exchanges[0].Status == "completed_validated" {
				t.Fatalf("invalid reply was not preserved as one incomplete exchange: %+v", exchanges)
			}
			if !bytes.Equal(exchanges[0].Response, []byte(test.body)) {
				t.Fatalf("raw invalid reply bytes were not captured: got=%q want=%q", exchanges[0].Response, test.body)
			}
		})
	}
}

func TestChooseLayaTruncationAndFailurePreservePartialExchange(t *testing.T) {
	plan := twoArithmeticHolePlan()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		number := requests.Add(1)
		if number == 1 {
			io.WriteString(writer, `{"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"subtract","probabilities":{"add":0.1,"subtract":0.9}}}}`)
			return
		}
		writer.WriteHeader(http.StatusBadGateway)
		io.WriteString(writer, strings.Repeat("x", 70_000))
	}))
	defer server.Close()

	selection, exchanges, err := ChooseLaya(context.Background(), plan, server.URL+"/v1/systemone")
	if err == nil {
		t.Fatal("second failed provider response was accepted")
	}
	if requests.Load() != 2 || selection.ProviderCalls != 2 || len(selection.Holes) != 1 || len(exchanges) != 2 {
		t.Fatalf("partial capture counts are wrong: requests=%d selection=%+v exchanges=%+v err=%v", requests.Load(), selection, exchanges, err)
	}
	if exchanges[0].Status != "completed_validated" || exchanges[1].Status == "completed_validated" {
		t.Fatalf("exchange completion states are wrong: %+v", exchanges)
	}
	if selection.Choices["inner"] != "subtract" || selection.Choices["outer"] != "add" {
		t.Fatalf("partial failure lost the completed selection or declared fallback: %v", selection.Choices)
	}
	if len(exchanges[1].Response) == 0 {
		t.Error("failed/truncated second response lost all raw response bytes")
	}
	if len(exchanges[1].Response) != 65_537 {
		t.Errorf("truncated reply capture length = %d, want bounded 65537-byte prefix", len(exchanges[1].Response))
	}
}

func TestChooseLayaCancellationIsNotRetriedAndLeavesOnePartialExchange(t *testing.T) {
	plan := intHolePlan([]string{"add", "subtract", "multiply"}, "subtract", "cancel request")
	started := make(chan struct{})
	release := make(chan struct{})
	var startOnce sync.Once
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		startOnce.Do(func() { close(started) })
		<-release
		io.WriteString(writer, `{"routing":{"model":"multilingual"},"answers":{"operation":{"choice":"add","probabilities":{"add":0.8,"subtract":0.1,"multiply":0.1}}}}`)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan struct {
		selection Selection
		exchanges []Exchange
		err       error
	}, 1)
	go func() {
		selection, exchanges, err := ChooseLaya(ctx, plan, server.URL+"/v1/systemone")
		resultCh <- struct {
			selection Selection
			exchanges []Exchange
			err       error
		}{selection, exchanges, err}
	}()
	select {
	case <-started:
		cancel()
		close(release)
	case <-time.After(2 * time.Second):
		cancel()
		close(release)
		t.Fatal("provider handler did not receive request")
	}
	select {
	case result := <-resultCh:
		if result.err == nil || requests.Load() != 1 || result.selection.ProviderCalls != 1 || len(result.exchanges) != 1 || result.exchanges[0].Status == "completed_validated" {
			t.Fatalf("canceled call was retried or falsely completed: requests=%d result=%+v", requests.Load(), result)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("canceled provider call did not settle")
	}
}

func intHolePlan(allowed []string, fallback, text string) bodyplan.Plan {
	return bodyplan.Plan{
		Schema: bodyplan.Schema, ID: "test-plan", Name: "Compose", ResultType: decision.TypeInt,
		Expressions: []bodyplan.Expr{
			{Kind: bodyplan.ExprInput, Name: "input"},
			{Kind: bodyplan.ExprInt, Int: 1},
			{Kind: bodyplan.ExprHole, Left: 0, Right: 1, HoleID: "op", Text: text, Allowed: append([]string(nil), allowed...), Fallback: fallback},
		},
		Statements: []bodyplan.Stmt{{Kind: bodyplan.StmtReturn, Expr: 2}}, Root: []int{0},
	}
}

func twoArithmeticHolePlan() bodyplan.Plan {
	return bodyplan.Plan{
		Schema: bodyplan.Schema, ID: "two-hole-plan", Name: "ComposeTwice", ResultType: decision.TypeInt,
		Expressions: []bodyplan.Expr{
			{Kind: bodyplan.ExprInput, Name: "input"},
			{Kind: bodyplan.ExprInt, Int: 1},
			{Kind: bodyplan.ExprHole, Left: 0, Right: 1, HoleID: "inner", Text: "inner integer operation", Allowed: []string{"add", "subtract"}, Fallback: "add"},
			{Kind: bodyplan.ExprHole, Left: 2, Right: 0, HoleID: "outer", Text: "outer integer operation", Allowed: []string{"add", "subtract"}, Fallback: "add"},
		},
		Statements: []bodyplan.Stmt{{Kind: bodyplan.StmtReturn, Expr: 3}}, Root: []int{0},
	}
}

func isOperationLabel(label string) bool {
	for _, candidate := range decision.Labels() {
		if label == candidate {
			return true
		}
	}
	return false
}

func loadTestModel(t *testing.T, logits [decision.LabelCount]float32, threshold float64) *decision.Model {
	t.Helper()
	dir := t.TempDir()
	definitions := []struct {
		name       string
		count, row int
		col        int
	}{
		{name: "w1", count: decision.FeatureDim * decision.HiddenDim, row: decision.HiddenDim, col: decision.FeatureDim},
		{name: "b1", count: decision.HiddenDim, row: 1, col: decision.HiddenDim},
		{name: "w2", count: decision.HiddenDim * decision.LabelCount, row: decision.LabelCount, col: decision.HiddenDim},
		{name: "b2", count: decision.LabelCount, row: 1, col: decision.LabelCount},
	}
	var weights []byte
	tensors := make([]decision.TensorMetadata, len(definitions))
	for i, definition := range definitions {
		tensors[i] = decision.TensorMetadata{
			Name: definition.name, Count: definition.count, Rows: definition.row, Cols: definition.col,
			Encoding: "float32_le", Offset: int64(len(weights)), Bytes: int64(definition.count * 4), Scale: 1,
		}
		weights = append(weights, make([]byte, definition.count*4)...)
	}
	logitOffset := int(tensors[3].Offset)
	for i, value := range logits {
		binary.LittleEndian.PutUint32(weights[logitOffset+i*4:logitOffset+(i+1)*4], math.Float32bits(value))
	}
	digest := sha256.Sum256(weights)
	labels := decision.Labels()
	thresholdCopy := threshold
	metadata := decision.Metadata{
		Schema: decision.MetadataSchema, Variant: "fp32", FeatureDim: decision.FeatureDim,
		HiddenDim: decision.HiddenDim, MaxBytes: decision.InputMaxBytes, Labels: append([]string(nil), labels[:]...),
		Temperature: 1, ConfidenceThreshold: &thresholdCopy, WeightsFile: "weights.bin",
		WeightsSHA256: hex.EncodeToString(digest[:]), Tensors: tensors,
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "weights.bin"), weights, 0o600); err != nil {
		t.Fatal(err)
	}
	metadataPath := filepath.Join(dir, "model.json")
	if err := os.WriteFile(metadataPath, metadataBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	model, err := decision.Load(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	return model
}
