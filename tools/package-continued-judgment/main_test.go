package main

import "testing"

func TestPrivateEvidencePatterns(t *testing.T) {
	for _, value := range []string{"Bearer placeholder", "/Users/example/file", "/private/var/example", "hf_abcdefghijklmnopqrstuv"} {
		if !privateText.MatchString(value) {
			t.Fatal("private evidence pattern missed")
		}
	}
	if privateText.MatchString("bounded Korean/English Gooo judgment") {
		t.Fatal("public synthetic evidence rejected")
	}
}
