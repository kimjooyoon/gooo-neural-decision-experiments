package main

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

func verifyReferenceProgress(v jointcohort.View, c jointfeedback.Capture, m *model) error {
	if len(c.Progress) != 1+len(c.Search.Attempts)+len(c.Feedback) {
		return errors.New("reference progress count differs")
	}
	caseRaw, err := json.Marshal(v.Cases)
	if err != nil {
		return err
	}
	initialCalls := 0
	if m != nil {
		initialCalls = 2
	}
	attempted, rounds, calls, best := 0, 0, initialCalls, -1
	latest, previous := "", ""
	var bestCases []pathplan.TestResult
	for i, p := range c.Progress {
		if i == 0 {
			if len(p.NewAttempts) != 0 || len(p.BestCases) != 0 {
				return errors.New("reference initial observation already tested cases")
			}
		} else if len(p.NewAttempts) == 1 {
			if attempted >= len(c.Search.Attempts) || !reflect.DeepEqual(p.NewAttempts[0], c.Search.Attempts[attempted]) {
				return errors.New("reference observed attempt differs")
			}
			a := c.Search.Attempts[attempted]
			attempted++
			if a.Passed > best {
				best, bestCases = a.Passed, a.Results
			}
		} else {
			if len(p.NewAttempts) != 0 || rounds >= len(c.Feedback) || c.Feedback[rounds].FromProgressSHA != previous {
				return errors.New("reference feedback observation has no immediate cause")
			}
			f := c.Feedback[rounds]
			rounds++
			calls, latest = f.CumulativeCalls, f.SHA
		}
		if p.Attempted != attempted || p.CaseSHA != jointcohort.SHA(caseRaw) || p.Declared != 4 || p.Unattempted != 4-attempted || p.TypeRejected != 0 || p.Cases != 16 || p.Selection.ModelCalls != calls || p.FeedbackRounds != rounds || p.FeedbackPredictions != calls-initialCalls || p.LatestFeedbackSHA != latest {
			return errors.New("reference progress arithmetic differs")
		}
		if i > 0 && (p.SelectedPassed != best || !reflect.DeepEqual(p.BestCases, bestCases)) {
			return errors.New("reference retained best values differ")
		}
		previous = p.SHA
	}
	if c.Progress[len(c.Progress)-1].Status != "TRAINING_COMPLETE" {
		return errors.New("reference final state incomplete")
	}
	return nil
}
