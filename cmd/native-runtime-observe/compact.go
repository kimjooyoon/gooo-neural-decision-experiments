package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const compactProtocolSHA = "2dae15f2fdf33991124815685e239f8b77e36d24200685b5caf906236de4fb81"
const compactAuditSHA = "1aff17949d9b066949edc98c57e195e1a322207488e3e245579e8da15c7d3125"
const compactPairReserve = 4 << 20

var compactVariants = [3]string{"fp32", "ptq_ternary", "qat_ternary"}

type compactBinding struct {
	Metadata string `json:"metadata_sha256"`
	Weights  string `json:"weights_sha256"`
	Schema   string `json:"schema"`
}
type compactStudy struct {
	Policies     []runtimePolicy
	MetadataPins map[string]string
	Bindings     map[string]compactBinding
	Source       string
	Output       string
	Pairs        []map[string]any
}

func newCompactStudy(root, output string) *compactStudy {
	if filepath.Dir(filepath.Clean(output)) != "runs" || !strings.HasPrefix(filepath.Base(output), "compact-shared-") {
		panic("compact raw output must use the preregistered runs/compact-shared- prefix")
	}
	for name, want := range map[string]string{
		"preexecution/shared-three-compact-runtime-preregistration-20261003.json": compactProtocolSHA,
		filepath.Join(root, "report.json"):                                        compactAuditSHA,
	} {
		pin, e := threestudent.FilePin(name)
		must(e)
		if pin.SHA != want {
			panic("immutable compact protocol/audit differs")
		}
	}
	head, e := exec.Command("git", "rev-parse", "HEAD").Output()
	must(e)
	dirty, e := exec.Command("git", "status", "--porcelain").Output()
	must(e)
	if len(dirty) != 0 {
		panic("clean collector source required")
	}
	c := &compactStudy{MetadataPins: map[string]string{}, Bindings: map[string]compactBinding{}, Source: strings.TrimSpace(string(head)), Output: output}
	var audit struct {
		Models []struct {
			Variant  string         `json:"variant"`
			Expanded compactBinding `json:"expanded"`
			Compact  compactBinding `json:"compact"`
		} `json:"models"`
	}
	b, e := os.ReadFile(filepath.Join(root, "report.json"))
	must(e)
	must(json.Unmarshal(b, &audit))
	if len(audit.Models) != 3 {
		panic("closed compact variants required")
	}
	for i, v := range compactVariants {
		if audit.Models[i].Variant != v {
			panic("variant order differs")
		}
		for _, format := range []string{"expanded", "compact"} {
			name := format + "-" + v
			path := filepath.Join(root, "models", v, "model.json")
			binding := audit.Models[i].Compact
			binding.Schema = jointdecision.SharedThreeSchema
			loader := jointdecision.LoadSharedThree
			if format == "expanded" {
				path = filepath.Join("runs/own-three-shared-judgment-mps-v2-20261003/shared-local/models", v, "model.json")
				binding = audit.Models[i].Expanded
				binding.Schema = jointdecision.ThreeSchema
				loader = jointdecision.LoadThree
			}
			path, e = filepath.Abs(path)
			must(e)
			model, e := loader(path)
			must(e)
			if model.MetadataSHA256() != binding.Metadata || model.WeightsSHA256() != binding.Weights || model.Schema() != binding.Schema {
				panic("native compact model pins differ")
			}
			c.Policies = append(c.Policies, runtimePolicy{name, path})
			c.MetadataPins[name] = "sha256:" + binding.Metadata
			c.Bindings[name] = binding
		}
	}
	c.CheckStorage(compactPairReserve)
	return c
}

func (c *compactStudy) Preexecution() map[string]any {
	return map[string]any{"collector_source_sha": c.Source, "protocol_sha256": compactProtocolSHA, "go_audit_sha256": compactAuditSHA,
		"model_bindings": c.Bindings, "raw_cap_bytes": 32 << 20, "pair_reserve_bytes": compactPairReserve, "per_process_capture_cap_bytes": 1 << 20,
		"process_timeout_seconds": 75, "seed": "", "policy_order": "Variants FP32/PTQ/QAT; expanded/compact within each variant, reversed after each view. Every generation immediately executes before the next generation.",
		"comparison_scope": "Generated Go source, complete unseeded search, semantic feedback observations and all native cases. Actual artifact, receipt-chain and timing identities remain separately recorded and are not asserted equal."}
}

func (c *compactStudy) CheckStorage(reserve int64) {
	paths, e := filepath.Glob("runs/compact-shared-*")
	must(e)
	var total int64
	for _, root := range paths {
		must(filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			s, e := os.Lstat(p)
			if e != nil {
				return e
			}
			if s.IsDir() {
				return nil
			}
			if !s.Mode().IsRegular() {
				return fmt.Errorf("compact evidence must be regular")
			}
			total += s.Size()
			return nil
		}))
	}
	if total+reserve > 32<<20 {
		panic("compact 32 MiB study cap; preserve completed prefix")
	}
	var disk syscall.Statfs_t
	must(syscall.Statfs("runs", &disk))
	if disk.Bavail*uint64(disk.Bsize) < 4<<30 {
		panic("compact study requires four GiB free storage")
	}
}

type pairedGeneration struct {
	Source string `json:"source"`
	Report struct {
		Paths struct {
			Search   pathplan.SearchResult      `json:"search"`
			Feedback []pathplan.FeedbackReceipt `json:"feedback_judgments"`
		} `json:"body_paths"`
	} `json:"report"`
}

func normalizedThree(p *pathplan.ThreeReceipt, schema string) *pathplan.ThreeReceipt {
	if p == nil {
		return nil
	}
	if p.Schema != schema {
		panic("wrong runtime model schema")
	}
	copy := *p
	copy.Schema = ""
	copy.PredictNS = 0
	return &copy
}

func normalizeGeneration(raw []byte, binding compactBinding) []byte {
	var g pairedGeneration
	must(json.Unmarshal(raw, &g))
	s := &g.Report.Paths.Search.Selection
	if s.MetadataSHA256 != binding.Metadata || s.WeightsSHA256 != binding.Weights || s.SeedSHA256 != "" || s.Three == nil || !s.ExternalCallsKnown || s.ExternalCalls != 0 || s.ModelCalls < 1 {
		panic("native unseeded selection must retain exact model identity")
	}
	s.MetadataSHA256, s.WeightsSHA256 = "", ""
	s.Three = normalizedThree(s.Three, binding.Schema)
	for i := range s.Receipts {
		s.Receipts[i].PredictNS = 0
	}
	for i := range g.Report.Paths.Feedback {
		f := &g.Report.Paths.Feedback[i]
		if f.MetadataSHA != binding.Metadata || f.WeightsSHA != binding.Weights {
			panic("feedback repinning differs")
		}
		f.MetadataSHA, f.WeightsSHA = "", ""
		f.SHA, f.PreviousSHA, f.FromProgressSHA = "", "", ""
		f.Three = normalizedThree(f.Three, binding.Schema)
	}
	b, e := json.Marshal(g)
	must(e)
	return b
}

func (c *compactStudy) CompareView(root, family, language, id string) {
	for _, v := range compactVariants {
		var generations, runtimes [2][]byte
		var cases [2][]byte
		for i, format := range []string{"expanded", "compact"} {
			name := format + "-" + v
			dir := filepath.Join(root, family+"-"+language+"-"+name)
			b, e := os.ReadFile(filepath.Join(dir, "generation.json"))
			must(e)
			generations[i] = normalizeGeneration(b, c.Bindings[name])
			b, e = os.ReadFile(filepath.Join(dir, "runtime.json"))
			must(e)
			runtimes[i] = b
			var r runtimeResult
			must(json.Unmarshal(b, &r))
			cases[i], e = json.Marshal(r.Observation.Cases)
			must(e)
		}
		if !bytes.Equal(generations[0], generations[1]) || !bytes.Equal(cases[0], cases[1]) {
			panic("expanded/compact native semantic pair differs; all prefix evidence retained")
		}
		c.Pairs = append(c.Pairs, map[string]any{"view_id": id, "variant": v, "semantic_generation_sha256": sha(generations[0]), "ordered_native_cases_sha256": sha(cases[0]), "expanded_runtime_sha256": sha(runtimes[0]), "compact_runtime_sha256": sha(runtimes[1])})
	}
	c.CheckStorage(64 << 10)
	save(filepath.Join(root, "paired-prefix.json"), map[string]any{"schema": "gooo/compact-shared-native-pair-prefix/v1", "completed_pairs": len(c.Pairs), "pairs": c.Pairs})
}

func (c *compactStudy) Finish(output string) {
	if len(c.Pairs) != 48 {
		panic("complete 48 native representation pairs required")
	}
	save(filepath.Join(output, "paired-report.json"), map[string]any{"schema": "gooo/compact-shared-native-pairs/v1", "status": "PASS", "collector_source_sha": c.Source, "protocol_sha256": compactProtocolSHA, "go_audit_sha256": compactAuditSHA, "pairs": c.Pairs, "paired_views": 48, "native_generations": 96, "new_optimizer_updates": 0, "scope": "Unseeded source, complete finite search, semantic feedback and native case equivalence. Model identities, receipt-chain hashes and timing values are validated separately and intentionally not equated. No new training holdout or accuracy improvement."})
	c.CheckStorage(0)
}
