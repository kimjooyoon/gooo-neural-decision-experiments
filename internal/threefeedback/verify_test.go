package threefeedback

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func capture(t *testing.T, v threecohort.View, m *decision.Model, seedIndex int) Capture {
	t.Helper()
	seed := fmt.Sprintf("three-source-path-teacher-%d", seedIndex)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	s, _, progress, feedback, err := v.Prepared.SearchFeedbackBatchesUnfixed(ctx, m, v.Cases, 8, 1, seed, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := Capture{Schema: "gooo/own-three-choice-teacher-capture/v1", ViewID: v.ID, SourceSHA: v.SourceSHA, InputSHA: threecohort.SHA([]byte(v.Text)), SeedIndex: seedIndex, Seed: seed, WallNS: time.Since(start).Nanoseconds(), Search: s, Progress: progress, Feedback: feedback, Derived: []Projection{}, TeacherInputs: []TeacherInput{}}
	for i, f := range feedback {
		inputs, err := AttemptedInputs(v, f, progress[2*i+1])
		if err != nil {
			t.Fatal(err)
		}
		c.TeacherInputs = append(c.TeacherInputs, inputs...)
		p, err := Derive(v, f, progress[2*i+1])
		if err != nil {
			t.Fatal(err)
		}
		if p != nil {
			c.Derived = append(c.Derived, *p)
		}
	}
	return c
}

func TestActualIndependentTeacherTraceAndMutations(t *testing.T) {
	m, err := decision.LoadPath("../../models/joint-composition-v1/independent/models/fp32/model.json")
	if err != nil {
		t.Fatal(err)
	}
	if m.MetadataSHA256() != TeacherMetadata || m.WeightsSHA256() != TeacherWeights {
		t.Fatal("teacher pins differ")
	}
	f, err := os.Open("testdata/native-goal7-rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), 1<<20)
	sessions, calls, attempts := 0, 0, 0
	var withFeedback *Capture
	var chosen threecohort.View
	for scan.Scan() {
		v, err := threecohort.ReconstructRow(scan.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		for seed := range 2 {
			c := capture(t, v, m, seed)
			if err = Verify(v, c); err != nil {
				t.Fatalf("%s seed%d: %v", v.ID, seed, err)
			}
			// Round-trip hashes and probabilities must survive actual JSON storage.
			raw, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			var saved Capture
			if err = threecohort.Decode(raw, &saved); err != nil {
				t.Fatal(err)
			}
			if err = Verify(v, saved); err != nil {
				t.Fatal(err)
			}
			sessions++
			calls += c.Search.Selection.ModelCalls
			attempts += len(c.Search.Attempts)
			if withFeedback == nil && len(c.Feedback) > 0 {
				copy := c
				withFeedback = &copy
				chosen = v
			}
		}
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err)
	}
	if sessions != 32 || withFeedback == nil {
		t.Fatal("declared representative validation scope missing")
	}
	t.Logf("validation only: sessions=%d predictions=%d candidates=%d; distinct from planned 4096 training captures", sessions, calls, attempts)
	raw, err := json.Marshal(withFeedback)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Capture){
		func(c *Capture) { c.Search.Attempts[0].Results[0].Actual++ },
		func(c *Capture) { c.Progress[0].Selection.ModelCalls = 1 },
		func(c *Capture) { c.Progress[0].InitialProposals["choice0"] = "outside_vocabulary" },
		func(c *Capture) { c.Feedback[0].CI = &pathplan.CIHint{Status: "PASS"} },
		func(c *Capture) { c.Feedback[0].ModelCalls++ },
		func(c *Capture) { c.Feedback[0].FirstFailure.Expected++ },
		func(c *Capture) { c.Derived[0].Parts[0] += " future result=PASS" },
		func(c *Capture) { c.TeacherInputs[0].Text += " invalid shortened source" },
		func(c *Capture) { c.Derived[0].Representable = !c.Derived[0].Representable },
		func(c *Capture) { c.SeedIndex = 2 },
	} {
		var c Capture
		if err = threecohort.Decode(raw, &c); err != nil {
			t.Fatal(err)
		}
		mutate(&c)
		if err = Verify(chosen, c); err == nil {
			t.Fatal("mutated actual trace accepted")
		}
	}
}
