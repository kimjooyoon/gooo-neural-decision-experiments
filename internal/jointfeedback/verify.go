package jointfeedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func Verify(v jointcohort.View, c Capture) error {
	if v.Split != "train" || c.Schema != "gooo/own-joint-feedback-teacher-capture/v2" || c.ViewID != v.ID || c.SourceSHA != v.SourceSHA || c.JointSHA != jointcohort.SHA([]byte(v.JointInput)) || c.Rotation < 0 || c.Rotation > 3 || c.Seed != fmt.Sprintf("own-feedback-teacher/v2/rotation-%d", c.Rotation) || c.WallNS <= 0 {
		return errors.New("training-only teacher binding differs")
	}
	s := c.Search
	if s.Status != "TRAINING_COMPLETE" || s.DeclaredCombinations != 4 || s.TrainingTotal != 16 || s.TypeRejected != 0 || s.Evaluated != len(s.Attempts) || len(s.Attempts) < 1 || len(s.Attempts) > 4 || s.SelectedTrainingPassed != 16 || s.Selection.MetadataSHA256 != TeacherMetadata || s.Selection.WeightsSHA256 != TeacherWeights || s.Selection.PlanSHA256 != v.Prepared.PlanSHA256() || s.Selection.SeedSHA256 != jointcohort.SHA([]byte(c.Seed)) {
		return errors.New("actual teacher search bounds or pins differ")
	}
	if err := verifyAttempts(v, s); err != nil {
		return err
	}
	progress, err := verifyProgress(v, c)
	if err != nil {
		return err
	}
	return verifyFeedback(v, c, progress)
}

func verifyAttempts(v jointcohort.View, s pathplan.SearchResult) error {
	plan, err := jointcompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
	if err != nil {
		return err
	}
	seen := uint8(0)
	for i, a := range s.Attempts {
		if a.Mask > 3 || seen&(1<<a.Mask) != 0 || a.Status != "EVALUATED" || a.Total != 16 || len(a.Results) != 16 {
			return errors.New("finite unique teacher attempts required")
		}
		seen |= 1 << a.Mask
		choices, e := jointcompositionstudy.Choices(plan, int(a.Mask))
		body, e2 := v.Prepared.Compile(choices)
		if e != nil || e2 != nil || !reflect.DeepEqual(a.Choices, choices) || a.GoooSHA != jointcohort.SHA([]byte(body.GoooSource())) {
			return errors.New("teacher emitted fragment identity differs")
		}
		passed := 0
		for j, test := range v.Cases {
			actual, e := jointcompositionstudy.Oracle(v.Family, v.Config, int(a.Mask), test.Input)
			if e != nil || a.Results[j] != (pathplan.TestResult{Input: test.Input, Expected: test.Expected, Actual: actual, Passed: actual == test.Expected}) {
				return errors.New("teacher actual ordered values differ from independent oracle")
			}
			if actual == test.Expected {
				passed++
			}
		}
		if a.Passed != passed || (i < len(s.Attempts)-1 && passed == 16) || (i == len(s.Attempts)-1 && passed != 16) {
			return errors.New("teacher finite acceptance or stopping differs")
		}
	}
	return nil
}

func verifyProgress(v jointcohort.View, c Capture) (map[string]pathplan.SessionProgress, error) {
	if len(c.Progress) != 1+len(c.Search.Attempts)+len(c.Feedback) {
		return nil, errors.New("initial, committed attempt and feedback observations required")
	}
	casesRaw, _ := json.Marshal(v.Cases)
	caseSHA := jointcohort.SHA(casesRaw)
	previous, bestPassed, bestCases := "", -1, []pathplan.TestResult(nil)
	bySHA := map[string]pathplan.SessionProgress{}
	attempted, calls, rounds, latest := 0, 1, 0, ""
	for i, p := range c.Progress {
		sha := p.SHA
		p.SHA = ""
		raw, err := json.Marshal(p)
		if i%2 == 1 {
			attempted++
		} else if i > 0 {
			if (i-2)/2 >= len(c.Feedback) {
				return nil, errors.New("missing teacher feedback observation")
			}
			f := c.Feedback[(i-2)/2]
			calls, rounds, latest = f.CumulativeCalls, f.Round, f.SHA
		}
		if err != nil || sha == "" || jointcohort.SHA(raw) != sha || p.Sequence != i+1 || p.PreviousSHA != previous || p.CaseSHA != caseSHA || p.Selection.PlanSHA256 != v.Prepared.PlanSHA256() || p.Attempted != attempted || p.Declared != 4 || p.Unattempted != 4-attempted || p.TypeRejected != 0 || p.Cases != 16 || p.Selection.ModelCalls != calls || p.FeedbackRounds != rounds || p.LatestFeedbackSHA != latest || p.FeedbackPredictions != calls-1 {
			return nil, errors.New("teacher progress chain differs")
		}
		if i%2 == 1 {
			a := c.Search.Attempts[attempted-1]
			if len(p.NewAttempts) != 1 || !reflect.DeepEqual(p.NewAttempts[0], a) {
				return nil, errors.New("teacher committed attempt differs")
			}
			if a.Passed > bestPassed {
				bestPassed, bestCases = a.Passed, a.Results
			}
		} else if len(p.NewAttempts) != 0 {
			return nil, errors.New("observation invented teacher attempts")
		}
		if i > 0 && (p.SelectedPassed != bestPassed || !reflect.DeepEqual(p.BestCases, bestCases)) {
			return nil, errors.New("teacher retained best case values differ")
		}
		if i == 0 && (p.Attempted != 0 || p.Selection.Joint == nil || p.Selection.Joint.Input != v.JointInput || p.Selection.Joint.Calls != 1 || p.Selection.ModelCalls != 1) {
			return nil, errors.New("teacher initial judgment preceded by tests or lost input")
		}
		p.SHA = sha
		bySHA[sha] = p
		previous = sha
	}
	last := c.Progress[len(c.Progress)-1]
	if !reflect.DeepEqual(last.Selection, c.Search.Selection) || last.Status != "TRAINING_COMPLETE" {
		return nil, errors.New("teacher final progress differs")
	}
	return bySHA, nil
}

func verifyFeedback(v jointcohort.View, c Capture, progress map[string]pathplan.SessionProgress) error {
	previous, calls := "", 1
	for i, f := range c.Feedback {
		sha := f.SHA
		f.SHA = ""
		raw, err := json.Marshal(f)
		p, ok := progress[f.FromProgressSHA]
		if err != nil || !ok || 2*i+1 >= len(c.Progress) || f.FromProgressSHA != c.Progress[2*i+1].SHA || sha == "" || jointcohort.SHA(raw) != sha || f.Round != i+1 || f.PreviousSHA != previous || f.MetadataSHA != TeacherMetadata || f.WeightsSHA != TeacherWeights || f.PlanSHA != v.Prepared.PlanSHA256() || f.CaseSHA != p.CaseSHA || f.Attempted != p.Attempted || f.Passed != p.SelectedPassed || f.Cases != 16 || f.TypeRejected != 0 || f.CI != nil || f.CIIsAuthority || p.Status == "TRAINING_COMPLETE" {
			return errors.New("teacher feedback cause or receipt differs")
		}
		var first *pathplan.TestResult
		for _, test := range p.BestCases {
			if !test.Passed {
				copy := test
				first = &copy
				break
			}
		}
		if first == nil || !reflect.DeepEqual(first, f.FirstFailure) {
			return errors.New("teacher feedback has no actual selected failure")
		}
		if 4-f.Attempted == 1 {
			if !f.RankingUnnecessary || f.ModelCalls != 0 || f.Joint != nil {
				return errors.New("sole remaining teacher path unnecessarily predicted")
			}
		} else if err = verifyFeedbackText(v, f, p); err != nil {
			return err
		}
		calls += f.ModelCalls
		if f.CumulativeCalls != calls {
			return errors.New("teacher cumulative prediction count differs")
		}
		previous = sha
	}
	if calls != c.Search.Selection.ModelCalls || len(c.Feedback) != len(c.Search.Attempts)-1 {
		return errors.New("teacher prediction/feedback totals differ")
	}
	return nil
}

func verifyFeedbackText(v jointcohort.View, f pathplan.FeedbackReceipt, p pathplan.SessionProgress) error {
	x := f.FirstFailure
	prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d", f.Attempted, f.Passed, f.Cases, f.TypeRejected, 4-f.Attempted, x.Input, x.Actual, x.Expected)
	plan, err := jointcompositionstudy.Fixture(v.Family, v.Config, v.Goal, v.Language)
	if err != nil {
		return err
	}
	labels := []string{p.Selection.Choices[plan.Decisions[0].ID], p.Selection.Choices[plan.Decisions[1].ID]}
	text, err := jointdecision.Feedback(v.JointInput, prefix+" selected="+strings.Join(labels, ","))
	if err != nil || f.Joint == nil || f.Joint.Input != text || f.Joint.InputSHA != jointcohort.SHA([]byte(text)) || f.Joint.Bytes != len(text) || f.Joint.Calls != f.ModelCalls {
		return errors.New("full original teacher input/actual feedback differs")
	}
	var vector [jointdecision.FeatureDim]float32
	err = jointdecision.FeaturesInto(text, &vector)
	if err != nil {
		if !f.ContextDeclined || f.ModelCalls != 0 || !f.Joint.Declined || f.Error == "" {
			return errors.New("teacher overflow requires recorded zero-call decline")
		}
	} else if f.ContextDeclined || f.ModelCalls != 1 || !f.Applied || f.Joint.Declined || f.Error != "" {
		return errors.New("valid teacher feedback prediction required")
	}
	return nil
}
