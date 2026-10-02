package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threefeedback"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const nativePhase = "own-three-native-execution-20261002"
const nativeProducer = "7184fe01327f491aad98027d97fc973e2f0ae645"
const nativeMain = "774eabb226f88a317c523ce4efa24a08032066a9"
const nativeRemotePrefix = "research/native-execution-20261002"
const nativeDesign = "docs/own-three-choice-native-execution-design-20261002.md"
const nativeResults = "docs/own-three-choice-native-execution-results-20261002.md"
const nativeProv = "publication/own-three-choice-native-execution-prov-20261002.jsonld"

// Replace only from the completed independent audit before publication.
const nativePredictions = 1760

func nativeNames() []string {
	names := []string{"amendment.md", "audit.json", "design.md", "protocol.md", "results.md", "prov.jsonld"}
	for _, n := range []string{"preexecution.json", "combined-sdk-audit.json", "report.json", "collection-attempt.json"} {
		names = append(names, nativePhase+"/"+n)
	}
	for _, family := range threecompositionstudy.Families {
		for goal := 0; goal < 8; goal++ {
			for _, language := range []string{"en", "ko"} {
				for _, policy := range []string{"selected", "uniform-initial", "set-initial", "set-feedback", "offline"} {
					name := fmt.Sprintf("%s-goal%d-%s-%s", family, goal, language, policy)
					names = append(names, nativePhase+"/"+name+"-native.json", nativePhase+"/"+name+"-execution.json")
				}
			}
		}
	}
	slices.Sort(names)
	return names
}
func validateNative(m manifest) error {
	names := nativeNames()
	if m.Schema != "gooo/own-three-native-public-bundle/v1" || m.Status != "PASS" || m.Producer != nativeProducer || m.NativeRevision != nativeMain || m.Sessions != 0 || m.NativeCalls != 640 || m.GoCalls != 640 || m.Invocations != 10240 || m.Predictions != nativePredictions || m.Values != 10240 || m.ModelsRevision != baseHF || m.Protocol != threefeedback.ProtocolSHA || len(m.Files) != len(names) || m.Bytes <= 0 || m.Bytes > 256<<20 || m.Archive.Bytes <= 0 || m.Archive.Bytes > 64<<20 {
		return errors.New("closed actual native generation/execution manifest differs")
	}
	var total int64
	for i, f := range m.Files {
		if f.Name != names[i] || filepath.Clean(f.Name) != f.Name || strings.Contains(f.Name, "..") || len(f.Pin.SHA) != 64 || strings.Trim(f.Pin.SHA, "0123456789abcdef") != "" || f.Pin.Bytes <= 0 || f.Pin.Bytes > 1<<20 {
			return errors.New("complete sorted closed native member required")
		}
		if f.Name == "protocol.md" && f.Pin.SHA != threefeedback.ProtocolSHA {
			return errors.New("original protocol changed")
		}
		if f.Name == "amendment.md" && f.Pin.SHA != tailAmendmentSHA {
			return errors.New("storage amendment changed")
		}
		total += f.Pin.Bytes
	}
	if total != m.Bytes {
		return errors.New("native decoded byte sum differs")
	}
	return nil
}
func packNative(bundle, index, audit string) error {
	var proof struct {
		Schema         string                      `json:"schema"`
		Status         string                      `json:"status"`
		Source         string                      `json:"producer_source_revision"`
		Native         string                      `json:"native_revision"`
		Calls          int                         `json:"actual_native_generations_audited"`
		Executions     int                         `json:"actual_verified_compiled_go_executions"`
		GoCalls        int                         `json:"actual_compile_and_run_calls_audited"`
		Values         int                         `json:"actual_independently_verified_ordered_invocations"`
		Predictions    int                         `json:"recorded_actual_model_predictions"`
		Known          bool                        `json:"all_native_model_prediction_counts_known"`
		NewNative      int                         `json:"new_native_calls"`
		NewGo          int                         `json:"new_go_execution_calls"`
		NewPredictions int                         `json:"new_model_predictions"`
		Updates        int                         `json:"new_optimizer_updates"`
		Promoted       bool                        `json:"default_model_promoted"`
		Files          map[string]threestudent.Pin `json:"native_phase_files"`
	}
	if err := read(audit, &proof); err != nil {
		return err
	}
	if proof.Schema != "gooo/own-three-native-independent-audit/v1" || proof.Status != "PASS" || proof.Source != nativeProducer || proof.Native != nativeMain || proof.Calls != 640 || proof.Executions != 640 || proof.GoCalls != 640 || proof.Values != 10240 || proof.Predictions != nativePredictions || !proof.Known || proof.NewNative != 0 || proof.NewGo != 0 || proof.NewPredictions != 0 || proof.Updates != 0 || proof.Promoted || len(proof.Files) != 1284 {
		return errors.New("full actual independently audited native phase required")
	}
	all := map[string]string{"amendment.md": tailAmendment, "audit.json": audit, "design.md": nativeDesign, "protocol.md": threefeedback.Protocol, "results.md": nativeResults, "prov.jsonld": nativeProv}
	phaseRoot := filepath.Join("runs", nativePhase)
	entries, err := os.ReadDir(phaseRoot)
	if err != nil || len(entries) != 1284 {
		return errors.New("closed complete native phase required")
	}
	for _, entry := range entries {
		name := filepath.Join(phaseRoot, entry.Name())
		actual, e := threestudent.FilePin(name)
		if e != nil || actual != proof.Files[entry.Name()] {
			return errors.New("actual native raw bytes differ")
		}
		all[nativePhase+"/"+entry.Name()] = name
	}
	names := nativeNames()
	if len(all) != len(names) {
		return errors.New("native appendix inventory differs")
	}
	m := manifest{Schema: "gooo/own-three-native-public-bundle/v1", Status: "PASS", Producer: nativeProducer, NativeRevision: nativeMain, NativeCalls: 640, GoCalls: 640, Invocations: 10240, Predictions: nativePredictions, Values: 10240, ModelsRevision: baseHF, Protocol: threefeedback.ProtocolSHA, Scope: "All original raw native stdout and independent compile-and-run receipts for five preplanned policies on 128 frozen config-20 bilingual views. Full source/input/SDK path/ordered output/resource audit; unchanged models and original storage failure. Finite completeness is not universal natural-language judgment or default-model promotion."}
	f, err := os.OpenFile(bundle, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	z := zip.NewWriter(f)
	for _, name := range names {
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
		h := sha256.New()
		var buffer [32768]byte
		n, e := io.CopyBuffer(io.MultiWriter(out, h), in, buffer[:])
		in.Close()
		if e != nil || n != p.Bytes || hex.EncodeToString(h.Sum(nil)) != p.SHA {
			z.Close()
			f.Close()
			return errors.New("native original bytes changed during packing")
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
	if err = validateNative(m); err != nil {
		return err
	}
	if err = save(index, m); err != nil {
		return err
	}
	return verifyWith(bundle, index, validateNative)
}
func stageNativeHF(bundle, index, audit, directory string) error {
	if err := verifyWith(bundle, index, validateNative); err != nil {
		return err
	}
	all := map[string]string{"bundle.zip": bundle, "manifest.json": index, "audit.json": audit, "results.md": nativeResults, "protocol.md": threefeedback.Protocol, "design.md": nativeDesign}
	m := appendix{Schema: "gooo/own-three-native-hf-appendix/v1", Repository: repository, Prefix: nativeRemotePrefix, Base: baseHF, Scope: "Append complete actual native generation and compiled-Go evidence, including full raw provenance inside the ZIP. Original models/source/training/SDK prefix/tail remain at their immutable revisions. No new training or default promotion."}
	return stageAppendix(directory, all, m)
}
