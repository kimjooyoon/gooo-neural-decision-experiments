// package-bilingual-judgment publishes an explicit experimental model appendix.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

const repository = "asketeddy/gooo-feedback-path-tiny-v1"
const trainingRoot = "runs/bilingual-judgment-mps-20261001"
const nativeRoot = "runs/bilingual-judgment-native-main-20261001"
const prefix = "research/bilingual-gooo-judgment-20261001/"
const maxFile = 16 << 20

var privacy = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var modelPins = map[string]string{
	"control/fp32":        "d19d98f6a4f2fc55524332a79db4e65ce019e5172f0e891d633e896c8f0115e6",
	"control/ptq_ternary": "88b2cdda4fd9bd9bcd12caea597eaf3ac407fb63915dd4def1621960640b8ec3",
	"control/qat_ternary": "91e4ac583e3e078e03d1ac88c303574c469e7683560c3dd74d736c846f6736d7",
	"paired/fp32":         "ccb48c6fd0f76458a231d0bd1266e88252708aa6f64093742aead6a5ede27dc3",
	"paired/ptq_ternary":  "5233de7a91a997358de890c6431cf0fa2fd6fcd4d3dfb10a8cd851349028b6c9",
	"paired/qat_ternary":  "47524ff50c41b5bde0be95a7447126ee20abfae59f2e2fdc1e3f78c42c5613d0",
}
var fixed = map[string]string{
	"data/pairs.jsonl":           "25e090b1c4d842b34cf7a1d3a994b7b6e4be70cb59389fe33420e5d8750574ce",
	"training/report.json":       "fc3b1ff3bee93306a66f5cd1cf80e379f93cae40cca3bf9e9f9569ac678994c8",
	"training/preexecution.json": "535d154552df0780b9c9600b272920878f9de405c9a3710785c511989b2f3c02",
	"judgment/report.json":       "89a07a45360b472b1e4343acbffb2912efbad5a56c2cfdd1c0d82ac808807775",
	"native/report.json":         "d1679a36e8a05d7af007f1effb4ebdccda501fbe2cac9b6269a33c4130d761cf",
	"native/audit.json":          "ca48a4b670ae115480908b75cc630ab2efa9208b1b60901d2d6a0867e2f53dd5",
}

func inventory() map[string]string {
	files := map[string]string{"README.md": "docs/bilingual-judgment-results.md", "LICENSE": "LICENSE",
		"preregistration.md": "docs/bilingual-judgment-preregistration.md",
		"data/pairs.jsonl":   "data/bilingual-gooo-pairs-v1/pairs.jsonl", "data/manifest.json": "data/bilingual-gooo-pairs-v1/manifest.json",
		"training/attempts.json":    "publication/bilingual-judgment-training-attempts-20261001.json",
		"main-ci/finalization.json": "publication/native-path-diagnosis-main-push-finalization-20261001.json"}
	for key := range modelPins {
		arm, variant, _ := strings.Cut(key, "/")
		for _, name := range []string{"model.json", "weights.bin"} {
			files["models/"+key+"/"+name] = trainingRoot + "/" + arm + "/models/" + variant + "/" + name
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "control/go-parity.json", "control/go-audit.json", "paired/go-parity.json", "paired/go-audit.json"} {
		files["training/"+name] = trainingRoot + "/" + name
	}
	for _, name := range []string{"preexecution.json", "report.json"} {
		files["judgment/"+name] = "runs/bilingual-judgment-go-20261001/" + name
	}
	for _, name := range []string{"preexecution.json", "report.json", "audit.json",
		"execution-3999a9a5a3218e6b4c1b4ac80564351334e60f30bb71cb27c749bc264474981c.json",
		"execution-e3114b2376adbc2821bb57135586eace3f125c0116c9dd5b6ca825935ea1db94.json"} {
		files["native/"+name] = nativeRoot + "/" + name
	}
	return files
}

type entry struct {
	Path  string `json:"path"`
	SHA   string `json:"sha256"`
	Bytes int    `json:"bytes"`
}
type manifest struct {
	Schema     string            `json:"schema"`
	Repository string            `json:"repository"`
	Prefix     string            `json:"prefix"`
	Models     map[string]string `json:"model_metadata_sha256"`
	Files      []entry           `json:"files"`
	Scope      string            `json:"scope"`
}

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func read(path string) ([]byte, error) {
	i, err := os.Lstat(path)
	if err != nil || !i.Mode().IsRegular() || i.Size() <= 0 || i.Size() > maxFile {
		return nil, errors.New("bounded regular publication file required")
	}
	return os.ReadFile(path)
}
func save(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}
func sources() (manifest, error) {
	value := manifest{Schema: "gooo/bilingual-judgment-publication/v1", Repository: repository, Prefix: prefix, Models: modelPins,
		Scope: "Experimental own-model control and bilingual consistency weights, reused synthetic Gooo pairs, negative comparisons, actual adopted-main compiler dogfood and explicit full-contract continuation. Full raw prediction/native captures are on GitHub. Existing core weights/card untouched. No private logs, paths, credentials, upstream Laya weights or default checkpoint promotion."}
	for key, pin := range modelPins {
		arm, variant, _ := strings.Cut(key, "/")
		model, err := decision.LoadPath(trainingRoot + "/" + arm + "/models/" + variant + "/model.json")
		if err != nil || model.MetadataSHA256() != pin {
			return value, errors.New("frozen trained model metadata or weight digest differs")
		}
	}
	files := inventory()
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	total := 0
	for _, name := range names {
		raw, err := read(files[name])
		if err != nil {
			return value, err
		}
		if !strings.HasSuffix(name, "/weights.bin") && privacy.Match(raw) {
			return value, errors.New("private publication text rejected")
		}
		if pin := fixed[name]; pin != "" && hash(raw) != pin {
			return value, errors.New("fixed evidence digest differs")
		}
		total += len(raw)
		if total > 20<<20 {
			return value, errors.New("publication exceeds total budget")
		}
		value.Files = append(value.Files, entry{name, hash(raw), len(raw)})
	}
	return value, nil
}
func pack(bundle string) error {
	if bundle == "" {
		return errors.New("bundle path required")
	}
	if _, err := os.Lstat(bundle); !os.IsNotExist(err) {
		return errors.New("fresh publication directory required")
	}
	value, err := sources()
	if err != nil {
		return err
	}
	if err = os.MkdirAll(bundle, 0755); err != nil {
		return err
	}
	for _, e := range value.Files {
		raw, err := read(inventory()[e.Path])
		if err != nil || hash(raw) != e.SHA {
			return errors.New("publication source changed during packaging")
		}
		target := filepath.Join(bundle, e.Path)
		if err = os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err = os.WriteFile(target, raw, 0644); err != nil {
			return err
		}
	}
	return save(filepath.Join(bundle, "publication-manifest.json"), value)
}
func verify(bundle, revision, output string) error {
	if revision != "" && !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable HF revision required")
	}
	expected, err := sources()
	if err != nil {
		return err
	}
	raw, err := read(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var actual manifest
	if err = json.Unmarshal(raw, &actual); err != nil || !reflect.DeepEqual(actual, expected) {
		return errors.New("manifest differs from fixed source allowlist")
	}
	allowed := map[string]entry{}
	for _, e := range actual.Files {
		allowed[e.Path] = e
	}
	allowed["publication-manifest.json"] = entry{"publication-manifest.json", hash(raw), len(raw)}
	count := 0
	if err = filepath.WalkDir(bundle, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("publication symlink rejected")
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(bundle, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		e, ok := allowed[rel]
		if !ok {
			return errors.New("unlisted publication file rejected")
		}
		data, err := read(path)
		if err != nil || len(data) != e.Bytes || hash(data) != e.SHA {
			return errors.New("publication bytes differ")
		}
		count++
		return nil
	}); err != nil {
		return err
	}
	if count != len(allowed) {
		return errors.New("missing publication file")
	}
	requests := 0
	if revision != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		client := &http.Client{Timeout: 20 * time.Second}
		fetch := func(url string) ([]byte, error) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return nil, err
			}
			response, err := client.Do(request)
			requests++
			if err != nil {
				return nil, errors.New("anonymous public request failed")
			}
			defer response.Body.Close()
			data, err := io.ReadAll(io.LimitReader(response.Body, maxFile+1))
			if err != nil || response.StatusCode != http.StatusOK || len(data) > maxFile {
				return nil, errors.New("bounded public response required")
			}
			return data, nil
		}
		metaRaw, err := fetch("https://huggingface.co/api/models/" + repository + "/revision/" + revision)
		if err != nil {
			return err
		}
		var meta struct {
			ID      string `json:"id"`
			SHA     string `json:"sha"`
			Private *bool  `json:"private"`
		}
		if err = json.Unmarshal(metaRaw, &meta); err != nil || meta.ID != repository || meta.SHA != revision || meta.Private == nil || *meta.Private {
			return errors.New("public immutable repository identity differs")
		}
		names := make([]string, 0, len(allowed))
		for name := range allowed {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			data, err := fetch("https://huggingface.co/" + repository + "/resolve/" + revision + "/" + prefix + name)
			if err != nil || len(data) != allowed[name].Bytes || hash(data) != allowed[name].SHA {
				return errors.New("anonymous immutable publication digest differs")
			}
		}
	}
	return save(output, map[string]any{"schema": "gooo/bilingual-judgment-verification/v1", "decision": "PASS", "repository": repository, "prefix": prefix, "revision": revision, "publication_files": count, "explicit_anonymous_requests": requests, "credentials_sent": false, "manifest_sha256": hash(raw), "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_steps": 0, "request_scope": "One public identity API and one explicit file request per selected file; automatic redirects and prior publications excluded."})
}
func main() {
	mode := flag.String("mode", "pack", "pack or verify")
	bundle := flag.String("bundle", "", "allowlisted bundle directory")
	revision := flag.String("revision", "", "immutable public revision")
	output := flag.String("output", "", "verification receipt")
	flag.Parse()
	var err error
	if *mode == "pack" {
		err = pack(*bundle)
	} else if *mode == "verify" {
		err = verify(*bundle, *revision, *output)
	} else {
		err = errors.New("unknown mode")
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
