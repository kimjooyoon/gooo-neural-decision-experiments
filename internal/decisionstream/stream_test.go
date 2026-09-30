package decisionstream

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

type fakeModel struct {
	decide func(decision.DecisionRequest, *decision.Workspace) (decision.DecisionResponse, error)
}

func (model fakeModel) Decide(request decision.DecisionRequest, workspace *decision.Workspace) (decision.DecisionResponse, error) {
	return model.decide(request, workspace)
}

func successfulModel() fakeModel {
	return fakeModel{decide: func(_ decision.DecisionRequest, workspace *decision.Workspace) (decision.DecisionResponse, error) {
		if workspace == nil {
			return decision.DecisionResponse{}, errors.New("workspace is nil")
		}
		return decision.DecisionResponse{Schema: decision.DecisionResponseSchema, Status: "decision", BestLabel: "add"}, nil
	}}
}

func requestLine(t testing.TB, correlationID, text string) []byte {
	t.Helper()
	request := Request{
		Schema: RequestSchema, CorrelationID: correlationID,
		Request: decision.DecisionRequest{
			Schema: decision.DecisionRequestSchema, Text: text,
			Left:  decision.Identifier{Name: "left", Type: decision.TypeInt},
			Right: decision.Identifier{Name: "right", Type: decision.TypeInt},
		},
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func decodeResults(t *testing.T, raw []byte) []Result {
	t.Helper()
	var results []Result
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var result Result
		if err := json.Unmarshal(line, &result); err != nil {
			t.Fatalf("decode result line %q: %v", line, err)
		}
		results = append(results, result)
	}
	return results
}

func TestRunProcessesRecordsAndRejectsInvalidInputPerRecord(t *testing.T) {
	valid := requestLine(t, "good-1", "add these values")
	unknown := bytes.Replace(valid, []byte(`"request":`), []byte(`"unexpected":true,"request":`), 1)
	wrongSchema := bytes.Replace(valid, []byte(`"schema":`), []byte(`"wire_schema":`), 1)
	wrongType := bytes.Replace(valid, []byte(`"type":"Int"`), []byte(`"type":"Bool"`), 1)
	nestedUnknown := bytes.Replace(valid, []byte(`"left":{"name":"left","type":"Int"}`), []byte(`"left":{"name":"left","type":"Int","source":"x"}`), 1)
	duplicate := bytes.Replace(valid, []byte(`"schema":`), []byte(`"schema":"`+RequestSchema+`","schema":`), 1)
	invalidJSON := []byte(`{"schema":`)
	oversized := bytes.Repeat([]byte{'x'}, MaxRecordBytes+1)
	inputBytes := bytes.Join([][]byte{
		valid, unknown, wrongSchema, wrongType, nestedUnknown, duplicate, invalidJSON, nil, oversized,
	}, []byte{'\n'})
	inputBytes = append(inputBytes, '\n')

	model := fakeModel{decide: func(request decision.DecisionRequest, workspace *decision.Workspace) (decision.DecisionResponse, error) {
		if workspace == nil {
			return decision.DecisionResponse{}, errors.New("workspace is nil")
		}
		if request.Left.Type != decision.TypeInt || request.Right.Type != decision.TypeInt {
			return decision.DecisionResponse{}, errors.New("type mismatch")
		}
		return decision.DecisionResponse{Status: "decision", BestLabel: "add"}, nil
	}}
	var output bytes.Buffer
	input := &closeCountingReader{ReadCloser: io.NopCloser(bytes.NewReader(inputBytes))}
	trackedOutput := &closeCountingWriter{Writer: &output}
	if err := Run(context.Background(), model, input, trackedOutput, 3); err != nil {
		t.Fatal(err)
	}
	if input.closeCount.Load() != 0 || trackedOutput.closeCount.Load() != 0 {
		t.Fatalf("normal EOF closed caller-owned streams: input=%d output=%d", input.closeCount.Load(), trackedOutput.closeCount.Load())
	}
	results := decodeResults(t, output.Bytes())
	if len(results) != 9 {
		t.Fatalf("got %d results, want one per record: %+v", len(results), results)
	}
	sort.Slice(results, func(left, right int) bool { return results[left].Sequence < results[right].Sequence })
	if results[0].Status != "completed" || results[0].CorrelationID != "good-1" || results[0].Response == nil {
		t.Fatalf("valid request failed: %+v", results[0])
	}
	for index := 1; index < len(results); index++ {
		if results[index].Status != "rejected" || results[index].Error == "" {
			t.Fatalf("record %d was not rejected: %+v", index+1, results[index])
		}
		if results[index].Sequence != uint64(index+1) {
			t.Fatalf("rejected record lost sequence: %+v", results[index])
		}
	}
	if results[1].CorrelationID != "good-1" || results[2].CorrelationID != "good-1" || results[3].CorrelationID != "good-1" || results[4].CorrelationID != "good-1" {
		t.Fatalf("parseable invalid requests should retain their correlation ID: %+v", results[1:5])
	}
	if results[5].CorrelationID != "" || results[8].CorrelationID != "" {
		t.Fatalf("ambiguous duplicate/oversized records must not claim a correlation ID: %+v", results)
	}
}

func TestRunUsesDecisionModelAndRejectsTypedOperandMismatch(t *testing.T) {
	model, err := decision.Load(writeStreamFixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	good := requestLine(t, "valid", "add the values")
	badType := bytes.Replace(good, []byte(`"right":{"name":"right","type":"Int"}`), []byte(`"right":{"name":"right","type":"Bool"}`), 1)
	inputBytes := append(append(append([]byte(nil), good...), '\n'), badType...)
	inputBytes = append(inputBytes, '\n')
	var output bytes.Buffer
	if err := Run(context.Background(), model, io.NopCloser(bytes.NewReader(inputBytes)), asWriteCloser(&output), 2); err != nil {
		t.Fatal(err)
	}
	results := decodeResults(t, output.Bytes())
	sort.Slice(results, func(left, right int) bool { return results[left].Sequence < results[right].Sequence })
	if len(results) != 2 || results[0].Status != "completed" || results[0].Response == nil || results[0].Response.TypedBinaryIR == nil {
		t.Fatalf("typed decision was not emitted: %+v", results)
	}
	if results[1].Status != "rejected" || results[1].Error == "" || results[1].CorrelationID != "valid" {
		t.Fatalf("typed operand mismatch was not rejected with its correlation ID: %+v", results[1])
	}
}

func TestRunWritesResultsBeforeInputEOF(t *testing.T) {
	reader := newGatedReader(append(requestLine(t, "first", "quick"), '\n'))
	writer := &observingWriter{written: make(chan struct{}, 1)}
	done := make(chan error, 1)
	go func() {
		done <- Run(context.Background(), successfulModel(), reader, writer, 2)
	}()

	select {
	case <-writer.written:
		// The input reader is still blocked waiting for the next line here.
	case <-time.After(time.Second):
		_ = reader.Close()
		t.Fatal("first result waited for input EOF")
	}
	_ = reader.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not finish after input close")
	}
	results := decodeResults(t, writer.Bytes())
	if len(results) != 1 || results[0].CorrelationID != "first" {
		t.Fatalf("unexpected output: %+v", results)
	}
}

func TestRunEmitsCompletionOrderWithCorrelationAndSequence(t *testing.T) {
	first := requestLine(t, "slow", "slow")
	second := requestLine(t, "fast", "fast")
	inputBytes := append(append(append([]byte{}, first...), '\n'), second...)
	inputBytes = append(inputBytes, '\n')
	model := fakeModel{decide: func(request decision.DecisionRequest, workspace *decision.Workspace) (decision.DecisionResponse, error) {
		if request.Text == "slow" {
			time.Sleep(80 * time.Millisecond)
		}
		return decision.DecisionResponse{Status: "decision", BestLabel: "add"}, nil
	}}
	var output bytes.Buffer
	if err := Run(context.Background(), model, io.NopCloser(bytes.NewReader(inputBytes)), asWriteCloser(&output), 2); err != nil {
		t.Fatal(err)
	}
	results := decodeResults(t, output.Bytes())
	if len(results) != 2 || results[0].Sequence != 2 || results[0].CorrelationID != "fast" || results[1].Sequence != 1 || results[1].CorrelationID != "slow" {
		t.Fatalf("results were not emitted in completion order with stable correlation: %+v", results)
	}
}

func TestRunBoundsWorkAndReusesPrivateWorkerWorkspace(t *testing.T) {
	const requestCount = 3000
	var input bytes.Buffer
	for index := range requestCount {
		input.Write(requestLine(t, fmt.Sprintf("id-%d", index), "queued"))
		input.WriteByte('\n')
	}
	reader := &countingReadCloser{reader: bytes.NewReader(input.Bytes())}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var mutex sync.Mutex
	active := make(map[*decision.Workspace]bool)
	seen := make(map[*decision.Workspace]int)
	model := fakeModel{decide: func(_ decision.DecisionRequest, workspace *decision.Workspace) (decision.DecisionResponse, error) {
		mutex.Lock()
		if active[workspace] {
			mutex.Unlock()
			return decision.DecisionResponse{}, errors.New("workspace shared concurrently")
		}
		active[workspace] = true
		seen[workspace]++
		mutex.Unlock()
		entered <- struct{}{}
		<-release
		mutex.Lock()
		delete(active, workspace)
		mutex.Unlock()
		return decision.DecisionResponse{Status: "decision", BestLabel: "add"}, nil
	}}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, model, reader, asWriteCloser(io.Discard), 2) }()
	for range 2 {
		select {
		case <-entered:
		case <-time.After(time.Second):
			cancel()
			close(release)
			t.Fatal("both bounded workers did not start")
		}
	}
	time.Sleep(50 * time.Millisecond)
	if consumed := reader.consumed.Load(); consumed >= int64(input.Len()) {
		cancel()
		close(release)
		<-done
		t.Fatalf("producer consumed all input while workers were blocked (%d/%d bytes)", consumed, input.Len())
	}
	cancel()
	close(release)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("bounded stream did not stop after cancellation")
	}
	mutex.Lock()
	defer mutex.Unlock()
	if len(seen) > 2 {
		t.Fatalf("created %d workspaces for two workers", len(seen))
	}
}

func TestRunCancellationUnblocksAWaitingReader(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, successfulModel(), reader, asWriteCloser(io.Discard), 1) }()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not close the blocked reader")
	}
}

func TestRunOutputErrorCancelsAndClosesInput(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	input := append(requestLine(t, "first", "quick"), '\n')
	writeDone := make(chan error, 1)
	go func() { _, err := writer.Write(input); writeDone <- err }()
	done := make(chan error, 1)
	go func() { done <- Run(context.Background(), successfulModel(), reader, failingWriter{}, 1) }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "writer failed") {
			t.Fatalf("Run error = %v, want writer failure", err)
		}
	case <-time.After(time.Second):
		t.Fatal("writer failure deadlocked the producer")
	}
	select {
	case <-writeDone:
	case <-time.After(time.Second):
		t.Fatal("closing input did not unblock the upstream writer")
	}
}

func TestRunCancellationClosesBlockedOutputWriter(t *testing.T) {
	inputBytes := append(requestLine(t, "cancel-write", "quick"), '\n')
	input := &closeCountingReader{ReadCloser: io.NopCloser(bytes.NewReader(inputBytes))}
	output := newBlockingWriter()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, successfulModel(), input, output, 1) }()
	select {
	case <-output.entered:
	case <-time.After(time.Second):
		cancel()
		select {
		case <-done:
		case <-time.After(time.Second):
		}
		t.Fatal("stream did not enter blocked output write")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not close blocked output writer")
	}
	if input.closeCount.Load() != 1 || output.closeCount.Load() != 1 {
		t.Fatalf("close counts: input=%d output=%d, want exactly one each", input.closeCount.Load(), output.closeCount.Load())
	}
}

func TestRunCancellationReleasesOutputBackpressureAndReader(t *testing.T) {
	reader, upstream := io.Pipe()
	input := &closeCountingReader{ReadCloser: reader}
	output := newBlockingWriter()
	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- Run(ctx, successfulModel(), input, output, 2) }()

	line := append(requestLine(t, "backpressure", "queued"), '\n')
	feedDone := make(chan error, 1)
	go func() {
		for range 4096 {
			if _, err := upstream.Write(line); err != nil {
				feedDone <- err
				return
			}
		}
		feedDone <- upstream.Close()
	}()
	select {
	case <-output.entered:
	case <-time.After(time.Second):
		cancel()
		_ = upstream.Close()
		select {
		case <-runDone:
		case <-time.After(time.Second):
		}
		select {
		case <-feedDone:
		case <-time.After(time.Second):
		}
		t.Fatal("stream did not reach its blocked result write")
	}
	select {
	case err := <-feedDone:
		cancel()
		select {
		case <-runDone:
		case <-time.After(time.Second):
		}
		t.Fatalf("upstream feed escaped bounded backpressure before cancellation: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	cancel()
	select {
	case err := <-runDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Run error = %v, want context cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation did not release blocked result output")
	}
	select {
	case <-feedDone:
	case <-time.After(time.Second):
		t.Fatal("closing the input reader did not release the upstream writer")
	}
	_ = upstream.Close()
	if input.closeCount.Load() != 1 || output.closeCount.Load() != 1 {
		t.Fatalf("close counts: input=%d output=%d, want exactly one each", input.closeCount.Load(), output.closeCount.Load())
	}
}

func TestRunCompletedTransportCanContainModelAbstention(t *testing.T) {
	line := append(requestLine(t, "abstain", "ambiguous instruction"), '\n')
	model := fakeModel{decide: func(_ decision.DecisionRequest, _ *decision.Workspace) (decision.DecisionResponse, error) {
		return decision.DecisionResponse{Schema: decision.DecisionResponseSchema, Status: "abstained"}, nil
	}}
	var output bytes.Buffer
	if err := Run(context.Background(), model, io.NopCloser(bytes.NewReader(line)), asWriteCloser(&output), 1); err != nil {
		t.Fatal(err)
	}
	results := decodeResults(t, output.Bytes())
	if len(results) != 1 || results[0].Status != "completed" || results[0].Response == nil || results[0].Response.Status != "abstained" || results[0].Response.TypedBinaryIR != nil {
		t.Fatalf("transport completion was confused with accepted IR: %+v", results)
	}
}

func TestRunPreservesCompletedResultsWhenInputReadFails(t *testing.T) {
	reader := &readFailureReader{first: append(requestLine(t, "before-error", "quick"), '\n')}
	var output bytes.Buffer
	err := Run(context.Background(), successfulModel(), reader, asWriteCloser(&output), 1)
	if err == nil || !strings.Contains(err.Error(), "upstream read failed") {
		t.Fatalf("Run error = %v, want upstream read failure", err)
	}
	results := decodeResults(t, output.Bytes())
	if len(results) != 1 || results[0].Sequence != 1 || results[0].CorrelationID != "before-error" || results[0].Status != "completed" {
		t.Fatalf("completed result was lost when a later read failed: %+v", results)
	}
}

type observingWriter struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	written chan struct{}
}

func (writer *observingWriter) Close() error { return nil }

func (writer *observingWriter) Write(value []byte) (int, error) {
	writer.mu.Lock()
	n, err := writer.buffer.Write(value)
	writer.mu.Unlock()
	select {
	case writer.written <- struct{}{}:
	default:
	}
	return n, err
}

func (writer *observingWriter) Bytes() []byte {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return append([]byte(nil), writer.buffer.Bytes()...)
}

type gatedReader struct {
	first   []byte
	read    bool
	release chan struct{}
	once    sync.Once
}

func newGatedReader(first []byte) *gatedReader {
	return &gatedReader{first: first, release: make(chan struct{})}
}

func (reader *gatedReader) Read(buffer []byte) (int, error) {
	if !reader.read {
		reader.read = true
		return copy(buffer, reader.first), nil
	}
	<-reader.release
	return 0, io.EOF
}

func (reader *gatedReader) Close() error {
	reader.once.Do(func() { close(reader.release) })
	return nil
}

type countingReadCloser struct {
	reader   *bytes.Reader
	consumed atomic.Int64
}

func (reader *countingReadCloser) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	reader.consumed.Add(int64(n))
	return n, err
}

func (reader *countingReadCloser) Close() error { return nil }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("writer failed") }
func (failingWriter) Close() error              { return nil }

type closeCountingReader struct {
	io.ReadCloser
	closeCount atomic.Int32
}

func (reader *closeCountingReader) Close() error {
	reader.closeCount.Add(1)
	return reader.ReadCloser.Close()
}

type closeCountingWriter struct {
	io.Writer
	closeCount atomic.Int32
}

func (writer *closeCountingWriter) Close() error {
	writer.closeCount.Add(1)
	return nil
}

type blockingWriter struct {
	entered    chan struct{}
	closed     chan struct{}
	enterOnce  sync.Once
	closeOnce  sync.Once
	closeCount atomic.Int32
}

func newBlockingWriter() *blockingWriter {
	return &blockingWriter{entered: make(chan struct{}), closed: make(chan struct{})}
}

func (writer *blockingWriter) Write([]byte) (int, error) {
	writer.enterOnce.Do(func() { close(writer.entered) })
	<-writer.closed
	return 0, errors.New("write interrupted by close")
}

func (writer *blockingWriter) Close() error {
	writer.closeOnce.Do(func() {
		writer.closeCount.Add(1)
		close(writer.closed)
	})
	return nil
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func asWriteCloser(writer io.Writer) io.WriteCloser { return nopWriteCloser{Writer: writer} }

type readFailureReader struct {
	first []byte
	read  bool
}

func (reader *readFailureReader) Read(buffer []byte) (int, error) {
	if !reader.read {
		reader.read = true
		return copy(buffer, reader.first), nil
	}
	return 0, errors.New("upstream read failed")
}

func (*readFailureReader) Close() error { return nil }

func writeStreamFixtureModel(t testing.TB) string {
	t.Helper()
	directory := t.TempDir()
	weights := make([]byte, 0, 50_912)
	tensors := []decision.TensorMetadata{
		{Name: "w1", Count: decision.FeatureDim * decision.HiddenDim, Rows: decision.HiddenDim, Cols: decision.FeatureDim, Encoding: "float32_le", Scale: 1},
		{Name: "b1", Count: decision.HiddenDim, Rows: 1, Cols: decision.HiddenDim, Encoding: "float32_le", Scale: 1},
		{Name: "w2", Count: decision.HiddenDim * decision.LabelCount, Rows: decision.LabelCount, Cols: decision.HiddenDim, Encoding: "float32_le", Scale: 1},
		{Name: "b2", Count: decision.LabelCount, Rows: 1, Cols: decision.LabelCount, Encoding: "float32_le", Scale: 1},
	}
	for index := range tensors {
		tensors[index].Offset = int64(len(weights))
		tensors[index].Bytes = int64(tensors[index].Count * 4)
		weights = append(weights, make([]byte, tensors[index].Count*4)...)
	}
	biasOffset := int(tensors[3].Offset)
	binary.LittleEndian.PutUint32(weights[biasOffset:biasOffset+4], 0x40000000)
	digest := sha256.Sum256(weights)
	labels := decision.Labels()
	metadata := decision.Metadata{
		Schema: decision.MetadataSchema, Variant: "fp32",
		FeatureDim: decision.FeatureDim, HiddenDim: decision.HiddenDim, MaxBytes: decision.InputMaxBytes,
		Labels: labels[:], Temperature: 1, WeightsFile: "weights.bin",
		WeightsSHA256: hex.EncodeToString(digest[:]), Tensors: tensors,
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	weightsPath := filepath.Join(directory, "weights.bin")
	metadataPath := filepath.Join(directory, "model.json")
	if err := os.WriteFile(weightsPath, weights, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return metadataPath
}
