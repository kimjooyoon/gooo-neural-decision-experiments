package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
)

type runtimePolicy struct{ Name, Model string }

func sharedPolicies(root string) ([]runtimePolicy, map[string]string) {
	raw, err := os.ReadFile(filepath.Join(root, "go-audit.json"))
	must(err)
	var audit struct {
		Schema      string `json:"schema"`
		Status      string `json:"status"`
		Updates     int    `json:"optimizer_updates_reconciled"`
		Exports     int    `json:"model_exports"`
		Parity      int    `json:"actual_parity_predictions"`
		Development int    `json:"actual_development_predictions"`
		Promoted    bool   `json:"default_model_promoted"`
		Models      map[string]struct {
			Metadata string `json:"metadata_sha256"`
			Weights  string `json:"weights_sha256"`
		} `json:"models"`
	}
	must(json.Unmarshal(raw, &audit))
	if audit.Schema != "gooo/shared-three-go-audit/v1" || audit.Status != "PASS" || audit.Updates != 3200 || audit.Exports != 6 || audit.Parity != 288 || audit.Development != 3072 || audit.Promoted || len(audit.Models) != 6 {
		panic("complete fixed Go audit required before native calls")
	}
	policies := make([]runtimePolicy, 0, 6)
	pins := map[string]string{}
	for _, arm := range []string{"dense", "shared-local"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			path, err := filepath.Abs(filepath.Join(root, arm, "models", variant, "model.json"))
			must(err)
			m, err := jointdecision.LoadThree(path)
			must(err)
			pin := audit.Models[arm+"/"+variant]
			if m.MetadataSHA256() != pin.Metadata || m.WeightsSHA256() != pin.Weights {
				panic("native model differs from audited export")
			}
			name := arm + "-" + variant
			policies = append(policies, runtimePolicy{name, path})
			pins[name] = "sha256:" + pin.Metadata
		}
	}
	return policies, pins
}

func checkSharedStorage() {
	phases, err := filepath.Glob("runs/own-three-shared-*")
	must(err)
	var total int64
	for _, phase := range phases {
		must(filepath.WalkDir(phase, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			s, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if s.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("shared evidence symlink")
			}
			if d.IsDir() {
				return nil
			}
			if !s.Mode().IsRegular() {
				return fmt.Errorf("nonregular evidence")
			}
			total += s.Size()
			return nil
		}))
	}
	// Reserve a complete bounded generation/runtime pair before the next call.
	if total+(20<<20) > 64<<20 || 990581179+total+(20<<20) > 2<<30 {
		panic("shared study cap; preserve completed prefix")
	}
}
