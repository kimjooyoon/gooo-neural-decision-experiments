package main

import (
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
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

func publicClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		h := r.URL.Hostname()
		if len(via) > 4 || r.URL.Scheme != "https" || r.URL.User != nil || !(h == "huggingface.co" || strings.HasSuffix(h, ".huggingface.co") || h == "hf.co" || strings.HasSuffix(h, ".hf.co")) {
			return errors.New("unexpected anonymous public redirect")
		}
		return nil
	}}
}

func get(c *http.Client, address string) (*http.Response, error) {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Host != "huggingface.co" {
		return nil, errors.New("public HF origin required")
	}
	r, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, errors.New("public request rejected")
	}
	response, err := c.Do(r)
	if err != nil {
		return nil, errors.New("anonymous HF fetch failed")
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		return nil, fmt.Errorf("anonymous HF status %d", response.StatusCode)
	}
	return response, nil
}

func jsonGet(c *http.Client, address string, value any) error {
	r, err := get(c, address)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(r.Body, 2<<20+1))
	if err != nil || len(raw) > 2<<20 {
		return errors.New("bounded public metadata required")
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}

func verifyHF(directory, revision, output string) error {
	if _, err := os.Lstat(output); !os.IsNotExist(err) {
		return errors.New("fresh anonymous public receipt required")
	}
	raw, err := os.ReadFile(filepath.Join(directory, "publication-manifest.json"))
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var m publication
	if err = json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.Schema != "gooo/three-choice-model-publication-allowlist/v1" || m.Repository != repository || m.Status != "STAGED_NOT_REMOTE_VERIFIED" || len(m.Files) != 34 || m.Bytes > 64<<20 {
		return errors.New("closed HF stage required")
	}
	c := publicClient()
	var meta struct {
		SHA      string `json:"sha"`
		Private  bool   `json:"private"`
		Siblings []struct {
			Name string `json:"rfilename"`
		} `json:"siblings"`
	}
	if revision == "" {
		if err = jsonGet(c, "https://huggingface.co/api/models/"+repository, &meta); err != nil {
			return err
		}
		revision = meta.SHA
	}
	if len(revision) != 40 || strings.Trim(revision, "0123456789abcdef") != "" {
		return errors.New("immutable public HF commit required")
	}
	if err = jsonGet(c, "https://huggingface.co/api/models/"+repository+"/revision/"+revision, &meta); err != nil {
		return err
	}
	if meta.Private || meta.SHA != revision {
		return errors.New("public exact HF revision required")
	}
	files := append([]entry{}, m.Files...)
	files = append(files, entry{"publication-manifest.json", sha(raw), int64(len(raw))})
	allowed := map[string]bool{".gitattributes": true}
	for _, p := range files {
		allowed[p.Path] = true
	}
	if len(meta.Siblings) < len(files) || len(meta.Siblings) > len(files)+1 {
		return errors.New("remote HF inventory size differs")
	}
	seen := map[string]bool{}
	for _, s := range meta.Siblings {
		if !allowed[s.Name] || seen[s.Name] {
			return errors.New("unexpected remote public file")
		}
		seen[s.Name] = true
	}
	var downloaded int64
	for _, p := range files {
		if !seen[p.Path] || p.Path != filepath.ToSlash(filepath.Clean(p.Path)) || strings.HasPrefix(p.Path, "../") || filepath.IsAbs(p.Path) || p.Bytes <= 0 || p.Bytes > 64<<20 {
			return errors.New("bounded expected remote member required")
		}
		local, err := threestudent.FilePin(filepath.Join(directory, filepath.FromSlash(p.Path)))
		if err != nil || local.SHA != p.SHA || local.Bytes != p.Bytes {
			return errors.New("local allowlisted source changed")
		}
		r, err := get(c, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+p.Path)
		if err != nil {
			return err
		}
		h := sha256.New()
		var buffer [32768]byte
		n, copyErr := io.CopyBuffer(h, io.LimitReader(r.Body, p.Bytes+1), buffer[:])
		closeErr := r.Body.Close()
		if copyErr != nil || closeErr != nil || n != p.Bytes || hex.EncodeToString(h.Sum(nil)) != p.SHA {
			return errors.New("actual anonymous remote bytes differ")
		}
		downloaded += n
	}
	receipt := map[string]any{"schema": "gooo/own-three-choice-hf-anonymous-byte-verification/v1", "status": "PASS", "repository": repository, "public_revision": revision, "private": false, "authentication_sent": false, "verified_upload_files_including_manifest": len(files), "downloaded_verified_bytes": downloaded, "models": 9, "new_optimizer_updates": 0, "new_model_predictions": 0, "new_native_calls": 0, "scope": "Anonymous HTTPS GET of every actual allowlisted model, complete evidence ZIP, source, result and manifest at one immutable public HF commit. All SHA-256 and sizes match the privacy-checked staged originals; closed remote inventory permits only the Hub-generated .gitattributes. This is publication evidence, not full SDK/native student completeness."}
	result, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, err = f.Write(append(result, '\n'))
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Printf("PASS: public HF %s revision %s, %d anonymous file hashes, %d bytes\n", repository, revision, len(files), downloaded)
	return nil
}
