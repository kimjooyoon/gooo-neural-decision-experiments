// own-three-sdk-study retains all eleven policies on the frozen bilingual
// calibration/development corpus. Go owns inference, code assembly and audits.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type preexecution struct {
	Schema     string                      `json:"schema"`
	Source     string                      `json:"source_revision"`
	Go         string                      `json:"go"`
	Protocol   string                      `json:"protocol_sha256"`
	Dataset    string                      `json:"source_dataset_sha256"`
	Models     map[string]pin              `json:"model_pins"`
	Policies   []string                    `json:"policy_order"`
	PriorFiles map[string]threestudent.Pin `json:"prior_raw_files"`
	PriorBytes int64                       `json:"prior_retained_raw_bytes"`
	RawCap     int64                       `json:"whole_study_raw_cap_bytes"`
	Sessions   int                         `json:"planned_sdk_sessions"`
	Views      int                         `json:"function_views_per_cell"`
	Budget     int                         `json:"candidate_budget"`
	Step       int                         `json:"candidates_per_advance"`
	Rounds     int                         `json:"maximum_feedback_rounds"`
	Seed       string                      `json:"seed"`
	CI         bool                        `json:"ci_hint_supplied"`
	Selector   string                      `json:"calibration_selector"`
}
type selection struct {
	Schema          string            `json:"schema"`
	Selected        string            `json:"selected_candidate"`
	Calibration     map[string]counts `json:"calibration"`
	Models          map[string]pin    `json:"model_pins"`
	DevelopmentSeen bool              `json:"development_seen"`
}
type report struct {
	Schema      string            `json:"schema"`
	Status      string            `json:"status"`
	Selected    string            `json:"selected_candidate"`
	Sessions    int               `json:"actual_sdk_sessions"`
	Predictions int               `json:"actual_model_predictions"`
	Calibration map[string]counts `json:"calibration"`
	Development map[string]counts `json:"development"`
	ModelSetup  map[string]int64  `json:"model_load_and_pin_wall_ns"`
	WallNS      int64             `json:"whole_collector_wall_ns"`
	CPUNS       int64             `json:"whole_collector_cpu_ns"`
	CPUPercent  float64           `json:"cpu_percent_of_one_core"`
	RSS         int64             `json:"process_lifetime_peak_rss_bytes"`
	NewUpdates  int               `json:"new_optimizer_updates"`
	Native      int               `json:"native_calls"`
	Promoted    bool              `json:"default_model_promoted"`
	Scope       string            `json:"scope"`
}

func main() {
	dataset := flag.String("dataset", "", "frozen actual native source dataset")
	models := flag.String("models", "models/own-three-feedback-v1", "all nine frozen own student exports")
	output := flag.String("output", "", "fresh ignored own-three SDK phase directory")
	revision := flag.String("source-revision", "", "exact clean public study source")
	audit := flag.String("audit-report", "", "zero-prediction independent replay receipt")
	prefix := flag.String("prefix", "", "immutable cap-stopped original phase for separate storage continuation")
	combined := flag.String("combined-audit-report", "", "zero-prediction combined original/tail audit receipt")
	bilingual := flag.String("bilingual-report", "", "post-hoc first-output equivalence diagnostic; no model calls or selector changes")
	native := flag.Bool("native", false, "actual pinned Gooo generation and compiled Go execution")
	nativeAudit := flag.String("native-audit-report", "", "zero-call source/native/SDK/compiled-output replay receipt")
	nativeBinary := flag.String("native-binary", "", "clean adopted Gooo executable with sibling body worker")
	goBinary := flag.String("go-binary", "", "exact Go 1.27.1 execution compiler")
	sdkTail := flag.String("sdk-tail", "", "complete immutable missing-only SDK tail")
	flag.Parse()
	var err error
	if *native || *nativeAudit != "" {
		if *prefix == "" || *sdkTail == "" || *audit != "" || *combined != "" || *bilingual != "" || (*native && *nativeAudit != "") {
			err = errors.New("native collection and independent audit require separate original/tail modes")
		} else {
			err = nativeStudy(*dataset, *models, *prefix, *sdkTail, *output, *revision, *nativeBinary, *goBinary, *nativeAudit)
		}
	} else if *bilingual != "" && *prefix != "" && *audit == "" && *combined == "" {
		err = bilingualAudit(*dataset, *models, *prefix, *output, *bilingual)
	} else if *bilingual != "" {
		err = errors.New("bilingual diagnostic requires separate prefix/tail audit mode")
	} else if *combined != "" && *prefix != "" && *audit == "" {
		err = auditContinuation(*dataset, *models, *prefix, *output, *combined)
	} else if *prefix != "" && *audit == "" && *combined == "" {
		err = continueStudy(*dataset, *models, *prefix, *output, *revision)
	} else if *combined != "" || (*prefix != "" && *audit != "") {
		err = errors.New("continuation collection and combined/original audit modes must be separate")
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
	if err != nil || len(revision) != 40 || strings.TrimSpace(string(head)) != revision || runtime.Version() != "go1.27.1" {
		return errors.New("exact clean Go 1.27.1 study source required")
	}
	dirty, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil || len(dirty) != 0 {
		return errors.New("clean published study source required")
	}
	raw, err := os.ReadFile(threefeedback.Protocol)
	if err != nil || threecohort.SHA(raw) != threefeedback.ProtocolSHA {
		return errors.New("immutable three-choice protocol differs")
	}
	return nil
}
func filename(split, id string) string {
	return split + "-" + strings.ReplaceAll(id, "/", "-") + ".jsonl"
}
func study(dataset, root, output, revision string) (failure error) {
	if err := source(revision); err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	absolute, err := filepath.Abs(output)
	if err != nil || filepath.Dir(absolute) != filepath.Join(cwd, "runs") || !strings.HasPrefix(filepath.Base(absolute), "own-three-") {
		return errors.New("fresh own-three phase directly under runs required")
	}
	if _, err = os.Lstat(absolute); !os.IsNotExist(err) {
		return errors.New("never overwrite or restart retained SDK phase")
	}
	prior, used, err := inventory()
	if err != nil {
		return err
	}
	store := &storage{Used: used, Prior: used}
	if err = store.beforeCall(); err != nil {
		return err
	}
	views, err := threecohort.Load(dataset)
	if err != nil {
		return err
	}
	start, cpuStart := time.Now(), cpuNS()
	models, ids, err := loadModels(root)
	if err != nil {
		return err
	}
	pins, setups := map[string]pin{}, map[string]int64{}
	for id, m := range models {
		if m != nil {
			pins[id], setups[id] = m.Pin, m.SetupNS
		}
	}
	if err = os.Mkdir(absolute, 0755); err != nil {
		return err
	}
	sessions, predictions := 0, 0
	defer func() {
		collectorWall, collectorCPU := time.Since(start).Nanoseconds(), cpuNS()-cpuStart
		status, message := "SDK_CAPTURED_PENDING_EXTERNAL_REPLAY", ""
		if failure != nil {
			status, message = "FAILED_PREFIX_RETAINED", failure.Error()
		}
		files := map[string]threestudent.Pin{}
		entries, e := os.ReadDir(absolute)
		if e != nil {
			if failure == nil {
				failure = e
			}
			return
		}
		for _, d := range entries {
			p, e := threestudent.FilePin(filepath.Join(absolute, d.Name()))
			if e != nil {
				if failure == nil {
					failure = e
				}
				return
			}
			files[d.Name()] = p
		}
		e = store.save(filepath.Join(absolute, "collection-attempt.json"), map[string]any{"schema": "gooo/own-three-sdk-collection-attempt/v1", "status": status, "error": message, "actual_sdk_sessions": sessions, "actual_model_predictions": predictions, "prior_raw_bytes": store.Prior, "retained_new_bytes_before_attempt": store.Used - store.Prior, "retained_files_before_attempt": files, "new_optimizer_updates": 0, "native_calls": 0, "whole_collector_wall_ns": collectorWall, "whole_collector_cpu_ns": collectorCPU, "cpu_percent_of_one_core": 100 * float64(collectorCPU) / float64(collectorWall), "process_lifetime_peak_rss_bytes": peakRSS()})
		if e != nil && failure == nil {
			failure = e
		}
	}()
	pre := preexecution{Schema: "gooo/own-three-sdk-preexecution/v1", Source: revision, Go: runtime.Version(), Protocol: threefeedback.ProtocolSHA, Dataset: threecohort.DatasetSHA, Models: pins, Policies: ids, PriorFiles: prior, PriorBytes: used, RawCap: threestudent.RawCap, Sessions: 11264, Views: 512, Budget: 8, Step: 1, Rounds: 7, Selector: "calibration extra candidates, actual predictions, bilingual disagreement, packed bytes, candidate name; offline excluded; no development selection"}
	if err = store.save(filepath.Join(absolute, "preexecution.json"), pre); err != nil {
		return err
	}
	collect := func(split string) (map[string]counts, error) {
		cells := map[string]counts{}
		for _, id := range ids {
			f, e := os.OpenFile(filepath.Join(absolute, filename(split, id)), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if e != nil {
				return nil, e
			}
			a := accumulator{}
			for _, v := range views {
				if v.Split != split {
					continue
				}
				if e = store.beforeCall(); e != nil {
					f.Sync()
					f.Close()
					return nil, e
				}
				capture, runtimeErr := one(v, models[id])
				raw, e := json.Marshal(observation{id, capture})
				if e != nil {
					f.Close()
					return nil, e
				}
				if len(raw)+1 > lineCap {
					f.Close()
					return nil, errors.New("full capture exceeds immutable one-MiB line cap")
				}
				if e = store.write(f, append(raw, '\n'), receiptReserve); e != nil {
					f.Close()
					return nil, e
				}
				sessions++
				predictions += capture.Search.Selection.ModelCalls
				if runtimeErr == nil {
					runtimeErr = verify(v, capture, models[id])
				}
				if runtimeErr == nil {
					runtimeErr = a.add(v, capture)
				}
				if runtimeErr != nil {
					f.Sync()
					f.Close()
					return nil, fmt.Errorf("%s %s %s: %w", split, id, v.ID, runtimeErr)
				}
			}
			if e = f.Sync(); e != nil {
				f.Close()
				return nil, e
			}
			if e = f.Close(); e != nil {
				return nil, e
			}
			cell, e := a.finish()
			if e != nil {
				return nil, e
			}
			cells[id] = cell
			fmt.Printf("%s %s views=%d extra=%d calls=%d curve=%v raw=%d\n", split, id, cell.Views, cell.Extra, cell.Predictions, cell.Curve, store.Used)
		}
		return cells, nil
	}
	cal, err := collect("calibration")
	if err != nil {
		return err
	}
	selected := choose(cal, models)
	if err = store.save(filepath.Join(absolute, "selection.json"), selection{Schema: "gooo/own-three-sdk-calibration-selection/v1", Selected: selected, Calibration: cal, Models: pins}); err != nil {
		return err
	}
	dev, err := collect("development")
	if err != nil {
		return err
	}
	if sessions != 11264 {
		return errors.New("full planned actual session denominator differs")
	}
	wall, cpu := time.Since(start).Nanoseconds(), cpuNS()-cpuStart
	r := report{Schema: "gooo/own-three-sdk-report/v1", Status: "PASS", Selected: selected, Sessions: sessions, Predictions: predictions, Calibration: cal, Development: dev, ModelSetup: setups, WallNS: wall, CPUNS: cpu, CPUPercent: 100 * float64(cpu) / float64(wall), RSS: peakRSS(), Scope: "Actual full eleven-policy SDK comparison and captured eight-mask typed code assembly. Each ordered value/source/progress/failure/frontier independently audited without prediction replay. Candidate budgets 1/2/4/6/8 are indices 0/1/3/5/7. No new training, native compiled execution, default promotion, GPU-utilization or causal host-utilization claim. SDK session wall includes source assembly/tests/context/inference; collector wall also includes audit and storage. Process lifetime RSS is separate from model tensors."}
	return store.save(filepath.Join(absolute, "report.json"), r)
}
