package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func TestExecuteEmitsTypedDecisionAndRejectsOpenEndedJSON(t *testing.T) {
	model, err := decision.Load(writeCLIFixtureModel(t))
	if err != nil {
		t.Fatal(err)
	}
	valid := `{"schema":"gooo/tiny-ir-decision-request/v1","text":"add the values","left":{"name":"left","type":"Int"},"right":{"name":"right","type":"Int"}}`
	var output bytes.Buffer
	if err := execute(model, strings.NewReader(valid), &output); err != nil {
		t.Fatal(err)
	}
	var response decision.DecisionResponse
	if err := json.Unmarshal(output.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "decision" || response.TypedBinaryIR == nil || response.TypedBinaryIR.Operation != "add" {
		t.Fatalf("unexpected CLI decision response: %+v", response)
	}
	if response.WorkspaceBytes != 1248 || response.PredictionBytes != 80 || response.PredictionValueBytes != 64 || response.DecodedBytes != 50912 || response.MatrixScaleBytes != 0 {
		t.Fatalf("runtime memory accounting missing from response: %+v", response)
	}

	for name, invalid := range map[string]string{
		"unknown field": strings.Replace(valid, `"text":`, `"arbitrary_source":"x","text":`, 1),
		"trailing JSON": valid + ` {}`,
		"duplicate key": strings.Replace(valid, `"schema":`, `"schema":"gooo/tiny-ir-decision-request/v1","schema":`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			if err := execute(model, strings.NewReader(invalid), &output); err == nil {
				t.Fatal("invalid request was accepted")
			}
		})
	}
}

func writeCLIFixtureModel(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
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
	lastBiasOffset := int(tensors[3].Offset)
	binary.LittleEndian.PutUint32(weights[lastBiasOffset:lastBiasOffset+4], 0x40000000)
	digest := sha256.Sum256(weights)
	metadata := decision.Metadata{
		Schema: decision.MetadataSchema, Variant: "fp32",
		FeatureDim: decision.FeatureDim, HiddenDim: decision.HiddenDim, MaxBytes: decision.InputMaxBytes,
		Labels:      func() []string { labels := decision.Labels(); return labels[:] }(),
		Temperature: 1, WeightsFile: "weights.bin", WeightsSHA256: hex.EncodeToString(digest[:]), Tensors: tensors,
	}
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	weightsPath := filepath.Join(dir, "weights.bin")
	metadataPath := filepath.Join(dir, "model.json")
	if err := os.WriteFile(weightsPath, weights, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadataPath, metadataRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	return metadataPath
}
