package main

import (
	"strings"
	"testing"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func TestClosedActualNativeManifestPreservesOriginalFullDenominator(t *testing.T) {
	m := manifest{Schema: "gooo/own-three-native-public-bundle/v1", Status: "PASS", Producer: nativeProducer, NativeRevision: nativeMain, NativeCalls: 640, GoCalls: 640, Invocations: 10240, Predictions: nativePredictions, Values: 10240, ModelsRevision: baseHF, Protocol: threefeedback.ProtocolSHA, Archive: threestudent.Pin{SHA: strings.Repeat("a", 64), Bytes: 1}}
	for _, name := range nativeNames() {
		sha := strings.Repeat("a", 64)
		if name == "protocol.md" {
			sha = threefeedback.ProtocolSHA
		}
		if name == "amendment.md" {
			sha = tailAmendmentSHA
		}
		m.Files = append(m.Files, member{name, threestudent.Pin{SHA: sha, Bytes: 1}})
		m.Bytes++
	}
	if len(m.Files) != 1290 || validateNative(m) != nil {
		t.Fatal("complete original native closed manifest required")
	}
	for _, mutate := range []func(*manifest){
		func(m *manifest) { m.NativeCalls-- },
		func(m *manifest) { m.GoCalls-- },
		func(m *manifest) { m.Invocations-- },
		func(m *manifest) { m.Sessions = 640 },
		func(m *manifest) { m.Predictions-- },
		func(m *manifest) { m.Status = "PREFIX_VERIFIED_INCOMPLETE" },
		func(m *manifest) { m.Files = m.Files[:len(m.Files)-1] },
	} {
		changed := m
		mutate(&changed)
		if validateNative(changed) == nil {
			t.Fatal("incomplete or reclassified native execution accepted")
		}
	}
	if validate(m) == nil || validateTail(m) == nil {
		t.Fatal("native phase substituted for immutable SDK prefix/tail")
	}
}

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
