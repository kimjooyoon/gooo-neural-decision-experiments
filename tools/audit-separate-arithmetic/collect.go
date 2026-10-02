package main

import (
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func collect(output, revision string) error {
	start := time.Now()
	if output == "" {
		return errors.New("fresh output directory required")
	}
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh output directory required")
	}
	sources, err := sourcePins(revision)
	if err != nil {
		return err
	}
	var fs syscall.Statfs_t
	if err = syscall.Statfs(".", &fs); err != nil || uint64(fs.Bavail)*uint64(fs.Bsize) < 4<<30 {
		return errors.New("4 GiB free-space preflight failed")
	}
	m, err := loadFrozen()
	if err != nil {
		return err
	}
	protocolPin, err := threestudent.FilePin(protocol)
	if err != nil {
		return err
	}
	z, err := zip.OpenReader(filepath.Join(bundle, "initial-audit.zip"))
	if err != nil {
		return err
	}
	defer z.Close()
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	r := report{Schema: "gooo/separate-arithmetic-collection/v1", Status: "COLLECTED", Source: revision, GoVersion: runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, Manifest: manifestSHA, Protocol: protocolPin, Sources: sources, Models: map[string]threestudent.Pin{}, Journals: map[string]threestudent.Pin{}, Lanes: lanes, Counts: map[string][4]totals{}, Changes: map[string]changes{}, RetainedDiff: map[string]changes{}}
	var used int64
	for _, arm := range arms {
		for _, variant := range variants {
			models, err := loadFour(m, output, arm, variant, &used, r.Models)
			if err != nil {
				return err
			}
			for _, form := range forms {
				name := arm + "--" + variant + "--" + form + ".jsonl.gz"
				rows, err := originalRows(z, name, m.Members["initial-audit.zip"][name])
				if err != nil {
					return err
				}
				counts, delta, retained, err := collectJournal(filepath.Join(output, name), rows, models, &used, start.Add(15*time.Minute))
				if err != nil {
					return err
				}
				r.Counts[name], r.Changes[name], r.RetainedDiff[name] = counts, delta, retained
				r.Rows += len(rows)
				r.Calls += len(rows) * 4
				pin, err := threestudent.FilePin(filepath.Join(output, name))
				if err != nil {
					return err
				}
				r.Journals[name] = pin
			}
		}
	}
	if r.Rows != 18432 || r.Calls != 73728 {
		return errors.New("collection denominator differs")
	}
	r.Bytes, r.WallSeconds = used, time.Since(start).Seconds()
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	if err = writeFresh(filepath.Join(output, "report.json"), append(raw, '\n'), &used); err != nil {
		return err
	}
	fmt.Printf("{\"status\":\"COLLECTED\",\"rows\":%d,\"actual_predictions\":%d,\"output_bytes\":%d,\"wall_seconds\":%.6f}\n", r.Rows, r.Calls, used, r.WallSeconds)
	return nil
}

func collectJournal(path string, rows []original, models [4]*jointdecision.ThreeModel, used *int64, deadline time.Time) (counts [4]totals, delta, retained changes, err error) {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return counts, delta, retained, err
	}
	z := gzip.NewWriter(budgetWriter{f, used})
	defer func() { err = errors.Join(err, z.Close(), f.Close()) }()
	enc := json.NewEncoder(z)
	seen := map[string]bool{}
	for _, row := range rows {
		if time.Now().After(deadline) {
			return counts, delta, retained, errors.New("collection deadline exceeded")
		}
		if seen[row.View] {
			return counts, delta, retained, errors.New("duplicate original view")
		}
		seen[row.View] = true
		pair := pairedRow{Original: row}
		for lane, model := range models {
			pair.Lanes[lane], err = observe(model, row)
			if err != nil {
				return counts, delta, retained, err
			}
			counts[lane].add(pair.Lanes[lane])
		}
		if pair.Lanes[0] != pair.Lanes[1] || pair.Lanes[2] != pair.Lanes[3] {
			return counts, delta, retained, errors.New("expanded/compact arrays or predictions differ")
		}
		if pair.Lanes[0].Features != pair.Lanes[2].Features {
			return counts, delta, retained, errors.New("arithmetic change altered input features")
		}
		delta.add(pair.Lanes[0], pair.Lanes[2])
		old := pair.Lanes[0]
		// Original journals did not retain hidden activations; do not imply a comparison.
		old.HiddenSHA = ""
		old.Prediction, old.Order = row.Prediction, row.Order
		old.Partial, old.CompleteAt = finiteOutcomes(old.Order, row.Passed)
		old.LogitsSHA, old.ProbsSHA = bitsSHA(row.Prediction.Logits[:]), bitsSHA(row.Prediction.Probabilities[:])
		retained.add(old, pair.Lanes[0])
		if err = enc.Encode(pair); err != nil {
			return counts, delta, retained, err
		}
	}
	return counts, delta, retained, nil
}
