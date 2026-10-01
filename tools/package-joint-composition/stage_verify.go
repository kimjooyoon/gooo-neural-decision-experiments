package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
)

func stagePaths() map[string]bool {
	paths := publicPaths()
	for _, name := range []string{"native-report.json", "native-preexecution.json", "native-independent-audit.json", "raw-evidence.zip"} {
		delete(paths, name)
	}
	for _, arm := range []string{"independent", "joint"} {
		for _, variant := range []string{"fp32", "ptq_ternary", "qat_ternary"} {
			delete(paths, "provenance/"+arm+"-"+variant+".ttl")
		}
	}
	return paths
}

func verifyStage(bundle, revision, output string) error {
	if !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(revision) {
		return errors.New("immutable intermediate revision required")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		return errors.New("fresh verification output required")
	}
	raw, err := os.ReadFile(filepath.Join(bundle, "publication-manifest.json"))
	if err != nil {
		return err
	}
	if err = decision.RejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	var m struct {
		Schema     string     `json:"schema"`
		Status     string     `json:"status"`
		Repository string     `json:"repository"`
		Files      []artifact `json:"files"`
		Models     int        `json:"model_exports"`
		Updates    int        `json:"optimizer_updates"`
		Calls      int        `json:"actual_sdk_predictions_all_stages"`
		Views      int        `json:"frozen_sdk_function_observations"`
		Native     bool       `json:"native_main_execution_claimed"`
	}
	if err = json.Unmarshal(raw, &m); err != nil {
		return err
	}
	if m.Schema != "gooo/own-joint-composition-intermediate-publication/v1" || m.Status != "SDK_VERIFIED_NATIVE_PENDING" || m.Repository != repository || m.Models != 6 || m.Updates != 480 || m.Calls != 25471 || m.Views != 5376 || m.Native || len(m.Files) != 28 {
		return errors.New("fixed intermediate scope differs")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	apiRaw, err := publicGet(client, "https://huggingface.co/api/models/"+repository+"/revision/"+revision)
	if err != nil {
		return err
	}
	var api struct {
		Private bool   `json:"private"`
		SHA     string `json:"sha"`
	}
	if err = json.Unmarshal(apiRaw, &api); err != nil || api.Private || api.SHA != revision {
		return errors.New("public immutable stage differs")
	}
	allow, seen := stagePaths(), map[string]bool{}
	for _, f := range m.Files {
		if !allow[f.Path] || seen[f.Path] {
			return errors.New("fixed stage path differs")
		}
		seen[f.Path] = true
		local, e := hashFile(filepath.Join(bundle, f.Path))
		if e != nil || local.SHA != f.SHA || local.Bytes != f.Bytes {
			return errors.New("local stage bytes differ")
		}
		remote, e := publicGet(client, "https://huggingface.co/"+repository+"/resolve/"+revision+"/"+f.Path+"?download=true")
		if e != nil {
			return e
		}
		if digest(remote) != f.SHA || int64(len(remote)) != f.Bytes {
			return errors.New("anonymous stage bytes differ")
		}
	}
	remote, err := publicGet(client, "https://huggingface.co/"+repository+"/resolve/"+revision+"/publication-manifest.json?download=true")
	if err != nil {
		return err
	}
	if !bytes.Equal(raw, remote) {
		return errors.New("stage manifest bytes differ")
	}
	return save(output, map[string]any{"schema": "gooo/own-joint-composition-intermediate-public-verification/v1", "status": "PASS", "repository": repository, "revision": revision, "manifest_sha256": digest(raw), "payload_files_verified": 28, "manifest_files_verified": 1, "anonymous_api_gets": 1, "anonymous_payload_gets": 29, "private": false, "credentials_sent": false, "model_exports": 6, "recorded_optimizer_updates": 480, "recorded_sdk_predictions_all_stages": 25471, "native_main_execution_claimed": false, "new_model_predictions": 0, "new_optimizer_updates": 0, "scope": "Anonymous immutable intermediate bytes only; SDK results are independently measured and native-main evidence remains pending."})
}
