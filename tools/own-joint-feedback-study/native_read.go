package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

func sdkObservations(sdk string, policies []nativePolicy, views []jointcohort.View) (map[string]map[string]jointfeedback.Capture, error) {
	wanted := map[string]bool{}
	for _, v := range views {
		if v.Config == 40 {
			wanted[v.ID] = true
		}
	}
	all := map[string]map[string]jointfeedback.Capture{}
	for _, p := range policies {
		if all[p.Candidate] != nil {
			continue
		}
		all[p.Candidate] = map[string]jointfeedback.Capture{}
		f, err := os.Open(filepath.Join(sdk, "development-"+strings.ReplaceAll(p.Candidate, "/", "-")+".jsonl"))
		if err != nil {
			return nil, err
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 32768), 1<<20)
		for s.Scan() {
			var o observation
			if err = decision.RejectDuplicateJSONKeys(s.Bytes()); err != nil {
				f.Close()
				return nil, err
			}
			if err = json.Unmarshal(s.Bytes(), &o); err != nil {
				f.Close()
				return nil, err
			}
			if o.Policy != p.Candidate {
				f.Close()
				return nil, errors.New("SDK observation policy differs")
			}
			if wanted[o.Capture.ViewID] {
				if _, ok := all[p.Candidate][o.Capture.ViewID]; ok {
					f.Close()
					return nil, errors.New("duplicate native prior view")
				}
				all[p.Candidate][o.Capture.ViewID] = o.Capture
			}
		}
		err = s.Err()
		closeErr := f.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(all[p.Candidate]) != 48 {
			return nil, errors.New("all 48 native SDK prior views required")
		}
	}
	return all, nil
}
func verifyNativePreexecution(out string, selection []byte, pins map[string]pin, policies []nativePolicy) error {
	var p struct {
		Native      string         `json:"native_revision"`
		SDK         string         `json:"sdk"`
		Protocol    string         `json:"protocol_sha256"`
		Dataset     string         `json:"dataset_sha256"`
		Selection   string         `json:"selection_sha256"`
		Pins        map[string]pin `json:"model_pins"`
		Policies    []nativePolicy `json:"policies"`
		Calls       int            `json:"planned_native_generations"`
		Executions  int            `json:"planned_actual_go_executions"`
		Invocations int            `json:"planned_ordered_invocations"`
		CI          bool           `json:"ci_hint_supplied"`
	}
	if err := read(filepath.Join(out, "preexecution.json"), &p); err != nil {
		return err
	}
	if p.Native != nativeDeployed || p.SDK != "v0.2.12-experimental" || p.Protocol != jointfeedback.ProtocolSHA || p.Dataset != jointcohort.DatasetSHA || p.Selection != jointcohort.SHA(selection) || !reflect.DeepEqual(p.Pins, pins) || !reflect.DeepEqual(p.Policies, policies) || p.Calls != 240 || p.Executions != 240 || p.Invocations != 3840 || p.CI {
		return errors.New("frozen native preexecution tuple differs")
	}
	return nil
}
