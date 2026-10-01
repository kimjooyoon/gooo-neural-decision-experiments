// Package jointfeedback retains actual own-model SDK failures as student data.
package jointfeedback

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const TeacherMetadata = "bbad1ae3bd5fbee3e782be40ea75c656be7f33f554ea64a79f034dc74669d931"
const TeacherWeights = "e1576dd7e7f206e60d1523eff3bb0af2b33c969c7021a502133d980de4985cf6"
const ProtocolSHA = "aad8b228c7c647690abbccd628473f7757517ba09d689a8d46e0e8c180ba6a28"
const Protocol = "docs/own-joint-completeness-feedback-preregistration-20261002.md"

type Capture struct {
	Schema    string                     `json:"schema"`
	ViewID    string                     `json:"function_view"`
	SourceSHA string                     `json:"original_source_sha256"`
	JointSHA  string                     `json:"original_joint_input_sha256"`
	Rotation  int                        `json:"rotation"`
	Seed      string                     `json:"seed"`
	WallNS    int64                      `json:"sdk_session_wall_ns"`
	Search    pathplan.SearchResult      `json:"search"`
	Progress  []pathplan.SessionProgress `json:"progress"`
	Feedback  []pathplan.FeedbackReceipt `json:"feedback"`
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
	Text      string     `json:"complete_joint_input"`
	InputSHA  string     `json:"input_sha256"`
	Target    [4]float32 `json:"finite_soft_targets"`
	Origins   []Origin   `json:"actual_feedback_origins"`
}

func Initial(v jointcohort.View) State {
	return State{ID: "initial/" + v.ID, ViewID: v.ID, Group: v.Group, Family: v.Family, Config: v.Config,
		Language: v.Language, Split: v.Split, Phase: "initial", SourceSHA: v.SourceSHA, Text: v.JointInput,
		InputSHA: jointcohort.SHA([]byte(v.JointInput)), Target: v.Target.Joint, Origins: []Origin{}}
}

func AddStates(states map[string]*State, v jointcohort.View, capture Capture, raw []byte, index int) {
	for _, f := range capture.Feedback {
		if f.Joint == nil || f.ModelCalls != 1 || !f.Applied || f.ContextDeclined || f.RankingUnnecessary || f.Error != "" {
			continue
		}
		inputSHA := jointcohort.SHA([]byte(f.Joint.Input))
		id := "feedback/" + v.ID + "/" + inputSHA
		s, exists := states[id]
		if !exists {
			initial := Initial(v)
			initial.ID, initial.Text, initial.InputSHA, initial.Phase = id, f.Joint.Input, inputSHA, "feedback"
			s = &initial
			states[id] = s
		}
		s.Origins = append(s.Origins, Origin{index, jointcohort.SHA(raw), f.Round, f.SHA, f.FromProgressSHA})
	}
}
