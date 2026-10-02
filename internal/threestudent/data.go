// Package threestudent binds optimizer rows to frozen Go-observed source and
// teacher failures. It performs no model calls and never generates gold text.
package threestudent

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
)

const StatesSHA = "38930441c45ec03336a66c670028208e525131f385699140d793da96cbf4028b"
const StatesBytes = 19345322
const TeacherAuditSHA = "dc1bc90a5468b2f45a4e32501f0814a73ee2be7de56481c1edea3b7d414bfc6a"
const TeacherReportSHA = "95281f2ee55efbb45146fbc58203c4ac43f12e8630df04f2cca9cfcd7bc61050"
const InitialSeed = 20261031
const ShuffleSeed = 20261032
const RawCap int64 = 768 << 20

type Pin struct {
	SHA   string `json:"sha256"`
	Bytes int64  `json:"bytes"`
}

func FilePin(name string) (Pin, error) {
	s, err := os.Lstat(name)
	if err != nil || !s.Mode().IsRegular() {
		return Pin{}, errors.New("regular nonsymlink evidence required")
	}
	f, err := os.Open(name)
	if err != nil {
		return Pin{}, err
	}
	defer f.Close()
	h := sha256.New()
	var buffer [32768]byte
	n, err := io.CopyBuffer(h, f, buffer[:])
	return Pin{hex.EncodeToString(h.Sum(nil)), n}, err
}

func Load(dataset, directory, audit string) ([]threefeedback.State, error) {
	for name, expected := range map[string]string{audit: TeacherAuditSHA, filepath.Join(directory, "report.json"): TeacherReportSHA, filepath.Join(directory, "states.jsonl"): StatesSHA} {
		p, err := FilePin(name)
		if err != nil || p.SHA != expected {
			return nil, errors.New("frozen independently audited teacher bytes required")
		}
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]threecohort.View, len(views))
	for _, v := range views {
		byID[v.ID] = v
	}
	f, err := os.Open(filepath.Join(directory, "states.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), 1<<20)
	states := make([]threefeedback.State, 0, 10739)
	previous := ""
	initial := map[string]bool{}
	for scan.Scan() {
		var s threefeedback.State
		if err = threecohort.Decode(scan.Bytes(), &s); err != nil {
			return nil, err
		}
		v, ok := byID[s.ViewID]
		if !ok || s.ID <= previous || s.Group != v.Group || s.Family != v.Family || s.Config != v.Config || s.Language != v.Language || s.Split != v.Split || s.SourceSHA != v.SourceSHA || s.Target != v.Target.Joint || s.InputSHA != threecohort.SHA([]byte(s.Text)) {
			return nil, errors.New("complete state identity, group or finite source target differs")
		}
		if s.Phase == "initial" {
			if !reflect.DeepEqual(s, threefeedback.Initial(v)) || initial[s.ViewID] {
				return nil, errors.New("initial source state differs")
			}
			initial[s.ViewID] = true
		} else {
			if s.Phase != "feedback" || s.Split != "train" || len(s.Origins) == 0 || s.ID != "feedback/"+v.ID+"/"+s.InputSHA {
				return nil, errors.New("actual training-only failure state required")
			}
			parts, err := jointdecision.ThreeParts(s.Text)
			if err != nil {
				return nil, err
			}
			for i, part := range parts {
				if !strings.HasPrefix(part, v.Parts[i]+"\nfeedback: ") {
					return nil, errors.New("failure input must preserve every source/intent part")
				}
			}
		}
		var features [jointdecision.ThreeFeatureDim]float32
		if err = jointdecision.FeaturesIntoThree(s.Text, &features); err != nil {
			return nil, err
		}
		previous = s.ID
		states = append(states, s)
	}
	if scan.Err() != nil {
		return nil, scan.Err()
	}
	if len(states) != 10739 || len(initial) != 3072 {
		return nil, errors.New("frozen student-state denominator differs")
	}
	return states, nil
}

type Row struct {
	Index          int        `json:"feature_row_index"`
	ID             string     `json:"state_id"`
	ViewID         string     `json:"function_view"`
	Group          string     `json:"program_contract_group"`
	Language       string     `json:"language"`
	Split          string     `json:"split"`
	Phase          string     `json:"phase"`
	InputSHA       string     `json:"input_sha256"`
	SourceSHA      string     `json:"original_source_sha256"`
	Target         [8]float32 `json:"finite_soft_targets"`
	InitialWeight  float64    `json:"initial_arm_row_weight"`
	FeedbackWeight float64    `json:"feedback_arm_row_weight"`
}

// Rows gives each function group total weight one and each language one half.
// Repeated session origins never increase a distinct continuation's weight.
func Rows(states []threefeedback.State) ([]Row, error) {
	counts, initial := map[string]int{}, map[string]bool{}
	for _, s := range states {
		key := s.Group + "/" + s.Language
		if s.Language != "en" && s.Language != "ko" {
			return nil, errors.New("bilingual language required")
		}
		if s.Phase == "initial" {
			if initial[key] {
				return nil, errors.New("duplicate initial language view")
			}
			initial[key] = true
		} else if s.Phase == "feedback" && s.Split == "train" {
			counts[key]++
		} else {
			return nil, errors.New("no holdout failure rows allowed")
		}
	}
	result := make([]Row, len(states))
	totals := map[string][2]float64{}
	for i, s := range states {
		key := s.Group + "/" + s.Language
		if !initial[key] {
			return nil, errors.New("continuation without original contract")
		}
		a, b := 0., 0.
		if s.Phase == "initial" {
			a, b = .5, .5
			if counts[key] > 0 {
				b = .25
			}
		} else {
			b = .25 / float64(counts[key])
		}
		result[i] = Row{i, s.ID, s.ViewID, s.Group, s.Language, s.Split, s.Phase, s.InputSHA, s.SourceSHA, s.Target, a, b}
		sums := totals[key]
		sums[0] += a
		sums[1] += b
		totals[key] = sums
	}
	for key, total := range totals {
		if math.Abs(total[0]-.5) > 1e-12 || math.Abs(total[1]-.5) > 1e-12 {
			return nil, errors.New("per-language weight differs")
		}
		group, lang, _ := strings.Cut(key, "/")
		other := "en"
		if lang == "en" {
			other = "ko"
		}
		if _, ok := totals[group+"/"+other]; !ok {
			return nil, errors.New("unpaired bilingual group")
		}
	}
	return result, nil
}
