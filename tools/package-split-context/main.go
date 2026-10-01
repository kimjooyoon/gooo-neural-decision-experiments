// package-split-context publishes the preregistered feature-channel experiment.
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
const trainingRoot = "runs/split-context-judgment-mps-20261001"
const nativeRoot = "runs/split-context-native-contract-20261001"
const prefix = "research/split-context-gooo-judgment-20261001/"
const maxFile = 16 << 20

var privacy = regexp.MustCompile(`(?:hf_|ghp_|github_pat_|sk-)[A-Za-z0-9_-]{20,}|/Users/|/private/var/|Bearer\s+[A-Za-z0-9]`)
var modelPins = map[string]string{
	"v1/fp32":        "be75203287811c9e7f7c81d460e41c954a3868a1a53dae481eecda7dc18e0b3b",
	"v1/ptq_ternary": "2a2a7eb962a02b403f061a43348476f0cb8a53dbac883b1738947eda00945dd2",
	"v1/qat_ternary": "0f3c545f772a90977799dfc7ccf419c63d995aa08e42130977ad134b02211ca6",
	"v2/fp32":        "7998ca6e5cbe28455e0467a19bc77c03f99683d8623f35ebe95f79e905624fbd",
	"v2/ptq_ternary": "2f996c11983485c4e2fb1e9c2b08444042d8725a3bf0947025b318014d28ea50",
	"v2/qat_ternary": "6a0162e814c589ac833b3f3ac2b7eeff4dd48ad75a015ee822523c534995196d",
}
var fixed = map[string]string{
	"data/pairs.jsonl":                  "25e090b1c4d842b34cf7a1d3a994b7b6e4be70cb59389fe33420e5d8750574ce",
	"training/report.json":              "903e636e5a2c267517ff72f65413a88c8a8f4020b792294ce17b5b5cfaa126e3",
	"training/preexecution.json":        "9fcaa38b1b0f1891af99f068d2f30a356d282fc5901cb1d90d20897394d400db",
	"judgment/report.json":              "7a17a74f7c1c0c41dc3aae8f91175b8edd2d123eec560305ca9356db48eb0805",
	"feature-parity.json":               "ef045f3ccc149ee154a772ffa683c6900392b2aeb13474a00753de00af74131d",
	"native-contract/capture.json":      "0dc799e78549371cd3ec5eb42acc5c0a19da03c456f9086da5d8950f88f1b742",
	"native-contract/outcome.json":      "428ba06df0905419ee19c87926c3382be396428ea58d6e731640db2b7786f56e",
	"native-contract/preexecution.json": "e104942c698272d7b718486356f20d4b7ef40a41771070af9f9bd0a4a0d2c241",
}

func inventory() map[string]string {
	files := map[string]string{"README.md": "docs/split-context-judgment-results.md", "LICENSE": "LICENSE",
		"preregistration.md": "docs/split-context-judgment-preregistration.md",
		"data/pairs.jsonl":   "data/bilingual-gooo-pairs-v1/pairs.jsonl", "data/manifest.json": "data/bilingual-gooo-pairs-v1/manifest.json",
		"feature-parity.json": "studies/split-context-features-v2/parity.json"}
	for key := range modelPins {
		arm, variant, _ := strings.Cut(key, "/")
		for _, name := range []string{"model.json", "weights.bin"} {
			files["models/"+key+"/"+name] = trainingRoot + "/" + arm + "/models/" + variant + "/" + name
		}
	}
	for _, name := range []string{"preexecution.json", "report.json", "v1/go-parity.json", "v1/go-audit.json", "v2/go-parity.json", "v2/go-audit.json"} {
		files["training/"+name] = trainingRoot + "/" + name
	}
	for _, name := range []string{"preexecution.json", "report.json"} {
		files["judgment/"+name] = "runs/split-context-judgment-go-20261001/" + name
	}
	for _, name := range []string{"preexecution.json", "capture.json", "outcome.json"} {
		files["native-contract/"+name] = nativeRoot + "/" + name
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
	value := manifest{Schema: "gooo/split-context-publication/v1", Repository: repository, Prefix: prefix, Models: modelPins,
		Scope: "Experimental identical-initialization v1/v2 feature comparison with increased continuation cost, six own-model exports, reused Gooo bilingual pairs, Go numerical parity and finite two-path continuation. One actual SDK 0.2.8 native probe rejects v2 before inference; no native v2 adoption. Raw predictions are on GitHub. Existing core weights/card untouched. No private logs, paths, credentials, upstream Laya weights or default promotion."}
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
	return save(output, map[string]any{"schema": "gooo/split-context-verification/v1", "decision": "PASS", "repository": repository, "prefix": prefix, "revision": revision, "publication_files": count, "explicit_anonymous_requests": requests, "credentials_sent": false, "manifest_sha256": hash(raw), "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_steps": 0, "request_scope": "One public identity API and one explicit file request per selected file; automatic redirects and prior publications excluded."})
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
