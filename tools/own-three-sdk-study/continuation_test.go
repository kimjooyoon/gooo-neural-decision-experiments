package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func TestContinuationPlansOnlyExactMissingIdentitiesAndPins(t *testing.T) {
	enterRepo(t)
	if err := amendment(); err != nil {
		t.Fatal(err)
	}
	models, ids, err := loadModels("models/own-three-feedback-v1")
	if err != nil {
		t.Fatal(err)
	}
	cell := make([]threecohort.View, 512)
	for i := range cell {
		cell[i] = threecohort.View{ID: string(rune(i + 1)), SourceSHA: "source", Text: "complete joint input", Parts: [3]string{"part0", "part1", "part2"}}
	}
	seen := map[string]int{}
	for _, id := range ids {
		seen[id] = 512
	}
	seen["set-initial/ptq_ternary"] = 75
	for _, id := range []string{"set-initial/qat_ternary", "uniform-initial/fp32", "uniform-initial/ptq_ternary", "uniform-initial/qat_ternary"} {
		seen[id] = 0
	}
	missing, err := deriveMissing(cell, ids, seen, models)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 2485 || missing[0].View != cell[75].ID || missing[0].Policy != "set-initial/ptq_ternary" || missing[436].View != cell[511].ID || missing[437].Policy != "set-initial/qat_ternary" {
		t.Fatal("wrong missing-only boundary/order")
	}
	for _, item := range missing {
		if item.Model != models[item.Policy].Pin || item.Input != threecohort.SHA([]byte(cell[0].Text)) || item.Parts[2] != threecohort.SHA([]byte("part2")) {
			t.Fatal("full input/model pins lost")
		}
	}
	pins := map[string]pin{}
	for id, m := range models {
		if m != nil {
			pins[id] = m.Pin
		}
	}
	original := map[string]threestudent.Pin{"file": {SHA: "original", Bytes: 1}}
	audit := threestudent.Pin{SHA: "independent", Bytes: 2}
	pre := tailPreexecution{Schema: "gooo/own-three-sdk-tail-preexecution/v1", Source: originalProducer, Go: "go1.27.1", Protocol: threefeedback.ProtocolSHA, Amendment: continuationSHA, Dataset: threecohort.DatasetSHA, OriginalPhase: "original", OriginalFiles: original, OriginalAudit: audit, Models: pins, Selected: "set-feedback/fp32", RawCap: continuationCap, Available: 4 << 30, Missing: missing, Budget: 8, Step: 1, Rounds: 7}
	if err = validateTailPre(pre, original, missing, pins, audit, "original"); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(pre)
	mutations := []func(*tailPreexecution){
		func(p *tailPreexecution) { p.Missing[0].View = cell[0].ID },
		func(p *tailPreexecution) { p.Missing[1] = p.Missing[0] },
		func(p *tailPreexecution) { p.Missing[0].Parts[1] = "truncated" },
		func(p *tailPreexecution) { p.Models["set-initial/ptq_ternary"] = pin{} },
		func(p *tailPreexecution) { p.Selected = "uniform-initial/fp32" },
		func(p *tailPreexecution) { p.RawCap++ },
		func(p *tailPreexecution) { p.Protocol = "amended original protocol" },
		func(p *tailPreexecution) { p.Available-- },
		func(p *tailPreexecution) { p.Seed = "post-hoc" },
		func(p *tailPreexecution) { p.CI = true },
	}
	for i, mutate := range mutations {
		var changed tailPreexecution
		if err = threecohort.Decode(raw, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		if err = validateTailPre(changed, original, missing, pins, audit, "original"); err == nil {
			t.Fatalf("mutation %d accepted", i)
		}
	}
	seen["set-initial/ptq_ternary"] = 74
	if _, err = deriveMissing(cell, ids, seen, models); err == nil {
		t.Fatal("changed original partial boundary accepted")
	}
}

func TestTwoStreamCellsReconcileActualRowsAndRejectOperationalDuplicates(t *testing.T) {
	enterRepo(t)
	models, _, err := loadModels("models/own-three-feedback-v1")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("internal/threefeedback/testdata/native-goal7-rows.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var cell []threecohort.View
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 32768), lineCap)
	for scan.Scan() {
		v, e := threecohort.ReconstructRow(scan.Bytes())
		if e != nil {
			t.Fatal(e)
		}
		cell = append(cell, v)
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err)
	}
	if len(cell) != 16 {
		t.Fatal("full native validation fixture required")
	}
	policy := "set-initial/ptq_ternary"
	var original, tail []byte
	whole := accumulator{}
	for i, v := range cell {
		c, e := one(v, models[policy])
		if e != nil {
			t.Fatal(e)
		}
		if e = whole.add(v, c); e != nil {
			t.Fatal(e)
		}
		raw, e := json.Marshal(observation{policy, c})
		if e != nil {
			t.Fatal(e)
		}
		if i < 7 {
			original = append(original, append(raw, '\n')...)
		} else {
			tail = append(tail, append(raw, '\n')...)
		}
	}
	dir := t.TempDir()
	prefixName, tailName := filepath.Join(dir, "prefix.jsonl"), filepath.Join(dir, "tail.jsonl")
	if err = os.WriteFile(prefixName, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(tailName, tail, 0600); err != nil {
		t.Fatal(err)
	}
	combined := accumulator{}
	n, err := cellRows(prefixName, policy, cell, 0, &combined, models[policy], true)
	if err != nil || n != 7 {
		t.Fatalf("original stream: %d %v", n, err)
	}
	n, err = cellRows(tailName, policy, cell, 7, &combined, models[policy], true)
	if err != nil || n != 9 || !reflect.DeepEqual(whole, combined) {
		t.Fatalf("exact combined actual metrics: %d %v", n, err)
	}
	if _, err = cellRows(tailName, policy, cell, 0, nil, models[policy], true); err == nil {
		t.Fatal("restarted cell or identity gap accepted")
	}
	if _, err = cellRows(prefixName, policy, cell, 7, nil, models[policy], true); err == nil {
		t.Fatal("duplicated completed observations accepted")
	}
	if _, err = cellRows(tailName, "uniform-initial/ptq_ternary", cell, 7, nil, models[policy], true); err == nil {
		t.Fatal("wrong model policy accepted")
	}
	if _, err = cellRows(tailName, policy, cell, 17, nil, models[policy], true); err == nil {
		t.Fatal("invalid offset accepted")
	}
}

func TestSeparateStorageCapLeavesOriginalCapAndTerminalReserveIntact(t *testing.T) {
	defaultStore := storage{}
	if defaultStore.limit() != threestudent.RawCap {
		t.Fatal("original protocol cap changed")
	}
	separate := storage{Cap: continuationCap, Used: threestudent.RawCap + 1}
	if err := separate.beforeCall(); err != nil {
		t.Fatal(err)
	}
	separate.Used = continuationCap - receiptReserve - lineCap + 1
	if err := separate.beforeCall(); err == nil {
		t.Fatal("over-cap next call accepted")
	}
	f, err := os.OpenFile(filepath.Join(t.TempDir(), "retained.json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	separate.Used = continuationCap - receiptReserve
	if err = separate.write(f, []byte("unretained call"), receiptReserve); err == nil {
		t.Fatal("terminal reserve consumed by a call")
	}
	if err = separate.write(f, []byte("terminal"), 0); err != nil {
		t.Fatal(err)
	}
	stat, err := f.Stat()
	if err != nil || stat.Size() != 8 {
		t.Fatal("failed call wrote bytes or terminal record lost")
	}
}
