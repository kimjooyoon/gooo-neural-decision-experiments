package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type countingReader struct {
	R io.Reader
	N int64
}

func (r *countingReader) Read(b []byte) (int, error) {
	n, e := r.R.Read(b)
	r.N += int64(n)
	return n, e
}

func requiredFiles() map[string]bool {
	r := map[string]bool{}
	r["provenance/experiment.ttl"] = true
	for _, n := range []string{"README.md", "LICENSE", "protocol.md", "results.md", "training-report.json", "training-preexecution.json", "model-audit.json", "teacher-report.json", "teacher-audit.json", "sdk-report.json", "sdk-preexecution.json", "calibration-selection.json", "sdk-audit.json", "native-report.json", "native-preexecution.json", "native-audit.json", "raw-evidence.zip"} {
		r[n] = true
	}
	for _, a := range arms {
		r[a+"/go-parity.json"] = true
		for _, v := range variants {
			for _, n := range []string{"model.json", "weights.bin"} {
				r[a+"/models/"+v+"/"+n] = true
			}
			r["provenance/"+a+"-"+v+".ttl"] = true
		}
	}
	return r
}
func verify(dir, out string) error {
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		return errors.New("fresh verification output required")
	}
	var m manifest
	if e := read(filepath.Join(dir, "publication-manifest.json"), &m); e != nil {
		return e
	}
	if m.Schema != "gooo/own-joint-feedback-publication/v2" || m.Repository != repository || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(m.Source) || !m.PrivateScanned || m.Steps != 3600 || m.Models != 9 || m.Generations != 240 || m.Executions != 240 || m.Invocations != 3840 || m.Predictions != 473 {
		return errors.New("publication provenance tuple differs")
	}
	required := requiredFiles()
	if len(m.Files) != len(required) {
		return errors.New("fixed public payload count differs")
	}
	seen := map[string]bool{}
	for _, a := range m.Files {
		if !required[a.Path] || seen[a.Path] || a.Bytes < 0 || a.Bytes > 64<<20 {
			return errors.New("unexpected bounded public payload")
		}
		seen[a.Path] = true
		actual, e := hashFile(filepath.Join(dir, a.Path))
		if e != nil || actual.Bytes != a.Bytes || actual.SHA != a.SHA {
			return errors.New("public payload bytes differ")
		}
		if a.Path != "raw-evidence.zip" && !strings.HasSuffix(a.Path, "weights.bin") {
			if e = scan(filepath.Join(dir, a.Path)); e != nil {
				return e
			}
		}
	}
	allow, err := rawMembers()
	if err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, a := range allow {
		if allowed[a.Public] {
			return errors.New("duplicate fixed archive path")
		}
		allowed[a.Public] = true
	}
	if len(m.Members) != len(allowed) {
		return errors.New("fixed raw archive count differs")
	}
	expected := map[string]entry{}
	for _, a := range m.Members {
		if !allowed[a.Path] || expected[a.Path].Path != "" || a.Bytes < 0 || a.Bytes > 256<<20 {
			return errors.New("unexpected raw member")
		}
		expected[a.Path] = a
	}
	z, err := zip.OpenReader(filepath.Join(dir, "raw-evidence.zip"))
	if err != nil {
		return err
	}
	defer z.Close()
	if len(z.File) != len(expected) {
		return errors.New("archive entry count differs")
	}
	total := int64(0)
	seen = map[string]bool{}
	for _, f := range z.File {
		a, ok := expected[f.Name]
		if !ok || seen[f.Name] || !f.Mode().IsRegular() || strings.Contains(f.Name, "..") || uint64(a.Bytes) != f.UncompressedSize64 {
			return errors.New("archive entry type/path/size differs")
		}
		seen[f.Name] = true
		r, e := f.Open()
		if e != nil {
			return e
		}
		h := sha256.New()
		counter := &countingReader{R: io.TeeReader(io.LimitReader(r, a.Bytes+1), h)}
		if strings.HasSuffix(f.Name, "weights.bin") {
			var b [32768]byte
			_, e = io.CopyBuffer(io.Discard, counter, b[:])
		} else {
			s := bufio.NewScanner(counter)
			s.Buffer(make([]byte, 32768), 1<<20)
			for s.Scan() {
				if privateText.Match(s.Bytes()) {
					r.Close()
					return errors.New("private text in raw public archive")
				}
			}
			e = s.Err()
		}
		ce := r.Close()
		if e != nil || ce != nil || counter.N != a.Bytes || hex.EncodeToString(h.Sum(nil)) != a.SHA {
			return errors.New("archive content hash, CRC or size differs")
		}
		total += counter.N
		if total > 640<<20 {
			return errors.New("archive expanded byte cap exceeded")
		}
	}
	return save(out, map[string]any{"schema": "gooo/own-joint-feedback-publication-verification/v2", "status": "PASS", "public_payloads": len(m.Files), "raw_archive_members": len(z.File), "expanded_archive_bytes": total, "model_exports": 9, "optimizer_updates": 3600, "recorded_native_generations": 240, "recorded_go_executions": 240, "recorded_ordered_invocations": 3840, "new_model_predictions": 0, "new_native_calls": 0, "private_paths_and_credentials_absent_from_payload_text": true})
}
func get(client *http.Client, url, dest string, cap int64) error {
	request, e := http.NewRequest(http.MethodGet, url, nil)
	if e != nil {
		return e
	}
	response, e := client.Do(request)
	if e != nil {
		return e
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("anonymous immutable Hub GET failed")
	}
	if e = os.MkdirAll(filepath.Dir(dest), 0755); e != nil {
		return e
	}
	f, e := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if e != nil {
		return e
	}
	var b [32768]byte
	n, e := io.CopyBuffer(f, io.LimitReader(response.Body, cap+1), b[:])
	ce := f.Close()
	if e != nil {
		return e
	}
	if ce != nil {
		return ce
	}
	if n > cap {
		return errors.New("anonymous response byte cap exceeded")
	}
	return nil
}
func fetch(revision, dir, out string) error {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(revision) {
		return errors.New("immutable public commit required")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		return errors.New("fresh anonymous verification bundle required")
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	client := &http.Client{Timeout: 60 * time.Second}
	prefix := "https://huggingface.co/" + repository + "/resolve/" + revision + "/"
	if e := get(client, prefix+"publication-manifest.json", filepath.Join(dir, "publication-manifest.json"), 1<<20); e != nil {
		return e
	}
	var m manifest
	if e := read(filepath.Join(dir, "publication-manifest.json"), &m); e != nil {
		return e
	}
	required := requiredFiles()
	if len(m.Files) != len(required) {
		return errors.New("anonymous public allowlist differs")
	}
	seen := map[string]bool{}
	for _, a := range m.Files {
		if !required[a.Path] || seen[a.Path] || a.Bytes < 0 || a.Bytes > 64<<20 {
			return errors.New("anonymous bounded path differs")
		}
		seen[a.Path] = true
		if e := get(client, prefix+a.Path, filepath.Join(dir, a.Path), a.Bytes); e != nil {
			return e
		}
	}
	if e := verify(dir, out); e != nil {
		return e
	}
	return nil
}
