package main

import "testing"

func TestClosedPhaseInventoryAndDecodedPrivacy(t *testing.T) {
	if len(expectedNames()) != 642 {
		t.Fatal("raw files missing")
	}
	for _, raw := range []string{`{"safe":"public","safe":"duplicate"}`, `{"path":"\u002fUsers\u002fprivate"}`, `{"token":"hf_abcdefghijklmnopqrstuvwxyz012345"}`} {
		if publicJSON([]byte(raw)) == nil {
			t.Fatal("private/duplicate JSON accepted")
		}
	}
	if err := publicJSON([]byte(`{"state":"gooo;joint3|","source":"sha256:public"}`)); err != nil {
		t.Fatal(err)
	}
	if err := validate(manifest{}); err == nil {
		t.Fatal("unbound manifest accepted")
	}
}
