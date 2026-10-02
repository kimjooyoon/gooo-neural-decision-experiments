// package-own-feedback-ci preserves the bounded synthetic Linux CI appendix.
package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
)

const ciHead = "c0a179dde93122faf449f6e42dbfdabce92cafb0"
const artifactSHA = "0c5366fe90751d758e5608073ebbdcc4b61dc26eb2575213b6b83215e237e6ff"
const runID = 36952987501
const artifactID = 11204911749

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}

func inventory() (map[string]bool, map[string]bool) {
	all, reports := map[string]bool{}, map[string]bool{}
	for _, n := range []string{"feedback-public-verification.json", "feedback-source-curriculum-audit.json", "feedback-teacher-audit.json", "feedback-sdk-frozen-audit.json", "feedback-native-frozen-audit.json", "feedback-native-frozen-process-summary.json", "feedback-new-go-model-audit.json"} {
		all[n], reports[n] = true, true
	}
	for _, n := range []string{"preexecution.json", "report.json", "independent-audit.json", "process-summary.json"} {
		p := "feedback-native-new/" + n
		all[p], reports[p] = true, true
	}
	for _, n := range []string{"preexecution.json", "report.json", "selection.json", "independent-audit-strengthened.json"} {
		p := "feedback-sdk-new/" + n
		all[p], reports[p] = true, true
	}
	for _, f := range jointcompositionstudy.Families {
		for g := 0; g < 4; g++ {
			for _, l := range []string{"en", "ko"} {
				for _, p := range []string{"offline", "selected", "uniform-initial", "set-initial", "set-feedback"} {
					for _, s := range []string{".json", "-execution.json"} {
						all[fmt.Sprintf("feedback-native-new/%s-goal%d-%s-%s%s", f, g, l, p, s)] = true
					}
				}
			}
		}
	}
	ids := []string{"offline", "reference-v1-independent", "reference-v1-joint"}
	for _, a := range []string{"uniform-initial", "set-initial", "set-feedback"} {
		for _, v := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			ids = append(ids, a+"-"+v)
		}
	}
	for _, s := range []string{"calibration", "development"} {
		for _, id := range ids {
			all["feedback-sdk-new/"+s+"-"+id+".jsonl"] = true
		}
	}
	return all, reports
}

func streamText(r io.Reader, w io.Writer, limit int64) (entry, error) {
	if limit < 0 {
		return entry{}, errors.New("nonnegative stream bound required")
	}
	h := sha256.New()
	c := &counter{W: w}
	s := bufio.NewScanner(io.TeeReader(io.LimitReader(r, limit+1), io.MultiWriter(c, h)))
	s.Buffer(make([]byte, 32768), 1<<20)
	for s.Scan() {
		if privateText.Match(s.Bytes()) {
			return entry{}, errors.New("private text in synthetic CI payload")
		}
	}
	if err := s.Err(); err != nil {
		return entry{}, err
	}
	if c.N > limit {
		return entry{}, errors.New("stream byte cap exceeded")
	}
	return entry{SHA: hex.EncodeToString(h.Sum(nil)), Bytes: c.N}, nil
}

type counter struct {
	N int64
	W io.Writer
}

func (c *counter) Write(b []byte) (int, error) { n, e := c.W.Write(b); c.N += int64(n); return n, e }

func rawJSON(b []byte, v any) error {
	if len(b) > 1<<20 {
		return errors.New("bounded report required")
	}
	if e := decision.RejectDuplicateJSONKeys(b); e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func requiredJobs() map[string]bool {
	r := map[string]bool{}
	for _, name := range []string{"own-model-sdk-context", "own-joint-feedback-v2-native", "compiler-model-dogfood", "joint-own-model-native", "native-body-integration", "runtime-and-artifacts", "native-typed-structural-inference", "compiler-context-own-model", "own-model-native-context", "paired-compiler-own-model"} {
		r[name] = true
	}
	return r
}
func validateRun(raw []byte) error {
	if privateText.Match(raw) {
		return errors.New("private run metadata")
	}
	var r struct {
		ID                 int64  `json:"databaseId"`
		Head               string `json:"headSha"`
		Status, Conclusion string
		Jobs               []struct{ Name, Status, Conclusion string }
	}
	if e := rawJSON(raw, &r); e != nil {
		return e
	}
	if r.ID != runID || r.Head != ciHead || r.Status != "completed" || r.Conclusion != "success" || len(r.Jobs) != 10 {
		return errors.New("exact completed ten-job CI run required")
	}
	seen := map[string]bool{}
	required := requiredJobs()
	for _, j := range r.Jobs {
		if !required[j.Name] || seen[j.Name] || j.Status != "completed" || j.Conclusion != "success" {
			return errors.New("successful unique CI jobs required")
		}
		seen[j.Name] = true
	}
	if !seen["own-joint-feedback-v2-native"] {
		return errors.New("v2 native job missing")
	}
	return nil
}

func inspectArchive(name string) ([]entry, map[string][]byte, error) {
	s, e := os.Lstat(name)
	if e != nil || !s.Mode().IsRegular() || s.Size() != 15146280 {
		return nil, nil, errors.New("exact regular Actions artifact required")
	}
	f, e := os.Open(name)
	if e != nil {
		return nil, nil, e
	}
	h := sha256.New()
	var b [32768]byte
	_, e = io.CopyBuffer(h, f, b[:])
	ce := f.Close()
	if e != nil || ce != nil || hex.EncodeToString(h.Sum(nil)) != artifactSHA {
		return nil, nil, errors.New("Actions artifact digest differs")
	}
	z, e := zip.OpenReader(name)
	if e != nil {
		return nil, nil, e
	}
	defer z.Close()
	allow, reports := inventory()
	if len(z.File) != len(allow) {
		return nil, nil, errors.New("fixed CI archive count differs")
	}
	seen := map[string]bool{}
	entries := []entry{}
	selected := map[string][]byte{}
	var total int64
	for _, f := range z.File {
		if !allow[f.Name] || seen[f.Name] || !f.Mode().IsRegular() || f.UncompressedSize64 > 256<<20 {
			return nil, nil, errors.New("CI artifact path/type/size differs")
		}
		seen[f.Name] = true
		total += int64(f.UncompressedSize64)
		if total > 512<<20 {
			return nil, nil, errors.New("expanded CI archive byte cap exceeded")
		}
		r, e := f.Open()
		if e != nil {
			return nil, nil, e
		}
		var out strings.Builder
		var w io.Writer = io.Discard
		if reports[f.Name] {
			if f.UncompressedSize64 > 1<<20 {
				r.Close()
				return nil, nil, errors.New("bounded CI report required")
			}
			w = &out
		}
		c := &counter{W: w}
		a, e := streamText(r, c, int64(f.UncompressedSize64))
		ce := r.Close()
		if e != nil || ce != nil || uint64(a.Bytes) != f.UncompressedSize64 {
			return nil, nil, errors.New("CI archive content/privacy/CRC differs")
		}
		a.Path = f.Name
		entries = append(entries, a)
		if reports[f.Name] {
			selected[f.Name] = []byte(out.String())
		}
	}
	return entries, selected, nil
}

func validateRecords(r map[string][]byte) error {
	for _, p := range []string{"feedback-native-new/preexecution.json", "feedback-sdk-new/preexecution.json"} {
		var x struct {
			Source string `json:"source_revision"`
		}
		if e := rawJSON(r[p], &x); e != nil {
			return e
		}
		if x.Source != ciHead {
			return errors.New("captured source head differs")
		}
	}
	var s struct {
		Status   string
		Sessions int `json:"actual_sdk_sessions"`
		Calls    int `json:"actual_model_predictions"`
	}
	if e := rawJSON(r["feedback-sdk-new/report.json"], &s); e != nil {
		return e
	}
	if s.Status != "PASS" || s.Sessions != 9216 || s.Calls != 18510 {
		return errors.New("SDK actual counts differ")
	}
	var n struct {
		Status      string
		Generations int `json:"native_generations_audited"`
		Executions  int `json:"actual_go_executions_audited"`
		Invocations int `json:"ordered_invocations_audited"`
		Calls       int `json:"recorded_model_predictions"`
	}
	if e := rawJSON(r["feedback-native-new/independent-audit.json"], &n); e != nil {
		return e
	}
	if n.Status != "PASS" || n.Generations != 240 || n.Executions != 240 || n.Invocations != 3840 || n.Calls != 473 {
		return errors.New("native audited counts differ")
	}
	var k struct {
		Status string
		Source string `json:"auditor_source_revision"`
		Calls  int    `json:"actual_model_predictions"`
		Parity int    `json:"actual_parity_predictions"`
	}
	if e := rawJSON(r["feedback-new-go-model-audit.json"], &k); e != nil {
		return e
	}
	if k.Status != "PASS" || k.Source != ciHead || k.Calls != 18450 || k.Parity != 432 {
		return errors.New("Go kernel audit differs")
	}
	return nil
}

func save(p string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}
func readReport(p string) ([]byte, error) {
	s, e := os.Lstat(p)
	if e != nil || !s.Mode().IsRegular() || s.Size() > 1<<20 {
		return nil, errors.New("bounded regular report required")
	}
	return os.ReadFile(p)
}

// Verify the anonymously retrieved bytes against the pinned Actions artifact,
// the independent manifest pin and every copied report's raw ZIP member.
func verify(bundle, output, pin string) error {
	if !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(pin) || output == "" {
		return errors.New("exact manifest pin and fresh verification output required")
	}
	if _, e := os.Stat(output); !os.IsNotExist(e) {
		return errors.New("fresh verification output required")
	}
	raw, e := readReport(filepath.Join(bundle, "appendix-manifest.json"))
	if e != nil {
		return e
	}
	h := sha256.Sum256(raw)
	if hex.EncodeToString(h[:]) != pin {
		return errors.New("independent appendix manifest pin differs")
	}
	var m struct {
		Schema    string
		Status    string
		Head      string  `json:"ci_head"`
		Run       int64   `json:"ci_run"`
		Artifact  int64   `json:"artifact_id"`
		SHA       string  `json:"artifact_sha256"`
		Publisher string  `json:"publisher_revision"`
		Members   []entry `json:"archive_members"`
		Scanned   bool    `json:"private_text_scanned"`
	}
	if e = rawJSON(raw, &m); e != nil {
		return e
	}
	if m.Schema != "gooo/own-joint-feedback-ci-appendix/v2" || m.Status != "PASS" || m.Head != ciHead || m.Run != runID || m.Artifact != artifactID || m.SHA != artifactSHA || !m.Scanned || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(m.Publisher) {
		return errors.New("CI appendix tuple differs")
	}
	entries, reports, e := inspectArchive(filepath.Join(bundle, "actions-artifact.zip"))
	if e != nil {
		return e
	}
	if !reflect.DeepEqual(entries, m.Members) {
		return errors.New("CI appendix member hashes differ")
	}
	if e = validateRecords(reports); e != nil {
		return e
	}
	for p, expected := range reports {
		actual, e := readReport(filepath.Join(bundle, p))
		if e != nil || string(actual) != string(expected) {
			return errors.New("CI appendix report differs from raw artifact")
		}
	}
	raw, e = readReport(filepath.Join(bundle, "run-metadata.json"))
	if e != nil {
		return e
	}
	if e = validateRun(raw); e != nil {
		return e
	}
	raw, e = readReport(filepath.Join(bundle, "README.md"))
	if e != nil || privateText.Match(raw) {
		return errors.New("private or invalid appendix README")
	}
	return save(output, map[string]any{"schema": "gooo/own-joint-feedback-ci-appendix-verification/v2", "status": "PASS", "ci_head": ciHead, "ci_run": runID, "artifact_sha256": artifactSHA, "manifest_sha256": pin, "raw_archive_members": len(entries), "public_reports_verified_against_raw_members": len(reports), "successful_ci_jobs": 10, "new_model_predictions": 0, "new_native_calls": 0})
}
func run(artifact, metadata, out, revision string) error {
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact committed publisher required")
	}
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	if e != nil || len(dirty) != 0 {
		return errors.New("clean publisher required")
	}
	entries, reports, e := inspectArchive(artifact)
	if e != nil {
		return e
	}
	if e = validateRecords(reports); e != nil {
		return e
	}
	raw, e := readReport(metadata)
	if e != nil {
		return e
	}
	if e = validateRun(raw); e != nil {
		return e
	}
	if out == "" {
		return errors.New("fresh publication directory required")
	}
	if e = os.Mkdir(out, 0755); e != nil {
		return e
	}
	for p, b := range reports {
		p = filepath.Join(out, p)
		if e = os.MkdirAll(filepath.Dir(p), 0755); e != nil {
			return e
		}
		if e = os.WriteFile(p, b, 0644); e != nil {
			return e
		}
	}
	if e = os.WriteFile(filepath.Join(out, "run-metadata.json"), raw, 0644); e != nil {
		return e
	}
	f, e := os.Open(artifact)
	if e != nil {
		return e
	}
	defer f.Close()
	w, e := os.OpenFile(filepath.Join(out, "actions-artifact.zip"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	var buffer [32768]byte
	_, e = io.CopyBuffer(w, f, buffer[:])
	ce := w.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if e = os.WriteFile(filepath.Join(out, "README.md"), []byte("# Own feedback v2 Linux CI evidence\n\nExact source CI c0a179d: ten jobs passed. This archive preserves 9,216 new SDK sessions, 240 new Gooo generations, 240 independently compiled Go executions and 3,840 ordered invocations. Kernel probes made 18,450 predictions, SDK sessions 18,510 and native generation 473; these are separate actual work. No new authored intentions or optimizer updates are claimed.\n\nThe verified raw Actions ZIP has 519 fixed synthetic members. All text was streamed for private host paths and credential patterns; ZIP CRC, size and SHA-256 were verified. Model and source evidence is the immutable main edition b4e9a3e50b50e893abc52a36f49eacf032aded50. The packaging tool performs zero inference and zero compiled execution.\n\nProcess measurements are environment-specific. Linux getrusage peak RSS can include memory inherited before exec; it is not steady-state model RAM. CPU is normalized to one core and can exceed 100%. Do not compare Linux CI and macOS timings as a controlled speedup. Original macOS measurements remain in the main edition.\n"), 0644); e != nil {
		return e
	}
	return save(filepath.Join(out, "appendix-manifest.json"), map[string]any{"schema": "gooo/own-joint-feedback-ci-appendix/v2", "status": "PASS", "ci_run": runID, "ci_head": ciHead, "publisher_revision": revision, "artifact_id": artifactID, "artifact_sha256": artifactSHA, "artifact_bytes": 15146280, "archive_members": entries, "public_reports": len(reports), "private_text_scanned": true, "successful_ci_jobs": 10, "recorded_sdk_sessions": 9216, "recorded_native_generations": 240, "recorded_compiled_go_executions": 240, "recorded_ordered_go_invocations": 3840, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0, "primary_model_revision": "b4e9a3e50b50e893abc52a36f49eacf032aded50"})
}

func main() {
	mode := flag.String("mode", "package", "package or verify")
	bundle := flag.String("bundle", "", "local or anonymous CI appendix")
	pin := flag.String("manifest-sha256", "", "independently pinned CI appendix manifest")
	artifact := flag.String("artifact", "", "pinned raw Actions ZIP")
	metadata := flag.String("run-metadata", "", "exact completed run JSON")
	out := flag.String("output", "", "fresh ignored publication directory")
	revision := flag.String("source-revision", "", "exact clean publisher revision")
	flag.Parse()
	var e error
	switch *mode {
	case "package":
		e = run(*artifact, *metadata, *out, *revision)
	case "verify":
		e = verify(*bundle, *out, *pin)
	default:
		e = errors.New("unknown CI publication mode")
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
