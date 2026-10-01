// joint-composition-study measures frozen own-model finite construction in Go.
package main

import (
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
	"sort"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const datasetSHA = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"
const protocolSHA = "03655ff443402b3e7d1b59fa780d4c8e68b750e1093b504a61de3fbf691d6a28"
const nativeMain = "f4813dc6251037767c8cff7295ccfdab2b044ff2"

var variants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}
var arms = [2]string{"independent", "joint"}

func hash(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func save(name string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, append(raw, '\n'), 0644)
}
func decodeFile(name string, value any) error {
	raw, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}
func fileHash(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	if _, err = io.CopyBuffer(h, f, buffer[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

type row struct {
	ID         string `json:"id"`
	Group      string `json:"program_contract_group"`
	Pair       string `json:"bilingual_decision_pair"`
	Family     string `json:"family"`
	Config     int    `json:"configuration"`
	Desired    int    `json:"desired_mask"`
	Language   string `json:"language"`
	Split      string `json:"split"`
	Feature    string `json:"feature_version"`
	Coordinate int    `json:"coordinate"`
	Input      struct {
		ID      string `json:"decision_id"`
		Text    string `json:"text"`
		SHA     string `json:"input_sha256"`
		Natural string `json:"natural_intent_sha256"`
	} `json:"input"`
	Options   [2]string                          `json:"eligible_labels"`
	Targets   [2]float32                         `json:"finite_soft_targets"`
	Finite    jointcompositionstudy.FiniteTarget `json:"full_contract_target"`
	SourceSHA string                             `json:"source_sha256"`
	JointText string                             `json:"joint_input"`
	JointSHA  string                             `json:"joint_input_sha256"`
}
type view struct {
	ID   string
	Rows [2]row
}

func loadViews(curriculum string) ([]view, error) {
	sha, err := fileHash(filepath.Join(curriculum, "dataset.jsonl"))
	if err != nil || sha != datasetSHA {
		return nil, errors.New("frozen valid dataset required")
	}
	f, err := os.Open(filepath.Join(curriculum, "dataset.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 32768)
	groups, seen := map[string]*view{}, map[string]bool{}
	counts, n := map[string]int{}, 0
	for scanner.Scan() {
		var r row
		if err = decision.RejectDuplicateJSONKeys(scanner.Bytes()); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		if seen[r.ID] || r.Coordinate < 0 || r.Coordinate > 1 || r.Input.SHA != "sha256:"+hash([]byte(r.Input.Text)) {
			return nil, errors.New("row/input identity differs")
		}
		seen[r.ID], n = true, n+1
		counts[r.Feature+"/"+r.Split]++
		if r.Split == "train" {
			continue
		}
		id := r.Group + "-" + r.Language + "-" + r.Feature
		if groups[id] == nil {
			groups[id] = &view{ID: id}
		}
		if groups[id].Rows[r.Coordinate].ID != "" {
			return nil, errors.New("duplicate coordinate")
		}
		groups[id].Rows[r.Coordinate] = r
	}
	if scanner.Err() != nil || n != 4608 {
		return nil, errors.New("bounded dataset count differs")
	}
	for _, feature := range []string{decision.SemanticContextIntentFeatureVersion} {
		for split, count := range map[string]int{"train": 3072, "calibration": 768, "development": 768} {
			if counts[feature+"/"+split] != count {
				return nil, errors.New("split count differs")
			}
		}
	}
	result := make([]view, 0, len(groups))
	for _, v := range groups {
		if _, _, err = prepareView(*v); err != nil {
			return nil, err
		}
		result = append(result, *v)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func prepareView(v view) (*pathplan.PreparedPlan, []pathplan.TestCase, error) {
	a, b := v.Rows[0], v.Rows[1]
	if a.ID == "" || b.ID == "" || a.Group != b.Group || a.Language != b.Language || a.Feature != b.Feature ||
		a.Family != b.Family || a.Config != b.Config || a.Desired != b.Desired || !reflect.DeepEqual(a.Finite, b.Finite) {
		return nil, nil, errors.New("paired coordinate provenance differs")
	}
	plan, err := jointcompositionstudy.Fixture(a.Family, a.Config, a.Desired, a.Language)
	if err != nil {
		return nil, nil, err
	}
	target, err := jointcompositionstudy.Target(plan, a.Family, a.Config, a.Desired)
	if err != nil || !reflect.DeepEqual(target, a.Finite) {
		return nil, nil, errors.New("independent full target differs")
	}
	base, err := pathplan.Prepare(plan)
	if err != nil {
		return nil, nil, err
	}
	body, err := json.Marshal(base.Fallback().GoooBody())
	if err != nil {
		return nil, nil, err
	}
	source := []byte("package freshcomposition\nnamespace freshcomposition\nentity Integer id \"freshcomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	if a.SourceSHA != hash(source) || b.SourceSHA != hash(source) {
		return nil, nil, errors.New("original native source differs")
	}
	for i, r := range v.Rows {
		if r.Input.ID != plan.Decisions[i].ID || r.Options != [2]string{plan.Decisions[i].Options[0].Label, plan.Decisions[i].Options[1].Label} || r.Targets != target.Marginals[i] {
			return nil, nil, errors.New("choice target/labels differ")
		}
		natural := plan.Decisions[i].Intent[strings.LastIndex(plan.Decisions[i].Intent, "intent: ")+8:]
		if r.Input.Natural != "sha256:"+hash([]byte(natural)) {
			return nil, nil, errors.New("natural suffix differs")
		}
		if a.Feature == decision.SemanticContextIntentFeatureVersion {
			fields, e := base.SourceFeatures(plan.Decisions[i].ID)
			if e != nil {
				return nil, nil, e
			}
			text, e := decision.EncodeSemanticContextInput(fields, natural)
			if e != nil || text != r.Input.Text {
				return nil, nil, errors.New("native semantic source array differs")
			}
		}
		plan.Decisions[i].Intent = r.Input.Text
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return nil, nil, err
	}
	joint, err := jointdecision.Encode([2]string{plan.Decisions[0].Intent, plan.Decisions[1].Intent})
	if err != nil || joint != a.JointText || joint != b.JointText || hash([]byte(joint)) != a.JointSHA || a.JointSHA != b.JointSHA {
		return nil, nil, errors.New("ordered full joint input differs")
	}
	cases, err := jointcompositionstudy.Cases(a.Family, a.Config, a.Desired)
	return prepared, cases, err
}

func preflight(output, revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean source required")
	}
	raw, err := os.ReadFile("docs/joint-path-composition-preregistration-20261002.md")
	if err != nil || hash(raw) != protocolSHA {
		return errors.New("frozen protocol differs")
	}
	raw, err = os.ReadFile("docs/joint-path-composition-prefixture-amendment-20261002.md")
	if err != nil || hash(raw) != "4f8b87715e3e23b6077f0dc80b2120d519b97135dff5c2c2ede1cd70583f33d1" {
		return errors.New("frozen prefixture amendment differs")
	}
	if _, err = os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh output required")
	}
	return os.MkdirAll(output, 0755)
}

func main() {
	curriculum := flag.String("curriculum", "", "frozen joint source-bound curriculum")
	models := flag.String("models", "", "six own model exports")
	output := flag.String("output", "", "fresh study output")
	revision := flag.String("source-revision", "", "exact clean runner source")
	flag.Parse()
	if flag.NArg() != 0 || *curriculum == "" || *models == "" || *output == "" || *revision == "" {
		fmt.Fprintln(os.Stderr, "complete study arguments required")
		os.Exit(2)
	}
	if err := run(*curriculum, *models, *output, *revision); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
