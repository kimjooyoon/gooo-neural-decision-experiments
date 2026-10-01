package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

type retainedProcess struct {
	Metrics     metrics         `json:"whole_process_metrics"`
	Setup       json.RawMessage `json:"constructor_record,omitempty"`
	StartupNS   int64           `json:"startup_until_constructor_record_ns,omitempty"`
	RoundtripNS []int64         `json:"request_enqueue_to_response_ns,omitempty"`
}

// This controller continuously drains results while writing a parallel batch.
// Sequential mode acknowledges each response before sending the next request.
func retainedChild(parent context.Context, binary, model string, workers int, lines [][]byte) ([]byte, retainedProcess, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	args := []string{"--workers", fmtInt(workers)}
	if model != "" {
		args = append(args, "--model", model)
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.WaitDelay = time.Second
	cmd.Env = append(os.Environ(), "GOOO_LAYA_URL=", "GOOO_LAYA_API_KEY=", "GOWORK=off", "GOTOOLCHAIN=local")
	configure(cmd)
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, retainedProcess{}, err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		return nil, retainedProcess{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, retainedProcess{}, err
	}
	started := time.Now()
	if err = cmd.Start(); err != nil {
		return nil, retainedProcess{}, err
	}
	defer input.Close()
	reader := bufio.NewReaderSize(stderr, 4096)
	setup, setupErr := reader.ReadSlice('\n')
	p := retainedProcess{Setup: append(json.RawMessage(nil), bytes.TrimSpace(setup)...),
		StartupNS: time.Since(started).Nanoseconds(), RoundtripNS: make([]int64, len(lines))}
	var extra bounded
	stderrDone := make(chan error, 1)
	go func() { _, err := io.Copy(&extra, reader); stderrDone <- err }()
	if setupErr != nil {
		cancel()
		_ = cmd.Wait()
		<-stderrDone
		return nil, p, errors.New("worker constructor record missing")
	}
	ack := make(chan struct{}, 1)
	writeDone := make(chan error, 1)
	var mutex sync.Mutex
	enqueued := make([]time.Time, len(lines))
	go func() {
		defer input.Close()
		for i, line := range lines {
			mutex.Lock()
			enqueued[i] = time.Now()
			mutex.Unlock()
			if _, err := input.Write(append(append([]byte(nil), line...), '\n')); err != nil {
				writeDone <- err
				return
			}
			if workers == 1 {
				select {
				case <-ack:
				case <-ctx.Done():
					writeDone <- ctx.Err()
					return
				}
			}
		}
		writeDone <- nil
	}()
	var raw bounded
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	count := 0
	var decodeErr error
	for scanner.Scan() {
		line := scanner.Bytes()
		if _, err = raw.Write(append(append([]byte(nil), line...), '\n')); err != nil {
			decodeErr = err
			break
		}
		var value struct {
			Sequence int `json:"sequence"`
		}
		if json.Unmarshal(line, &value) != nil || value.Sequence < 1 || value.Sequence > len(lines) {
			decodeErr = errors.New("worker result sequence invalid")
			break
		}
		mutex.Lock()
		sent := enqueued[value.Sequence-1]
		mutex.Unlock()
		if sent.IsZero() || p.RoundtripNS[value.Sequence-1] != 0 {
			decodeErr = errors.New("worker sequence repeated")
			break
		}
		p.RoundtripNS[value.Sequence-1] = time.Since(sent).Nanoseconds()
		count++
		if workers == 1 {
			ack <- struct{}{}
		}
	}
	if scanner.Err() != nil || decodeErr != nil || count != len(lines) {
		cancel()
		_ = input.Close()
	}
	writeErr := <-writeDone
	// Stderr is drained before Wait closes the pipe, preserving complete evidence.
	stderrErr := <-stderrDone
	waitErr := cmd.Wait()
	p.Metrics = metrics{Wall: time.Since(started).Nanoseconds()}
	if cmd.ProcessState != nil {
		p.Metrics.User = cmd.ProcessState.UserTime().Nanoseconds()
		p.Metrics.System = cmd.ProcessState.SystemTime().Nanoseconds()
		p.Metrics.RSS = maxRSS(cmd.ProcessState)
		p.Metrics.CPU = 100 * float64(p.Metrics.User+p.Metrics.System) / float64(p.Metrics.Wall)
	}
	if waitErr != nil || writeErr != nil || stderrErr != nil || extra.Len() != 0 || decodeErr != nil || scanner.Err() != nil || count != len(lines) {
		return raw.Bytes(), p, errors.New("bounded native worker failed; captures retained")
	}
	return raw.Bytes(), p, nil
}
