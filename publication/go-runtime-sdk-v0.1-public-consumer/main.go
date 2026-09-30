package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"testing"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

const (
	modulePath = "github.com/kimjooyoon/gooo-decision-runtime"
	originRev  = "e18908b88eecaacbb0b80df4feb8cc7b9b4380bb"
)

type receipt struct {
	Schema          string            `json:"schema"`
	ModulePath      string            `json:"module_path"`
	ModuleVersion   string            `json:"module_version"`
	ModuleH1        string            `json:"module_h1"`
	GoVersion       string            `json:"go_version"`
	OriginRevision  string            `json:"origin_revision"`
	ConsumerSHA256  map[string]string `json:"consumer_source_sha256"`
	SourceSHA256    map[string]string `json:"source_sha256"`
	WorkspaceBytes  int               `json:"workspace_bytes"`
	PredictionBytes int               `json:"prediction_bytes"`
	Models          []modelResult     `json:"models"`
}

type modelResult struct {
	Variant                  string  `json:"variant"`
	WeightsSHA256            string  `json:"weights_sha256"`
	DecisionStatus           string  `json:"decision_status"`
	BestLabel                string  `json:"best_label"`
	TypedOperation           string  `json:"typed_operation,omitempty"`
	TypedResultType          string  `json:"typed_result_type,omitempty"`
	GoExpression             string  `json:"go_expression,omitempty"`
	PredictIntoAllocsPerCall float64 `json:"predict_into_allocs_per_call"`
	ConcurrentPredictCalls   int     `json:"concurrent_predict_calls"`
	ConcurrentWorkers        int     `json:"concurrent_workers"`
}

func main() {
	if len(os.Args) != 5 {
		fail("expected three model metadata paths and the resolved module directory")
	}
	sourceHashes, err := verifyModuleSource(os.Args[4])
	if err != nil {
		fail("public module source provenance verification failed")
	}
	consumerHashes, err := hashConsumerFiles()
	if err != nil {
		fail("consumer source fingerprinting failed")
	}
	version, checksum, err := publicModuleBuildInfo()
	if err != nil {
		fail("public module build information is missing or replaced")
	}

	results := make([]modelResult, 0, 3)
	variants := []string{"fp32", "ptq_ternary", "qat_ternary"}
	for index, variant := range variants {
		model, err := decision.Load(os.Args[index+1])
		if err != nil || model.Variant() != variant {
			fail("a pinned model bundle could not be loaded")
		}
		actualWeightsSHA, err := verifyBundleWeights(os.Args[index+1], model)
		if err != nil {
			fail("actual model weight bytes did not match loaded model metadata")
		}

		text := "Add the left value to the right value."
		left := decision.Identifier{Name: "left", Type: decision.TypeInt}
		right := decision.Identifier{Name: "right", Type: decision.TypeInt}
		request := decision.DecisionRequest{
			Schema: decision.DecisionRequestSchema,
			Text:   text,
			Left:   left,
			Right:  right,
		}
		var workspace decision.Workspace
		response, err := model.Decide(request, &workspace)
		if err != nil {
			fail("typed decision call failed")
		}
		result := modelResult{
			Variant:        variant,
			WeightsSHA256:  actualWeightsSHA,
			DecisionStatus: response.Status,
			BestLabel:      response.BestLabel,
		}
		if response.Status == "decision" {
			if response.TypedBinaryIR == nil {
				fail("accepted decision omitted typed IR")
			}
			expression, err := decision.AssembleGoExpression(*response.TypedBinaryIR)
			if err != nil {
				fail("typed IR could not be rendered")
			}
			result.TypedOperation = response.TypedBinaryIR.Operation
			result.TypedResultType = string(response.TypedBinaryIR.ResultType)
			result.GoExpression = expression
		}

		var prediction decision.Prediction
		allocs := testing.AllocsPerRun(100, func() {
			if err := model.PredictInto(text, &workspace, &prediction); err != nil {
				panic(err)
			}
		})
		if allocs != 0 {
			fail("PredictInto valid hot path allocation count was not zero")
		}
		result.PredictIntoAllocsPerCall = allocs
		result.ConcurrentWorkers = 4
		result.ConcurrentPredictCalls, err = runConcurrent(model, text, model.PredictLabel(&prediction))
		if err != nil {
			fail("shared-model concurrent prediction check failed")
		}
		results = append(results, result)
	}

	if err := exerciseClosedIRTable(); err != nil {
		fail("closed typed operation table verification failed")
	}
	out := receipt{
		Schema:          "gooo/public-decision-runtime-consumer-check/v1",
		ModulePath:      modulePath,
		ModuleVersion:   version,
		ModuleH1:        checksum,
		GoVersion:       runtime.Version(),
		OriginRevision:  originRev,
		ConsumerSHA256:  consumerHashes,
		SourceSHA256:    sourceHashes,
		WorkspaceBytes:  decision.WorkspaceBytes(),
		PredictionBytes: decision.PredictionBytes(),
		Models:          results,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		fail("could not encode verification receipt")
	}
}

func hashConsumerFiles() (map[string]string, error) {
	hashes := make(map[string]string, 3)
	for _, name := range []string{"main.go", "go.mod", "go.sum"} {
		contents, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(contents)
		hashes[name] = hex.EncodeToString(digest[:])
	}
	return hashes, nil
}

func verifyModuleSource(moduleDir string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(moduleDir, "source-provenance.json"))
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Schema           string `json:"schema"`
		Module           string `json:"module"`
		OriginRepository string `json:"origin_repository"`
		OriginRevision   string `json:"origin_revision"`
		Files            []struct {
			Origin string `json:"origin_path"`
			Copied string `json:"copied_path"`
			SHA256 string `json:"sha256"`
			Bytes  int    `json:"bytes"`
		} `json:"files"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil ||
		manifest.Schema != "gooo/decision-runtime-source-provenance/v1" ||
		manifest.Module != modulePath ||
		manifest.OriginRepository != "github.com/kimjooyoon/gooo-neural-decision-experiments" ||
		manifest.OriginRevision != originRev {
		return nil, fmt.Errorf("unexpected source provenance")
	}
	want := map[string]struct {
		origin string
		digest string
		bytes  int
	}{
		"model.go":  {"internal/decision/model.go", "5444e9a649a1cd2bd84f6b157b22e7fc912bc912ce046f5bda1f63cb6c6c5a02", 22717},
		"bridge.go": {"internal/decision/bridge.go", "08227b64151ee8296c743e409d5a7dbb61f65d4d841d4dc0e0ab3a6119607236", 3745},
		"ir.go":     {"internal/decision/ir.go", "14b259d00a581fe8924f973befa0149052f1f8dacb1f2f5fa03eef1eea24312d", 3625},
		"LICENSE":   {"LICENSE", "3ad2cd8fe84a937a0005a2934e377432f2f86fe10ff86c4242cc48f49ebf4947", 1074},
	}
	hashes := make(map[string]string, len(want))
	for _, entry := range manifest.Files {
		expected, ok := want[entry.Copied]
		if !ok || entry.Origin != expected.origin || entry.SHA256 != expected.digest || entry.Bytes != expected.bytes {
			return nil, fmt.Errorf("manifest changed")
		}
		contents, err := os.ReadFile(filepath.Join(moduleDir, entry.Copied))
		if err != nil || len(contents) != expected.bytes {
			return nil, fmt.Errorf("source file changed")
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		if actual != expected.digest {
			return nil, fmt.Errorf("source digest changed")
		}
		hashes[entry.Copied] = actual
		delete(want, entry.Copied)
	}
	if len(want) != 0 || len(hashes) != 4 {
		return nil, fmt.Errorf("source inventory changed")
	}
	return hashes, nil
}

func verifyBundleWeights(metadataPath string, model *decision.Model) (string, error) {
	metadataRaw, err := os.ReadFile(metadataPath)
	if err != nil {
		return "", err
	}
	var metadata struct {
		WeightsFile   string `json:"weights_file"`
		WeightsSHA256 string `json:"weights_sha256"`
	}
	if err := json.Unmarshal(metadataRaw, &metadata); err != nil || metadata.WeightsFile == "" || filepath.Base(metadata.WeightsFile) != metadata.WeightsFile {
		return "", fmt.Errorf("invalid bundle metadata")
	}
	weights, err := os.ReadFile(filepath.Join(filepath.Dir(metadataPath), metadata.WeightsFile))
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(weights)
	actual := hex.EncodeToString(digest[:])
	if actual != metadata.WeightsSHA256 || actual != model.WeightsSHA256() {
		return "", fmt.Errorf("weight digest mismatch")
	}
	return actual, nil
}

func publicModuleBuildInfo() (string, string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", fmt.Errorf("build information unavailable")
	}
	for _, dependency := range info.Deps {
		if dependency.Path != modulePath {
			continue
		}
		if dependency.Replace != nil || dependency.Version != "v0.1.0-experimental" || dependency.Sum == "" {
			return "", "", fmt.Errorf("module dependency is not the public tagged release")
		}
		return dependency.Version, dependency.Sum, nil
	}
	return "", "", fmt.Errorf("SDK dependency absent from build information")
}

func runConcurrent(model *decision.Model, text, expectedLabel string) (int, error) {
	const workers = 4
	const callsPerWorker = 16
	var wait sync.WaitGroup
	errorsFound := make(chan error, workers)
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			var workspace decision.Workspace
			var prediction decision.Prediction
			for call := 0; call < callsPerWorker; call++ {
				if err := model.PredictInto(text, &workspace, &prediction); err != nil {
					errorsFound <- err
					return
				}
				if model.PredictLabel(&prediction) != expectedLabel {
					errorsFound <- fmt.Errorf("concurrent prediction differed")
					return
				}
			}
		}()
	}
	wait.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			return 0, err
		}
	}
	return workers * callsPerWorker, nil
}

func exerciseClosedIRTable() error {
	labels := decision.Labels()
	intLeft := decision.Identifier{Name: "left", Type: decision.TypeInt}
	intRight := decision.Identifier{Name: "right", Type: decision.TypeInt}
	boolLeft := decision.Identifier{Name: "leftFlag", Type: decision.TypeBool}
	boolRight := decision.Identifier{Name: "rightFlag", Type: decision.TypeBool}
	for _, label := range labels {
		left, right := intLeft, intRight
		if label == "and" || label == "or" {
			left, right = boolLeft, boolRight
		}
		ir, err := decision.BuildTypedBinary(label, left, right)
		if err != nil {
			return err
		}
		if _, err := decision.AssembleGoExpression(ir); err != nil {
			return err
		}
	}
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
