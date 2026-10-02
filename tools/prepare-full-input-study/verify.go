package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type bankManifest struct {
	Schema      string                      `json:"schema"`
	Status      string                      `json:"status"`
	Source      string                      `json:"source_revision"`
	Protocol    string                      `json:"protocol_sha256"`
	SourceRows  int                         `json:"source_rows"`
	Rows        int                         `json:"prepared_rows"`
	Files       map[string]threestudent.Pin `json:"files"`
	Counts      map[string]int              `json:"counts_by_split_phase"`
	Forms       map[string]int              `json:"forms_by_split"`
	Rejected    map[string]int              `json:"rejected_full_forms"`
	Versions    map[string]string           `json:"feature_versions"`
	Groups      map[string]int              `json:"groups"`
	InitialSHA  string                      `json:"initial_state_sha256"`
	Dim         int                         `json:"feature_dim"`
	Parameters  int                         `json:"parameters"`
	Steps       int                         `json:"planned_optimizer_updates"`
	Updates     int                         `json:"optimizer_updates"`
	Predictions int                         `json:"model_predictions"`
	StudyBytes  int64                       `json:"new_study_bytes_before_manifest"`
	StudyCap    int64                       `json:"new_study_cap_bytes"`
	WholeCap    int64                       `json:"whole_own_three_cap_bytes"`
}

// Reconstruct the frozen training forms separately from the preparation code.
func expectedText(original, language, form string) (string, error) {
	parts, err := jointdecision.ThreeParts(original)
	if err != nil {
		return "", err
	}
	var pre, post string
	if language == "en" {
		switch form {
		case "original":
		case "request-prefix":
			pre = "Request: "
		case "please-prefix":
			pre = "Please follow this instruction: "
		case "request-suffix":
			post = " This is the request."
		case "please-suffix":
			post = " Please follow this instruction."
		default:
			return "", errors.New("unknown English form")
		}
	} else if language == "ko" {
		switch form {
		case "original":
		case "request-prefix":
			pre = "요청: "
		case "please-prefix":
			pre = "다음 지시를 따라 주세요: "
		case "request-suffix":
			post = " 이것이 요청입니다."
		case "please-suffix":
			post = " 이 지시를 따라 주세요."
		default:
			return "", errors.New("unknown Korean form")
		}
	} else {
		return "", errors.New("unknown language")
	}
	out := "gooo;joint3|"
	for _, part := range parts {
		at := strings.Index(part, ";intent: ")
		if at < 0 {
			return "", errors.New("source header missing")
		}
		at += len(";intent: ")
		full := part[:at] + pre + part[at:] + post
		out += strconv.Itoa(len(full)) + ":" + full
	}
	return out, nil
}

func verifyPrepared(dataset, teacher, audit, directory, revision, output string) error {
	if err := exactSource(revision); err != nil {
		return err
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh replay report required")
	}
	raw, err := os.ReadFile(filepath.Join(directory, "manifest.json"))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("bounded manifest required")
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var m bankManifest
	if err = threecohort.Decode(raw, &m); err != nil {
		return err
	}
	protocol, err := os.ReadFile(fullinputstudy.Protocol)
	if err != nil {
		return err
	}
	if m.Schema != "gooo/full-input-training-bank/v1" || m.Status != "PREPARED_PENDING_INDEPENDENT_REPLAY" || m.Source != revision || m.Protocol != threecohort.SHA(protocol) || m.SourceRows != 10739 || m.Rows < 10739 || m.Rows > 49599 || m.Dim != 768 || m.Parameters != 2072 || m.Steps != 6400 || m.Updates != 0 || m.Predictions != 0 || len(m.Files) != 7 || len(m.Versions) != 2 || m.Versions["positioned"] != jointdecision.ThreeFeatureVersion || m.Versions["bag"] != jointdecision.ThreeBagFeatureVersion || m.StudyCap != studyCap || m.WholeCap != wholeCap || m.StudyBytes < 0 || m.StudyBytes > studyCap-reserve {
		return errors.New("frozen bank contract differs")
	}
	for _, name := range []string{"preexecution.json", "initial-fp32.bin", "order-control.json", "rows.jsonl", "rejections.jsonl", "positioned-f32le.bin", "bag-f32le.bin"} {
		pin, ok := m.Files[name]
		if !ok || pin.Bytes < 0 || pin.Bytes > studyCap {
			return errors.New("bank file extent differs")
		}
		info, e := os.Lstat(filepath.Join(directory, name))
		if e != nil || !info.Mode().IsRegular() || info.Size() != pin.Bytes {
			return errors.New("bounded regular bank file required")
		}
		got, e := threestudent.FilePin(filepath.Join(directory, name))
		if e != nil || got != pin {
			return errors.New("bank file digest differs")
		}
	}
	if m.InitialSHA != threecohort.SHA(threestudent.InitialWeights()) || m.Files["initial-fp32.bin"].SHA != m.InitialSHA || m.Files["positioned-f32le.bin"].Bytes != int64(m.Rows)*768*4 || m.Files["bag-f32le.bin"].Bytes != int64(m.Rows)*768*4 {
		return errors.New("initializer or feature extent differs")
	}
	controlRaw, err := os.ReadFile(filepath.Join(directory, "order-control.json"))
	if err != nil {
		return err
	}
	var control fullinputstudy.OrderControl
	if err = threecohort.Decode(controlRaw, &control); err != nil {
		return err
	}
	expectedControl, err := fullinputstudy.MakeOrderControl()
	if err != nil || control != expectedControl {
		return errors.New("negative order control differs")
	}
	states, err := threestudent.Load(dataset, teacher, audit)
	if err != nil {
		return err
	}
	rows, err := threestudent.Rows(states)
	if err != nil {
		return err
	}
	data, err := os.Open(filepath.Join(directory, "rows.jsonl"))
	if err != nil {
		return err
	}
	defer data.Close()
	scan := bufio.NewScanner(data)
	scan.Buffer(make([]byte, 32768), 1<<20)
	reject, err := os.Open(filepath.Join(directory, "rejections.jsonl"))
	if err != nil {
		return err
	}
	defer reject.Close()
	rscan := bufio.NewScanner(reject)
	rscan.Buffer(make([]byte, 32768), 1<<20)
	var featureFiles [2]*os.File
	for i, name := range []string{"positioned-f32le.bin", "bag-f32le.bin"} {
		featureFiles[i], err = os.Open(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		defer featureFiles[i].Close()
	}
	counts, formsCount, rejected := map[string]int{}, map[string]int{}, map[string]int{}
	groups := map[string]map[string]bool{"train": {}, "calibration": {}, "development": {}}
	weights := map[string]float64{}
	index, rejections := 0, 0
	for original, state := range states {
		forms := []string{"original"}
		if state.Split == "train" {
			forms = []string{"original", "request-prefix", "please-prefix", "request-suffix", "please-suffix"}
		}
		accepted := map[string]string{}
		for _, form := range forms {
			text, e := expectedText(state.Text, state.Language, form)
			if e != nil {
				return e
			}
			var scratch [768]float32
			if e = jointdecision.FeaturesIntoThree(text, &scratch); e == nil {
				accepted[form] = text
				continue
			}
			var sizes [3]int
			rest := strings.TrimPrefix(text, "gooo;joint3|")
			for i := range sizes {
				colon := strings.IndexByte(rest, ':')
				if colon < 0 {
					return errors.New("expected rejection framing")
				}
				sizes[i], e = strconv.Atoi(rest[:colon])
				if e != nil || sizes[i] > len(rest)-colon-1 {
					return errors.New("expected rejection extent")
				}
				rest = rest[colon+1+sizes[i]:]
			}
			if rest != "" {
				return errors.New("trailing expected rejection framing")
			}
			if !rscan.Scan() {
				return errors.New("missing retained full rejection")
			}
			var record struct {
				Index     int    `json:"original_row_index"`
				ID        string `json:"original_state_id"`
				Original  string `json:"original_input_sha256"`
				Form      string `json:"form"`
				Text      string `json:"text"`
				SHA       string `json:"input_sha256"`
				PartBytes [3]int `json:"part_bytes"`
				Reason    string `json:"reason"`
			}
			if e = threecohort.Decode(rscan.Bytes(), &record); e != nil {
				return e
			}
			if record.Index != original || record.ID != state.ID || record.Original != state.InputSHA || record.Form != form || record.Text != text || record.SHA != threecohort.SHA([]byte(text)) || record.Reason != "complete input exceeds fixed representation bound" {
				return errors.New("full rejection differs")
			}
			if record.PartBytes != sizes {
				return errors.New("rejected part byte length differs")
			}
			rejected[state.Split+"/"+state.Language+"/"+form]++
			rejections++
		}
		for _, form := range forms {
			text, ok := accepted[form]
			if !ok {
				continue
			}
			if !scan.Scan() {
				return errors.New("prepared row missing")
			}
			var got preparedRow
			if err = threecohort.Decode(scan.Bytes(), &got); err != nil {
				return err
			}
			want := rows[original]
			want.Index = index
			want.ID = state.ID + "/form/" + form
			want.InputSHA = threecohort.SHA([]byte(text))
			want.InitialWeight /= float64(len(accepted))
			want.FeedbackWeight /= float64(len(accepted))
			expected := preparedRow{want, original, state.ID, state.InputSHA, rows[original].FeedbackWeight, form, len(accepted), text}
			if !reflect.DeepEqual(got, expected) {
				return fmt.Errorf("source-bound prepared row %d differs", index)
			}
			for version, f := range featureFiles {
				var actual [768 * 4]byte
				if _, err = io.ReadFull(f, actual[:]); err != nil {
					return err
				}
				var features [768]float32
				if version == 0 {
					err = jointdecision.FeaturesIntoThree(text, &features)
				} else {
					err = jointdecision.FeaturesIntoThreeBag(text, &features)
				}
				if err != nil {
					return err
				}
				var expectedBytes [768 * 4]byte
				for i, value := range features {
					binary.LittleEndian.PutUint32(expectedBytes[i*4:], math.Float32bits(value))
				}
				if !bytes.Equal(actual[:], expectedBytes[:]) {
					return errors.New("independently reconstructed features differ")
				}
			}
			counts[state.Split+"/"+state.Phase]++
			formsCount[state.Split+"/"+form]++
			groups[state.Split][state.Group] = true
			weights[state.Split+"/"+state.Group] += want.FeedbackWeight
			index++
		}
	}
	if scan.Scan() || scan.Err() != nil || rscan.Scan() || rscan.Err() != nil {
		return errors.New("extra or invalid bank journal data")
	}
	for _, f := range featureFiles {
		var one [1]byte
		if n, e := f.Read(one[:]); n != 0 || e != io.EOF {
			return errors.New("trailing feature bytes")
		}
	}
	actualGroups := map[string]int{"train": len(groups["train"]), "calibration": len(groups["calibration"]), "development": len(groups["development"])}
	if index != m.Rows || !reflect.DeepEqual(counts, m.Counts) || !reflect.DeepEqual(formsCount, m.Forms) || !reflect.DeepEqual(rejected, m.Rejected) || !reflect.DeepEqual(actualGroups, m.Groups) {
		return errors.New("recomputed preparation denominators differ")
	}
	for _, weight := range weights {
		if math.Abs(weight-1) > 1e-10 {
			return errors.New("reconstructed group weights differ")
		}
	}
	report := map[string]any{"schema": "gooo/full-input-training-bank-replay/v1", "status": "PASS", "source_revision": revision, "manifest_sha256": threecohort.SHA(raw), "protocol_sha256": m.Protocol, "source_rows": 10739, "prepared_rows": index, "full_rejections_replayed": rejections, "feature_values_recomputed": index * 768 * 2, "groups": actualGroups, "model_predictions": 0, "optimizer_updates": 0, "native_executions": 0, "scope": "Independent source/form/target/weight reconstruction and both Go feature arrays; no trained-model quality claim."}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err = f.Write(append(encoded, '\n')); err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}
