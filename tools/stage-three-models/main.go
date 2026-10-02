// stage-three-models copies only digest-pinned public training artifacts and
// source files. HF CLI transports the closed folder; Go verifies its bytes.
package main

import (
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
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const repository = "asketeddy/gooo-three-choice-feedback-tiny-v1"
const trained = "own-three-training-mps-20261002"

var privateText = regexp.MustCompile(`/Users/|/home/|/private/var/|hf_[A-Za-z0-9]{20,}|gh[pousr]_[A-Za-z0-9]{20,}|github_pat_|Authorization: Bearer`)

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}
type publication struct {
	Schema     string  `json:"schema"`
	Repository string  `json:"repository"`
	Status     string  `json:"status"`
	Files      []entry `json:"files"`
	Bytes      int64   `json:"total_bytes"`
	Scope      string  `json:"scope"`
}

func main() {
	mode := flag.String("mode", "models", "models, hf or verify-hf")
	destination := flag.String("destination", "", "fresh public copy or staged HF directory")
	index := flag.String("training-manifest", "publication/own-three-choice-training-bundle-20261002.json", "exact complete raw bundle manifest")
	revision := flag.String("public-revision", "", "immutable HF commit for anonymous verify-hf")
	receipt := flag.String("verification-report", "", "fresh anonymous receipt")
	flag.Parse()
	var err error
	if *mode == "publish-hf" {
		err = publishHF(*destination)
	} else if *mode == "provenance" {
		err = provenance(*destination)
	} else if *mode == "verify-hf" {
		err = verifyHF(*destination, *revision, *receipt)
	} else {
		err = stage(*mode, *destination, *index)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func stage(mode, destination, index string) error {
	if mode != "models" && mode != "hf" {
		return errors.New("unknown stage mode")
	}
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		return errors.New("fresh public destination required")
	}
	raw, err := os.ReadFile(index)
	if err != nil {
		return err
	}
	var bundle struct {
		Schema string `json:"schema"`
		Status string `json:"status"`
		Files  []struct {
			Name  string `json:"name"`
			SHA   string `json:"sha256"`
			Bytes int64  `json:"bytes"`
		} `json:"files"`
		Archive threestudent.Pin `json:"archive"`
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &bundle); err != nil {
		return err
	}
	if bundle.Schema != "gooo/three-choice-training-public-bundle/v1" || bundle.Status != "TRAINED_GO_KERNEL_AUDITED" || len(bundle.Files) != 642 {
		return errors.New("complete audited training publication required")
	}
	pinned := map[string]threestudent.Pin{}
	for _, p := range bundle.Files {
		pinned[p.Name] = threestudent.Pin{SHA: p.SHA, Bytes: p.Bytes}
	}
	sources := map[string]string{}
	for _, arm := range []string{"uniform-initial", "set-initial", "set-feedback"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, n := range []string{"model.json", "weights.bin"} {
				name := trained + "/" + arm + "/models/" + variant + "/" + n
				path := filepath.Join("runs", filepath.FromSlash(name))
				pin, err := threestudent.FilePin(path)
				if err != nil || pin != pinned[name] {
					return errors.New("actual student model byte pin differs")
				}
				target := arm + "/models/" + variant + "/" + n
				if mode == "hf" {
					target = "models/" + arm + "/" + variant + "/" + n
				}
				sources[target] = path
			}
		}
	}
	for _, n := range []string{"preexecution.json", "report.json"} {
		name := trained + "/" + n
		pin, err := threestudent.FilePin(filepath.Join("runs", name))
		if err != nil || pin != pinned[name] {
			return errors.New("actual optimization report pin differs")
		}
		target := n
		if mode == "hf" {
			target = "training/" + n
		}
		sources[target] = filepath.Join("runs", name)
	}
	if mode == "hf" {
		for target, path := range map[string]string{
			"README.md": "docs/own-three-choice-hf-model-card-20261002.md", "LICENSE": "LICENSE",
			"evidence/source-teacher.zip": "publication/own-three-choice-curriculum-20261002.zip", "evidence/source-teacher-manifest.json": "publication/own-three-choice-curriculum-bundle-20261002.json",
			"evidence/training.zip": "publication/own-three-choice-training-20261002.zip", "evidence/training-manifest.json": index,
			"verification/go-kernel-audit.json": "publication/own-three-choice-model-go-audit-20261002.json", "verification/preparation-audit.json": "publication/own-three-choice-training-input-audit-20261002.json",
			"source/preregistration.md": "docs/own-three-choice-completeness-preregistration-20261002.md", "source/training-design.md": "docs/own-three-choice-training-design-20261002.md",
			"source/optimizer.py": "training/train_own_three_feedback_v1.py", "source/export-core.py": "training/train_pilot_v2.py", "source/training-results.md": "docs/own-three-choice-training-results-20261002.md",
			"provenance.jsonld": "publication/own-three-choice-training-prov-20261002.jsonld",
		} {
			sources[target] = path
		}
		for target, member := range map[string]string{"verification/go-kernel-audit.json": "go-model-audit.json", "verification/preparation-audit.json": "preparation-audit.json"} {
			pin, err := threestudent.FilePin(sources[target])
			if err != nil || pin != pinned[member] {
				return errors.New("actual independent audit pin differs")
			}
		}
		pin, err := threestudent.FilePin(sources["evidence/training.zip"])
		if err != nil || pin != bundle.Archive {
			return errors.New("complete training ZIP differs")
		}
	}
	if err = os.MkdirAll(destination, 0755); err != nil {
		return err
	}
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	m := publication{Schema: "gooo/three-choice-model-publication-allowlist/v1", Repository: repository, Status: "STAGED_NOT_REMOTE_VERIFIED", Scope: "Nine own fresh three-choice exports and complete raw source/teacher/training evidence. Go kernel and fixed optimization budget verified; full SDK/native study and default promotion pending. No host paths, tokens or private source are staged."}
	var buffer [32768]byte
	for _, name := range names {
		p, err := threestudent.FilePin(sources[name])
		if err != nil || p.Bytes > 64<<20 {
			return errors.New("bounded regular source artifact required")
		}
		if !strings.HasSuffix(name, ".bin") && !strings.HasSuffix(name, ".zip") {
			raw, err := os.ReadFile(sources[name])
			if err != nil || privateText.Match(raw) {
				return errors.New("private staged text rejected")
			}
			if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".jsonld") {
				if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
					return err
				}
				var v any
				if err = json.Unmarshal(raw, &v); err != nil {
					return err
				}
				decoded, _ := json.Marshal(v)
				if privateText.Match(decoded) {
					return errors.New("private decoded JSON rejected")
				}
			}
		}
		path := filepath.Join(destination, filepath.FromSlash(name))
		if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		from, err := os.Open(sources[name])
		if err != nil {
			return err
		}
		to, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			from.Close()
			return err
		}
		h := sha256.New()
		n, copyErr := io.CopyBuffer(io.MultiWriter(to, h), from, buffer[:])
		fromErr := from.Close()
		toErr := to.Close()
		if copyErr != nil || fromErr != nil || toErr != nil || n != p.Bytes || hex.EncodeToString(h.Sum(nil)) != p.SHA {
			return errors.New("staged byte identity differs")
		}
		m.Files = append(m.Files, entry{name, p.SHA, p.Bytes})
		m.Bytes += p.Bytes
	}
	if m.Bytes > 64<<20 {
		return errors.New("staged folder cap exceeded; original artifacts preserved")
	}
	raw, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(destination, "publication-manifest.json"), append(raw, '\n'), 0644)
	if err == nil {
		fmt.Printf("STAGED: %s, %d allowlisted files, %d bytes; remote verification pending\n", mode, len(m.Files), m.Bytes)
	}
	return err
}

func sha(raw []byte) string { return threecohort.SHA(raw) }

func publishHF(directory string) error {
	raw, err := os.ReadFile(filepath.Join(directory, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var m publication
	if err = threecohort.Decode(raw, &m); err != nil {
		return err
	}
	if m.Schema != "gooo/three-choice-model-publication-allowlist/v1" || m.Repository != repository || m.Status != "STAGED_NOT_REMOTE_VERIFIED" || len(m.Files) != 34 {
		return errors.New("closed public HF allowlist required")
	}
	args := []string{"upload", repository, directory, ".", "--repo-type", "model", "--commit-message", "Publish fresh three-choice own models and complete training evidence; SDK/native study pending", "--include", "publication-manifest.json"}
	for _, p := range m.Files {
		if filepath.IsAbs(p.Path) || strings.HasPrefix(p.Path, "../") || filepath.ToSlash(filepath.Clean(p.Path)) != p.Path {
			return errors.New("relative public path required")
		}
		pin, err := threestudent.FilePin(filepath.Join(directory, filepath.FromSlash(p.Path)))
		if err != nil || pin.SHA != p.SHA || pin.Bytes != p.Bytes {
			return errors.New("allowlisted upload bytes changed")
		}
		args = append(args, p.Path)
	}
	command := exec.Command("hf", args...)
	command.Stdout, command.Stderr = os.Stdout, os.Stderr
	return command.Run()
}
