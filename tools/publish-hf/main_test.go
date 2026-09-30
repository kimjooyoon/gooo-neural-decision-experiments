package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type testRoundTripper func(*http.Request) (*http.Response, error)

func (roundTrip testRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func publisherRepoRoot(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate publisher test source")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
}

func makePublisherFixture(t *testing.T) (bundle, policy string) {
	t.Helper()
	root := publisherRepoRoot(t)
	temporary := t.TempDir()
	bundle = filepath.Join(temporary, "bundle")
	policy = filepath.Join(temporary, "allowlist.json")
	if err := os.Mkdir(bundle, 0755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "publication/public-export-allowlist-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var allowlist allowlist
	if err := json.Unmarshal(raw, &allowlist); err != nil {
		t.Fatal(err)
	}
	for _, entry := range allowlist.Files {
		sourcePath := entry.Path
		if sourcePath == "README.md" {
			sourcePath = "HF-MODEL-CARD.md"
		}
		source := filepath.Join(root, filepath.FromSlash(sourcePath))
		destination := filepath.Join(bundle, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
			t.Fatal(err)
		}
		content, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("read export fixture source %s: %v", sourcePath, err)
		}
		if err := os.WriteFile(destination, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(policy, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return bundle, policy
}

func TestDefaultPublisherPreflightIsReadOnlyAndUsesFixedSeventeenFileSet(t *testing.T) {
	bundle, policy := makePublisherFixture(t)
	// Supply only a dummy token to the in-process test. The mocked transport
	// intercepts every HTTP request; this test cannot contact the Hub.
	t.Setenv("HF_TOKEN", "unit-test-only-token")
	t.Setenv("HF_TOKEN_PATH", "")
	t.Setenv("HF_HOME", "")
	var methods, paths []string
	previousTransport := http.DefaultTransport
	http.DefaultTransport = testRoundTripper(func(request *http.Request) (*http.Response, error) {
		if request.URL.Scheme != "https" || request.URL.Host != "huggingface.co" {
			return nil, fmt.Errorf("unexpected network destination %s", request.URL)
		}
		methods = append(methods, request.Method)
		paths = append(paths, request.URL.Path)
		status, body := http.StatusNotFound, `{"error":"not found"}`
		switch request.URL.Path {
		case "/api/whoami-v2":
			if request.Method != http.MethodGet {
				return nil, fmt.Errorf("identity check used %s", request.Method)
			}
			status, body = http.StatusOK, `{"name":"asketeddy"}`
		case "/api/models/asketeddy/gooo-ir-operator-tiny-v1/refs":
			if request.Method != http.MethodGet {
				return nil, fmt.Errorf("refs check used %s", request.Method)
			}
			status = http.StatusNotFound
		default:
			return nil, fmt.Errorf("unexpected Hub request: %s %s", request.Method, request.URL.Path)
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previousTransport })

	var stdout, stderr strings.Builder
	err := run([]string{"--bundle", bundle, "--allowlist", policy}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("default publisher preflight failed: %v; stderr=%s", err, stderr.String())
	}
	var result preflightResult
	if err := json.Unmarshal([]byte(stdout.String()), &result); err != nil {
		t.Fatalf("decode default preflight result: %v; output=%s", err, stdout.String())
	}
	if result.Mode != "dry_run" || result.Decision != "READY_TO_PUBLISH" || result.Repository != expectedNamespace+"/"+expectedRepoName {
		t.Fatalf("default invocation did not return read-only preflight: %+v", result)
	}
	if result.FileCount != 17 || len(result.Files) != 17 || len(result.Models) != 3 {
		t.Fatalf("default preflight did not preserve the fixed 17-file/3-model inventory: files=%d models=%d", result.FileCount, len(result.Models))
	}
	if result.AuthenticatedUser != expectedNamespace || result.RepositoryExists {
		t.Fatalf("mocked identity or absent-repository result differs: %+v", result)
	}
	wantMethods := []string{http.MethodGet, http.MethodGet}
	wantPaths := []string{"/api/whoami-v2", "/api/models/asketeddy/gooo-ir-operator-tiny-v1/refs"}
	if fmt.Sprint(methods) != fmt.Sprint(wantMethods) || fmt.Sprint(paths) != fmt.Sprint(wantPaths) {
		t.Fatalf("default mode attempted unexpected HTTP operations: methods=%v paths=%v", methods, paths)
	}
}

func TestChildVerifierEnvironmentDropsHubCredentials(t *testing.T) {
	input := []string{
		"PATH=/usr/bin",
		"HF_TOKEN=unit-test-only-token",
		"HF_TOKEN_PATH=/tmp/token",
		"HF_HOME=/tmp/hf",
		"HF_ENDPOINT=https://example.invalid",
		"HUGGINGFACE_HUB_CACHE=/tmp/cache",
		"LANG=C.UTF-8",
		"not-an-environment-assignment",
	}
	got := childEnvironmentWithoutHubSecrets(input)
	want := []string{"PATH=/usr/bin", "LANG=C.UTF-8"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("verifier child environment=%v, want %v", got, want)
	}
}
