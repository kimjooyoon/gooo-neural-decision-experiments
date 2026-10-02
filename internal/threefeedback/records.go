// Package threefeedback retains real independent-teacher observations and
// derives complete runtime-shaped three-choice student contexts from failures.
package threefeedback

import (
	"fmt"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

const TeacherMetadata = "bb8e6fb5d9d0b4d660ac862f8c0e6d0cbe91748cf78b491ea46b031dee1d25e8"
const TeacherWeights = "30367ac83e9d1129ea48dd1e108a3e25d6ed083c1782486cf99dcdbb8b78f6c4"
const Protocol = "docs/own-three-choice-completeness-preregistration-20261002.md"
const ProtocolSHA = "5d840f7b3379339d684b85b7896a895538bd1a030395a096af3e52c5fc781cd3"
const ProtocolRevision = "d74a8a455ceed5949fcbad482375405b4704dc9a"

type Projection struct {
	Round         int       `json:"feedback_round"`
	FeedbackSHA   string    `json:"actual_teacher_feedback_sha256"`
	ProgressSHA   string    `json:"actual_teacher_from_progress_sha256"`
	Prefix        string    `json:"observed_failure_context"`
	Text          string    `json:"complete_three_input"`
	Parts         [3]string `json:"complete_ordered_inputs"`
	InputSHA      string    `json:"input_sha256"`
	PartSHA       [3]string `json:"ordered_input_sha256"`
	Bytes         int       `json:"complete_input_bytes"`
	PartBytes     [3]int    `json:"ordered_input_bytes"`
	Representable bool      `json:"representable_without_truncation"`
	Reason        string    `json:"representation_error,omitempty"`
	Scope         string    `json:"scope"`
}
type Capture struct {
	Schema        string                     `json:"schema"`
	ViewID        string                     `json:"function_view"`
	SourceSHA     string                     `json:"original_source_sha256"`
	InputSHA      string                     `json:"original_three_input_sha256"`
	SeedIndex     int                        `json:"seed_index"`
	Seed          string                     `json:"seed"`
	WallNS        int64                      `json:"sdk_session_wall_ns"`
	RuntimeError  string                     `json:"runtime_error,omitempty"`
	Search        pathplan.SearchResult      `json:"search"`
	Progress      []pathplan.SessionProgress `json:"progress"`
	Feedback      []pathplan.FeedbackReceipt `json:"feedback"`
	Derived       []Projection               `json:"derived_student_contexts"`
	TeacherInputs []TeacherInput             `json:"reconstructed_attempted_teacher_inputs"`
}

type TeacherInput struct {
	Round       int    `json:"feedback_round"`
	DecisionID  string `json:"decision_id"`
	Text        string `json:"complete_input"`
	SHA         string `json:"input_sha256"`
	Bytes       int    `json:"input_bytes"`
	OriginalSHA string `json:"original_intent_sha256"`
	Predicted   bool   `json:"actual_prediction_receipt_present"`
	Declined    bool   `json:"actual_context_decline_receipt_present"`
	Scope       string `json:"scope"`
}

// AttemptedInputs reconstructs the independent runtime's full individual texts,
// including its first oversized zero-call input. Only actual judgments mark a
// prediction; a reconstruction is not itself evidence of another model call.
func AttemptedInputs(v threecohort.View, f pathplan.FeedbackReceipt, p pathplan.SessionProgress) ([]TeacherInput, error) {
	result := []TeacherInput{}
	if f.RankingUnnecessary {
		return result, nil
	}
	first := f.FirstFailure
	if first == nil {
		return result, fmt.Errorf("actual failure required for teacher input reconstruction")
	}
	prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d", f.Attempted, f.Passed, f.Cases, f.TypeRejected, 8-f.Attempted, first.Input, first.Actual, first.Expected)
	fixed := map[string]bool{}
	for _, c := range f.FixedCoordinates {
		fixed[c.DecisionID] = true
	}
	for _, choice := range v.Plan.Decisions {
		if fixed[choice.ID] {
			continue
		}
		text, err := decision.SemanticContextFeedbackInput(choice.Intent, prefix+" selected="+p.Selection.Choices[choice.ID])
		if err != nil {
			return result, err
		}
		r := TeacherInput{Round: f.Round, DecisionID: choice.ID, Text: text, SHA: threecohort.SHA([]byte(text)), Bytes: len(text), OriginalSHA: threecohort.SHA([]byte(choice.Intent)), Declined: f.ContextDeclined && f.DeclinedDecision == choice.ID, Scope: "Full individual teacher context reconstructed from the actual source/progress/feedback receipt; predictions are counted only by actual judgment receipts."}
		for _, j := range f.Judgments {
			if j.DecisionID == choice.ID {
				r.Predicted = true
			}
		}
		result = append(result, r)
		if len(text) > decision.InputMaxBytes {
			break
		}
	}
	return result, nil
}

type Origin struct {
	SessionIndex int    `json:"session_index"`
	SessionSHA   string `json:"session_sha256"`
	Round        int    `json:"feedback_round"`
	FeedbackSHA  string `json:"feedback_sha256"`
	ProgressSHA  string `json:"from_progress_sha256"`
}
type State struct {
	ID        string     `json:"id"`
	ViewID    string     `json:"function_view"`
	Group     string     `json:"program_contract_group"`
	Family    string     `json:"family"`
	Config    int        `json:"configuration"`
	Language  string     `json:"language"`
	Split     string     `json:"split"`
	Phase     string     `json:"phase"`
	SourceSHA string     `json:"original_source_sha256"`
	Text      string     `json:"complete_three_input"`
	InputSHA  string     `json:"input_sha256"`
	Target    [8]float32 `json:"finite_soft_targets"`
	Origins   []Origin   `json:"actual_feedback_origins"`
}

func Initial(v threecohort.View) State {
	return State{"initial/" + v.ID, v.ID, v.Group, v.Family, v.Config, v.Language, v.Split, "initial", v.SourceSHA, v.Text, threecohort.SHA([]byte(v.Text)), v.Target.Joint, []Origin{}}
}

// Derive uses the three-choice runtime's complete ordered selected-label prefix,
// rather than pretending the independent teacher made a joint prediction.
// A sole remaining mask needs no student ranking and supplies no training row.
func Derive(v threecohort.View, f pathplan.FeedbackReceipt, p pathplan.SessionProgress) (*Projection, error) {
	if 8-f.Attempted == 1 {
		return nil, nil
	}
	first := f.FirstFailure
	if first == nil {
		return nil, fmt.Errorf("actual selected failure required")
	}
	prefix := fmt.Sprintf("feedback: tried=%d passed=%d/%d rejected=%d remaining=%d mismatch=%d:%d:%d", f.Attempted, f.Passed, f.Cases, f.TypeRejected, 8-f.Attempted, first.Input, first.Actual, first.Expected)
	var selected [3]string
	for i, c := range v.Plan.Decisions {
		selected[i] = p.Selection.Choices[c.ID]
	}
	prefix += " selected=" + strings.Join(selected[:], ",")
	text, parts, err := jointdecision.FeedbackThreeWithParts(v.Text, prefix)
	if err != nil {
		return nil, err
	}
	r := Projection{Round: f.Round, FeedbackSHA: f.SHA, ProgressSHA: p.SHA, Prefix: prefix, Text: text, Parts: parts, InputSHA: threecohort.SHA([]byte(text)), Bytes: len(text), Scope: "Derived complete three-choice runtime context from actual independent-teacher failure; zero student model calls. No gold mask, future result or CI hint is added."}
	for i, part := range parts {
		r.PartSHA[i] = threecohort.SHA([]byte(part))
		r.PartBytes[i] = len(part)
	}
	var features [jointdecision.ThreeFeatureDim]float32
	if err = jointdecision.FeaturesIntoThree(text, &features); err != nil {
		r.Reason = err.Error()
	} else {
		r.Representable = true
	}
	return &r, nil
}

func AddStates(states map[string]*State, v threecohort.View, c Capture, raw []byte, index int) {
	for _, p := range c.Derived {
		if !p.Representable {
			continue
		}
		id := "feedback/" + v.ID + "/" + p.InputSHA
		s, found := states[id]
		if !found {
			initial := Initial(v)
			initial.ID, initial.Text, initial.InputSHA, initial.Phase = id, p.Text, p.InputSHA, "feedback"
			s = &initial
			states[id] = s
		}
		s.Origins = append(s.Origins, Origin{index, threecohort.SHA(raw), p.Round, p.FeedbackSHA, p.ProgressSHA})
	}
}
