package decision

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadVariantsAndPredictIntoDoesNotAllocate(t *testing.T) {
	for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		t.Run(variant, func(t *testing.T) {
			metadataPath, packedBytes := writeFixtureModel(t, variant, 0.5)
			model, err := Load(metadataPath)
			if err != nil {
				t.Fatal(err)
			}
			if model.Variant() != variant {
				t.Fatalf("variant = %q, want %q", model.Variant(), variant)
			}
			wantResident, wantMatrix, wantBias := 50_912, 50_688, 224
			wantScaleBytes := 0
			if variant != "fp32" {
				wantResident, wantMatrix = 12_896, 12_672
				wantScaleBytes = 8
			}
			if model.PackedFileBytes() != packedBytes || model.ResidentTensorBytes() != wantResident || model.MatrixTensorBytes() != wantMatrix || model.BiasTensorBytes() != wantBias || model.MatrixScaleBytes() != wantScaleBytes {
				t.Fatalf("packed/resident/matrix/bias/scales bytes = %d/%d/%d/%d/%d; want packed=%d resident=%d matrix=%d bias=%d scales=%d", model.PackedFileBytes(), model.ResidentTensorBytes(), model.MatrixTensorBytes(), model.BiasTensorBytes(), model.MatrixScaleBytes(), packedBytes, wantResident, wantMatrix, wantBias, wantScaleBytes)
			}
			var workspace Workspace
			var prediction Prediction
			if err := model.PredictInto("ADD the value to the balance", &workspace, &prediction); err != nil {
				t.Fatal(err)
			}
			if WorkspaceBytes() != 1248 || PredictionValueArrayBytes() != 64 || PredictionBytes() != 80 {
				t.Fatalf("unexpected workspace/prediction bytes = %d/%d/%d", WorkspaceBytes(), PredictionValueArrayBytes(), PredictionBytes())
			}
			if got := model.PredictLabel(&prediction); got != "add" || prediction.Abstained {
				t.Fatalf("prediction = %q, abstained=%t confidence=%f", got, prediction.Abstained, prediction.Confidence)
			}
			if !closeTo(float64(probabilitySum(prediction.Probabilities)), 1, 1e-6) {
				t.Fatalf("probabilities do not sum to one: %.9f", probabilitySum(prediction.Probabilities))
			}
			allocs := testing.AllocsPerRun(1000, func() {
				if err := model.PredictInto("ADD the value to the balance", &workspace, &prediction); err != nil {
					panic(err)
				}
			})
			if allocs != 0 {
				t.Fatalf("valid PredictInto hot path allocated %.2f objects per call", allocs)
			}
		})
	}
}

func TestFeatureHashingLowercasesASCIIAndNormalizes(t *testing.T) {
	var upper, lower [FeatureDim]float32
	buildFeatures("Add Two", &upper)
	buildFeatures("aDD tWO", &lower)
	if upper != lower {
		t.Fatal("ASCII case folding produced different feature vectors")
	}
	var norm float64
	for _, value := range upper {
		norm += float64(value * value)
	}
	if !closeTo(norm, 1, 1e-6) {
		t.Fatalf("L2 norm squared = %.9f, want 1", norm)
	}
	var short [FeatureDim]float32
	buildFeatures("A", &short)
	for _, value := range short {
		if value != 0 {
			t.Fatal("one-byte input should have no byte bigram/trigram features")
		}
	}
}

func TestPredictRejectsOversizeAndInvalidUTF8(t *testing.T) {
	metadataPath, _ := writeFixtureModel(t, "fp32", 0.5)
	model, err := Load(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var workspace Workspace
	var prediction Prediction
	if err := model.PredictInto(string(make([]byte, InputMaxBytes+1)), &workspace, &prediction); err == nil {
		t.Fatal("oversized input accepted")
	}
	if err := model.PredictInto(string([]byte{0xff, 'a'}), &workspace, &prediction); err == nil {
		t.Fatal("invalid UTF-8 accepted")
	}
}

func TestConfidenceThresholdAbstains(t *testing.T) {
	metadataPath, _ := writeFixtureModel(t, "fp32", 0.99)
	model, err := Load(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var workspace Workspace
	var prediction Prediction
	if err := model.PredictInto("add two values", &workspace, &prediction); err != nil {
		t.Fatal(err)
	}
	if !prediction.Abstained || prediction.Confidence >= 0.99 {
		t.Fatalf("expected low-confidence abstention, confidence=%f abstained=%t", prediction.Confidence, prediction.Abstained)
	}
}

func TestDecodeTernaryBase3FiveAndPadding(t *testing.T) {
	// Five trits encode -1, 0, 1, -1, 0 as 0 + 3 + 18 + 0 + 81 = 102.
	// The second byte encodes 1, 0, -1 plus two required zero-padding trits.
	dst := make([]int8, 8)
	if err := decodeTernaryTensor([]byte{102, 113}, dst); err != nil {
		t.Fatal(err)
	}
	want := []int8{-1, 0, 1, -1, 0, 1, 0, -1}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("decoded[%d] = %d, want %d", i, dst[i], want[i])
		}
	}
	if err := decodeTernaryTensor([]byte{243, 0}, dst); err == nil {
		t.Fatal("out-of-range base-3 byte accepted")
	}
	badPadding := make([]int8, 6)
	if err := decodeTernaryTensor([]byte{121, 0}, badPadding); err == nil {
		t.Fatal("nonzero unused padding trit accepted")
	}
}

func TestTernaryPredictUsesInt8FlatMatricesAndPerMatrixScales(t *testing.T) {
	model := &Model{
		metadata:       Metadata{MaxBytes: InputMaxBytes},
		ternaryWeights: make([]int8, FeatureDim*HiddenDim+HiddenDim*LabelCount),
		biases:         make([]float32, HiddenDim+LabelCount),
		w1Scale:        0.25,
		w2Scale:        0.5,
		temperature:    1,
		threshold:      0,
	}
	var features [FeatureDim]float32
	buildFeatures("aa", &features)
	featureIndex := hashNgram("aa", 0, 2)
	model.ternaryWeights[featureIndex] = 1
	model.ternaryWeights[HiddenDim*FeatureDim] = 1
	var workspace Workspace
	var prediction Prediction
	if err := model.PredictInto("aa", &workspace, &prediction); err != nil {
		t.Fatal(err)
	}
	wantLogit := features[featureIndex] * 0.25 * 0.5
	wantProbability := float32(math.Exp(float64(wantLogit)) / (math.Exp(float64(wantLogit)) + LabelCount - 1))
	if !closeTo(float64(prediction.Probabilities[0]), float64(wantProbability), 1e-6) {
		t.Fatalf("first-label probability = %.8f, want %.8f", prediction.Probabilities[0], wantProbability)
	}
	if model.MatrixTensorBytes() != FeatureDim*HiddenDim+HiddenDim*LabelCount || model.BiasTensorBytes() != (HiddenDim+LabelCount)*4 {
		t.Fatalf("unexpected ternary tensor storage: matrix=%d bias=%d", model.MatrixTensorBytes(), model.BiasTensorBytes())
	}
}

func TestLoadRejectsDigestAndUnknownMetadataKeys(t *testing.T) {
	metadataPath, _ := writeFixtureModel(t, "fp32", 0.5)
	metadataRaw, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	weightsPath := filepath.Join(filepath.Dir(metadataPath), "weights.bin")
	weightsRaw, err := os.ReadFile(weightsPath)
	if err != nil {
		t.Fatal(err)
	}
	originalWeights := append([]byte(nil), weightsRaw...)
	weightsRaw[len(weightsRaw)-1] ^= 1
	if err := os.WriteFile(weightsPath, weightsRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(metadataPath); err == nil {
		t.Fatal("weights digest mismatch accepted")
	}
	if err := os.WriteFile(weightsPath, originalWeights, 0o600); err != nil {
		t.Fatal(err)
	}
	metadataRaw = bytes.TrimSpace(metadataRaw)
	metadataRaw = append(metadataRaw[:len(metadataRaw)-1], []byte(`,"unexpected":true}`)...)
	if err := os.WriteFile(metadataPath, metadataRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(metadataPath); err == nil {
		t.Fatal("unknown metadata field accepted")
	}
}

func TestLoadRejectsDuplicateMetadataKeysAndInvalidFloat32Ranges(t *testing.T) {
	metadataPath, _ := writeFixtureModel(t, "fp32", 0.5)
	metadataRaw, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	metadataRaw = bytes.Replace(metadataRaw, []byte(`"variant":"fp32"`), []byte(`"variant":"fp32","variant":"fp32"`), 1)
	if err := os.WriteFile(metadataPath, metadataRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(metadataPath); err == nil {
		t.Fatal("duplicate metadata member was accepted")
	}
	if err := RejectDuplicateJSONKeys([]byte(`{"outer":{"value":1,"value":2}}`)); err == nil {
		t.Fatal("nested duplicate JSON key was accepted")
	}

	labels := Labels()
	if err := validateMetadata(Metadata{Schema: MetadataSchema, Variant: "fp32", FeatureDim: FeatureDim,
		HiddenDim: HiddenDim, MaxBytes: InputMaxBytes, Labels: labels[:], Temperature: math.SmallestNonzeroFloat64}); err == nil {
		t.Fatal("temperature that underflows float32 was accepted")
	}
	if err := validateEncoding("qat_ternary", TensorMetadata{Name: "w1", Count: FeatureDim * HiddenDim,
		Encoding: "ternary_base3_5", Bytes: 2458, Scale: math.MaxFloat64}); err == nil {
		t.Fatal("ternary scale that overflows float32 was accepted")
	}
}

func TestTensorByteCounts(t *testing.T) {
	if got, err := TensorByteCount(12288, "ternary_base3_5"); err != nil || got != 2458 {
		t.Fatalf("ternary tensor bytes = %d, err=%v", got, err)
	}
	if got, err := TensorByteCount(48, "float32_le"); err != nil || got != 192 {
		t.Fatalf("float tensor bytes = %d, err=%v", got, err)
	}
}

func BenchmarkPredictInto(b *testing.B) {
	for _, variant := range []string{"fp32", "qat_ternary"} {
		b.Run(variant, func(b *testing.B) {
			metadataPath, _ := writeFixtureModel(b, variant, 0.5)
			model, err := Load(metadataPath)
			if err != nil {
				b.Fatal(err)
			}
			var workspace Workspace
			var prediction Prediction
			text := "Multiply the quantity by the unit price"
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := model.PredictInto(text, &workspace, &prediction); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(model.PackedFileBytes()), "packed-B")
			b.ReportMetric(float64(model.MatrixTensorBytes()), "matrix-B")
			b.ReportMetric(float64(model.BiasTensorBytes()), "bias-B")
			b.ReportMetric(float64(model.MatrixScaleBytes()), "scale-B")
			b.ReportMetric(float64(model.ResidentTensorBytes()), "tensor-B")
			b.ReportMetric(float64(WorkspaceBytes()), "workspace-B")
			b.ReportMetric(float64(PredictionValueArrayBytes()), "prediction-arrays-B")
			b.ReportMetric(float64(PredictionBytes()), "prediction-B")
		})
	}
}

type testingTB interface {
	Helper()
	Fatalf(string, ...any)
	TempDir() string
}

func writeFixtureModel(t testingTB, variant string, threshold float64) (string, int) {
	t.Helper()
	dir := t.TempDir()
	segments := tensorSegments()
	labels := Labels()
	metadata := Metadata{
		Schema: MetadataSchema, Variant: variant,
		FeatureDim: FeatureDim, HiddenDim: HiddenDim, MaxBytes: InputMaxBytes,
		Labels: labels[:], Temperature: 1, WeightsFile: "weights.bin",
		Tensors: make([]TensorMetadata, 0, 4), ConfidenceThreshold: &threshold,
	}
	// Populate the biases so the synthetic bundle predicts "add" without text input.
	total := FeatureDim*HiddenDim + HiddenDim + HiddenDim*LabelCount + LabelCount
	decoded := make([]float32, total)
	decoded[total-LabelCount] = 2
	var packed []byte
	for _, name := range []string{"w1", "b1", "w2", "b2"} {
		segment := segments[name]
		encoding := "float32_le"
		if (name == "w1" || name == "w2") && variant != "fp32" {
			encoding = "ternary_base3_5"
		}
		tensor := TensorMetadata{Name: name, Count: segment.count, Rows: segment.rows, Cols: segment.cols,
			Encoding: encoding, Offset: int64(len(packed)), Scale: 1}
		if encoding == "ternary_base3_5" {
			tensor.Scale = 0.25
			n := (segment.count + 4) / 5
			for i := 0; i < n; i++ {
				packed = append(packed, 121) // five zero weights, including required padding trits
			}
		} else {
			for i := 0; i < segment.count; i++ {
				packed = binary.LittleEndian.AppendUint32(packed, math.Float32bits(decoded[segment.floatOffset+i]))
			}
		}
		tensor.Bytes = int64(len(packed)) - tensor.Offset
		metadata.Tensors = append(metadata.Tensors, tensor)
	}
	metadata.WeightsSHA256 = shaHex(packed)
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal test metadata: %v", err)
	}
	weightsPath := filepath.Join(dir, metadata.WeightsFile)
	if err := os.WriteFile(weightsPath, packed, 0o600); err != nil {
		t.Fatalf("write test weights: %v", err)
	}
	metadataPath := filepath.Join(dir, "model.json")
	if err := os.WriteFile(metadataPath, metadataRaw, 0o600); err != nil {
		t.Fatalf("write test metadata: %v", err)
	}
	return metadataPath, len(packed)
}

func shaHex(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

func probabilitySum(values [LabelCount]float32) float32 {
	var sum float32
	for _, value := range values {
		sum += value
	}
	return sum
}

func closeTo(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}
