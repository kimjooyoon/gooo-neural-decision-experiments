package main

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecohort"
)

func anonymousClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second, CheckRedirect: func(r *http.Request, via []*http.Request) error {
		if len(via) > 3 || r.URL.Scheme != "https" || r.URL.Host != "raw.githubusercontent.com" {
			return errors.New("unexpected public download redirect")
		}
		return nil
	}}
}
func download(client *http.Client, url string, destination io.Writer, cap int64) error {
	r, err := client.Get(url)
	if err != nil {
		return errors.New("anonymous public request failed")
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK || r.ContentLength > cap {
		return errors.New("bounded public response required")
	}
	var buffer [32768]byte
	n, err := io.CopyBuffer(destination, io.LimitReader(r.Body, cap+1), buffer[:])
	if err != nil {
		return err
	}
	if n > cap {
		return errors.New("public download cap exceeded")
	}
	return nil
}
func verifyRemote(revision, manifestPath, destination, report string) error {
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(revision) {
		return errors.New("immutable public GitHub revision required")
	}
	if _, err := os.Lstat(report); !os.IsNotExist(err) {
		return errors.New("fresh public verification receipt required")
	}
	s, err := os.Lstat(manifestPath)
	if err != nil || !s.Mode().IsRegular() || s.Size() > lineCap {
		return errors.New("bounded regular trusted manifest required")
	}
	local, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	var m manifest
	if err = threecohort.Decode(local, &m); err != nil {
		return err
	}
	if err = validate(m); err != nil {
		return err
	}
	base := "https://raw.githubusercontent.com/kimjooyoon/gooo-neural-decision-experiments/" + revision + "/publication/"
	client := anonymousClient()
	metadata, err := os.CreateTemp(filepath.Dir(destination), "three-public-manifest-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(metadata.Name())
	if err = download(client, base+"own-three-choice-curriculum-bundle-20261002.json", metadata, lineCap); err != nil {
		metadata.Close()
		return err
	}
	if err = metadata.Close(); err != nil {
		return err
	}
	remoteManifest, err := fileHash(metadata.Name())
	if err != nil || remoteManifest.SHA != threecohort.SHA(local) {
		return errors.New("anonymous public manifest bytes differ")
	}
	archive, err := os.CreateTemp(filepath.Dir(destination), "three-public-evidence-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(archive.Name())
	if err = download(client, base+"own-three-choice-curriculum-20261002.zip", archive, archiveCap); err != nil {
		archive.Close()
		return err
	}
	if err = archive.Sync(); err != nil {
		archive.Close()
		return err
	}
	if err = archive.Close(); err != nil {
		return err
	}
	if err = verify(archive.Name(), manifestPath, destination); err != nil {
		return err
	}
	return writeJSON(report, map[string]any{"schema": "gooo/three-choice-source-teacher-anonymous-byte-verification/v1", "status": "PASS", "public_source_revision": revision, "public_manifest_sha256": remoteManifest.SHA, "public_archive_sha256": m.ArchiveSHA, "public_archive_bytes": m.ArchiveBytes, "decoded_member_count": len(m.Files), "decoded_bytes": m.DecodedBytes, "authentication_supplied": false, "all_decoded_sha_crc_and_privacy_checks": true, "new_model_predictions": 0, "new_native_calls": 0, "new_optimizer_updates": 0, "scope": "Anonymous immutable public GitHub manifest/archive and decoded member bytes verified, including full source/teacher raw evidence. This transport/byte receipt alone is not a new semantic audit, model training or trained native study."})
}
