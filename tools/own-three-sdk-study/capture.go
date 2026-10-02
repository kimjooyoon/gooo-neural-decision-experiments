package main

import (
	"context"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

type observation struct {
	Policy  string                `json:"policy"`
	Capture threefeedback.Capture `json:"capture"`
}

func one(v threecohort.View, m *model) (threefeedback.Capture, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	var s pathplan.SearchResult
	var progress []pathplan.SessionProgress
	var feedback []pathplan.FeedbackReceipt
	var err error
	if m == nil {
		s, _, progress, err = v.Prepared.SearchBatches(ctx, nil, v.Cases, 8, 1, "")
	} else if m.Three != nil {
		s, _, progress, feedback, err = v.Prepared.SearchThreeFeedbackBatches(ctx, m.Three, v.Cases, 8, 1, "", 7, nil)
	} else {
		s, _, progress, feedback, err = v.Prepared.SearchFeedbackBatchesUnfixed(ctx, m.Independent, v.Cases, 8, 1, "", 7, nil)
	}
	c := threefeedback.Capture{Schema: "gooo/own-three-choice-sdk-capture/v1", ViewID: v.ID, SourceSHA: v.SourceSHA, InputSHA: threecohort.SHA([]byte(v.Text)), SeedIndex: -1, WallNS: time.Since(start).Nanoseconds(), Search: s, Progress: progress, Feedback: feedback, Derived: []threefeedback.Projection{}, TeacherInputs: []threefeedback.TeacherInput{}}
	if err != nil {
		c.RuntimeError = err.Error()
		return c, err
	}
	if m != nil && m.Independent != nil {
		for i, f := range feedback {
			inputs, e := threefeedback.AttemptedInputs(v, f, progress[2*i+1])
			if e != nil {
				return c, e
			}
			c.TeacherInputs = append(c.TeacherInputs, inputs...)
			projection, e := threefeedback.Derive(v, f, progress[2*i+1])
			if e != nil {
				return c, e
			}
			if projection != nil {
				c.Derived = append(c.Derived, *projection)
			}
		}
	}
	return c, nil
}
func verify(v threecohort.View, c threefeedback.Capture, m *model) error {
	if m == nil {
		return threefeedback.VerifyThreeObservation(v, c, "", "", "")
	}
	if m.Independent != nil {
		return threefeedback.VerifyIndependentObservation(v, c)
	}
	return threefeedback.VerifyThreeObservation(v, c, m.Pin.Metadata, m.Pin.Weights, m.Variant)
}
