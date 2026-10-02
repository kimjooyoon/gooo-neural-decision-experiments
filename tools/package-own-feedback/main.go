// package-own-feedback exports a fixed, bounded synthetic own-model publication.
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
	"regexp"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

const repository = "asketeddy/gooo-joint-feedback-tiny-v2"
const base = "runs/own-joint-feedback-"
const teacher = base + "curriculum-20261002"
const sdk = base + "sdk-study-20261002"
const native = base + "native-main-20261002"
const models = "models/own-joint-feedback-v2"

var arms = []string{"uniform-initial", "set-initial", "set-feedback"}
var variants = []string{"fp32", "ptq_ternary", "qat_ternary"}
var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}
type member struct{ Public, Local string }
type manifest struct {
	Schema         string  `json:"schema"`
	Repository     string  `json:"repository"`
	Source         string  `json:"publisher_revision"`
	Files          []entry `json:"files"`
	Members        []entry `json:"archive_members"`
	PrivateScanned bool    `json:"private_text_scanned"`
	Steps          int     `json:"optimizer_updates"`
	Models         int     `json:"model_exports"`
	Generations    int     `json:"native_generations"`
	Executions     int     `json:"compiled_go_executions"`
	Invocations    int     `json:"ordered_go_invocations"`
	Predictions    int     `json:"native_model_predictions"`
}

func save(p string, v any) error {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	return os.WriteFile(p, append(b, '\n'), 0644)
}
func read(p string, v any) error {
	b, e := os.ReadFile(p)
	if e != nil {
		return e
	}
	if e = decision.RejectDuplicateJSONKeys(b); e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
func hashFile(p string) (entry, error) {
	s, e := os.Lstat(p)
	if e != nil || !s.Mode().IsRegular() || s.Size() > 256<<20 {
		return entry{}, errors.New("bounded regular publication file required")
	}
	f, e := os.Open(p)
	if e != nil {
		return entry{}, e
	}
	defer f.Close()
	h := sha256.New()
	var b [32768]byte
	n, e := io.CopyBuffer(h, f, b[:])
	if e != nil {
		return entry{}, e
	}
	return entry{SHA: hex.EncodeToString(h.Sum(nil)), Bytes: n}, nil
}
func scan(p string) error {
	f, e := os.Open(p)
	if e != nil {
		return e
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 32768), 1<<20)
	for s.Scan() {
		if privateText.Match(s.Bytes()) {
			return errors.New("private text in fixed publication input")
		}
	}
	return s.Err()
}
func copyFile(from, to string, binary bool) error {
	if _, e := hashFile(from); e != nil {
		return e
	}
	if !binary {
		if e := scan(from); e != nil {
			return e
		}
	}
	if e := os.MkdirAll(filepath.Dir(to), 0755); e != nil {
		return e
	}
	f, e := os.Open(from)
	if e != nil {
		return e
	}
	defer f.Close()
	out, e := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return e
	}
	var b [32768]byte
	_, e = io.CopyBuffer(out, f, b[:])
	ce := out.Close()
	if e != nil {
		return e
	}
	return ce
}
func archive(p string, members []member) ([]entry, error) {
	f, e := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	z := zip.NewWriter(f)
	seen := map[string]bool{}
	result := []entry{}
	var total int64
	for _, m := range members {
		if m.Public == "" || strings.Contains(m.Public, "..") || filepath.IsAbs(m.Public) || seen[m.Public] {
			return nil, errors.New("unsafe archive member")
		}
		seen[m.Public] = true
		a, e := hashFile(m.Local)
		if e != nil {
			return nil, e
		}
		if e = scan(m.Local); e != nil {
			return nil, e
		}
		total += a.Bytes
		if total > 640<<20 {
			return nil, errors.New("archive expanded byte cap")
		}
		h := &zip.FileHeader{Name: m.Public, Method: zip.Deflate}
		h.SetMode(0644)
		w, e := z.CreateHeader(h)
		if e != nil {
			return nil, e
		}
		r, e := os.Open(m.Local)
		if e != nil {
			return nil, e
		}
		var b [32768]byte
		_, e = io.CopyBuffer(w, r, b[:])
		ce := r.Close()
		if e != nil {
			return nil, e
		}
		if ce != nil {
			return nil, ce
		}
		a.Path = m.Public
		result = append(result, a)
	}
	if e = z.Close(); e != nil {
		return nil, e
	}
	return result, f.Sync()
}
func validateEvidence() error {
	var t struct {
		Status  string `json:"status"`
		States  int    `json:"student_state_rows"`
		Teacher struct {
			Sessions int `json:"actual_sdk_sessions"`
			Calls    int `json:"actual_teacher_predictions"`
		} `json:"actual_teacher"`
		Files map[string]string `json:"files_sha256"`
	}
	if e := read(teacher+"/independent-audit.json", &t); e != nil {
		return e
	}
	for _, n := range []string{"preexecution.json", "states.jsonl", "teacher-sessions.jsonl"} {
		a, e := hashFile(teacher + "/" + n)
		if e != nil || a.SHA != t.Files[n] {
			return errors.New("audited teacher source bytes differ")
		}
	}
	var s struct {
		Status   string `json:"status"`
		Sessions int    `json:"sdk_sessions_audited"`
		Calls    int    `json:"recorded_model_predictions"`
	}
	if e := read(sdk+"/independent-audit-strengthened.json", &s); e != nil {
		return e
	}
	var n struct {
		Status      string `json:"status"`
		Calls       int    `json:"native_generations_audited"`
		Executions  int    `json:"actual_go_executions_audited"`
		Invocations int    `json:"ordered_invocations_audited"`
		Predictions int    `json:"recorded_model_predictions"`
	}
	if e := read(native+"/independent-audit.json", &n); e != nil {
		return e
	}
	var training struct {
		Status string `json:"status"`
		Steps  int    `json:"optimizer_steps"`
	}
	if e := read(models+"/report.json", &training); e != nil {
		return e
	}
	if t.Status != "PASS" || t.States != 6463 || t.Teacher.Sessions != 6144 || t.Teacher.Calls != 12399 || s.Status != "PASS" || s.Sessions != 9216 || s.Calls != 18510 || n.Status != "PASS" || n.Calls != 240 || n.Executions != 240 || n.Invocations != 3840 || n.Predictions != 473 || training.Status != "TRAINED_AND_EXPORTED" || training.Steps != 3600 {
		return errors.New("complete own feedback audits required")
	}
	return nil
}
func packageBundle(out, revision string) error {
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	if e != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact committed publisher required")
	}
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	if e != nil || len(dirty) != 0 {
		return errors.New("clean publication source required")
	}
	if e = validateEvidence(); e != nil {
		return e
	}
	if _, e = os.Stat(out); !os.IsNotExist(e) {
		return errors.New("fresh bundle required")
	}
	if e = os.MkdirAll(out, 0755); e != nil {
		return e
	}
	files := map[string]string{"LICENSE": "LICENSE", "protocol.md": "docs/own-joint-completeness-feedback-preregistration-20261002.md", "results.md": "docs/own-joint-feedback-results-20261002.md", "training-report.json": models + "/report.json", "training-preexecution.json": models + "/preexecution.json", "model-audit.json": "publication/own-joint-feedback-go-parity-and-memory-20261002.json", "teacher-report.json": teacher + "/report.json", "teacher-audit.json": teacher + "/independent-audit.json", "sdk-report.json": sdk + "/report.json", "sdk-preexecution.json": sdk + "/preexecution.json", "calibration-selection.json": sdk + "/selection.json", "sdk-audit.json": sdk + "/independent-audit-strengthened.json", "native-report.json": native + "/report.json", "native-preexecution.json": native + "/preexecution.json", "native-audit.json": native + "/independent-audit.json"}
	for _, arm := range arms {
		files[arm+"/go-parity.json"] = models + "/" + arm + "/go-parity.json"
		for _, v := range variants {
			model := models + "/" + arm + "/models/" + v + "/model.json"
			if _, e = jointdecision.Load(model); e != nil {
				return e
			}
			for _, f := range []string{"model.json", "weights.bin"} {
				files[arm+"/models/"+v+"/"+f] = models + "/" + arm + "/models/" + v + "/" + f
			}
		}
	}
	names := []string{}
	for n := range files {
		names = append(names, n)
	}
	slices.Sort(names)
	payloads := []entry{}
	for _, n := range names {
		if e = copyFile(files[n], filepath.Join(out, n), strings.HasSuffix(n, "weights.bin")); e != nil {
			return fmt.Errorf("%s: %w", n, e)
		}
		a, e := hashFile(filepath.Join(out, n))
		if e != nil {
			return e
		}
		a.Path = n
		payloads = append(payloads, a)
	}
	if e = os.WriteFile(filepath.Join(out, "README.md"), []byte(modelCard()), 0644); e != nil {
		return e
	}
	a, e := hashFile(filepath.Join(out, "README.md"))
	if e != nil {
		return e
	}
	a.Path = "README.md"
	payloads = append(payloads, a)
	provenance, e := writeProvenance(out, revision)
	if e != nil {
		return e
	}
	for _, p := range provenance {
		a, e := hashFile(filepath.Join(out, p))
		if e != nil {
			return e
		}
		a.Path = p
		payloads = append(payloads, a)
	}
	members, e := rawMembers()
	if e != nil {
		return e
	}
	entries, e := archive(filepath.Join(out, "raw-evidence.zip"), members)
	if e != nil {
		return e
	}
	a, e = hashFile(filepath.Join(out, "raw-evidence.zip"))
	if e != nil {
		return e
	}
	a.Path = "raw-evidence.zip"
	payloads = append(payloads, a)
	slices.SortFunc(payloads, func(a, b entry) int { return strings.Compare(a.Path, b.Path) })
	return save(filepath.Join(out, "publication-manifest.json"), manifest{"gooo/own-joint-feedback-publication/v2", repository, revision, payloads, entries, true, 3600, 9, 240, 240, 3840, 473})
}
func main() {
	mode := flag.String("mode", "package", "package, verify or fetch")
	out := flag.String("output", "", "bundle or verification output")
	revision := flag.String("source-revision", "", "committed publisher")
	bundle := flag.String("bundle", "", "local frozen bundle")
	publicRevision := flag.String("public-revision", "", "immutable public Hub commit")
	flag.Parse()
	var e error
	switch *mode {
	case "package":
		e = packageBundle(*out, *revision)
	case "verify":
		e = verify(*bundle, *out)
	case "fetch":
		e = fetch(*publicRevision, *bundle, *out)
	default:
		e = errors.New("unknown publication mode")
	}
	if e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}
