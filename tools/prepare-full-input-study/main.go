package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/fullinputstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type preparedRow struct {
	threestudent.Row
	OriginalIndex    int     `json:"original_row_index"`
	OriginalID       string  `json:"original_state_id"`
	OriginalInputSHA string  `json:"original_input_sha256"`
	OriginalWeight   float64 `json:"original_feedback_arm_row_weight"`
	Form             string  `json:"form"`
	AcceptedForms    int     `json:"accepted_forms"`
	Text             string  `json:"text"`
}

func main() {
	dataset := flag.String("dataset", "runs/own-three-composition-curriculum-fixed-20261002/dataset.jsonl", "source-bound corpus")
	teacher := flag.String("teacher", "runs/own-three-teacher-curriculum-20261002", "frozen teacher evidence")
	audit := flag.String("teacher-audit", "publication/own-three-choice-teacher-audit-20261002.json", "frozen independent teacher audit")
	output := flag.String("output", "", "fresh own-three-full-input phase under runs")
	revision := flag.String("source-revision", "", "exact clean published source")
	verify := flag.String("verify-report", "", "verify an existing prepared bank by independent reconstruction")
	storageReport := flag.String("storage-report", "", "inventory a future phase without creating it")
	flag.Parse()
	var err error
	if *storageReport != "" {
		err = writeStoragePreflight(*output, *revision, *storageReport)
	} else if *verify != "" {
		err = verifyPrepared(*dataset, *teacher, *audit, *output, *revision, *verify)
	} else {
		err = prepare(*dataset, *teacher, *audit, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func prepare(dataset, teacher, audit, output, revision string) (failure error) {
	if err := exactSource(revision); err != nil {
		return err
	}
	store, err := newStorage(output)
	if err != nil {
		return err
	}
	states, err := threestudent.Load(dataset, teacher, audit)
	if err != nil {
		return err
	}
	rows, err := threestudent.Rows(states)
	if err != nil {
		return err
	}
	protocol, err := os.ReadFile(fullinputstudy.Protocol)
	if err != nil {
		return err
	}
	if err = os.Mkdir(output, 0755); err != nil {
		return err
	}
	defer func() {
		if failure != nil {
			_ = store.json("failure.json", map[string]any{"status": "FAILED_PREFIX_RETAINED", "error": failure.Error(), "optimizer_updates": 0, "model_predictions": 0})
		}
	}()
	if err = store.json("preexecution.json", map[string]any{"schema": "gooo/full-input-preparation/v1", "source_revision": revision, "protocol_sha256": threecohort.SHA(protocol), "dataset_sha256": threecohort.DatasetSHA, "teacher_states_sha256": threestudent.StatesSHA, "prior_own_three_bytes": store.prior, "new_study_cap_bytes": studyCap, "whole_own_three_cap_bytes": wholeCap, "available_bytes": store.available, "optimizer_updates": 0, "model_predictions": 0, "source_rows": len(rows)}); err != nil {
		return err
	}
	if err = store.write("initial-fp32.bin", threestudent.InitialWeights(), false); err != nil {
		return err
	}
	control, err := fullinputstudy.MakeOrderControl()
	if err != nil {
		return err
	}
	if err = store.json("order-control.json", control); err != nil {
		return err
	}
	for _, name := range []string{"rows.jsonl", "rejections.jsonl", "positioned-f32le.bin", "bag-f32le.bin"} {
		if err = store.write(name, nil, false); err != nil {
			return err
		}
	}
	counts, formsCount, rejected := map[string]int{}, map[string]int{}, map[string]int{}
	groups := map[string]map[string]bool{"train": {}, "calibration": {}, "development": {}}
	totals := map[string]float64{}
	index := 0
	for original, s := range states {
		forms := []string{"original"}
		if s.Split == "train" {
			forms = fullinputstudy.TrainingForms[:]
		}
		accepted := make([]fullinputstudy.Input, 0, len(forms))
		acceptedNames := make([]string, 0, len(forms))
		for _, form := range forms {
			input, e := fullinputstudy.Apply(s.Text, s.Language, form)
			if e != nil {
				return e
			}
			if input.Declined {
				rejected[s.Split+"/"+s.Language+"/"+form]++
				if e = store.line("rejections.jsonl", map[string]any{"original_row_index": original, "original_state_id": s.ID, "original_input_sha256": s.InputSHA, "form": form, "text": input.Text, "input_sha256": threecohort.SHA([]byte(input.Text)), "part_bytes": input.PartBytes, "reason": "complete input exceeds fixed representation bound"}); e != nil {
					return e
				}
				continue
			}
			accepted = append(accepted, input)
			acceptedNames = append(acceptedNames, form)
		}
		if len(accepted) == 0 || acceptedNames[0] != "original" || accepted[0].Text != s.Text {
			return errors.New("original state lost")
		}
		for i, input := range accepted {
			row := rows[original]
			row.Index, index = index, index+1
			row.ID = s.ID + "/form/" + acceptedNames[i]
			row.InputSHA = threecohort.SHA([]byte(input.Text))
			row.InitialWeight /= float64(len(accepted))
			row.FeedbackWeight /= float64(len(accepted))
			out := preparedRow{row, original, s.ID, s.InputSHA, rows[original].FeedbackWeight, acceptedNames[i], len(accepted), input.Text}
			if err = store.line("rows.jsonl", out); err != nil {
				return err
			}
			for _, representation := range []string{"positioned", "bag"} {
				var features [768]float32
				if representation == "positioned" {
					err = jointdecision.FeaturesIntoThree(input.Text, &features)
				} else {
					err = jointdecision.FeaturesIntoThreeBag(input.Text, &features)
				}
				if err != nil {
					return err
				}
				var raw [768 * 4]byte
				for k, v := range features {
					binary.LittleEndian.PutUint32(raw[k*4:], math.Float32bits(v))
				}
				if err = store.write(representation+"-f32le.bin", raw[:], true); err != nil {
					return err
				}
			}
			counts[s.Split+"/"+s.Phase]++
			formsCount[s.Split+"/"+acceptedNames[i]]++
			groups[s.Split][s.Group] = true
			totals[s.Split+"/"+s.Group] += row.FeedbackWeight
		}
	}
	for _, total := range totals {
		if math.Abs(total-1) > 1e-10 {
			return errors.New("unequal function group weight")
		}
	}
	if len(groups["train"]) != 1024 || len(groups["calibration"]) != 256 || len(groups["development"]) != 256 {
		return errors.New("group splits changed")
	}
	pins := map[string]threestudent.Pin{}
	for _, name := range []string{"preexecution.json", "initial-fp32.bin", "order-control.json", "rows.jsonl", "rejections.jsonl", "positioned-f32le.bin", "bag-f32le.bin"} {
		pins[name], err = threestudent.FilePin(filepath.Join(output, name))
		if err != nil {
			return err
		}
	}
	manifest := map[string]any{"schema": "gooo/full-input-training-bank/v1", "status": "PREPARED_PENDING_INDEPENDENT_REPLAY", "source_revision": revision, "protocol_sha256": threecohort.SHA(protocol), "source_rows": 10739, "prepared_rows": index, "files": pins, "counts_by_split_phase": counts, "forms_by_split": formsCount, "rejected_full_forms": rejected, "feature_versions": map[string]string{"positioned": jointdecision.ThreeFeatureVersion, "bag": jointdecision.ThreeBagFeatureVersion}, "feature_dim": 768, "parameters": 2072, "groups": map[string]int{"train": 1024, "calibration": 256, "development": 256}, "initial_state_sha256": threecohort.SHA(threestudent.InitialWeights()), "planned_optimizer_updates": 6400, "optimizer_updates": 0, "model_predictions": 0, "new_study_bytes_before_manifest": store.used, "new_study_cap_bytes": studyCap, "whole_own_three_cap_bytes": wholeCap}
	if err = store.json("manifest.json", manifest); err != nil {
		return err
	}
	encoded, _ := json.Marshal(map[string]any{"prepared_rows": index, "rejected_full_forms": rejected, "status": "PREPARED_PENDING_INDEPENDENT_REPLAY", "new_bytes": store.used})
	fmt.Println(string(encoded))
	return nil
}
