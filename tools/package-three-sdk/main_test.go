package main

import (
	"strings"
	"testing"
)

func TestDecodedPrivacyAndClosedPrefixManifest(t *testing.T) {
	for _, text := range []string{`{"text":"\u002fUsers\u002fprivate"}`, `{"text":"github_pat_sensitive"}`, `{"text":"synthetic","text":"duplicate"}`} {
		if err := publicJSON([]byte(text)); err == nil {
			t.Fatal("private or ambiguous JSON accepted")
		}
	}
	if err := privacy(strings.NewReader("{\"synthetic\":true}\n"), "capture.jsonl"); err != nil {
		t.Fatal(err)
	}
	if err := validate(manifest{Schema: "gooo/own-three-sdk-prefix-public-bundle/v1", Status: "PASS"}); err == nil {
		t.Fatal("full completion substituted for incomplete original prefix")
	}
}
