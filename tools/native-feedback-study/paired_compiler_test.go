package main

import (
	"math"
	"testing"
)

func TestPairedSelectionUsesCandidateCostBeforeLanguageAgreement(t *testing.T) {
	scores := map[string]pairedScore{}
	for _, arm := range pairedArms {
		for _, variant := range pairedVariants {
			scores[arm+"-"+variant] = pairedScore{Total: pairedCounts{Views: 160, Calls: 160, CompleteTDD: 160, After: 1920, Cases: 1920, Extras: 80, Disagree: 1, FiniteNLL: 0.6}}
		}
	}
	better := scores["js03-fp32"]
	better.Total.Extras = 70
	better.Total.Disagree = 80
	scores["js03-fp32"] = better
	id, err := choosePaired(scores)
	if err != nil || id != "js03-fp32" {
		t.Fatal("agreement cannot override assembly cost", id, err)
	}
	better.Total.Extras = 80
	better.Total.Disagree = 1
	better.Total.FiniteNLL = 0.6
	scores["js03-fp32"] = better
	id, err = choosePaired(scores)
	if err != nil || id != "js0-fp32" {
		t.Fatal("deterministic lexicographic tie differs", id, err)
	}
	better.Total.FiniteNLL = math.NaN()
	scores["js03-fp32"] = better
	if _, err = choosePaired(scores); err == nil {
		t.Fatal("nonfinite candidate accepted")
	}
}

func TestPairedModelsAndCanonicalArithmeticRejectForgedActual(t *testing.T) {
	t.Chdir("../..")
	loaded, err := loadPairedModels("runs/paired-compiler-context-mps-20261002")
	if err != nil || len(loaded) != 9 {
		t.Fatal("own-model pins missing", err)
	}
	rows, err := compilerTrainingRows("runs/compiler-context-curriculum-validated-20261001")
	if err != nil {
		t.Fatal(err)
	}
	var selected []compilerTrainingRow
	for _, row := range rows {
		if row.Split == "calibration" {
			selected = append(selected, row)
			if len(selected) == 2 {
				break
			}
		}
	}
	observations, err := pairedEvaluate(selected, "calibration", loaded["js03-fp32"])
	if err != nil {
		t.Fatal(err)
	}
	score, err := summarizePaired(selected, "calibration", observations)
	if err != nil || score.Total.Views != 2 || score.Total.CompleteTDD != 2 {
		t.Fatal("bounded paired arithmetic differs", err)
	}
	observations[0].Search.Attempts[0].Results[0].Actual++
	if _, err = summarizePaired(selected, "calibration", observations); err == nil {
		t.Fatal("forged arithmetic accepted")
	}
}
