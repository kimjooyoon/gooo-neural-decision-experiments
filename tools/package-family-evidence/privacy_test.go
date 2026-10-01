package main

import (
	"bytes"
	"testing"
)

func TestPrivacyNecessaryPrefixEquivalence(t *testing.T) {
	for _, prefix := range []string{"", "hf_", "ghp_", "github_pat_", "sk-", "Bearer", "/Users/", "/private/var/", "bearer", "HF_", "판단"} {
		for _, gap := range []string{"", " ", "\t", "\n", "\r", "\f", "\v", "\u00a0", "_", "-"} {
			for n := 0; n <= 24; n++ {
				raw := append([]byte("ordinary gooo 한글 영어 "), []byte(prefix+gap+string(bytes.Repeat([]byte("A"), n)))...)
				if privateText(raw) != privacy.Match(raw) {
					t.Fatal("necessary-prefix optimization changed rejection")
				}
			}
		}
	}
	// A short benign prefix must not hide a later matched alternative.
	for _, raw := range [][]byte{[]byte("hf_ short /Users/synthetic"), []byte("Bearer. ghp_" + string(bytes.Repeat([]byte("A"), 20)))} {
		if privateText(raw) != privacy.Match(raw) || !privateText(raw) {
			t.Fatal("later private alternative missed")
		}
	}
}

func BenchmarkPrivacySafeEvidence(b *testing.B) {
	raw := bytes.Repeat([]byte("{\"safe\":\"gooo 판단 input\"}\n"), 40000)
	for _, variant := range []struct {
		name  string
		check func([]byte) bool
	}{{"regex", privacy.Match}, {"necessary_prefix", privateText}} {
		b.Run(variant.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			for b.Loop() {
				if variant.check(raw) {
					b.Fatal("synthetic safe text rejected")
				}
			}
		})
	}
}
