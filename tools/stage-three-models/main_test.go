package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestPublicRedirectBoundary(t *testing.T) {
	c := publicClient()
	for _, address := range []string{"http://huggingface.co/file", "https://example.com/file", "https://huggingface.co.evil.example/file"} {
		u, _ := url.Parse(address)
		if c.CheckRedirect(&http.Request{URL: u}, nil) == nil {
			t.Fatal("unexpected public redirect accepted")
		}
	}
	u, _ := url.Parse("https://cas-bridge.xethub.hf.co/file")
	if err := c.CheckRedirect(&http.Request{URL: u}, nil); err != nil {
		t.Fatal(err)
	}
}
