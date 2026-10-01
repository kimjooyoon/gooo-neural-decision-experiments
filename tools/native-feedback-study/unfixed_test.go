package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestUnfixedFrozenCaptureAndDerivedCoordinateRejection(t *testing.T) {
	t.Chdir("../..")
	rows, err := compoundRows()
	if err != nil {
		t.Fatal(err)
	}
	arms, err := compoundArms()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		for _, arm := range arms {
			if !arm.Feedback {
				continue
			}
			id := r.ID + "-" + arm.Name + "-true.json"
			raw, err := read(filepath.Join("runs/unfixed-feedback-sdk-20261001/captures", id))
			var capture unfixedCapture
			if err != nil || json.Unmarshal(raw, &capture) != nil {
				t.Fatal("frozen SDK capture unavailable")
			}
			count, err := verifyUnfixed(capture, r, arm)
			if err != nil {
				t.Fatal(err)
			}
			if count == 0 {
				continue
			}
			// Every digest stays valid. Changing the declared mode must reject
			// receipts with omitted predictions and constant-coordinate proofs.
			capture.Unfixed = false
			if _, err := verifyUnfixed(capture, r, arm); err == nil {
				t.Fatal("fixed-coordinate receipt accepted as legacy prediction")
			}
			capture.Unfixed = true
			capture.Source = "package main\nfunc ComposePaths(input int64) int64 { return 0 }\n"
			if _, err := verifyUnfixed(capture, r, arm); err == nil {
				t.Fatal("different emitted body accepted")
			}
			return
		}
	}
	t.Fatal("no frozen fixed-coordinate receipt checked")
}
