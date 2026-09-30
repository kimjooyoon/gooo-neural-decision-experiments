package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	repository      = "kimjooyoon/gooo-neural-decision-experiments"
	releaseTag      = "v0.2.0-experimental"
	expectedCommit  = "72813219c891285a5c6af43406cb0688d3a8b4c8"
	apiRoot         = "https://api.github.com"
	maxJSONBytes    = 2 << 20
	maxAssetBytes   = 64 << 20
	reviewDirectory = "review/release-v0.2-publication-20260930"
	receiptFileName = "verification-receipt.json"
)

var expectedAssetSHA = map[string]string{
	"darwin_arm64": "251c5c49c74dc5c70cc5ca853f8d609b4484a1b56fe07cc70c576dfa2f832c28",
	"linux_amd64":  "1035398c14850bdcbfefd4ac21dedc03e2b4ccba76e10e879b77b5759a978b2f",
	"linux_arm64":  "40b6260947f62f8dc5c0910fe11a1ad5169e0801ff72c5f3cb003a16c23abcbe",
	"sha256sums":   "810329d664ea2d1ec9b6cd94362409ab1cc8fe830d54c8540b69b499f427022c",
}

type repositoryResponse struct {
	Private    bool   `json:"private"`
	Visibility string `json:"visibility"`
	HTMLURL    string `json:"html_url"`
}

type releaseResponse struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	HTMLURL    string         `json:"html_url"`
	Assets     []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name               string  `json:"name"`
	State              string  `json:"state"`
	Size               int64   `json:"size"`
	Digest             *string `json:"digest"`
	BrowserDownloadURL string  `json:"browser_download_url"`
}

type gitObject struct {
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type gitRef struct {
	Object gitObject `json:"object"`
}

type gitTag struct {
	Object gitObject `json:"object"`
}

type assetReceipt struct {
	Name              string `json:"name"`
	DownloadURL       string `json:"download_url"`
	APIState          string `json:"api_state"`
	APIBytes          int64  `json:"api_bytes"`
	APIAssetDigest    string `json:"api_asset_digest,omitempty"`
	DownloadedBytes   int64  `json:"downloaded_bytes,omitempty"`
	DownloadedSHA256  string `json:"downloaded_sha256,omitempty"`
	ExpectedSHA256    string `json:"expected_sha256"`
	DigestMatched     bool   `json:"digest_matched"`
	ExpectedAssetRole string `json:"expected_asset_role"`
}

type receipt struct {
	Schema                  string         `json:"schema"`
	CheckedAtUTC            string         `json:"checked_at_utc"`
	VerifierSourceSHA256    string         `json:"verifier_source_sha256"`
	VerificationScope       string         `json:"verification_scope"`
	RepositoryURL           string         `json:"repository_url"`
	ReleaseURL              string         `json:"release_url"`
	TagURL                  string         `json:"tag_url"`
	SourceCommitURL         string         `json:"source_commit_url"`
	ExpectedCommit          string         `json:"expected_source_commit"`
	ResolvedCommit          string         `json:"resolved_source_commit,omitempty"`
	Public                  bool           `json:"repository_public"`
	Draft                   bool           `json:"release_draft"`
	Prerelease              bool           `json:"release_prerelease"`
	AssetCount              int            `json:"asset_count"`
	Assets                  []assetReceipt `json:"assets"`
	ChecksumManifestMatched bool           `json:"checksum_manifest_matched"`
	Failures                []string       `json:"failures"`
	Status                  string         `json:"status"`
}

func main() {
	receiptValue, tempDir, err := verify()
	if tempDir != "" && receiptValue.Status != "PASS" {
		fmt.Fprintf(os.Stderr, "Verification did not pass; private raw capture retained under %s\n", tempDir)
	} else if tempDir != "" {
		_ = os.RemoveAll(tempDir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if writeErr := writeReceipt(receiptValue); writeErr != nil {
		fmt.Fprintf(os.Stderr, "Could not write path-neutral verification receipt: %v\n", writeErr)
		os.Exit(2)
	}
	encoded, _ := json.MarshalIndent(receiptValue, "", "  ")
	fmt.Println(string(encoded))
	if receiptValue.Status != "PASS" {
		os.Exit(1)
	}
}

func verify() (receipt, string, error) {
	r := receipt{
		Schema:            "gooo/public-release-verification/v1",
		CheckedAtUTC:      time.Now().UTC().Format(time.RFC3339),
		VerificationScope: "Anonymous release metadata and fresh asset downloads were checked by size and SHA-256. No uploaded binary was executed; Linux amd64 and Linux arm64 were only downloaded and integrity-checked.",
		RepositoryURL:     "https://github.com/" + repository,
		ReleaseURL:        "https://github.com/" + repository + "/releases/tag/" + releaseTag,
		TagURL:            "https://github.com/" + repository + "/tree/" + releaseTag,
		SourceCommitURL:   "https://github.com/" + repository + "/commit/" + expectedCommit,
		ExpectedCommit:    expectedCommit,
		Assets:            []assetReceipt{},
		Failures:          []string{},
		Status:            "INCOMPLETE",
	}
	verifierBytes, verifierErr := os.ReadFile(filepath.Join(reviewDirectory, "verify_release.go"))
	if verifierErr != nil {
		r.Failures = append(r.Failures, "verifier source could not be hashed")
	} else {
		verifierHash := sha256.Sum256(verifierBytes)
		r.VerifierSourceSHA256 = hex.EncodeToString(verifierHash[:])
	}
	tempDir, err := os.MkdirTemp("", "gooo-release-check-")
	if err != nil {
		return r, "", errors.New("could not create a fresh temporary verification directory")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var repo repositoryResponse
	if err := getJSON(ctx, client, apiRoot+"/repos/"+repository, filepath.Join(tempDir, "repository.json"), &repo); err != nil {
		r.Failures = append(r.Failures, safeRequestFailure("repository metadata", err))
		return finish(r, tempDir, errors.New("public repository metadata could not be verified"))
	}
	r.Public = !repo.Private && repo.Visibility == "public"
	if !r.Public {
		r.Failures = append(r.Failures, "repository is not reported public")
	}
	if canonical, ok := canonicalRepoURL(repo.HTMLURL); !ok || canonical != r.RepositoryURL {
		r.Failures = append(r.Failures, "repository metadata URL did not match the canonical public repository URL")
	}

	var rel releaseResponse
	releaseAPIURL := apiRoot + "/repos/" + repository + "/releases/tags/" + releaseTag
	if err := getJSON(ctx, client, releaseAPIURL, filepath.Join(tempDir, "release.json"), &rel); err != nil {
		r.Failures = append(r.Failures, safeRequestFailure("release metadata", err))
		return finish(r, tempDir, errors.New("release metadata could not be verified"))
	}
	r.Draft = rel.Draft
	r.Prerelease = rel.Prerelease
	r.AssetCount = len(rel.Assets)
	if rel.TagName != releaseTag {
		r.Failures = append(r.Failures, "release tag name did not match the requested tag")
	}
	if rel.Draft {
		r.Failures = append(r.Failures, "release is still a draft")
	}
	if !rel.Prerelease {
		r.Failures = append(r.Failures, "release is not marked prerelease")
	}
	if rel.HTMLURL != r.ReleaseURL {
		r.Failures = append(r.Failures, "release URL did not match the canonical public release URL")
	}

	resolved, tagErr := resolveTag(ctx, client, releaseTag, tempDir)
	if tagErr != nil {
		r.Failures = append(r.Failures, "tag could not be resolved to a source commit")
	} else {
		r.ResolvedCommit = resolved
		if resolved != expectedCommit {
			r.Failures = append(r.Failures, "tag resolved to a different source commit")
		}
	}
	if len(rel.Assets) != 4 {
		r.Failures = append(r.Failures, fmt.Sprintf("release has %d uploaded assets; expected exactly 4", len(rel.Assets)))
		return finish(r, tempDir, nil)
	}

	sort.Slice(rel.Assets, func(i, j int) bool { return rel.Assets[i].Name < rel.Assets[j].Name })
	usedExpected := make(map[string]bool, len(expectedAssetSHA))
	assetPaths := make(map[string]string, len(rel.Assets))
	assetNamesByRole := make(map[string]string, len(rel.Assets))
	for index, asset := range rel.Assets {
		entry := assetReceipt{
			Name:              asset.Name,
			DownloadURL:       asset.BrowserDownloadURL,
			APIState:          asset.State,
			APIBytes:          asset.Size,
			ExpectedAssetRole: "unmatched",
		}
		if asset.Digest != nil {
			entry.APIAssetDigest = *asset.Digest
		}
		if asset.State != "uploaded" {
			r.Failures = append(r.Failures, "an asset is not in uploaded state")
		}
		if asset.Size < 0 || asset.Size > maxAssetBytes {
			r.Failures = append(r.Failures, "an asset size is outside the bounded download limit")
			r.Assets = append(r.Assets, entry)
			continue
		}
		if !canonicalDownloadURL(asset.BrowserDownloadURL, asset.Name) {
			r.Failures = append(r.Failures, "an asset download URL is not the canonical GitHub release URL")
			r.Assets = append(r.Assets, entry)
			continue
		}
		path := filepath.Join(tempDir, fmt.Sprintf("asset-%02d.bin", index+1))
		sha, size, downloadErr := downloadAsset(ctx, client, asset.BrowserDownloadURL, path)
		if downloadErr != nil {
			r.Failures = append(r.Failures, "an uploaded asset could not be downloaded anonymously")
			r.Assets = append(r.Assets, entry)
			continue
		}
		entry.DownloadedBytes = size
		entry.DownloadedSHA256 = sha
		if size != asset.Size {
			r.Failures = append(r.Failures, "downloaded asset size did not match the GitHub API size")
		}
		role, expected, ok := expectedRoleForHash(sha)
		if !ok {
			r.Failures = append(r.Failures, "downloaded asset SHA-256 did not match the expected release inventory")
		} else {
			entry.ExpectedAssetRole = role
			entry.ExpectedSHA256 = expected
			entry.DigestMatched = true
			if usedExpected[role] {
				r.Failures = append(r.Failures, "two uploaded assets matched the same expected release role")
			}
			usedExpected[role] = true
			assetPaths[role] = path
			assetNamesByRole[role] = asset.Name
		}
		if asset.Digest != nil && !strings.EqualFold(strings.TrimPrefix(*asset.Digest, "sha256:"), sha) {
			entry.DigestMatched = false
			r.Failures = append(r.Failures, "downloaded asset SHA-256 did not match the GitHub API digest")
		}
		r.Assets = append(r.Assets, entry)
	}
	for role := range expectedAssetSHA {
		if !usedExpected[role] {
			r.Failures = append(r.Failures, "an expected release asset role is missing")
		}
	}
	if len(assetPaths) == 4 {
		if err := verifyChecksumManifest(assetPaths, assetNamesByRole); err != nil {
			r.Failures = append(r.Failures, "SHA256SUMS did not bind the three downloaded binaries")
		} else {
			r.ChecksumManifestMatched = true
		}
	}

	return finish(r, tempDir, nil)
}

func finish(r receipt, tempDir string, err error) (receipt, string, error) {
	if len(r.Failures) == 0 {
		r.Status = "PASS"
	} else {
		r.Status = "FAIL"
	}
	return r, tempDir, err
}

func getJSON(ctx context.Context, client *http.Client, requestURL, privatePath string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return errors.New("request construction failed")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "gooo-public-release-verifier/1")
	resp, err := client.Do(req)
	if err != nil {
		return errors.New("anonymous HTTP request failed")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxJSONBytes+1))
	if err != nil || len(body) > maxJSONBytes {
		return errors.New("GitHub API response exceeded the bounded JSON read")
	}
	_ = os.WriteFile(privatePath, body, 0o600)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return errors.New("GitHub API returned invalid JSON")
	}
	return nil
}

func resolveTag(ctx context.Context, client *http.Client, tag, tempDir string) (string, error) {
	refURL := apiRoot + "/repos/" + repository + "/git/ref/tags/" + url.PathEscape(tag)
	var ref gitRef
	if err := getJSON(ctx, client, refURL, filepath.Join(tempDir, "tag-ref.json"), &ref); err != nil {
		return "", err
	}
	object := ref.Object
	for depth := 0; depth < 4; depth++ {
		switch object.Type {
		case "commit":
			return object.SHA, nil
		case "tag":
			var tagObject gitTag
			tagURL := apiRoot + "/repos/" + repository + "/git/tags/" + url.PathEscape(object.SHA)
			if err := getJSON(ctx, client, tagURL, filepath.Join(tempDir, fmt.Sprintf("tag-object-%d.json", depth)), &tagObject); err != nil {
				return "", err
			}
			object = tagObject.Object
		default:
			return "", errors.New("tag points to a non-commit object")
		}
	}
	return "", errors.New("tag annotation nesting exceeded bound")
}

func downloadAsset(ctx context.Context, client *http.Client, downloadURL, path string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", 0, errors.New("asset request construction failed")
	}
	req.Header.Set("User-Agent", "gooo-public-release-verifier/1")
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, errors.New("anonymous asset request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("asset endpoint returned HTTP %d", resp.StatusCode)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", 0, errors.New("private asset capture could not be created")
	}
	hasher := sha256.New()
	count, copyErr := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(resp.Body, maxAssetBytes+1))
	closeErr := file.Close()
	if copyErr != nil || closeErr != nil || count > maxAssetBytes {
		return "", count, errors.New("asset download failed or exceeded the bounded size")
	}
	return hex.EncodeToString(hasher.Sum(nil)), count, nil
}

func verifyChecksumManifest(paths, names map[string]string) error {
	path, ok := paths["sha256sums"]
	if !ok {
		return errors.New("checksum manifest is missing")
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 64<<10 {
		return errors.New("checksum manifest read failed or exceeded its size bound")
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		return errors.New("checksum manifest must contain three binary entries")
	}
	seenNames := map[string]bool{}
	seenDigests := map[string]bool{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			return errors.New("invalid checksum line")
		}
		digest, name := strings.ToLower(fields[0]), strings.TrimPrefix(fields[1], "*")
		if len(digest) != 64 || seenNames[name] || seenDigests[digest] {
			return errors.New("duplicate or malformed checksum line")
		}
		seenNames[name] = true
		seenDigests[digest] = true
		found := false
		for role, expectedName := range names {
			if role == "sha256sums" || expectedName != name {
				continue
			}
			want, known := expectedAssetSHA[role]
			if !known || want != digest {
				return errors.New("checksum line digest did not match the expected asset")
			}
			actual, hashErr := fileSHA256(paths[role])
			if hashErr != nil || actual != digest {
				return errors.New("checksum line digest did not match the downloaded asset")
			}
			found = true
		}
		if !found {
			return errors.New("checksum line names an unknown downloaded asset")
		}
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func expectedRoleForHash(digest string) (string, string, bool) {
	for role, expected := range expectedAssetSHA {
		if digest == expected {
			return role, expected, true
		}
	}
	return "", "", false
}

func canonicalRepoURL(value string) (string, bool) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.Path != "/"+repository || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false
	}
	return "https://github.com" + parsed.Path, true
}

func canonicalDownloadURL(value, assetName string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	wantPath := "/" + repository + "/releases/download/" + releaseTag + "/" + assetName
	return parsed.Path == wantPath
}

func safeRequestFailure(label string, err error) string {
	var statusErr interface{ StatusCode() int }
	if errors.As(err, &statusErr) {
		return fmt.Sprintf("%s request returned HTTP %d", label, statusErr.StatusCode())
	}
	if strings.Contains(err.Error(), "HTTP ") {
		return fmt.Sprintf("%s request returned an HTTP error", label)
	}
	return fmt.Sprintf("%s request failed", label)
}

func writeReceipt(r receipt) error {
	if err := os.MkdirAll(reviewDirectory, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(reviewDirectory, receiptFileName), data, 0o644)
}
