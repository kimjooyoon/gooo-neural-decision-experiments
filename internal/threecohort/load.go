// Package threecohort reconstructs the frozen native three-choice curriculum.
package threecohort

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

const DatasetSHA = "9a887dc09caf2f2b2b947641509328a2ee6f25dcefb6b52efe178fe8aff4fb3a"
const DatasetBytes = 10620412
const ManifestSHA = "1b1b8d4724c79ed5d7cb4aff79194a03451826c8a5acc7f80f97c18fdfa3b371"
const AuditSHA = "4c0c5347a53abdd3212f9897b198c6c0b42795ce82ac53e0faaf7051adae267f"

type View struct {
	ID        string                             `json:"id"`
	Group     string                             `json:"program_contract_group"`
	Family    string                             `json:"family"`
	Config    int                                `json:"configuration"`
	Goal      int                                `json:"desired_mask"`
	Language  string                             `json:"language"`
	Split     string                             `json:"split"`
	SourceSHA string                             `json:"original_source_sha256"`
	Parts     [3]string                          `json:"ordered_source_inputs"`
	Text      string                             `json:"complete_three_input"`
	Target    threecompositionstudy.FiniteTarget `json:"full_contract_target"`
	Prepared  *pathplan.PreparedPlan             `json:"-"`
	Plan      pathplan.Plan                      `json:"-"`
	Cases     []pathplan.TestCase                `json:"-"`
}

type input struct {
	ID       string `json:"decision_id"`
	Text     string `json:"text"`
	SHA      string `json:"input_sha256"`
	Natural  string `json:"natural_intent_sha256"`
	Original string `json:"original_intent_sha256"`
	Features string `json:"source_features_sha256"`
	Bytes    int    `json:"bytes"`
	Replaced bool   `json:"caller_prefix_replaced"`
}
type row struct {
	ID          string                             `json:"id"`
	Group       string                             `json:"program_contract_group"`
	Family      string                             `json:"family"`
	Config      int                                `json:"configuration"`
	Goal        int                                `json:"desired_mask"`
	Language    string                             `json:"language"`
	Split       string                             `json:"split"`
	SourceSHA   string                             `json:"source_sha256"`
	DocumentSHA string                             `json:"document_sha256"`
	CaptureSHA  string                             `json:"capture_sha256"`
	Inputs      []input                            `json:"inputs"`
	Text        string                             `json:"complete_three_input"`
	InputSHA    string                             `json:"input_sha256"`
	Target      threecompositionstudy.FiniteTarget `json:"full_contract_target"`
}

func SHA(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func Decode(raw []byte, v any) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

// ReconstructRow verifies one captured row against authored source and oracle.
// Full collection byte/order authority is established separately by Load.
func ReconstructRow(raw []byte) (View, error) {
	var r row
	if err := Decode(raw, &r); err != nil {
		return View{}, err
	}
	return prepare(r)
}

// Load binds all complete source inputs and ordered oracle labels to bytes
// published before any three-choice teacher observation or optimizer update.
func Load(filename string) ([]View, error) {
	s, err := os.Lstat(filename)
	if err != nil || !s.Mode().IsRegular() || s.Size() != DatasetBytes {
		return nil, errors.New("exact regular frozen curriculum required")
	}
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	scan := bufio.NewScanner(io.TeeReader(f, h))
	scan.Buffer(make([]byte, 32768), 1<<20)
	views := make([]View, 0, 3072)
	for scan.Scan() {
		if len(views) >= 3072 {
			return nil, errors.New("unexpected extra curriculum row")
		}
		var r row
		if err = Decode(scan.Bytes(), &r); err != nil {
			return nil, err
		}
		n := len(views)
		family := threecompositionstudy.Families[n/(24*8*2)]
		config, goal, language := (n/(8*2))%24, (n/2)%8, [2]string{"en", "ko"}[n%2]
		if r.Family != family || r.Config != config || r.Goal != goal || r.Language != language {
			return nil, errors.New("frozen family/configuration/goal/language order differs")
		}
		v, err := prepare(r)
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", n, err)
		}
		views = append(views, v)
	}
	if scan.Err() != nil {
		return nil, scan.Err()
	}
	if len(views) != 3072 || hex.EncodeToString(h.Sum(nil)) != DatasetSHA {
		return nil, errors.New("frozen curriculum bytes or denominator differs")
	}
	return views, nil
}

func prepare(r row) (View, error) {
	plan, err := threecompositionstudy.Fixture(r.Family, r.Config, r.Goal, r.Language)
	if err != nil {
		return View{}, err
	}
	group := fmt.Sprintf("%s-c%02d-goal%d", r.Family, r.Config, r.Goal)
	if r.ID != group+"-"+r.Language || r.Group != group || r.Split != threecompositionstudy.Split(r.Config) || len(r.Inputs) != 3 {
		return View{}, errors.New("complete three-coordinate source identity differs")
	}
	decoded, err := hex.DecodeString(r.CaptureSHA)
	if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != r.CaptureSHA {
		return View{}, errors.New("native capture identity missing")
	}
	target, err := threecompositionstudy.Target(plan, r.Family, r.Config, r.Goal)
	if err != nil || !reflect.DeepEqual(target, r.Target) {
		return View{}, errors.New("independent full passing-mask target differs")
	}
	cases, err := threecompositionstudy.Cases(r.Family, r.Config, r.Goal)
	if err != nil {
		return View{}, err
	}
	original, err := pathplan.Prepare(plan)
	if err != nil {
		return View{}, err
	}
	body, err := json.Marshal(original.Fallback().GoooBody())
	if err != nil {
		return View{}, err
	}
	source := []byte("package threecomposition\nnamespace threecomposition\nentity Integer id \"threecomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	doc := struct {
		Schema string              `json:"schema"`
		Plan   pathplan.Plan       `json:"path_plan"`
		Cases  []pathplan.TestCase `json:"test_cases"`
		Max    int                 `json:"max_attempts"`
	}{"gooo/body-codegen-typed-path-plan/v1", plan, cases, 8}
	docRaw, err := json.Marshal(doc)
	if err != nil || r.SourceSHA != SHA(source) || r.DocumentSHA != "sha256:"+SHA(docRaw) {
		return View{}, errors.New("original native source/document differs")
	}
	var parts [3]string
	for i, in := range r.Inputs {
		choice := plan.Decisions[i]
		_, natural, found := strings.Cut(choice.Intent, "intent: ")
		fields, err := original.SourceFeatures(choice.ID)
		if err != nil || !found {
			return View{}, errors.New("source facts or authored intention missing")
		}
		text, err := decision.EncodeSemanticContextInput(fields, natural)
		if err != nil || in.ID != choice.ID || in.Text != text || in.Bytes != len(text) || !in.Replaced || in.SHA != "sha256:"+SHA([]byte(text)) || in.Natural != "sha256:"+SHA([]byte(natural)) || in.Original != "sha256:"+SHA([]byte(choice.Intent)) || in.Features != "sha256:"+SHA(fields[:]) {
			return View{}, errors.New("complete ordered source input differs")
		}
		parts[i] = text
		plan.Decisions[i].Intent = text
	}
	text, err := jointdecision.EncodeThree(parts)
	if err != nil || r.Text != text || r.InputSHA != SHA([]byte(text)) {
		return View{}, errors.New("complete framed three-input identity differs")
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return View{}, err
	}
	return View{r.ID, r.Group, r.Family, r.Config, r.Goal, r.Language, r.Split, r.SourceSHA, parts, text, target, prepared, plan, cases}, nil
}
