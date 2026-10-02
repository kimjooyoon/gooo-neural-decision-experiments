package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointfeedback"
)

func loadStates(directory string) (map[string]jointfeedback.State, error) {
	var audit struct {
		Status string            `json:"status"`
		Files  map[string]string `json:"files_sha256"`
		Rows   int               `json:"student_state_rows"`
	}
	if err := read(filepath.Join(directory, "independent-audit.json"), &audit); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(filepath.Join(directory, "states.jsonl"))
	if err != nil || audit.Status != "PASS" || audit.Rows != 6463 || jointcohort.SHA(raw) != "2fb6ecc9e2085a311456d2fa6c158b3cd8ee0b6c481ed7395df8acf8c28f4aa2" || audit.Files["states.jsonl"] != jointcohort.SHA(raw) {
		return nil, errors.New("audited frozen student states required")
	}
	f, err := os.Open(filepath.Join(directory, "states.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 32768), 1<<20)
	states := map[string]jointfeedback.State{}
	for scanner.Scan() {
		var s jointfeedback.State
		if err = json.Unmarshal(scanner.Bytes(), &s); err != nil {
			return nil, err
		}
		if _, ok := states[s.ID]; ok {
			return nil, errors.New("duplicate student state")
		}
		states[s.ID] = s
	}
	if err = scanner.Err(); err != nil {
		return nil, err
	}
	if len(states) != 6463 {
		return nil, errors.New("student state count differs")
	}
	return states, nil
}

func verifyPre(raw []byte) error {
	if err := decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var pre struct {
		Protocol string `json:"protocol_sha256"`
		Dataset  string `json:"source_dataset_sha256"`
		States   string `json:"student_states_sha256"`
		Initial  string `json:"initial_state_sha256"`
		Seed     int    `json:"initial_seed"`
		Shuffle  int    `json:"shuffle_seed"`
		Steps    int    `json:"planned_optimizer_steps"`
	}
	if err := json.Unmarshal(raw, &pre); err != nil {
		return err
	}
	if pre.Protocol != jointfeedback.ProtocolSHA || pre.Dataset != jointcohort.DatasetSHA || pre.States != "2fb6ecc9e2085a311456d2fa6c158b3cd8ee0b6c481ed7395df8acf8c28f4aa2" || pre.Initial != "c3cea331f1413eda049404bd0caecdea5806e1c0990edcf2608d169ad770929c" || pre.Seed != 20261027 || pre.Shuffle != 20261028 || pre.Steps != 3600 {
		return errors.New("matched fresh own initialization and frozen dataset/protocol differ")
	}
	return nil
}
