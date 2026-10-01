package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func publicGet(client *http.Client, url string) ([]byte, error) {
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anonymous GET status %d", response.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (32<<20)+1))
	if err != nil || len(raw) > 32<<20 {
		return nil, errors.New("bounded anonymous response required")
	}
	return raw, nil
}
func digest(raw []byte) string { h := sha256.Sum256(raw); return hex.EncodeToString(h[:]) }
func publicPaths() map[string]bool {
	result := map[string]bool{}
	for _, name := range []string{"README.md", "LICENSE", "protocol.md", "results.md", "training-report.json", "training-preexecution.json", "study-report.json", "study-preexecution.json", "calibration-selection.json", "independent-audit.json", "native-report.json", "native-preexecution.json", "curriculum-manifest.json", "curriculum-audit.json", "raw-evidence.zip"} {
		result[name] = true
	}
	for _, arm := range []string{"v2", "v3"} {
		result[arm+"/go-parity.json"] = true
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			result[arm+"/models/"+variant+"/model.json"] = true
			result[arm+"/models/"+variant+"/weights.bin"] = true
			result["provenance/"+arm+"-"+variant+".ttl"] = true
		}
	}
	return result
}
func verifyArchive(raw []byte, members []artifact) (int64, error) {
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return 0, err
	}
	allow := map[string]bool{}
	for _, m := range rawMembers("", "") {
		allow[m.Public] = true
	}
	if len(members) != 327 || len(z.File) != 327 || len(allow) != 327 {
		return 0, errors.New("fixed archive count differs")
	}
	expected := map[string]artifact{}
	for _, m := range members {
		if !allow[m.Path] || expected[m.Path].Path != "" || m.Bytes < 0 || m.Bytes > 32<<20 {
			return 0, errors.New("fixed member allowlist differs")
		}
		expected[m.Path] = m
	}
	var total int64
	seen := map[string]bool{}
	for _, f := range z.File {
		m, ok := expected[f.Name]
		if !ok || seen[f.Name] || !f.Mode().IsRegular() || strings.Contains(f.Name, "..") || int64(f.UncompressedSize64) != m.Bytes {
			return 0, errors.New("archive entry type/path/size differs")
		}
		seen[f.Name] = true
		r, e := f.Open()
		if e != nil {
			return 0, e
		}
		h := sha256.New()
		var buffer [32768]byte
		n, e := io.CopyBuffer(h, io.LimitReader(r, (32<<20)+1), buffer[:])
		closeErr := r.Close()
		if e != nil || closeErr != nil || n != m.Bytes || hex.EncodeToString(h.Sum(nil)) != m.SHA {
			return 0, errors.New("archive CRC/SHA/byte count differs")
		}
		total += n
		if total > 256<<20 {
			return 0, errors.New("total archive expansion bound exceeded")
		}
	}
	return total, nil
}
func verify(bundle, revision, output string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable public revision required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh verification report required")
	}
	manifestRaw, err := os.ReadFile(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	var manifest struct {
		Repository string     `json:"repository"`
		Files      []artifact `json:"files"`
		Members    []artifact `json:"archive_members"`
		Models     int        `json:"model_exports"`
		Updates    int        `json:"optimizer_updates"`
		Calls      int        `json:"actual_native_calls"`
		Scanned    bool       `json:"credentials_and_host_paths_scanned"`
	}
	if err = json.Unmarshal(manifestRaw, &manifest); err != nil {
		return err
	}
	allow := publicPaths()
	if manifest.Repository != repository || manifest.Models != 6 || manifest.Updates != 960 || manifest.Calls != 144 || !manifest.Scanned || len(manifest.Files) != len(allow) {
		return errors.New("fixed own-model publication differs")
	}
	client := &http.Client{Timeout: 30 * time.Second} // No credentials, token lookup or cookie jar.
	raw, err := publicGet(client, "https://huggingface.co/api/models/"+repository+"/revision/"+revision)
	if err != nil {
		return err
	}
	var api struct {
		Private bool   `json:"private"`
		SHA     string `json:"sha"`
	}
	if err = json.Unmarshal(raw, &api); err != nil || api.Private || api.SHA != revision {
		return errors.New("public immutable model revision differs")
	}
	gets, total, zipBytes := 0, int64(0), int64(0)
	seen := map[string]bool{}
	for _, f := range manifest.Files {
		if !allow[f.Path] || seen[f.Path] || f.Bytes < 0 || f.Bytes > 32<<20 {
			return errors.New("payload allowlist differs")
		}
		seen[f.Path] = true
		local, e := hashFile(filepath.Join(bundle, f.Path))
		if e != nil || local.SHA != f.SHA || local.Bytes != f.Bytes {
			return errors.New("local publication bytes differ")
		}
		raw, e = publicGet(client, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+f.Path+"?download=true")
		if e != nil {
			return e
		}
		gets++
		if digest(raw) != f.SHA || int64(len(raw)) != f.Bytes {
			return errors.New("anonymous immutable payload digest differs")
		}
		if f.Path == "raw-evidence.zip" {
			total, e = verifyArchive(raw, manifest.Members)
			if e != nil {
				return e
			}
			zipBytes = f.Bytes
		}
	}
	raw, err = publicGet(client, "https://huggingface.co/"+repository+"/resolve/"+revision+"/publication-manifest.json?download=true")
	if err != nil {
		return err
	}
	gets++
	if !bytes.Equal(raw, manifestRaw) {
		return errors.New("anonymous manifest bytes differ")
	}
	return save(output, map[string]any{"schema": "gooo/own-semantic-composition-public-verification/v1", "status": "PASS", "repository": repository, "revision": revision,
		"private": false, "credentials_sent": false, "anonymous_api_gets": 1, "anonymous_payload_gets": gets, "payload_files_verified": len(manifest.Files),
		"manifest_files_verified": 1, "manifest_sha256": digest(manifestRaw), "archive_members_verified": len(manifest.Members), "archive_bytes": zipBytes,
		"archive_uncompressed_bytes": total, "model_exports": 6, "recorded_optimizer_updates": 960, "recorded_native_calls": 144,
		"new_model_predictions": 0, "new_optimizer_updates": 0, "scope": "Exact anonymous immutable public bytes, six own-model exports, fixed archive regular paths/CRC/SHA and expansion bounds. Quality results are separately measured, not inferred from publication."})
}
