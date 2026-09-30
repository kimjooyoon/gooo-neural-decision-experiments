package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	decision "github.com/kimjooyoon/gooo-decision-runtime"
)

const (
	modulePath       = "github.com/kimjooyoon/gooo-decision-runtime"
	moduleVersion    = "v0.1.0-experimental"
	originRevision   = "e18908b88eecaacbb0b80df4feb8cc7b9b4380bb"
	measuredCalls    = 50000
	explicitWarmups  = 128
	allocDiagnostics = 100
	inputText        = "Add the left value to the right value."
)

type receipt struct {
	Schema              string            `json:"schema"`
	ModulePath          string            `json:"module_path"`
	ModuleVersion       string            `json:"module_version"`
	ModuleH1            string            `json:"module_h1"`
	OriginRevision      string            `json:"origin_revision"`
	GoVersion           string            `json:"go_version"`
	GOOS                string            `json:"goos"`
	GOARCH              string            `json:"goarch"`
	SourceSHA256        map[string]string `json:"source_sha256"`
	BenchmarkSHA256     map[string]string `json:"benchmark_source_sha256"`
	InputSHA256         string            `json:"fixed_input_sha256"`
	WarmupCallsPerModel int               `json:"explicit_warmup_calls_per_model"`
	MeasuredCalls       int               `json:"measured_calls_per_model"`
	AllocDiagnostics    int               `json:"alloc_diagnostic_calls_per_model"`
	TrainingRuns        int               `json:"training_runs"`
	LayaCalls           int               `json:"laya_calls"`
	Models              []modelResult     `json:"models"`
	ProcessPeakRSS      peakRSSResult     `json:"process_peak_rss"`
	MeasurementNotes    []string          `json:"measurement_notes"`
}

type modelResult struct {
	Variant              string  `json:"variant"`
	WeightsSHA256        string  `json:"weights_sha256"`
	MeasuredCalls        int     `json:"measured_calls"`
	ExplicitWarmupCalls  int     `json:"explicit_warmup_calls"`
	MedianLatencyNS      int64   `json:"median_latency_ns"`
	P95LatencyNS         int64   `json:"p95_latency_ns"`
	MeanLatencyNS        float64 `json:"mean_latency_ns"`
	MeasuredWallNS       int64   `json:"measured_loop_wall_ns"`
	UserCPUNS            int64   `json:"process_user_cpu_ns"`
	SystemCPUNS          int64   `json:"process_system_cpu_ns"`
	TotalCPUNS           int64   `json:"process_total_cpu_ns"`
	CPUPercentOneCore    float64 `json:"process_cpu_percent_of_one_core"`
	AllocDiagnosticCalls int     `json:"alloc_diagnostic_calls"`
	AllocsPerRun         float64 `json:"allocs_per_diagnostic_call"`
}

type peakRSSResult struct {
	ValueBytes int64  `json:"value_bytes"`
	Scope      string `json:"scope"`
	UnitSource string `json:"unit_source"`
}

type cpuUsage struct {
	userNS   int64
	systemNS int64
}

func main() {
	if len(os.Args) != 6 {
		fail("expected three model metadata paths, SDK module directory, and output directory")
	}
	if runtime.GOOS != "darwin" {
		fail("this runner's peak RSS unit contract is verified for Darwin only")
	}
	if err := ensureFreshOutput(os.Args[5]); err != nil {
		fail("output directory must exist and benchmark.json must not exist")
	}

	sourceHashes, err := verifyModuleSource(os.Args[4])
	if err != nil {
		fail("public SDK source provenance verification failed")
	}
	benchmarkHashes, err := hashFiles([]string{"main.go", "go.mod", "go.sum"})
	if err != nil {
		fail("benchmark source fingerprinting failed")
	}
	version, checksum, err := publicModuleBuildInfo()
	if err != nil {
		fail("SDK is not the exact public tagged module without a replace")
	}
	if err := verifyDirectModuleRequirement(); err != nil {
		fail("go.mod does not pin the public SDK directly without replacement")
	}

	inputDigest := sha256.Sum256([]byte(inputText))
	results := make([]modelResult, 0, 3)
	for index, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
		model, err := decision.Load(os.Args[index+1])
		if err != nil || model.Variant() != variant {
			fail("a pinned model bundle could not be loaded")
		}
		weightsSHA, err := verifyBundleWeights(os.Args[index+1], model)
		if err != nil {
			fail("loaded model weight bytes did not match model metadata")
		}

		var workspace decision.Workspace
		var prediction decision.Prediction
		for i := 0; i < explicitWarmups; i++ {
			if err := model.PredictInto(inputText, &workspace, &prediction); err != nil {
				fail("explicit model warmup failed")
			}
		}

		latencies := make([]int64, 0, measuredCalls)
		before, err := readCPUUsage()
		if err != nil {
			fail("could not read process CPU time before measurement")
		}
		wallStart := time.Now()
		for i := 0; i < measuredCalls; i++ {
			callStart := time.Now()
			if err := model.PredictInto(inputText, &workspace, &prediction); err != nil {
				fail("measured prediction failed")
			}
			latencies = append(latencies, time.Since(callStart).Nanoseconds())
		}
		measuredWallNS := time.Since(wallStart).Nanoseconds()
		after, err := readCPUUsage()
		if err != nil {
			fail("could not read process CPU time after measurement")
		}
		userNS := after.userNS - before.userNS
		systemNS := after.systemNS - before.systemNS
		totalCPUNS := userNS + systemNS
		if userNS < 0 || systemNS < 0 || measuredWallNS <= 0 {
			fail("process CPU or wall measurement was invalid")
		}

		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		median := medianNS(latencies)
		p95 := latencies[int(math.Ceil(0.95*float64(len(latencies))))-1]
		mean := float64(sumNS(latencies)) / float64(len(latencies))
		allocs := testing.AllocsPerRun(allocDiagnostics, func() {
			if err := model.PredictInto(inputText, &workspace, &prediction); err != nil {
				panic(err)
			}
		})
		results = append(results, modelResult{
			Variant:              variant,
			WeightsSHA256:        weightsSHA,
			MeasuredCalls:        measuredCalls,
			ExplicitWarmupCalls:  explicitWarmups,
			MedianLatencyNS:      median,
			P95LatencyNS:         p95,
			MeanLatencyNS:        mean,
			MeasuredWallNS:       measuredWallNS,
			UserCPUNS:            userNS,
			SystemCPUNS:          systemNS,
			TotalCPUNS:           totalCPUNS,
			CPUPercentOneCore:    float64(totalCPUNS) / float64(measuredWallNS) * 100,
			AllocDiagnosticCalls: allocDiagnostics,
			AllocsPerRun:         allocs,
		})
	}

	usage, err := readPeakRSS()
	if err != nil {
		fail("could not read process peak resident set size")
	}
	out := receipt{
		Schema:              "gooo/public-sdk-predictinto-benchmark/v1",
		ModulePath:          modulePath,
		ModuleVersion:       version,
		ModuleH1:            checksum,
		OriginRevision:      originRevision,
		GoVersion:           runtime.Version(),
		GOOS:                runtime.GOOS,
		GOARCH:              runtime.GOARCH,
		SourceSHA256:        sourceHashes,
		BenchmarkSHA256:     benchmarkHashes,
		InputSHA256:         hex.EncodeToString(inputDigest[:]),
		WarmupCallsPerModel: explicitWarmups,
		MeasuredCalls:       measuredCalls,
		AllocDiagnostics:    allocDiagnostics,
		TrainingRuns:        0,
		LayaCalls:           0,
		Models:              results,
		ProcessPeakRSS: peakRSSResult{
			ValueBytes: usage.Maxrss,
			Scope:      "This process lifetime high-water mark sampled after all three variants, warmups, measured loops, and allocation diagnostics; not current RSS, not a per-model value, and not a delta.",
			UnitSource: "Darwin GETRUSAGE(2): RUSAGE_SELF describes the current process; ru_maxrss is maximum resident set size in bytes.",
		},
		MeasurementNotes: []string{
			"PredictInto hot-loop latency uses one fixed input and one reused Workspace and Prediction per variant; it makes no correctness or accuracy claim.",
			"The 128 explicit warmup calls occur before each measured loop. AllocsPerRun is a separate diagnostic of 100 counted repetitions and performs its own uncounted warmup.",
			"Measured wall and process CPU intervals cover the 50,000-call loop, including per-call timer reads and latency-sample appends; CPU percent is process CPU divided by wall time as a one-core ratio, not host utilization.",
			"Per-call timing and Go scheduler/OS activity add measurement overhead and run-to-run noise; results are observations from one bounded run.",
			"Ternary storage variants describe their bundle encoding; this benchmark makes no claim that the prediction kernel executes with 1.58-bit arithmetic.",
			"No training or Laya/provider calls were made.",
		},
	}
	if err := writeFreshJSON(os.Args[5], out); err != nil {
		fail("could not write benchmark receipt without overwriting")
	}
	fmt.Println("benchmark receipt written")
}

func verifyDirectModuleRequirement() error {
	contents, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}
	requirementFound := false
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "//") {
			continue
		}
		if fields[0] == "replace" {
			return fmt.Errorf("replace directive present")
		}
		if fields[0] == "require" && len(fields) == 3 && fields[1] == modulePath && fields[2] == moduleVersion {
			requirementFound = true
		}
		if fields[0] == modulePath && len(fields) == 2 && fields[1] == moduleVersion {
			requirementFound = true
		}
	}
	if !requirementFound {
		return fmt.Errorf("exact direct requirement absent")
	}
	return nil
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
		if dependency.Replace != nil || dependency.Version != moduleVersion || dependency.Sum == "" {
			return "", "", fmt.Errorf("not the public tagged module")
		}
		return dependency.Version, dependency.Sum, nil
	}
	return "", "", fmt.Errorf("SDK dependency absent")
}

func verifyModuleSource(moduleDir string) (map[string]string, error) {
	want := map[string]struct {
		digest string
		bytes  int
	}{
		"model.go":  {"5444e9a649a1cd2bd84f6b157b22e7fc912bc912ce046f5bda1f63cb6c6c5a02", 22717},
		"bridge.go": {"08227b64151ee8296c743e409d5a7dbb61f65d4d841d4dc0e0ab3a6119607236", 3745},
		"ir.go":     {"14b259d00a581fe8924f973befa0149052f1f8dacb1f2f5fa03eef1eea24312d", 3625},
	}
	hashes := make(map[string]string, len(want))
	for name, expected := range want {
		contents, err := os.ReadFile(filepath.Join(moduleDir, name))
		if err != nil || len(contents) != expected.bytes {
			return nil, fmt.Errorf("source file missing or size mismatch")
		}
		digest := sha256.Sum256(contents)
		actual := hex.EncodeToString(digest[:])
		if actual != expected.digest {
			return nil, fmt.Errorf("source digest mismatch")
		}
		hashes[name] = actual
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
		return "", fmt.Errorf("invalid model metadata")
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

func hashFiles(names []string) (map[string]string, error) {
	result := make(map[string]string, len(names))
	for _, name := range names {
		contents, err := os.ReadFile(name)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(contents)
		result[name] = hex.EncodeToString(digest[:])
	}
	return result, nil
}

func readCPUUsage() (cpuUsage, error) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return cpuUsage{}, err
	}
	return cpuUsage{
		userNS:   timevalNS(usage.Utime),
		systemNS: timevalNS(usage.Stime),
	}, nil
}

func timevalNS(value syscall.Timeval) int64 {
	return int64(value.Sec)*int64(time.Second) + int64(value.Usec)*int64(time.Microsecond)
}

func readPeakRSS() (syscall.Rusage, error) {
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		return syscall.Rusage{}, err
	}
	return usage, nil
}

func medianNS(values []int64) int64 {
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return values[middle-1]/2 + values[middle]/2 + (values[middle-1]%2+values[middle]%2)/2
}

func sumNS(values []int64) int64 {
	var sum int64
	for _, value := range values {
		sum += value
	}
	return sum
}

func ensureFreshOutput(outputDir string) error {
	info, err := os.Lstat(outputDir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output directory must already exist")
	}
	_, err = os.Lstat(filepath.Join(outputDir, "benchmark.json"))
	if err == nil {
		return fmt.Errorf("output exists")
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("could not inspect output path")
	}
	return nil
}

func writeFreshJSON(outputDir string, value any) error {
	path := filepath.Join(outputDir, "benchmark.json")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
