// Package jointcohort reconstructs the immutable source-bound joint corpus.
package jointcohort

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

const DatasetSHA = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"

type View struct {
	ID         string                             `json:"id"`
	Group      string                             `json:"program_contract_group"`
	Family     string                             `json:"family"`
	Config     int                                `json:"configuration"`
	Goal       int                                `json:"desired_mask"`
	Language   string                             `json:"language"`
	Split      string                             `json:"split"`
	SourceSHA  string                             `json:"original_source_sha256"`
	JointInput string                             `json:"joint_input"`
	Target     jointcompositionstudy.FiniteTarget `json:"full_contract_target"`
	Prepared   *pathplan.PreparedPlan             `json:"-"`
	Cases      []pathplan.TestCase                `json:"-"`
}

type sourceRow struct {
	ID         string `json:"id"`
	Group      string `json:"program_contract_group"`
	Family     string `json:"family"`
	Config     int    `json:"configuration"`
	Goal       int    `json:"desired_mask"`
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
	Joint     string                             `json:"joint_input"`
	JointSHA  string                             `json:"joint_input_sha256"`
	SourceSHA string                             `json:"source_sha256"`
	Options   [2]string                          `json:"eligible_labels"`
	Marginals [2]float32                         `json:"finite_soft_targets"`
	Target    jointcompositionstudy.FiniteTarget `json:"full_contract_target"`
}

func SHA(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func Load(filename string) ([]View, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 32<<20 {
		return nil, errors.New("bounded regular curriculum required")
	}
	h := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(f, h))
	scanner.Buffer(make([]byte, 4096), 32768)
	paired, seen, n := map[string][2]*sourceRow{}, map[string]bool{}, 0
	for scanner.Scan() {
		var r sourceRow
		if err = decision.RejectDuplicateJSONKeys(scanner.Bytes()); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		if seen[r.ID] || r.Coordinate < 0 || r.Coordinate > 1 || r.Feature != decision.SemanticContextIntentFeatureVersion {
			return nil, errors.New("unique canonical semantic rows required")
		}
		seen[r.ID], n = true, n+1
		id := r.Group + "/" + r.Language
		pair := paired[id]
		if pair[r.Coordinate] != nil {
			return nil, errors.New("duplicate source coordinate")
		}
		pair[r.Coordinate] = &r
		paired[id] = pair
	}
	if scanner.Err() != nil || n != 4608 || len(paired) != 2304 || hex.EncodeToString(h.Sum(nil)) != DatasetSHA {
		return nil, errors.New("frozen source bytes or denominators differ")
	}
	views := make([]View, 0, len(paired))
	for _, pair := range paired {
		view, err := prepare(pair)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	slices.SortFunc(views, func(a, b View) int { return cmp.Compare(a.ID, b.ID) })
	return views, nil
}
