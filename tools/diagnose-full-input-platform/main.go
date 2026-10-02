// Compare retained observations without invoking a model or changing tolerances.
package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type row struct {
	View       string                        `json:"view_id"`
	Group      string                        `json:"program_contract_group"`
	Language   string                        `json:"language"`
	Input      string                        `json:"input_sha256"`
	Source     string                        `json:"source_sha256"`
	Prediction jointdecision.ThreePrediction `json:"prediction"`
	Passed     [8]int                        `json:"passed_cases_by_mask"`
	Order      [8]int                        `json:"static_ranked_mask_order"`
	Text       string                        `json:"complete_input"`
}

type difference struct {
	File                string    `json:"journal"`
	View                string    `json:"view_id"`
	Input               string    `json:"input_sha256"`
	Masks               [2]uint16 `json:"selected_masks"`
	Orders              [2][8]int `json:"ranked_mask_orders"`
	MaxLogitDelta       float64   `json:"maximum_absolute_logit_delta"`
	MaxProbabilityDelta float64   `json:"maximum_absolute_probability_delta"`
}

func main() {
	local := flag.String("local", "", "retained arm64 audit")
	remote := flag.String("remote", "", "retained Linux audit")
	output := flag.String("output", "", "fresh diagnostic report")
	flag.Parse()
	if err := diagnose(*local, *remote, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func readRows(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	s := bufio.NewScanner(z)
	s.Buffer(make([]byte, 32768), 1<<20)
	var rows []row
	for s.Scan() {
		var r row
		if err = threecohort.Decode(s.Bytes(), &r); err != nil {
			return nil, err
		}
		rows = append(rows, r)
		if len(rows) > 512 {
			return nil, errors.New("extra observation")
		}
	}
	if s.Err() != nil || len(rows) != 512 {
		return nil, errors.New("complete observation journal required")
	}
	return rows, nil
}

func diagnose(local, remote, output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh diagnosis required")
	}
	pins := map[string][2]threestudent.Pin{}
	rows, logits, probabilities, masks, orders, softmaxOnly := 0, 0, 0, 0, 0, 0
	maxLogit, maxProbability := 0., 0.
	var changed []difference
	for _, arm := range []string{"positioned-original", "positioned-varied", "bag-original", "bag-varied"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			for _, form := range []string{"original", "task-prefix", "complete-suffix"} {
				name := arm + "--" + variant + "--" + form + ".jsonl.gz"
				a, err := readRows(filepath.Join(local, name))
				if err != nil {
					return err
				}
				b, err := readRows(filepath.Join(remote, name))
				if err != nil {
					return err
				}
				pa, err := threestudent.FilePin(filepath.Join(local, name))
				if err != nil {
					return err
				}
				pb, err := threestudent.FilePin(filepath.Join(remote, name))
				if err != nil {
					return err
				}
				pins[name] = [2]threestudent.Pin{pa, pb}
				for i, x := range a {
					y := b[i]
					if x.View != y.View || x.Group != y.Group || x.Language != y.Language || x.Input != y.Input || x.Source != y.Source || x.Text != y.Text || x.Passed != y.Passed {
						return errors.New("source/input/target mismatch")
					}
					rows++
					ld, pd := 0., 0.
					for j := range x.Prediction.Logits {
						ld = math.Max(ld, math.Abs(float64(x.Prediction.Logits[j])-float64(y.Prediction.Logits[j])))
						pd = math.Max(pd, math.Abs(float64(x.Prediction.Probabilities[j])-float64(y.Prediction.Probabilities[j])))
					}
					if ld > 0 {
						logits++
					}
					if pd > 0 {
						probabilities++
					}
					if ld == 0 && pd > 0 {
						softmaxOnly++
					}
					maxLogit = math.Max(maxLogit, ld)
					maxProbability = math.Max(maxProbability, pd)
					if x.Prediction.Mask != y.Prediction.Mask {
						masks++
					}
					if x.Order != y.Order {
						orders++
						changed = append(changed, difference{name, x.View, x.Input, [2]uint16{x.Prediction.Mask, y.Prediction.Mask}, [2][8]int{x.Order, y.Order}, ld, pd})
					}
				}
			}
		}
	}
	source, err := threestudent.FilePin("tools/diagnose-full-input-platform/main.go")
	if err != nil {
		return err
	}
	value := map[string]any{"schema": "gooo/full-input-platform-diagnosis/v1", "status": "OBSERVED", "paired_rows": rows, "different_logits_rows": logits, "different_probability_rows": probabilities, "equal_logits_different_probability_rows": softmaxOnly, "different_selected_mask_rows": masks, "different_rank_order_rows": orders, "maximum_absolute_logit_delta": maxLogit, "maximum_absolute_probability_delta": maxProbability, "changed_rank_rows": changed, "journals": pins, "diagnostic_source_sha256": source.SHA, "new_model_predictions": 0, "new_optimizer_updates": 0, "scope": "Complete paired retained initial development journals; exact source/text/target binding. Rows retain local then Linux ordering. Diagnostic comparison does not change model computation or pass the failed cross-platform check."}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(raw) > 4<<20 {
		return errors.New("bounded diagnosis required")
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(raw, '\n'))
	closed := f.Close()
	if err != nil || closed != nil {
		return errors.Join(err, closed)
	}
	fmt.Printf("{\"rows\":%d,\"logit_changes\":%d,\"probability_changes\":%d,\"mask_changes\":%d,\"rank_changes\":%d,\"max_logit_delta\":%g,\"max_probability_delta\":%g}\n", rows, logits, probabilities, masks, orders, maxLogit, maxProbability)
	return nil
}
