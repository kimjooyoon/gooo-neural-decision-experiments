package main

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func view(t *testing.T, split, language string) threecohort.View {
	t.Helper()
	var fields [64]byte
	fields[0], fields[5] = 128, 128
	prefix, err := wrapper(split, language)
	if err != nil {
		t.Fatal(err)
	}
	part, err := decision.EncodeSemanticContextInput(fields, prefix+"Keep every byte. 모든 글자를 보존하라.")
	if err != nil {
		t.Fatal(err)
	}
	v := threecohort.View{Split: split, Language: language, Parts: [3]string{part, part, part}}
	v.Text, err = jointdecision.EncodeThree(v.Parts)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestFormsPreserveCompleteBodyAndSource(t *testing.T) {
	for _, language := range []string{"en", "ko"} {
		for _, split := range []string{"train", "calibration", "development"} {
			v := view(t, split, language)
			var original [768]float32
			if err := jointdecision.FeaturesIntoThree(v.Text, &original); err != nil {
				t.Fatal(err)
			}
			for _, form := range forms {
				text, err := transformed(v, form)
				if err != nil {
					t.Fatal(err)
				}
				if form == "original" && text != v.Text {
					t.Fatal("original changed")
				}
				parts, err := jointdecision.ThreeParts(text)
				if err != nil {
					t.Fatal(err)
				}
				for i, part := range parts {
					header, _, _ := strings.Cut(v.Parts[i], ";intent: ")
					if !strings.HasPrefix(part, header+";intent: ") || !strings.Contains(part, "Keep every byte. 모든 글자를 보존하라.") {
						t.Fatal("source/body changed")
					}
				}
				if _, _, err = featurePin(text, &original); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestRejectWrongWrapperAndSourceChange(t *testing.T) {
	v := view(t, "train", "en")
	v.Split = "development"
	if _, err := transformed(v, "bare"); err == nil {
		t.Fatal("absent expected wrapper accepted")
	}
	v.Split = "train"
	if _, err := transformed(v, "unknown"); err == nil {
		t.Fatal("unknown form")
	}
	var original [768]float32
	if err := jointdecision.FeaturesIntoThree(v.Text, &original); err != nil {
		t.Fatal(err)
	}
	original[0] = 0
	if _, _, err := featurePin(v.Text, &original); err == nil {
		t.Fatal("changed source channel")
	}
}

func TestPairMetricsKeepValidAlternativesAndWrongAgreement(t *testing.T) {
	var p pairCounts
	p.add(1, 2, 6)
	p.add(1, 1, 6)
	p.add(0, 0, 6)
	p.add(0, 1, 6)
	if p.Pairs != 4 || p.Disagree != 2 || p.BothValid != 2 || p.DifferentValid != 1 || p.AgreeValid != 1 || p.SameWrong != 1 {
		t.Fatalf("%+v", p)
	}
}

func TestRawJournalFreshPrivateAndBounded(t *testing.T) {
	dir := t.TempDir()
	var used int64
	j, err := openJournal(dir, "rows.jsonl", &used)
	if err != nil {
		t.Fatal(err)
	}
	defer j.file.Close()
	if err = j.append(map[string]string{"text": "공개 evidence"}); err != nil {
		t.Fatal(err)
	}
	before := used
	if err = j.append("/Users/example/private"); err == nil || used != before {
		t.Fatal("private data appended")
	}
	used = capBytes
	if err = j.append("overflow"); err == nil {
		t.Fatal("cap exceeded")
	}
	if _, err = openJournal(dir, "rows.jsonl", &used); err == nil {
		t.Fatal("overwrote prior prefix")
	}
	if _, err = pin(filepath.Join(dir, "rows.jsonl")); err != nil {
		t.Fatal(err)
	}
}
