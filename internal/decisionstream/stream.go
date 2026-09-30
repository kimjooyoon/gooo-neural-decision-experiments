// Package decisionstream implements bounded NDJSON inference over a loaded
// decision model. It keeps the model immutable and gives each worker its own
// reusable prediction workspace.
package decisionstream

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const (
	RequestSchema  = "gooo/tiny-ir-decision-stream-request/v1"
	ResultSchema   = "gooo/tiny-ir-decision-stream-result/v1"
	MaxRecordBytes = 16 * 1024
	MaxWorkers     = 8
	readerBuffer   = 4 * 1024
)

// Decider is implemented by decision.Model. The small interface also makes
// queue and cancellation behavior testable without loading extra models.
type Decider interface {
	Decide(decision.DecisionRequest, *decision.Workspace) (decision.DecisionResponse, error)
}

type Request struct {
	Schema        string                   `json:"schema"`
	CorrelationID string                   `json:"correlation_id"`
	Request       decision.DecisionRequest `json:"request"`
}

type Result struct {
	Schema        string                     `json:"schema"`
	Sequence      uint64                     `json:"sequence"`
	CorrelationID string                     `json:"correlation_id,omitempty"`
	Status        string                     `json:"status"`
	Response      *decision.DecisionResponse `json:"response,omitempty"`
	Error         string                     `json:"error,omitempty"`
}

type record struct {
	sequence uint64
	raw      []byte
	tooLarge bool
}

// Run reads one JSON object per line and writes one result line per input
// record. The model is already loaded and is shared read-only. Exactly
// workers worker goroutines are created; both channels are bounded by workers.
// On cancellation it closes input and output to interrupt pending I/O. Both
// Close methods must promptly unblock a concurrent Read or Write.
func Run(parent context.Context, model Decider, input io.ReadCloser, output io.WriteCloser, workers int) error {
	if parent == nil {
		return errors.New("context is required")
	}
	if model == nil {
		return errors.New("model is required")
	}
	if input == nil {
		return errors.New("input is required")
	}
	if output == nil {
		return errors.New("output is required")
	}
	if workers < 1 || workers > MaxWorkers {
		return fmt.Errorf("workers must be between 1 and %d", MaxWorkers)
	}

	ctx, cancel := context.WithCancel(parent)
	closeIOOnce := sync.Once{}
	closeIO := func() {
		closeIOOnce.Do(func() {
			_ = output.Close()
			_ = input.Close()
		})
	}
	watcherStop := make(chan struct{})
	watcherDone := make(chan struct{})
	var lifecycleMu sync.Mutex
	normalCompletion := false
	go func() {
		defer close(watcherDone)
		select {
		case <-ctx.Done():
			lifecycleMu.Lock()
			shouldClose := !normalCompletion
			lifecycleMu.Unlock()
			if shouldClose {
				closeIO()
			}
		case <-watcherStop:
		}
	}()
	var watcherStopOnce sync.Once
	stopWatcher := func() {
		watcherStopOnce.Do(func() { close(watcherStop) })
		<-watcherDone
	}
	defer func() {
		stopWatcher()
		cancel()
	}()
	jobs := make(chan record, workers)
	results := make(chan Result, workers)
	producerDone := make(chan error, 1)

	var workerGroup sync.WaitGroup
	workerGroup.Add(workers)
	for range workers {
		go func() {
			defer workerGroup.Done()
			worker(ctx, model, jobs, results)
		}()
	}
	go func() {
		workerGroup.Wait()
		close(results)
	}()
	go func() {
		defer close(jobs)
		producerDone <- produce(ctx, input, jobs)
	}()

	encoder := json.NewEncoder(output)
	for {
		select {
		case <-ctx.Done():
			closeIO()
			<-watcherDone
			drain(results)
			<-producerDone
			return ctx.Err()
		case result, ok := <-results:
			if !ok {
				producerErr := <-producerDone
				lifecycleMu.Lock()
				normalCompletion = true
				lifecycleMu.Unlock()
				stopWatcher()
				if err := parent.Err(); err != nil {
					return err
				}
				return producerErr
			}
			if err := encoder.Encode(result); err != nil {
				cancel()
				closeIO()
				<-watcherDone
				drain(results)
				<-producerDone
				if parentErr := parent.Err(); parentErr != nil {
					return parentErr
				}
				return fmt.Errorf("write result: %w", err)
			}
		}
	}
}

func produce(ctx context.Context, input io.Reader, jobs chan<- record) error {
	reader := bufio.NewReaderSize(input, readerBuffer)
	var sequence uint64
	for {
		raw, tooLarge, err := readRecord(reader, MaxRecordBytes)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("read input: %w", err)
		}
		sequence++
		item := record{sequence: sequence, raw: raw, tooLarge: tooLarge}
		select {
		case <-ctx.Done():
			return nil
		case jobs <- item:
		}
	}
}

func readRecord(reader *bufio.Reader, limit int) ([]byte, bool, error) {
	line := make([]byte, 0, min(limit, readerBuffer))
	seen := false
	tooLarge := false
	for {
		fragment, err := reader.ReadSlice('\n')
		if len(fragment) != 0 {
			seen = true
		}
		newline := len(fragment) != 0 && fragment[len(fragment)-1] == '\n'
		if newline {
			fragment = fragment[:len(fragment)-1]
		}
		if !tooLarge {
			if len(fragment) > limit-len(line) {
				tooLarge = true
				line = nil
			} else {
				line = append(line, fragment...)
			}
		}
		if newline {
			return trimCR(line), tooLarge, nil
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF {
			if !seen {
				return nil, false, io.EOF
			}
			return trimCR(line), tooLarge, nil
		}
		if err != nil {
			return nil, false, err
		}
	}
}

func trimCR(line []byte) []byte {
	if len(line) > 0 && line[len(line)-1] == '\r' {
		return line[:len(line)-1]
	}
	return line
}

func worker(ctx context.Context, model Decider, jobs <-chan record, results chan<- Result) {
	var workspace decision.Workspace
	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-jobs:
			if !ok {
				return
			}
			if ctx.Err() != nil {
				return
			}
			result := evaluate(model, item, &workspace)
			select {
			case <-ctx.Done():
				return
			case results <- result:
			}
		}
	}
}

func evaluate(model Decider, item record, workspace *decision.Workspace) Result {
	result := Result{Schema: ResultSchema, Sequence: item.sequence, Status: "rejected"}
	reject := func(err error) Result {
		result.Error = err.Error()
		return result
	}
	if item.tooLarge {
		return reject(fmt.Errorf("request exceeds %d bytes", MaxRecordBytes))
	}
	if len(item.raw) == 0 {
		return reject(errors.New("empty NDJSON record"))
	}
	if !utf8.Valid(item.raw) {
		return reject(errors.New("request is not valid UTF-8"))
	}
	if err := decision.RejectDuplicateJSONKeys(item.raw); err != nil {
		return reject(fmt.Errorf("validate request JSON: %w", err))
	}
	decoder := json.NewDecoder(bytes.NewReader(item.raw))
	decoder.DisallowUnknownFields()
	var request Request
	decodeErr := decoder.Decode(&request)
	if validCorrelationID(request.CorrelationID) {
		result.CorrelationID = request.CorrelationID
	}
	if decodeErr != nil {
		return reject(fmt.Errorf("decode request: %w", decodeErr))
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return reject(errors.New("request has trailing JSON content"))
		}
		return reject(fmt.Errorf("request has trailing content: %w", err))
	}
	if request.Schema != RequestSchema {
		return reject(fmt.Errorf("schema must be %q", RequestSchema))
	}
	if !validCorrelationID(request.CorrelationID) {
		return reject(errors.New("correlation_id must be 1-128 UTF-8 bytes without control characters"))
	}
	result.CorrelationID = request.CorrelationID
	response, err := model.Decide(request.Request, workspace)
	if err != nil {
		return reject(fmt.Errorf("decision rejected: %w", err))
	}
	result.Status = "completed"
	result.Response = &response
	return result
}

func validCorrelationID(value string) bool {
	if value == "" || len(value) > 128 || !utf8.ValidString(value) || strings.TrimSpace(value) == "" {
		return false
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return false
		}
	}
	return true
}

func drain(results <-chan Result) {
	for range results {
	}
}
