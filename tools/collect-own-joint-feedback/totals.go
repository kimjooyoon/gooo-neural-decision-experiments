package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

type totals struct {
	Sessions        int    `json:"actual_sdk_sessions"`
	Predictions     int    `json:"actual_teacher_predictions"`
	InitialCalls    int    `json:"initial_teacher_predictions"`
	FeedbackCalls   int    `json:"feedback_teacher_predictions"`
	Attempts        int    `json:"actual_candidate_attempts"`
	CaseInvocations int    `json:"actual_ordered_evaluator_invocations"`
	Declines        int    `json:"zero_call_context_declines"`
	Unnecessary     int    `json:"sole_remaining_zero_call_feedback"`
	InitialComplete int    `json:"initial_complete_functions"`
	Curve           [4]int `json:"complete_functions_by_budgets_1_to_4"`
}

func (t *totals) add(c jointfeedback.Capture) {
	t.Sessions++
	t.InitialCalls++
	t.Predictions += c.Search.Selection.ModelCalls
	t.Attempts += len(c.Search.Attempts)
	t.CaseInvocations += 16 * len(c.Search.Attempts)
	if len(c.Search.Attempts) == 1 {
		t.InitialComplete++
	}
	for i := len(c.Search.Attempts) - 1; i < 4; i++ {
		t.Curve[i]++
	}
	for _, f := range c.Feedback {
		t.FeedbackCalls += f.ModelCalls
		if f.ContextDeclined {
			t.Declines++
		}
		if f.RankingUnnecessary {
			t.Unnecessary++
		}
	}
}

func fileSHA(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	if _, err = io.CopyBuffer(h, f, buffer[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
