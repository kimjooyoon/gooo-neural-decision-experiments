package main

import "testing"

func TestPrivacyDecodesEscapedStringsAndNativePayload(t *testing.T) {
	for _, raw := range []string{
		`{"path":"\u002fUsers\u002fprivate"}`,
		`{"a":1,"a":2}`,
		`{"native_receipt":"eyJwYXRoIjoiL1VzZXJzL3ByaXZhdGUifQ=="}`,
	} {
		if err := jsonPrivacy([]byte(raw)); err == nil {
			t.Fatal("private/duplicate/encoded evidence accepted")
		}
	}
	if err := jsonPrivacy([]byte(`{"native_receipt":"eyJtb2RlbF9wcmVkaWN0aW9ucyI6MH0="}`)); err != nil {
		t.Fatal(err)
	}
}
func TestClosedOrderedNamesExcludeTraversal(t *testing.T) {
	for _, name := range names() {
		if name == "" || name[0] == '/' {
			t.Fatal("absolute member")
		}
	}
	m := manifest{}
	if err := validate(m); err == nil {
		t.Fatal("unbound arbitrary archive accepted")
	}
}
