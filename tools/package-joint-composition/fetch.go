package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

// Retrieval stays anonymous and pinned; it executes no downloaded code.
func fetchBundle(bundle, revision, manifestSHA string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) || !regexp.MustCompile(`^[a-f0-9]{64}$`).MatchString(manifestSHA) {
		return errors.New("immutable revision and manifest digest required")
	}
	if _, err := os.Stat(bundle); !os.IsNotExist(err) {
		return errors.New("fresh download directory required")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	base := "https://huggingface.co/" + repository + "/resolve/" + revision + "/"
	raw, err := publicGet(client, base+"publication-manifest.json?download=true")
	if err != nil {
		return err
	}
	if digest(raw) != manifestSHA {
		return errors.New("pinned publication manifest differs")
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var manifest struct {
		Repository string     `json:"repository"`
		Files      []artifact `json:"files"`
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	allow := publicPaths()
	if manifest.Repository != repository || len(manifest.Files) != len(allow) {
		return errors.New("fixed public payload inventory differs")
	}
	seen := map[string]bool{}
	for _, entry := range manifest.Files {
		if !allow[entry.Path] || seen[entry.Path] || entry.Bytes < 0 || entry.Bytes > 32<<20 {
			return errors.New("fixed download path/size differs")
		}
		seen[entry.Path] = true
	}
	if err = os.MkdirAll(bundle, 0700); err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(bundle, "publication-manifest.json"), raw, 0600); err != nil {
		return err
	}
	for _, entry := range manifest.Files {
		raw, err = publicGet(client, base+entry.Path+"?download=true")
		if err != nil {
			return err
		}
		if digest(raw) != entry.SHA || int64(len(raw)) != entry.Bytes {
			return errors.New("download payload digest/size differs")
		}
		name := filepath.Join(bundle, entry.Path)
		if err = os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return err
		}
		if err = os.WriteFile(name, raw, 0600); err != nil {
			return err
		}
	}
	return nil
}

func extractEvidence(bundle, output string) error {
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh fixed evidence directory required")
	}
	var manifest struct {
		Members []artifact `json:"archive_members"`
	}
	raw, err := os.ReadFile(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	raw, err = os.ReadFile(filepath.Join(bundle, "raw-evidence.zip"))
	if err != nil {
		return err
	}
	if _, err = verifyArchive(raw, manifest.Members); err != nil {
		return err
	}
	z, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return err
	}
	if err = os.MkdirAll(output, 0700); err != nil {
		return err
	}
	for _, entry := range z.File {
		name := filepath.Join(output, entry.Name)
		if err = os.MkdirAll(filepath.Dir(name), 0700); err != nil {
			return err
		}
		f, e := os.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		r, e := entry.Open()
		if e != nil {
			f.Close()
			return e
		}
		var buffer [32768]byte
		_, copyErr := io.CopyBuffer(f, io.LimitReader(r, (32<<20)+1), buffer[:])
		rErr, fErr := r.Close(), f.Close()
		if copyErr != nil {
			return copyErr
		}
		if rErr != nil {
			return rErr
		}
		if fErr != nil {
			return fErr
		}
	}
	return nil
}
