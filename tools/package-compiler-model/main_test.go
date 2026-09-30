package main

import "testing"

func TestPrivateText(t *testing.T) {
	for _, value := range []string{"Bearer example", "Bearer\texample", "/Users/example", "/private/var/example", "hf_abcdefghijklmnopqrstuvwxyz"} {
		if !privateText.MatchString(value) {
			t.Errorf("private text was not detected: %q", value)
		}
	}
	if privateText.MatchString("Public synthetic Gooo instruction; weights have a SHA-256 digest") {
		t.Fatal("public text rejected")
	}
}
