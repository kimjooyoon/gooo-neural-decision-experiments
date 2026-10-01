package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type sessionComposeResult struct {
	Schema     string                   `json:"schema"`
	Progress   pathplan.SessionProgress `json:"progress"`
	GoooSource string                   `json:"gooo_source,omitempty"`
	GoSource   string                   `json:"go_source,omitempty"`
	Error      string                   `json:"error,omitempty"`
}

func emitSession(output io.Writer, progress pathplan.SessionProgress, body *bodyplan.Program, failure error) error {
	value := sessionComposeResult{Schema: "gooo/typed-path-session-compose-result/v1", Progress: progress}
	if body != nil {
		value.GoooSource, value.GoSource = body.GoooSource(), body.GoSource()
	}
	if failure != nil {
		value.Error = failure.Error()
	}
	return json.NewEncoder(output).Encode(value)
}
func composeSession(ctx context.Context, prepared *pathplan.PreparedPlan, model *decision.Model, cases []pathplan.TestCase, total, step int, seed string, output io.Writer) error {
	return composeSessionFeedback(ctx, prepared, model, cases, total, step, seed, 0, nil, output)
}

func composeSessionFeedback(ctx context.Context, prepared *pathplan.PreparedPlan, model *decision.Model, cases []pathplan.TestCase, total, step int, seed string, rounds int, ci *pathplan.CIHint, output io.Writer) error {
	session, startErr := prepared.NewSession(ctx, model, cases, seed)
	if session == nil {
		return startErr
	}
	initial, err := session.Observe()
	if err != nil {
		return err
	}
	if err := emitSession(output, initial, nil, startErr); err != nil {
		return err
	}
	if startErr != nil {
		return startErr
	}
	attempted := 0
	for attempted < total {
		progress, body, failure := session.Advance(ctx, min(step, total-attempted))
		if progress.Schema != "" {
			if err := emitSession(output, progress, body, failure); err != nil {
				return err
			}
			attempted = progress.Attempted
		}
		if failure != nil && !errors.Is(failure, pathplan.ErrNoTypedCandidate) {
			return failure
		}
		if progress.Status == "TRAINING_COMPLETE" || progress.Exhausted {
			return failure
		}
		if len(progress.NewAttempts) == 0 {
			return errors.New("incremental search made no candidate progress")
		}
		if attempted >= total {
			return failure
		}
		if rounds > 0 {
			receipt, err := session.Reconsider(ctx, model, ci)
			if receipt.Schema != "" {
				if writeErr := json.NewEncoder(output).Encode(receipt); writeErr != nil {
					return writeErr
				}
			}
			if err != nil {
				return err
			}
			rounds--
		}
	}
	return nil
}
