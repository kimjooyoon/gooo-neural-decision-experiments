package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

const datasetSHA = "2d1c9888a648d590e77253666b6e5ec848d375e47a3c99daf29dbde8bcbc7383"

type row struct {
	ID         string `json:"id"`
	Group      string `json:"program_contract_group"`
	Family     string `json:"family"`
	Config     int    `json:"configuration"`
	Goal       int    `json:"desired_mask"`
	Lang       string `json:"language"`
	Split      string `json:"split"`
	Feature    string `json:"feature_version"`
	Coordinate int    `json:"coordinate"`
	Input      struct {
		Text string `json:"text"`
	} `json:"input"`
	Joint    string                             `json:"joint_input"`
	Target   jointcompositionstudy.FiniteTarget `json:"full_contract_target"`
	Marginal [2]float32                         `json:"finite_soft_targets"`
}

func digest(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }

func readCurriculum(name string) (map[string]map[string][]sample, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 32<<20 {
		return nil, errors.New("bounded regular frozen dataset required")
	}
	h := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(f, h))
	scanner.Buffer(make([]byte, 4096), 32768)
	layers, paired, seen := map[string]map[string][]sample{}, map[string][2]*row{}, map[string]bool{}
	targets, n := map[string]jointcompositionstudy.FiniteTarget{}, 0
	for scanner.Scan() {
		var r row
		if err = decision.RejectDuplicateJSONKeys(scanner.Bytes()); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, err
		}
		if seen[r.ID] || r.Coordinate < 0 || r.Coordinate > 1 || r.Feature != decision.SemanticContextIntentFeatureVersion {
			return nil, errors.New("unique frozen semantic coordinates required")
		}
		if err = verifyTarget(r, targets); err != nil {
			return nil, err
		}
		seen[r.ID], n = true, n+1
		id := r.Group + "/" + r.Lang
		pair := paired[id]
		if pair[r.Coordinate] != nil {
			return nil, errors.New("duplicate function coordinate")
		}
		pair[r.Coordinate] = &r
		paired[id] = pair
		if err = addSingle(layers, r); err != nil {
			return nil, err
		}
	}
	if scanner.Err() != nil || n != 4608 || hex.EncodeToString(h.Sum(nil)) != datasetSHA || len(paired) != 2304 {
		return nil, errors.New("frozen dataset bytes or denominators differ")
	}
	for _, pair := range paired {
		if err = addJoint(layers, pair); err != nil {
			return nil, err
		}
	}
	return layers, nil
}

func verifyTarget(r row, memo map[string]jointcompositionstudy.FiniteTarget) error {
	target, found := memo[r.Group]
	if !found {
		plan, err := jointcompositionstudy.Fixture(r.Family, r.Config, r.Goal, r.Lang)
		if err != nil {
			return err
		}
		target, err = jointcompositionstudy.Target(plan, r.Family, r.Config, r.Goal)
		if err != nil {
			return err
		}
		memo[r.Group] = target
	}
	if r.Split != jointcompositionstudy.Split(r.Config) || !reflect.DeepEqual(target, r.Target) || r.Marginal != target.Marginals[r.Coordinate] {
		return errors.New("independent finite target reconstruction differs")
	}
	return nil
}

func item(r row, target [4]float32) sample {
	v := sample{ID: r.Group + "/" + r.Lang, Family: r.Family, Config: r.Config, Goal: r.Goal, Lang: r.Lang, Split: r.Split, Target: target}
	for i, mass := range target {
		if mass > 0 {
			v.Support |= 1 << i
		}
	}
	return v
}

func add(layers map[string]map[string][]sample, layer, sha string, v sample) {
	for _, key := range []string{layer, layer + "/" + v.Split} {
		if layers[key] == nil {
			layers[key] = map[string][]sample{}
		}
		layers[key][sha] = append(layers[key][sha], v)
	}
}

func featureDigest(features []float32) string {
	var raw [jointdecision.FeatureDim * 4]byte
	for i, f := range features {
		binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(f))
	}
	return digest(raw[:len(features)*4])
}

func addSingle(layers map[string]map[string][]sample, r row) error {
	var vector [decision.FeatureDim]float32
	if err := decision.SemanticContextFeaturesInto(r.Input.Text, &vector); err != nil {
		return err
	}
	v := item(r, [4]float32{r.Marginal[0], r.Marginal[1]})
	v.ID += "/" + r.ID
	add(layers, "independent_complete_text", digest([]byte(r.Input.Text)), v)
	add(layers, "independent_feature_vector", featureDigest(vector[:]), v)
	return nil
}

func addJoint(layers map[string]map[string][]sample, pair [2]*row) error {
	if pair[0] == nil || pair[1] == nil || pair[0].Joint != pair[1].Joint {
		return errors.New("complete joint coordinate pair required")
	}
	text, err := jointdecision.Encode([2]string{pair[0].Input.Text, pair[1].Input.Text})
	if err != nil || text != pair[0].Joint || !strings.HasPrefix(text, "gooo;joint2|") {
		return errors.New("complete joint text differs")
	}
	var vector [jointdecision.FeatureDim]float32
	if err = jointdecision.FeaturesInto(text, &vector); err != nil {
		return err
	}
	v := item(*pair[0], pair[0].Target.Joint)
	add(layers, "joint_complete_text", digest([]byte(text)), v)
	add(layers, "joint_feature_vector", featureDigest(vector[:]), v)
	return nil
}
