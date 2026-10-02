package threefeedback

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

func sameHash(v any, want string) bool {
	raw, e := json.Marshal(v)
	return e == nil && want != "" && threecohort.SHA(raw) == want
}
func progressHash(p pathplan.SessionProgress) bool { s := p.SHA; p.SHA = ""; return sameHash(p, s) }
func feedbackHash(f pathplan.FeedbackReceipt) bool { s := f.SHA; f.SHA = ""; return sameHash(f, s) }

// Verify performs no operational model inference. It reconstructs all ordered
// int64 outputs, immutable progress/feedback chains and the eight-mask frontier
// from the actual recorded distributions, including seeded initial selection.
func Verify(v threecohort.View, c Capture) error {
	if v.Split != "train" || c.Schema != "gooo/own-three-choice-teacher-capture/v1" || c.ViewID != v.ID || c.SourceSHA != v.SourceSHA || c.InputSHA != threecohort.SHA([]byte(v.Text)) || c.SeedIndex < 0 || c.SeedIndex > 1 || c.Seed != fmt.Sprintf("three-source-path-teacher-%d", c.SeedIndex) || c.WallNS <= 0 || c.RuntimeError != "" {
		return errors.New("completed training-only frozen teacher capture required")
	}
	s := c.Search
	if s.Schema != "gooo/typed-path-tdd-search/v1" || s.Status != "TRAINING_COMPLETE" || s.DeclaredCombinations != 8 || s.TrainingTotal != 16 || s.SelectedTrainingPassed != 16 || s.TypeRejected != 0 || s.Evaluated != len(s.Attempts) || len(s.Attempts) < 1 || len(s.Attempts) > 8 || s.Unattempted != 8-len(s.Attempts) || len(c.Feedback) != len(s.Attempts)-1 || len(c.Progress) != 2*len(s.Attempts) {
		return errors.New("eight-mask actual session bounds differ")
	}
	if err := verifyAttempts(v, s); err != nil {
		return err
	}
	return verifyTrace(v, c)
}

func verifyAttempts(v threecohort.View, s pathplan.SearchResult) error {
	var seen uint8
	for i, a := range s.Attempts {
		if a.Mask >= 8 || seen&(1<<a.Mask) != 0 || a.Status != "EVALUATED" || a.Total != 16 || len(a.Results) != 16 {
			return errors.New("unique complete typed attempt required")
		}
		seen |= 1 << a.Mask
		choices, err := threecompositionstudy.Choices(v.Plan, int(a.Mask))
		body, e := v.Prepared.Compile(choices)
		if err != nil || e != nil || !reflect.DeepEqual(choices, a.Choices) || a.GoooSHA != threecohort.SHA([]byte(body.GoooSource())) {
			return errors.New("source-bound emitted fragment differs")
		}
		passed := 0
		for j, test := range v.Cases {
			actual, e := threecompositionstudy.Oracle(v.Family, v.Config, int(a.Mask), test.Input)
			if e != nil || a.Results[j] != (pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: actual, Passed: actual == test.Expected}) {
				return errors.New("actual ordered values differ from independent arithmetic")
			}
			if actual == test.Expected {
				passed++
			}
		}
		if a.Passed != passed || passed != v.Target.Passed[a.Mask] || (i < len(s.Attempts)-1 && passed == 16) || (i == len(s.Attempts)-1 && passed != 16) {
			return errors.New("actual finite stopping rule differs")
		}
	}
	return nil
}

type frontier struct {
	scheduled, queued, committed uint8
	weights                      [3][2]float64
}

func (q *frontier) add(mask int) bool {
	bit := uint8(1 << mask)
	if q.scheduled&bit != 0 {
		return false
	}
	q.scheduled |= bit
	q.queued |= bit
	return true
}
func (q *frontier) score(mask int) float64 {
	var sum float64
	for i := range 3 {
		sum += q.weights[i][(mask>>i)&1]
	}
	return sum
}
func (q *frontier) pop() int {
	best := -1
	for mask := range 8 {
		if q.queued&(1<<mask) != 0 && (best < 0 || q.score(mask) > q.score(best)) {
			best = mask
		}
	}
	if best >= 0 {
		q.queued &^= 1 << best
		q.committed |= 1 << best
	}
	return best
}
func (q *frontier) fixed(i int) (int, bool) {
	var n [2]int
	for mask := range 8 {
		if q.committed&(1<<mask) != 0 {
			n[(mask>>i)&1]++
		}
	}
	for option := range 2 {
		if n[1-option] == 4 {
			return option, true
		}
	}
	return 0, false
}

func copySelection(s pathplan.Selection) pathplan.Selection {
	s.Receipts = append([]pathplan.Receipt(nil), s.Receipts...)
	m := map[string]string{}
	for k, v := range s.Choices {
		m[k] = v
	}
	s.Choices = m
	return s
}
func validatePrediction(r pathplan.Receipt, choice pathplan.Choice, weights [2]float64) error {
	if r.ID != choice.ID || r.Kind != choice.Kind || r.IntentSHA256 != threecohort.SHA([]byte(choice.Intent)) || r.PredictNS < 0 {
		return errors.New("prediction source identity differs")
	}
	labels := decision.PathLabels()
	sum := float64(0)
	top := 0
	var eligible [2]float64
	for i, p := range r.Probabilities {
		if p < 0 || p > 1 || math.IsNaN(float64(p)) || math.IsInf(float64(p), 0) {
			return errors.New("invalid observed probability")
		}
		sum += float64(p)
		if p > r.Probabilities[top] {
			top = i
		}
		for j, o := range choice.Options {
			if labels[i] == o.Label {
				eligible[j] = float64(p)
			}
		}
	}
	if math.Abs(sum-1) > 2e-6 || r.Proposed != labels[top] || r.Confidence != r.Probabilities[top] {
		return errors.New("global prediction receipt differs")
	}
	if weights[0] < 0 || weights[1] < 0 || math.IsNaN(weights[0]+weights[1]) || math.Abs(weights[0]+weights[1]-1) > 1e-12 {
		return errors.New("eligible observed distribution differs")
	}
	// Raw probabilities are float32; the SDK ranks using logits. Verify their
	// normalized ratio within rounding tolerance, without pretending to replay logits.
	if eligible[0]+eligible[1] > 1e-30 && math.Abs(eligible[0]/(eligible[0]+eligible[1])-weights[0]) > 2e-6 {
		return errors.New("eligible/global probability relation differs")
	}
	return nil
}
func sampled(v threecohort.View, c Capture, choice pathplan.Choice, w [2]float64) string {
	bound := []byte(v.Prepared.PlanSHA256() + "\x00" + choice.ID + "\x00" + TeacherMetadata + "\x00" + TeacherWeights + "\x00" + c.Seed)
	total := w[0] + w[1]
	for i, o := range choice.Options {
		bound = append(bound, 0)
		bound = append(bound, o.Label...)
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], math.Float64bits(w[i]/total))
		bound = append(bound, b[:]...)
	}
	h := sha256.Sum256(bound)
	draw := float64(binary.BigEndian.Uint64(h[:8])>>11) / (1 << 53)
	if draw < w[0]/total {
		return choice.Options[0].Label
	}
	return choice.Options[1].Label
}

func verifyTrace(v threecohort.View, c Capture) error {
	initial := c.Progress[0]
	sel := initial.Selection
	if sel.Schema != "gooo/typed-body-path-selection/v1" || sel.MetadataSHA256 != TeacherMetadata || sel.WeightsSHA256 != TeacherWeights || sel.ModelVariant != "fp32" || sel.PlanSHA256 != v.Prepared.PlanSHA256() || sel.SeedSHA256 != threecohort.SHA([]byte(c.Seed)) || sel.ModelCalls != 3 || sel.ExternalCalls != 0 || !sel.ExternalCallsKnown || sel.Joint != nil || sel.Three != nil || len(sel.Receipts) != 3 || len(initial.EligibleProbabilities) != 3 || !reflect.DeepEqual(sel.Choices, v.Prepared.Defaults()) {
		return errors.New("initial independent teacher binding differs")
	}
	q := frontier{}
	initialMask := 0
	abstentions := 0
	proposals := map[string]string{}
	for i, choice := range v.Plan.Decisions {
		r := sel.Receipts[i]
		w := initial.EligibleProbabilities[i]
		if err := validatePrediction(r, choice, w); err != nil {
			return err
		}
		label := sampled(v, c, choice, w)
		if r.Selected != label || r.Mode != "typed_probability_rank_before_finite_validation" {
			return errors.New("seeded initial proposal differs")
		}
		if r.Confidence < 1 {
			abstentions++
		}
		proposals[choice.ID] = label
		if label == choice.Options[1].Label {
			initialMask |= 1 << i
		}
		for j, p := range w {
			q.weights[i][j] = math.Log(math.Max(p, 1e-12))
		}
	}
	q.add(initialMask)
	casesRaw, _ := json.Marshal(v.Cases)
	casesSHA := threecohort.SHA(casesRaw)
	selection := copySelection(sel)
	previous, feedbackSHA := "", ""
	calls, rounds, bestPassed := 3, 0, -1
	var bestCases []pathplan.TestResult
	derived := []Projection{}
	teacherInputs := []TeacherInput{}
	for i, p := range c.Progress {
		var wantedAttempts []pathplan.SearchAttempt
		if i%2 == 1 {
			a := c.Search.Attempts[i/2]
			if q.pop() != int(a.Mask) {
				return errors.New("recorded attempt differs from independently reconstructed frontier")
			}
			wantedAttempts = []pathplan.SearchAttempt{a}
			if a.Passed > bestPassed {
				bestPassed, bestCases = a.Passed, a.Results
				for k, label := range a.Choices {
					selection.Choices[k] = label
				}
				for j := range selection.Receipts {
					selection.Receipts[j].Selected = a.Choices[selection.Receipts[j].ID]
					selection.Receipts[j].Mode = "finite_tdd_selection"
				}
			}
			if a.Passed != 16 {
				for j := range 3 {
					q.add(int(a.Mask) ^ (1 << j))
				}
			}
		} else if i > 0 {
			f := c.Feedback[i/2-1]
			cause := c.Progress[i-1]
			if !feedbackHash(f) || f.Schema != "gooo/typed-path-feedback-judgment/v1" || f.Round != rounds+1 || f.PreviousSHA != feedbackSHA || f.FromProgressSHA != cause.SHA || f.PlanSHA != v.Prepared.PlanSHA256() || f.CaseSHA != casesSHA || f.MetadataSHA != TeacherMetadata || f.WeightsSHA != TeacherWeights || f.Attempted != cause.Attempted || f.Passed != bestPassed || f.Cases != 16 || f.TypeRejected != 0 || f.CI != nil || f.CIIsAuthority || f.Joint != nil || f.Three != nil {
				return errors.New("actual feedback source/cause/hash differs")
			}
			if err := verifyReconsider(v, f, cause, &q); err != nil {
				return err
			}
			inputs, err := AttemptedInputs(v, f, cause)
			if err != nil {
				return err
			}
			teacherInputs = append(teacherInputs, inputs...)
			projection, err := Derive(v, f, cause)
			if err != nil {
				return err
			}
			if projection != nil {
				derived = append(derived, *projection)
			}
			calls += f.ModelCalls
			for _, judgment := range f.Judgments {
				if judgment.Prediction.Confidence < 1 {
					abstentions++
				}
			}
			if f.CumulativeCalls != calls {
				return errors.New("actual feedback call accounting differs")
			}
			rounds, feedbackSHA = f.Round, f.SHA
			selection.ModelCalls = calls
		}
		attempted := bits.OnesCount8(q.committed)
		status := "PARTIAL"
		if bestPassed == 16 {
			status = "TRAINING_COMPLETE"
		}
		selectedPassed := max(bestPassed, 0)
		if !progressHash(p) || p.Schema != "gooo/typed-path-session-progress/v1" || p.Sequence != i+1 || p.PreviousSHA != previous || p.CaseSHA != casesSHA || p.Status != status || p.Declared != 8 || p.Attempted != attempted || p.Evaluated != attempted || p.Unattempted != 8-attempted || p.Cases != 16 || p.TypeRejected != 0 || p.SelectedPassed != selectedPassed || p.ModelAbstentions != abstentions || !reflect.DeepEqual(p.BestCases, bestCases) || !reflect.DeepEqual(p.NewAttempts, wantedAttempts) || !reflect.DeepEqual(p.Selection, selection) || !reflect.DeepEqual(p.InitialProposals, proposals) || !reflect.DeepEqual(p.EligibleProbabilities, initial.EligibleProbabilities) || !p.Initialized || p.Interrupted || p.InitializationError != "" || p.FeedbackRounds != rounds || p.FeedbackPredictions != calls-3 || p.LatestFeedbackSHA != feedbackSHA || p.ScheduledBytes != 8 || p.FrontierNodes != bits.OnesCount8(q.queued) || p.Exhausted != (attempted == 8) || p.PredictionsThisAdvance != 0 {
			return fmt.Errorf("progress %d differs from actual committed trace", i)
		}
		previous = p.SHA
	}
	last := c.Progress[len(c.Progress)-1]
	if !reflect.DeepEqual(c.TeacherInputs, teacherInputs) || !reflect.DeepEqual(c.Derived, derived) || !reflect.DeepEqual(c.Search.Selection, last.Selection) || !reflect.DeepEqual(c.Search.InitialProposals, proposals) || !reflect.DeepEqual(c.Search.EligibleProbabilities, initial.EligibleProbabilities) || c.Search.ModelAbstentionsObserved != last.ModelAbstentions {
		return errors.New("final search or derived student contexts differ")
	}
	return nil
}

func verifyReconsider(v threecohort.View, f pathplan.FeedbackReceipt, p pathplan.SessionProgress, q *frontier) error {
	var first *pathplan.TestResult
	for _, r := range p.BestCases {
		if !r.Passed {
			copy := r
			first = &copy
			break
		}
	}
	if first == nil || !reflect.DeepEqual(first, f.FirstFailure) {
		return errors.New("feedback requires actual first selected failure")
	}
	if 8-f.Attempted == 1 {
		if !f.RankingUnnecessary || f.ContextDeclined || f.ModelCalls != 0 || len(f.Judgments) != 0 || len(f.FixedCoordinates) != 0 || f.Applied || f.AddedMask || f.Error != "" {
			return errors.New("sole remaining path should make zero calls")
		}
		return nil
	}
	prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=0 remaining=%d mismatch=%d:%d:%d", f.Attempted, f.Passed, f.Cases, 8-f.Attempted, first.Input, first.Actual, first.Expected)
	var texts [3]string
	var fixed [3]bool
	expectedFixed := []pathplan.FixedCoordinate(nil)
	proposal, declined := 0, -1
	for i, choice := range v.Plan.Decisions {
		if option, ok := q.fixed(i); ok {
			fixed[i] = true
			proposal |= option << i
			expectedFixed = append(expectedFixed, pathplan.FixedCoordinate{DecisionID: choice.ID, Option: option, Selected: choice.Options[option].Label, OtherTried: 4, Remaining: 8 - f.Attempted})
			continue
		}
		text, err := decision.SemanticContextFeedbackInput(choice.Intent, prefix+" selected="+p.Selection.Choices[choice.ID])
		if err != nil {
			return err
		}
		texts[i] = text
		if len(text) > decision.InputMaxBytes {
			declined = i
			break
		}
	}
	if !reflect.DeepEqual(f.FixedCoordinates, expectedFixed) || f.RankingUnnecessary {
		return errors.New("committed-mask fixed coordinates differ")
	}
	if declined >= 0 {
		choice := v.Plan.Decisions[declined]
		if !f.ContextDeclined || f.ModelCalls != 0 || len(f.Judgments) != 0 || f.Applied || f.AddedMask || f.DeclinedDecision != choice.ID || f.DeclinedBytes != len(texts[declined]) || f.DeclinedInputSHA != threecohort.SHA([]byte(texts[declined])) || f.DeclinedIntentSHA != threecohort.SHA([]byte(choice.Intent)) || !strings.Contains(f.Error, "original intent was not truncated") {
			return errors.New("full-input zero-call decline differs")
		}
		return nil
	}
	if f.ContextDeclined || !f.Applied || f.Error != "" || f.ModelCalls != 3-len(expectedFixed) || len(f.Judgments) != f.ModelCalls {
		return errors.New("actual unfixed teacher prediction count differs")
	}
	var weights [3][2]float64
	at := 0
	for i, choice := range v.Plan.Decisions {
		if fixed[i] {
			continue
		}
		j := f.Judgments[at]
		at++
		if j.DecisionID != choice.ID || j.Input != texts[i] || j.InputSHA != threecohort.SHA([]byte(texts[i])) || j.Prediction.Mode != "feedback_probability_rank_before_finite_validation" {
			return errors.New("complete source-preserving feedback prediction differs")
		}
		if err := validatePrediction(j.Prediction, choice, j.Eligible); err != nil {
			return err
		}
		option := 0
		if j.Eligible[1] > j.Eligible[0] {
			option = 1
		}
		if j.Prediction.Selected != choice.Options[option].Label {
			return errors.New("observed feedback proposal differs")
		}
		proposal |= option << i
		for k, w := range j.Eligible {
			weights[i][k] = math.Log(math.Max(w, 1e-12))
		}
	}
	q.weights = weights
	if q.add(proposal) != f.AddedMask {
		return errors.New("feedback scheduled-mask identity differs")
	}
	return nil
}
