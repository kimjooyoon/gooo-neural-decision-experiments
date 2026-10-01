package main

import "testing"

func TestNativePathDiagnosisFixedInventoryAndPrivateText(t *testing.T) {
	t.Chdir("../..")
	names, err := nativePathDiagnosisNames()
	if err != nil || len(names) != 30 {
		t.Fatal("native diagnosis inventory", err)
	}
	m, files, err := buildKind("native-path-diagnosis-feature")
	if err != nil || len(m.Raw) != 30 || len(files) != 9 || m.Prefix != "research/native-path-diagnosis-feature-20261001/" {
		t.Fatal("native diagnosis package", err)
	}
	for _, raw := range files {
		if privateText(raw) {
			t.Fatal("private native diagnosis payload")
		}
	}
}
