package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

type appendix struct {
	Schema     string   `json:"schema"`
	Repository string   `json:"repository"`
	Prefix     string   `json:"remote_prefix"`
	Base       string   `json:"unchanged_models_base_revision"`
	Files      []member `json:"files"`
	Scope      string   `json:"scope"`
}

func stageHF(bundle, index, audit, directory string) error {
	if err := verify(bundle, index); err != nil {
		return err
	}
	if _, err := os.Lstat(directory); !os.IsNotExist(err) {
		return errors.New("fresh closed HF appendix stage required")
	}
	if err := os.Mkdir(directory, 0755); err != nil {
		return err
	}
	all := map[string]string{"bundle.zip": bundle, "manifest.json": index, "audit.json": audit, "results.md": "docs/own-three-choice-sdk-prefix-results-20261002.md", "protocol.md": threefeedbackProtocol(), "design.md": "docs/own-three-choice-sdk-design-20261002.md"}
	m := appendix{Schema: "gooo/own-three-sdk-prefix-hf-appendix/v1", Repository: repository, Prefix: remotePrefix, Base: baseHF, Scope: "Appendix only: exact cap-stopped SDK prefix, unchanged nine own models and frozen reference at the existing immutable base. No weight replacement, complete-comparison claim or compiler promotion."}
	names := []string{"audit.json", "bundle.zip", "design.md", "manifest.json", "protocol.md", "results.md"}
	for _, name := range names {
		in, err := os.Open(all[name])
		if err != nil {
			return err
		}
		if name != "bundle.zip" {
			if err = privacy(in, name); err != nil {
				in.Close()
				return err
			}
			if _, err = in.Seek(0, io.SeekStart); err != nil {
				in.Close()
				return err
			}
		}
		out, err := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			in.Close()
			return err
		}
		var buffer [32768]byte
		_, err = io.CopyBuffer(out, in, buffer[:])
		in.Close()
		out.Close()
		if err != nil {
			return err
		}
		p, err := threestudent.FilePin(filepath.Join(directory, name))
		if err != nil {
			return err
		}
		original, err := threestudent.FilePin(all[name])
		if err != nil || p != original {
			return errors.New("HF appendix copy differs")
		}
		m.Files = append(m.Files, member{name, p})
	}
	return save(filepath.Join(directory, "publication-manifest.json"), m)
}
func threefeedbackProtocol() string {
	return "docs/own-three-choice-completeness-preregistration-20261002.md"
}
func publicGet(c *http.Client, address string) (*http.Response, error) {
	r, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	response, err := c.Do(r)
	if err != nil {
		return nil, errors.New("anonymous public HF request failed")
	}
	if response.StatusCode != 200 {
		response.Body.Close()
		return nil, errors.New("anonymous public HF status differs")
	}
	return response, nil
}
func verifyHF(directory, revision, output string) error {
	if len(revision) != 40 || strings.Trim(revision, "0123456789abcdef") != "" {
		return errors.New("exact immutable HF revision required")
	}
	var m appendix
	if err := read(filepath.Join(directory, "publication-manifest.json"), &m); err != nil {
		return err
	}
	if m.Schema != "gooo/own-three-sdk-prefix-hf-appendix/v1" || m.Repository != repository || m.Prefix != remotePrefix || m.Base != baseHF || len(m.Files) != 6 {
		return errors.New("closed staged appendix differs")
	}
	self, err := threestudent.FilePin(filepath.Join(directory, "publication-manifest.json"))
	if err != nil {
		return err
	}
	files := append(append([]member{}, m.Files...), member{"publication-manifest.json", self})
	c := &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		host := r.URL.Hostname()
		if len(via) > 4 || r.URL.Scheme != "https" || r.URL.User != nil || !(host == "huggingface.co" || strings.HasSuffix(host, ".huggingface.co") || host == "hf.co" || strings.HasSuffix(host, ".hf.co")) {
			return errors.New("public HF redirect rejected")
		}
		return nil
	}}
	meta, err := publicGet(c, "https://huggingface.co/api/models/"+repository+"/revision/"+revision)
	if err != nil {
		return err
	}
	raw, err := io.ReadAll(io.LimitReader(meta.Body, 1<<20+1))
	meta.Body.Close()
	if err != nil || len(raw) > 1<<20 {
		return errors.New("bounded public HF metadata required")
	}
	var info struct {
		SHA      string `json:"sha"`
		Private  bool   `json:"private"`
		Siblings []struct {
			Name string `json:"rfilename"`
		} `json:"siblings"`
	}
	if err = readJSON(raw, &info); err != nil {
		return err
	}
	if info.SHA != revision || info.Private {
		return errors.New("exact public HF revision required")
	}
	allowed := map[string]bool{}
	for _, f := range files {
		allowed[remotePrefix+"/"+f.Name] = true
	}
	count := 0
	for _, f := range info.Siblings {
		if strings.HasPrefix(f.Name, remotePrefix+"/") {
			if !allowed[f.Name] {
				return errors.New("unknown remote appendix member")
			}
			count++
		}
	}
	if count != 7 {
		return errors.New("complete seven-file public appendix required")
	}
	var total int64
	for _, f := range files {
		local, err := threestudent.FilePin(filepath.Join(directory, f.Name))
		if err != nil || local != f.Pin || filepath.Base(f.Name) != f.Name || f.Pin.Bytes > 64<<20 {
			return errors.New("closed local appendix bytes changed")
		}
		r, err := publicGet(c, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+remotePrefix+"/"+f.Name)
		if err != nil {
			return err
		}
		h := sha256.New()
		var buffer [32768]byte
		n, err := io.CopyBuffer(h, io.LimitReader(r.Body, f.Pin.Bytes+1), buffer[:])
		r.Body.Close()
		if err != nil || n != f.Pin.Bytes || hex.EncodeToString(h.Sum(nil)) != f.Pin.SHA {
			return errors.New("actual anonymous public appendix bytes differ")
		}
		total += n
	}
	return save(output, map[string]any{"schema": "gooo/own-three-sdk-prefix-hf-byte-verification/v1", "status": "PASS", "repository": repository, "revision": revision, "remote_prefix": remotePrefix, "unchanged_model_base_revision": baseHF, "public": true, "authenticated_requests": 0, "actual_files_verified": 7, "actual_bytes_verified": total, "new_model_predictions": 0, "new_optimizer_updates": 0, "native_calls": 0, "scope": "All seven exact appended public files downloaded anonymously at the immutable revision and matched byte-for-byte. Original model/source/training artifacts retain their previously verified immutable base; no full SDK or native completion claim."})
}
func readJSON(raw []byte, value any) error {
	if err := publicJSON(raw); err != nil {
		return err
	}
	return json.Unmarshal(raw, value)
}
