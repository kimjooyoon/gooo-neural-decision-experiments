package threefeedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

// VerifyThreeObservation reconstructs the entire unseeded eight-mask frontier,
// including complete failure-conditioned inputs and zero-call declines. Empty
// model pins identify the disconnected control. No model is run by this audit.
func VerifyThreeObservation(v threecohort.View, c Capture, metadata, weights, variant string) error {
	joint := metadata != ""
	n := len(c.Search.Attempts)
	wantProgress := n + 1
	if joint {
		wantProgress = n * 2
	}
	if c.Schema != "gooo/own-three-choice-sdk-capture/v1" || c.ViewID != v.ID || c.SourceSHA != v.SourceSHA || c.InputSHA != threecohort.SHA([]byte(v.Text)) || c.SeedIndex != -1 || c.Seed != "" || c.WallNS <= 0 || c.RuntimeError != "" || len(c.Progress) != wantProgress || len(c.Derived) != 0 || len(c.TeacherInputs) != 0 || (!joint && len(c.Feedback) != 0) || (joint && len(c.Feedback) != n-1) {
		return errors.New("complete unseeded three-choice or offline capture required")
	}
	if err := VerifyFiniteAttempts(v, c.Search); err != nil {
		return err
	}
	initial := c.Progress[0]
	sel := initial.Selection
	initialCalls := 0
	if joint {
		initialCalls = 1
	}
	if sel.Schema != "gooo/typed-body-path-selection/v1" || sel.PlanSHA256 != v.Prepared.PlanSHA256() || sel.MetadataSHA256 != metadata || sel.WeightsSHA256 != weights || sel.ModelVariant != variant || sel.SeedSHA256 != "" || sel.ModelCalls != initialCalls || sel.ExternalCalls != 0 || !sel.ExternalCallsKnown || sel.Joint != nil || len(sel.Receipts) != 3 || len(initial.EligibleProbabilities) != 0 || !reflect.DeepEqual(sel.Choices, v.Prepared.Defaults()) {
		return errors.New("initial model/source/choice binding differs")
	}
	q := frontier{joint: joint}
	mask := 0
	if joint {
		if err := validateThree(sel.Three, v.Text, v.Parts); err != nil {
			return err
		}
		mask = int(sel.Three.Sampled)
		for m, p := range sel.Three.Probabilities {
			q.maskWeights[m] = math.Log(math.Max(float64(p), 1e-12))
		}
	} else if sel.Three != nil {
		return errors.New("disconnected control contains model receipt")
	}
	proposals := map[string]string{}
	for i, choice := range v.Plan.Decisions {
		label, mode, proposed := choice.Fallback, "declared_fallback", ""
		if joint {
			label = choice.Options[(mask>>i)&1].Label
			proposed, mode = label, "three_choice_mask_rank_before_finite_validation"
		} else {
			if label == choice.Options[1].Label {
				mask |= 1 << i
			}
			for j, option := range choice.Options {
				if option.Label != label {
					q.weights[i][j] = -1
				}
			}
		}
		want := pathplan.Receipt{ID: choice.ID, Kind: choice.Kind, IntentSHA256: threecohort.SHA([]byte(choice.Intent)), Selected: label, Proposed: proposed, Mode: mode}
		if sel.Receipts[i] != want {
			return errors.New("initial declared coordinate receipt differs")
		}
		proposals[choice.ID] = label
	}
	q.add(mask)
	caseRaw, _ := json.Marshal(v.Cases)
	caseSHA := threecohort.SHA(caseRaw)
	selection := copySelection(sel)
	previous, latest := "", ""
	calls, rounds, attempted, best := initialCalls, 0, 0, -1
	var bestCases []pathplan.TestResult
	for i, p := range c.Progress {
		var newAttempts []pathplan.SearchAttempt
		isAttempt := i > 0 && (!joint || i%2 == 1)
		if isAttempt {
			a := c.Search.Attempts[attempted]
			if q.pop() != int(a.Mask) {
				return errors.New("attempt differs from independent observed frontier")
			}
			attempted++
			newAttempts = []pathplan.SearchAttempt{a}
			if a.Passed > best {
				best, bestCases = a.Passed, a.Results
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
				if joint {
					for mask := range 8 {
						q.add(mask)
					}
				}
			}
		} else if i > 0 {
			f, cause := c.Feedback[rounds], c.Progress[i-1]
			if !feedbackHash(f) || f.Schema != "gooo/typed-path-three-choice-feedback-judgment/v1" || f.Round != rounds+1 || f.PreviousSHA != latest || f.FromProgressSHA != cause.SHA || f.PlanSHA != v.Prepared.PlanSHA256() || f.CaseSHA != caseSHA || f.MetadataSHA != metadata || f.WeightsSHA != weights || f.Attempted != attempted || f.Passed != best || f.Cases != 16 || f.TypeRejected != 0 || f.CI != nil || f.CIIsAuthority || f.Joint != nil || len(f.Judgments) != 0 || len(f.FixedCoordinates) != 0 || f.AddedMask {
				return errors.New("three-choice feedback source/cause/hash differs")
			}
			if err := verifyThreeFailure(v, f, cause, &q); err != nil {
				return err
			}
			calls += f.ModelCalls
			if f.CumulativeCalls != calls {
				return errors.New("feedback actual call count differs")
			}
			rounds++
			latest = f.SHA
			selection.ModelCalls = calls
		}
		status := "PARTIAL"
		if best == 16 {
			status = "TRAINING_COMPLETE"
		}
		if !progressHash(p) || p.Schema != "gooo/typed-path-session-progress/v1" || p.Sequence != i+1 || p.PreviousSHA != previous || p.CaseSHA != caseSHA || p.Status != status || p.Declared != 8 || p.Attempted != attempted || p.Evaluated != attempted || p.Unattempted != 8-attempted || p.Cases != 16 || p.TypeRejected != 0 || p.SelectedPassed != max(best, 0) || p.ModelAbstentions != 0 || !reflect.DeepEqual(p.BestCases, bestCases) || !reflect.DeepEqual(p.NewAttempts, newAttempts) || !reflect.DeepEqual(p.Selection, selection) || !reflect.DeepEqual(p.InitialProposals, proposals) || len(p.EligibleProbabilities) != 0 || !p.Initialized || p.Interrupted || p.InitializationError != "" || p.FeedbackRounds != rounds || p.FeedbackPredictions != calls-initialCalls || p.LatestFeedbackSHA != latest || p.ScheduledBytes != 8 || p.FrontierNodes != bits.OnesCount8(q.queued) || p.FrontierStorage < p.FrontierNodes*16 || p.FrontierStorage > 128 || p.Exhausted != (attempted == 8) || p.PredictionsThisAdvance != 0 {
			return fmt.Errorf("three-choice/offline progress %d differs from reconstructed committed state", i)
		}
		previous = p.SHA
	}
	last := c.Progress[len(c.Progress)-1]
	if !reflect.DeepEqual(c.Search.Selection, last.Selection) || !reflect.DeepEqual(c.Search.InitialProposals, proposals) || len(c.Search.EligibleProbabilities) != 0 || c.Search.ModelAbstentionsObserved != 0 {
		return errors.New("final captured search differs")
	}
	return nil
}

func validateThree(r *pathplan.ThreeReceipt, text string, parts [3]string) error {
	if r == nil || r.Schema != jointdecision.ThreeSchema || r.Feature != jointdecision.ThreeFeatureVersion || r.Input != text || r.InputSHA != threecohort.SHA([]byte(text)) || r.Bytes != len(text) || r.Decisions != 3 || r.Calls != 1 || r.Declined || r.Error != "" || !r.PredictionValid || r.PredictNS < 0 {
		return errors.New("full three-choice prediction identity differs")
	}
	for i, part := range parts {
		if r.PartSHA[i] != threecohort.SHA([]byte(part)) {
			return errors.New("ordered complete input identity differs")
		}
	}
	sum, top := 0., 0
	for i, p := range r.Probabilities {
		if p < 0 || p > 1 || math.IsNaN(float64(p)) || math.IsInf(float64(p), 0) {
			return errors.New("invalid full-mask observed probability")
		}
		sum += float64(p)
		if p > r.Probabilities[top] {
			top = i
		}
	}
	if math.Abs(sum-1) > 2e-6 || r.Proposed != uint16(top) || r.Sampled != r.Proposed {
		return errors.New("unseeded complete-mask proposal differs")
	}
	return nil
}

func verifyThreeFailure(v threecohort.View, f pathplan.FeedbackReceipt, p pathplan.SessionProgress, q *frontier) error {
	var first *pathplan.TestResult
	for _, r := range p.BestCases {
		if !r.Passed {
			copy := r
			first = &copy
			break
		}
	}
	if first == nil || !reflect.DeepEqual(first, f.FirstFailure) {
		return errors.New("actual first failed retained case differs")
	}
	if 8-f.Attempted == 1 {
		if !f.RankingUnnecessary || f.Three != nil || f.ContextDeclined || f.ModelCalls != 0 || f.Applied || f.Error != "" {
			return errors.New("sole remaining mask must make zero calls")
		}
		return nil
	}
	projection, err := Derive(v, f, p)
	if err != nil || projection == nil {
		return errors.New("full runtime failure projection required")
	}
	if f.RankingUnnecessary {
		return errors.New("ranking unnecessarily skipped")
	}
	if !projection.Representable {
		r := f.Three
		if r == nil || r.Input != projection.Text || r.InputSHA != projection.InputSHA || r.PartSHA != projection.PartSHA || r.Bytes != projection.Bytes || r.Schema != jointdecision.ThreeSchema || r.Feature != jointdecision.ThreeFeatureVersion || r.Decisions != 3 || r.Calls != 0 || r.PredictNS != 0 || r.Probabilities != ([8]float32{}) || r.PredictionValid || !r.Declined || r.Error != projection.Reason || !f.ContextDeclined || f.ModelCalls != 0 || f.Applied || f.DeclinedDecision != "three-choice" || f.DeclinedBytes != projection.Bytes || f.DeclinedInputSHA != projection.InputSHA || f.DeclinedIntentSHA != threecohort.SHA([]byte(v.Text)) || f.Error != pathplan.ErrFeedbackContextBound.Error() {
			return errors.New("complete zero-call representation decline differs")
		}
		return nil
	}
	if f.ContextDeclined || f.ModelCalls != 1 || !f.Applied || f.Error != "" {
		return errors.New("actual full-mask feedback call differs")
	}
	if err := validateThree(f.Three, projection.Text, projection.Parts); err != nil {
		return err
	}
	for mask, mass := range f.Three.Probabilities {
		q.maskWeights[mask] = math.Log(math.Max(float64(mass), 1e-12))
	}
	return nil
}
