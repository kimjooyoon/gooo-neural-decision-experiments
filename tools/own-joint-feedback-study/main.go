// own-joint-feedback-study measures bounded student assembly in actual Go SDK sessions.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type observation struct {
	Policy  string                `json:"policy"`
	Capture jointfeedback.Capture `json:"capture"`
}

func main() {
	dataset := flag.String("dataset", "", "frozen native source curriculum")
	models := flag.String("models", "", "nine committed own student models")
	output := flag.String("output", "", "fresh retained SDK study directory")
	revision := flag.String("source-revision", "", "clean exact Go study source")
	audit := flag.String("audit-report", "", "fresh zero-prediction audit of retained SDK study")
	mode := flag.String("mode", "sdk", "sdk or native")
	sdk := flag.String("sdk-study", "", "frozen all-policy SDK study for native comparison")
	native := flag.String("native", "", "clean adopted-main Gooo executable")
	goBinary := flag.String("go-bin", "", "exact Go 1.27.1 execution compiler")
	flag.Parse()
	var err error
	if *mode == "native" {
		err = nativeStudy(*dataset, *models, *sdk, *output, *revision, *native, *goBinary, *audit)
	} else if *mode != "sdk" {
		err = errors.New("unknown study mode")
	} else if *audit != "" {
		err = auditStudy(*dataset, *models, *output, *audit)
	} else {
		err = study(*dataset, *models, *output, *revision)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func source(revision string) error {
	head, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision {
		return errors.New("exact committed study source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean study source required")
	}
	raw, err := os.ReadFile(jointfeedback.Protocol)
	if err != nil || jointcohort.SHA(raw) != jointfeedback.ProtocolSHA {
		return errors.New("frozen study protocol differs")
	}
	return nil
}

func study(dataset, root, output, revision string) error {
	if err := source(revision); err != nil {
		return err
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh SDK study directory required")
	}
	views, err := jointcohort.Load(dataset)
	if err != nil {
		return err
	}
	models, ids, err := loadModels(root)
	if err != nil {
		return err
	}
	pins := map[string]pin{}
	for id, m := range models {
		if m != nil {
			pins[id] = m.Pin
		}
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	if err = save(filepath.Join(output, "preexecution.json"), map[string]any{"schema": "gooo/own-joint-feedback-sdk-preexecution/v2",
		"source_revision": revision, "protocol_sha256": jointfeedback.ProtocolSHA, "source_dataset_sha256": jointcohort.DatasetSHA,
		"model_pins": pins, "planned_sdk_sessions": 9216, "function_views_per_cell": 384, "candidate_budget": 4,
		"ci_hint_supplied": false, "default_model_promoted": false}); err != nil {
		return err
	}
	calibration, err := split(output, "calibration", views, models, ids)
	if err != nil {
		return err
	}
	selected := choose(calibration, models)
	if err = save(filepath.Join(output, "selection.json"), map[string]any{"schema": "gooo/own-joint-feedback-calibration-selector/v2",
		"selected_candidate": selected, "calibration": calibration, "model_pins": pins,
		"ordering": "extra candidates, actual predictions, bilingual disagreement, packed bytes, candidate name; offline is a control", "development_seen": false}); err != nil {
		return err
	}
	development, err := split(output, "development", views, models, ids)
	if err != nil {
		return err
	}
	sessions, predictions := 0, 0
	for _, cells := range []map[string]counts{calibration, development} {
		for _, c := range cells {
			sessions += c.Views
			predictions += c.Predictions
		}
	}
	if sessions != 9216 {
		return errors.New("planned actual SDK session count differs")
	}
	return save(filepath.Join(output, "report.json"), map[string]any{"schema": "gooo/own-joint-feedback-sdk-report/v2", "status": "PASS",
		"selected_candidate": selected, "actual_sdk_sessions": sessions, "actual_model_predictions": predictions,
		"calibration": calibration, "development": development, "new_optimizer_updates": 0, "native_calls": 0,
		"scope": "Actual retained SDK sessions and independent ordered finite arithmetic. All 12 policies and both splits retained. Four-mask completeness includes deterministic enumeration and does not imply universal semantic correctness. Native compiled execution is separate."})
}

func choose(cal map[string]counts, models map[string]*model) string {
	ids := []string{}
	for id, m := range models {
		if m != nil {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, func(a, b string) int {
		ca, cb := cal[a], cal[b]
		for _, pair := range [][2]int{{ca.Extra, cb.Extra}, {ca.Predictions, cb.Predictions}, {ca.Disagreement, cb.Disagreement}, {models[a].Pin.Packed, models[b].Pin.Packed}} {
			if pair[0] < pair[1] {
				return -1
			}
			if pair[0] > pair[1] {
				return 1
			}
		}
		return strings.Compare(a, b)
	})
	return ids[0]
}

func split(output, name string, views []jointcohort.View, models map[string]*model, ids []string) (map[string]counts, error) {
	all := map[string]counts{}
	for _, id := range ids {
		file, err := os.OpenFile(filepath.Join(output, name+"-"+strings.ReplaceAll(id, "/", "-")+".jsonl"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return nil, err
		}
		writer := bufio.NewWriterSize(file, 32768)
		c, pairs := counts{}, map[string]map[string]uint16{}
		for _, v := range views {
			if v.Split != name {
				continue
			}
			o, err := one(v, models[id])
			if err != nil {
				writer.Flush()
				file.Close()
				return nil, err
			}
			if err = c.add(v, o); err != nil {
				file.Close()
				return nil, err
			}
			if pairs[v.Group] == nil {
				pairs[v.Group] = map[string]uint16{}
			}
			pairs[v.Group][v.Language] = o.Search.Attempts[0].Mask
			raw, err := json.Marshal(observation{id, o})
			if err != nil {
				file.Close()
				return nil, err
			}
			if _, err = writer.Write(append(raw, '\n')); err != nil {
				file.Close()
				return nil, err
			}
		}
		if err = writer.Flush(); err != nil {
			file.Close()
			return nil, err
		}
		if err = file.Close(); err != nil {
			return nil, err
		}
		for _, langs := range pairs {
			if len(langs) != 2 {
				return nil, errors.New("two complete languages required")
			}
			c.Pairs++
			if langs["en"] != langs["ko"] {
				c.Disagreement++
			}
		}
		if c.Views != 384 || c.Curve[3] != 384 {
			return nil, errors.New("finite full-budget completeness or denominators differ")
		}
		all[id] = c
		fmt.Printf("%s %s views=%d extra=%d calls=%d curve=%v\n", name, id, c.Views, c.Extra, c.Predictions, c.Curve)
	}
	return all, nil
}

func one(v jointcohort.View, m *model) (jointfeedback.Capture, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := time.Now()
	var search pathplan.SearchResult
	var progress []pathplan.SessionProgress
	var feedback []pathplan.FeedbackReceipt
	var err error
	if m == nil {
		search, _, progress, err = v.Prepared.SearchBatches(ctx, nil, v.Cases, 4, 1, "")
	} else if m.Joint != nil {
		search, _, progress, feedback, err = v.Prepared.SearchJointFeedbackBatches(ctx, m.Joint, v.Cases, 4, 1, "", 3, nil)
	} else {
		search, _, progress, feedback, err = v.Prepared.SearchFeedbackBatchesUnfixed(ctx, m.Independent, v.Cases, 4, 1, "", 3, nil)
	}
	o := jointfeedback.Capture{Schema: "gooo/own-joint-feedback-student-capture/v2", ViewID: v.ID, SourceSHA: v.SourceSHA,
		JointSHA: jointcohort.SHA([]byte(v.JointInput)), Rotation: -1, WallNS: time.Since(started).Nanoseconds(), Search: search, Progress: progress, Feedback: feedback}
	if err != nil {
		return o, err
	}
	if m != nil && m.Joint != nil {
		err = jointfeedback.VerifyJointObservation(v, o, m.Pin.Metadata, m.Pin.Weights)
	} else {
		err = jointfeedback.VerifyFiniteAttempts(v, search)
	}
	return o, err
}
