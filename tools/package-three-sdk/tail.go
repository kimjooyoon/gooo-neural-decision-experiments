package main

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const tailPhase = "own-three-sdk-storage-tail-20261002"
const tailProducer = "59f921c520d369d58800ea5c9505f4ff60f1b307"
const tailRemotePrefix = "research/sdk-storage-tail-20261002"
const tailAmendment = "docs/own-three-choice-sdk-storage-continuation-preregistration-20261002.md"
const tailAmendmentSHA = "b127f62dc5c45694afc9703be713bd9db834bee0699f1c31614e62729a721e10"
const tailResults = "docs/own-three-choice-sdk-storage-tail-results-20261002.md"
const tailDesign = "docs/own-three-choice-sdk-storage-continuation-implementation-20261002.md"

func tailNames() []string {
	names := []string{"audit.json", "bilingual-audit.json", "protocol.md", "amendment.md", "design.md", "results.md"}
	for _, name := range []string{"collection-attempt.json", "preexecution.json", "original-prefix-audit.json", "development-set-initial-ptq_ternary.jsonl", "development-set-initial-qat_ternary.jsonl", "development-uniform-initial-fp32.jsonl", "development-uniform-initial-ptq_ternary.jsonl", "development-uniform-initial-qat_ternary.jsonl"} {
		names = append(names, tailPhase+"/"+name)
	}
	slices.Sort(names)
	return names
}
func validateTail(m manifest) error {
	if m.NativeRevision != "" || m.NativeCalls != 0 || m.GoCalls != 0 || m.Invocations != 0 {
		return errors.New("original SDK tail cannot claim native execution")
	}
	if m.Schema != "gooo/own-three-sdk-tail-public-bundle/v1" || m.Status != "PASS_WITH_SEPARATE_STORAGE_AMENDMENT" || m.Producer != tailProducer || m.Sessions != 2485 || m.Predictions != 9142 || m.Values != 148736 || m.ModelsRevision != baseHF || m.Protocol != threefeedback.ProtocolSHA || len(m.Files) != 14 || m.Bytes > 256<<20 || m.Archive.Bytes > 64<<20 {
		return errors.New("closed actual storage tail manifest differs")
	}
	names := tailNames()
	var total int64
	for i, f := range m.Files {
		if f.Name != names[i] || filepath.Clean(f.Name) != f.Name || strings.Contains(f.Name, "..") || len(f.Pin.SHA) != 64 || f.Pin.Bytes < 0 {
			return errors.New("closed sorted unique tail members required")
		}
		if f.Name == "amendment.md" && f.Pin.SHA != tailAmendmentSHA {
			return errors.New("separate resource amendment changed")
		}
		if f.Name == "protocol.md" && f.Pin.SHA != threefeedback.ProtocolSHA {
			return errors.New("original resource protocol changed")
		}
		total += f.Pin.Bytes
	}
	if total != m.Bytes {
		return errors.New("tail decoded byte sum differs")
	}
	return nil
}
func packTail(bundle, index, audit string) error {
	var proof struct {
		Schema       string                      `json:"schema"`
		Status       string                      `json:"status"`
		Source       string                      `json:"tail_producer_source_revision"`
		Sessions     int                         `json:"actual_unique_sdk_sessions"`
		TailSessions int                         `json:"tail_captured_sdk_sessions"`
		Predictions  int                         `json:"tail_model_predictions"`
		Values       int                         `json:"actual_independently_verified_ordered_values"`
		TailFiles    map[string]threestudent.Pin `json:"tail_phase_files"`
	}
	if err := read(audit, &proof); err != nil {
		return err
	}
	if proof.Schema != "gooo/own-three-sdk-combined-independent-audit/v1" || proof.Status != "PASS_WITH_SEPARATE_STORAGE_AMENDMENT" || proof.Source != tailProducer || proof.Sessions != 11264 || proof.TailSessions != 2485 || proof.Predictions != 9142 || proof.Values != 658992 || len(proof.TailFiles) != 8 {
		return errors.New("actual full combined independent audit required")
	}
	all := map[string]string{"audit.json": audit, "bilingual-audit.json": "publication/own-three-choice-sdk-bilingual-functional-audit-20261002.json", "protocol.md": threefeedback.Protocol, "amendment.md": tailAmendment, "design.md": tailDesign, "results.md": tailResults}
	root := filepath.Join("runs", tailPhase)
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 8 {
		return errors.New("complete closed original tail inventory required")
	}
	for _, d := range entries {
		name := filepath.Join(root, d.Name())
		actual, e := threestudent.FilePin(name)
		if e != nil || actual != proof.TailFiles[d.Name()] {
			return errors.New("retained actual tail bytes differ")
		}
		all[tailPhase+"/"+d.Name()] = name
	}
	var bilingual struct {
		Status       string `json:"status"`
		Observations int    `json:"actual_session_observations"`
		Calls        int    `json:"new_model_predictions"`
	}
	if err = read(all["bilingual-audit.json"], &bilingual); err != nil {
		return err
	}
	if bilingual.Status != "PASS" || bilingual.Observations != 11264 || bilingual.Calls != 0 {
		return errors.New("actual zero-prediction bilingual diagnostic required")
	}
	m := manifest{Schema: "gooo/own-three-sdk-tail-public-bundle/v1", Status: proof.Status, Producer: tailProducer, Sessions: 2485, Predictions: 9142, Values: 148736, ModelsRevision: baseHF, Protocol: threefeedback.ProtocolSHA, Scope: "Every original tail byte, combined independent 11264-session audit and post-hoc finite bilingual output diagnostic. Original cap-stopped prefix remains separately published at HF ba1f540ee5bf75fceab53d1446b9385c793a4396/research/sdk-prefix-20261002. Original failure is unchanged; this is a separate storage amendment, with unchanged models, no new training/native execution/default promotion."}
	f, err := os.OpenFile(bundle, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	z := zip.NewWriter(f)
	for _, name := range tailNames() {
		p, e := threestudent.FilePin(all[name])
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
		in, e := os.Open(all[name])
		if e != nil {
			z.Close()
			f.Close()
			return e
		}
		if e = privacy(in, name); e != nil {
			in.Close()
			z.Close()
			f.Close()
			return e
		}
		if _, e = in.Seek(0, io.SeekStart); e != nil {
			in.Close()
			z.Close()
			f.Close()
			return e
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(0644)
		out, e := z.CreateHeader(header)
		if e != nil {
			in.Close()
			z.Close()
			f.Close()
			return e
		}
		var buffer [32768]byte
		n, e := io.CopyBuffer(out, in, buffer[:])
		in.Close()
		if e != nil || n != p.Bytes {
			z.Close()
			f.Close()
			return errors.New("tail original copy differs")
		}
		m.Files = append(m.Files, member{name, p})
		m.Bytes += n
	}
	if err = z.Close(); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	m.Archive, err = threestudent.FilePin(bundle)
	if err != nil {
		return err
	}
	if err = validateTail(m); err != nil {
		return err
	}
	if err = save(index, m); err != nil {
		return err
	}
	return verifyWith(bundle, index, validateTail)
}
func stageTailHF(bundle, index, audit, directory string) error {
	if err := verifyWith(bundle, index, validateTail); err != nil {
		return err
	}
	all := map[string]string{"bundle.zip": bundle, "manifest.json": index, "audit.json": audit, "results.md": tailResults, "protocol.md": threefeedbackProtocol(), "design.md": tailDesign}
	m := appendix{Schema: "gooo/own-three-sdk-tail-hf-appendix/v1", Repository: repository, Prefix: tailRemotePrefix, Base: baseHF, Scope: "Append only actual missing tail evidence and full amended SDK reconciliation. Existing weights and failed prefix remain unchanged. No native/compiler-completion or new model-training claim."}
	return stageAppendix(directory, all, m)
}
