package jointfeedback

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func fixture(t *testing.T) (jointcohort.View, *jointdecision.Model) {
	t.Helper()
	p, err := jointcompositionstudy.Fixture("schedule_operand", 0, 1, "en")
	if err != nil {
		t.Fatal(err)
	}
	original, err := pathplan.Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range p.Decisions {
		fields, e := original.SourceFeatures(c.ID)
		if e != nil {
			t.Fatal(e)
		}
		text, e := decision.EncodeSemanticContextInput(fields, c.Intent[strings.LastIndex(c.Intent, "intent: ")+8:])
		if e != nil {
			t.Fatal(e)
		}
		p.Decisions[i].Intent = text
	}
	prepared, err := pathplan.Prepare(p)
	if err != nil {
		t.Fatal(err)
	}
	text, err := prepared.JointInput()
	if err != nil {
		t.Fatal(err)
	}
	cases, err := jointcompositionstudy.Cases("schedule_operand", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	target, err := jointcompositionstudy.Target(p, "schedule_operand", 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	m, err := jointdecision.Load("../../models/joint-composition-v1/joint/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	return jointcohort.View{ID: "fixture/en", Group: "fixture", Family: "schedule_operand", Config: 0, Goal: 1, Language: "en", Split: "train", SourceSHA: jointcohort.SHA([]byte("unit fixture")), JointInput: text, Target: target, Prepared: prepared, Cases: cases}, m
}

func capture(t *testing.T) (jointcohort.View, Capture) {
	t.Helper()
	v, m := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for rotation := 0; rotation < 4; rotation++ {
		seed := fmt.Sprintf("own-feedback-teacher/v2/rotation-%d", rotation)
		s, _, p, f, err := v.Prepared.SearchJointFeedbackBatches(ctx, m, v.Cases, 4, 1, seed, 3, nil)
		if err != nil {
			t.Fatal(err)
		}
		c := Capture{Schema: "gooo/own-joint-feedback-teacher-capture/v2", ViewID: v.ID, SourceSHA: v.SourceSHA, JointSHA: jointcohort.SHA([]byte(v.JointInput)), Rotation: rotation, Seed: seed, WallNS: 1, Search: s, Progress: p, Feedback: f}
		if err = Verify(v, c); err != nil {
			t.Fatal(err)
		}
		if len(f) > 0 && f[0].ModelCalls == 1 {
			return v, c
		}
	}
	t.Fatal("fixed own teacher did not produce a test continuation")
	return v, Capture{}
}

func TestActualTeacherFailuresAndStateOrigins(t *testing.T) {
	v, c := capture(t)
	raw, _ := json.Marshal(c)
	states := map[string]*State{}
	AddStates(states, v, c, raw, 7)
	AddStates(states, v, c, raw, 8)
	if len(states) == 0 {
		t.Fatal("actual continuation was lost")
	}
	for _, s := range states {
		if s.Phase != "feedback" || s.Split != "train" || s.Target != v.Target.Joint || len(s.Origins) != 2 || s.Origins[0].SessionIndex != 7 || s.Origins[1].SessionIndex != 8 || s.Origins[0].SessionSHA != jointcohort.SHA(raw) {
			t.Fatal("deduplicated states lost their actual origins")
		}
		var vector [jointdecision.FeatureDim]float32
		if err := jointdecision.FeaturesInto(s.Text, &vector); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTeacherTamperingAndEvaluationSplitReject(t *testing.T) {
	v, c := capture(t)
	for _, mutate := range []func(*Capture){
		func(x *Capture) { x.SourceSHA = "changed" },
		func(x *Capture) { x.Search.Attempts[0].Results[0].Actual++ },
		func(x *Capture) { x.Feedback[0].FirstFailure.Expected++ },
		func(x *Capture) { x.Progress[1].PreviousSHA = "changed" },
		func(x *Capture) { x.Search.Selection.ModelCalls++ },
	} {
		raw, _ := json.Marshal(c)
		var changed Capture
		if err := json.Unmarshal(raw, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		if err := Verify(v, changed); err == nil {
			t.Fatal("tampered actual teacher record accepted")
		}
	}
	v.Split = "development"
	if err := Verify(v, c); err == nil {
		t.Fatal("development teacher entered training data")
	}
}
