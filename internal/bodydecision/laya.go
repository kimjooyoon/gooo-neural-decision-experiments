package bodydecision

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/bodyplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/strictjson"
)

type Exchange struct {
	HoleID   string          `json:"hole_id"`
	Request  json.RawMessage `json:"request"`
	Response []byte          `json:"response_bytes_base64,omitempty"`
	Status   string          `json:"status"`
	WallNS   int64           `json:"wall_ns"`
}

// ChooseLaya is an optional Go HTTP connector to the separately installed Laya
// reference service. It does not put Python in the tiny-model execution path.
// Requests contain only synthetic hole text and the declared operation choices.
// No retries are performed; unresolved calls return partial capture evidence.
func ChooseLaya(ctx context.Context, plan bodyplan.Plan, endpoint string) (Selection, []Exchange, error) {
	if ctx == nil {
		return Selection{}, nil, errors.New("Laya request requires a context")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	choices, err := Validate(plan)
	if err != nil {
		return Selection{}, nil, err
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.Path != "/v1/systemone" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return Selection{}, nil, errors.New("Laya study endpoint must be loopback HTTP /v1/systemone")
	}
	ip := net.ParseIP(parsed.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return Selection{}, nil, errors.New("Laya study endpoint must use a literal loopback address")
	}
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	defer client.CloseIdleConnections()
	result := Selection{Choices: choices, ModelVariant: "laya_multilingual", Holes: make([]HoleReceipt, 0, len(choices))}
	var exchanges []Exchange
	descriptions := map[string]string{
		"add":        "Add the left and right values: left + right.",
		"subtract":   "Subtract the right value from the left: left - right.",
		"multiply":   "Multiply the two values: left * right.",
		"less_than":  "Test whether left is strictly smaller than right: left < right.",
		"less_equal": "Test whether left is smaller than or equal to right: left <= right.",
		"equal":      "Test whether both values are equal: left == right.",
		"and":        "Require both Boolean values to be true: left && right.",
		"or":         "Require at least one Boolean value to be true: left || right.",
	}
	for _, expr := range plan.Expressions {
		if expr.Kind != "hole" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return result, exchanges, err
		}
		criteria := make(map[string]string, len(expr.Allowed))
		for _, operation := range expr.Allowed {
			criteria[operation] = descriptions[operation]
		}
		payload := struct {
			Model     string            `json:"model"`
			State     map[string]string `json:"state"`
			Questions map[string]any    `json:"questions"`
		}{"multilingual", map[string]string{"request": expr.Text}, map[string]any{"operation": map[string]any{"type": "choice", "instructions": "Choose the declared binary operation matching the instruction. Return one listed operation.", "criteria": criteria}}}
		body, err := json.Marshal(payload)
		if err != nil {
			return result, exchanges, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return result, exchanges, err
		}
		request.Header.Set("Content-Type", "application/json")
		exchanges = append(exchanges, Exchange{HoleID: expr.HoleID, Request: body, Status: "unresolved"})
		capture := &exchanges[len(exchanges)-1]
		started := time.Now()
		response, requestErr := client.Do(request)
		result.ProviderCalls++
		if requestErr != nil {
			capture.WallNS = time.Since(started).Nanoseconds()
			return result, exchanges, requestErr
		}
		raw, readErr := io.ReadAll(io.LimitReader(response.Body, 65537))
		closeErr := response.Body.Close()
		capture.WallNS = time.Since(started).Nanoseconds()
		capture.Response = append([]byte(nil), raw...)
		// A fully buffered response can win the transport's cancellation race.
		// Keep its bytes as partial evidence without committing a validated choice.
		if err := ctx.Err(); err != nil {
			return result, exchanges, err
		}
		if readErr != nil || closeErr != nil || len(raw) > 65536 || response.StatusCode != http.StatusOK {
			return result, exchanges, errors.New("Laya response failed or exceeded 65536 bytes")
		}
		var rawFields map[string]json.RawMessage
		if err := strictjson.Decode(raw, &rawFields); err != nil {
			return result, exchanges, err
		}
		var reply struct {
			Routing struct {
				Model string `json:"model"`
			} `json:"routing"`
			Answers map[string]struct {
				Choice        string             `json:"choice"`
				Probabilities map[string]float64 `json:"probabilities"`
			} `json:"answers"`
		}
		if err := json.Unmarshal(raw, &reply); err != nil {
			return result, exchanges, err
		}
		answer, found := reply.Answers["operation"]
		if reply.Routing.Model != "multilingual" || !found || !contains(expr.Allowed, answer.Choice) {
			return result, exchanges, errors.New("Laya routing or choice does not match the declared request")
		}
		digest := sha256.Sum256([]byte(expr.Text))
		receipt := HoleReceipt{ID: expr.HoleID, TextSHA256: hex.EncodeToString(digest[:]), Mode: "laya_closed_choice", Proposed: answer.Choice, Selected: answer.Choice, PredictNS: capture.WallNS}
		for _, operation := range expr.Allowed {
			probability, exists := answer.Probabilities[operation]
			if !exists || math.IsNaN(probability) || math.IsInf(probability, 0) || probability < 0 || probability > 1 {
				return result, exchanges, fmt.Errorf("invalid Laya probability for %s", operation)
			}
			receipt.Probabilities = append(receipt.Probabilities, decision.LabelProbability{Label: operation, Probability: float32(probability)})
		}
		if len(answer.Probabilities) != len(expr.Allowed) {
			return result, exchanges, errors.New("unexpected Laya probability labels")
		}
		if err := ctx.Err(); err != nil {
			return result, exchanges, err
		}
		result.Choices[expr.HoleID] = answer.Choice
		result.Holes = append(result.Holes, receipt)
		capture.Status = "completed_validated"
	}
	if _, err := bodyplan.Compile(plan, result.Choices); err != nil {
		return result, exchanges, err
	}
	if err := ctx.Err(); err != nil {
		return result, exchanges, err
	}
	return result, exchanges, nil
}
