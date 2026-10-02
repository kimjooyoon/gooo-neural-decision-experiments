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

func TestClosedTailManifestRejectsReclassifiedFailureAndChangedResourcePlan(t *testing.T) {
	m := manifest{Schema: "gooo/own-three-sdk-tail-public-bundle/v1", Status: "PASS_WITH_SEPARATE_STORAGE_AMENDMENT", Producer: tailProducer, Sessions: 2485, Predictions: 9142, Values: 148736, ModelsRevision: baseHF, Protocol: "5d840f7b3379339d684b85b7896a895538bd1a030395a096af3e52c5fc781cd3"}
	for _, name := range tailNames() {
		sha := strings.Repeat("a", 64)
		if name == "amendment.md" {
			sha = tailAmendmentSHA
		}
		if name == "protocol.md" {
			sha = m.Protocol
		}
		m.Files = append(m.Files, member{Name: name})
		m.Files[len(m.Files)-1].Pin.SHA = sha
	}
	if err := validateTail(m); err != nil {
		t.Fatal(err)
	}
	m.Status = "PASS"
	if err := validateTail(m); err == nil {
		t.Fatal("separate amendment hidden")
	}
	m.Status = "PASS_WITH_SEPARATE_STORAGE_AMENDMENT"
	m.Files[0].Name = "../original.json"
	if err := validateTail(m); err == nil {
		t.Fatal("unknown/traversal member accepted")
	}
}

func TestFullMissingIdentityMetadataHasSeparateBoundAndFullPrivacyScan(t *testing.T) {
	large := `{"identities":"` + strings.Repeat("a", 2<<20) + `"}`
	if err := privacy(strings.NewReader(large), tailPhase+"/preexecution.json"); err != nil {
		t.Fatal(err)
	}
	if err := privacy(strings.NewReader(large), "other.json"); err == nil {
		t.Fatal("ordinary metadata bound weakened")
	}
	if err := privacy(strings.NewReader(large+"\n"), "capture.jsonl"); err == nil {
		t.Fatal("session capture bound weakened")
	}
	private := `{"identities":"` + strings.Repeat("a", 2<<20) + `\u002fUsers\u002fprivate"}`
	if err := privacy(strings.NewReader(private), tailPhase+"/preexecution.json"); err == nil {
		t.Fatal("end of full metadata privacy scan omitted")
	}
}
