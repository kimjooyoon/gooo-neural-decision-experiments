package main

import "testing"

func TestBudgetAppendixKeepsTheLargerRawLimitSpecific(t *testing.T) {
	t.Chdir("../..")
	m, files, err := buildKind("native-budget-main")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Raw) != 342 || len(files) != 8 || m.Prefix != "research/native-budget-main-20261001/" {
		t.Fatal("budget allowlist differs")
	}
	total := 0
	for _, e := range m.Raw {
		total += e.Bytes
	}
	if total <= maxTotal || total > archiveBudget("native-budget-main") {
		t.Fatal("raw byte accounting differs")
	}
	if err := checkArchive(files["evidence.tar.gz"], m.Raw); err == nil {
		t.Fatal("legacy raw cap widened")
	}
	if err := checkArchiveBudget(files["evidence.tar.gz"], m.Raw, archiveBudget("native-budget-main")); err != nil {
		t.Fatal(err)
	}
	if archiveBudget("retained-native-feature") != maxTotal || archiveBudget("retained-completeness") != maxTotal {
		t.Fatal("old archive budget changed")
	}
}
