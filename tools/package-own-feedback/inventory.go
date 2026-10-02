package main

import (
	"fmt"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointcompositionstudy"
	"path/filepath"
)

func rawMembers() ([]member, error) {
	result := []member{}
	for _, n := range []string{"preexecution.json", "report.json", "independent-audit.json", "teacher-sessions.jsonl", "states.jsonl"} {
		result = append(result, member{"teacher/" + n, teacher + "/" + n})
	}
	for _, n := range []string{"preexecution.json", "report.json", "selection.json", "independent-audit.json", "independent-audit-strengthened.json"} {
		result = append(result, member{"sdk/" + n, sdk + "/" + n})
	}
	ids := []string{"offline", "reference-v1-independent", "reference-v1-joint"}
	for _, a := range arms {
		for _, v := range variants {
			ids = append(ids, a+"-"+v)
		}
	}
	for _, split := range []string{"calibration", "development"} {
		for _, id := range ids {
			n := split + "-" + id + ".jsonl"
			result = append(result, member{"sdk/" + n, sdk + "/" + n})
		}
	}
	for _, n := range []string{"preexecution.json", "report.json", "independent-audit.json"} {
		result = append(result, member{"native/" + n, native + "/" + n})
	}
	for _, family := range jointcompositionstudy.Families {
		for goal := 0; goal < 4; goal++ {
			for _, language := range []string{"en", "ko"} {
				for _, policy := range []string{"selected", "uniform-initial", "set-initial", "set-feedback", "offline"} {
					for _, suffix := range []string{".json", "-execution.json"} {
						n := fmt.Sprintf("%s-goal%d-%s-%s%s", family, goal, language, policy, suffix)
						result = append(result, member{"native/" + n, native + "/" + n})
					}
				}
			}
		}
	}
	failed := base + "native-main-attempt1-original-plan-20261002"
	result = append(result, member{"native-first-attempt/preexecution.json", failed + "/preexecution.json"})
	for _, policy := range []string{"selected", "uniform-initial", "set-initial", "set-feedback", "offline"} {
		n := "assignment_reference-goal0-en-" + policy + ".json"
		result = append(result, member{"native-first-attempt/" + n, failed + "/" + n})
		if policy != "offline" {
			n = "assignment_reference-goal0-en-" + policy + "-execution.json"
			result = append(result, member{"native-first-attempt/" + n, failed + "/" + n})
		}
	}
	for _, n := range []string{"manifest.json", "collection-attempt.json", "exports.jsonl", "preexecution.json", "dataset.jsonl", "audit.json"} {
		result = append(result, member{"original-curriculum/" + n, "runs/joint-composition-curriculum-20261002/" + n})
	}
	for _, arm := range []string{"joint", "independent"} {
		for _, n := range []string{"model.json", "weights.bin"} {
			result = append(result, member{"reference-models/" + arm + "/" + n, "models/joint-composition-v1/" + arm + "/models/fp32/" + n})
		}
	}
	for _, p := range ownSourceFiles() {
		result = append(result, member{"source/" + p, p})
	}
	for _, p := range runtimeOrigins() {
		result = append(result, member{"source/runtime/" + p, p})
	}
	for _, p := range []string{"go.mod", "training/train_own_joint_feedback_v2.py", "training/train_pilot_v2.py", "training/semantic_features_v3.py"} {
		result = append(result, member{"source/" + p, p})
	}
	for i := range result {
		result[i].Local = filepath.Clean(result[i].Local)
	}
	return result, nil
}
func ownSourceFiles() []string {
	return []string{
		"internal/jointcohort/load.go", "internal/jointcohort/prepare.go", "internal/jointcohort/prepare_test.go",
		"internal/jointcompositionstudy/fixture.go", "internal/jointcompositionstudy/fixture_test.go", "internal/jointcompositionstudy/natural.go", "internal/jointcompositionstudy/oracle.go",
		"internal/jointfeedback/records.go", "internal/jointfeedback/verify.go", "internal/jointfeedback/verify_test.go",
		"tools/audit-own-joint-feedback-models/main.go", "tools/audit-own-joint-feedback-models/source.go",
		"tools/collect-own-joint-feedback/audit.go", "tools/collect-own-joint-feedback/main.go", "tools/collect-own-joint-feedback/totals.go",
		"tools/own-joint-feedback-study/audit.go", "tools/own-joint-feedback-study/main.go", "tools/own-joint-feedback-study/main_test.go", "tools/own-joint-feedback-study/metrics.go", "tools/own-joint-feedback-study/models.go",
		"tools/own-joint-feedback-study/native.go", "tools/own-joint-feedback-study/native_capture.go", "tools/own-joint-feedback-study/native_offline_test.go", "tools/own-joint-feedback-study/native_process.go", "tools/own-joint-feedback-study/native_read.go", "tools/own-joint-feedback-study/native_test.go",
		"tools/own-joint-feedback-study/parts.go", "tools/own-joint-feedback-study/process_other.go", "tools/own-joint-feedback-study/process_unix.go", "tools/own-joint-feedback-study/process_unix_test.go", "tools/own-joint-feedback-study/reference_progress.go",
	}
}
func runtimeOrigins() []string {
	return []string{
		"LICENSE", "internal/bodyplan/bodyplan.go", "internal/bodyplan/bodyplan_test.go",
		"internal/decision/bridge.go", "internal/strictjson/decode.go", "internal/strictjson/decode_test.go",
		"internal/decision/ir.go", "internal/decision/ir_test.go",
		"internal/jointdecision/input.go", "internal/jointdecision/input_test.go", "internal/jointdecision/load.go",
		"internal/jointdecision/model.go", "internal/jointdecision/model_test.go",
		"internal/decision/model.go", "internal/decision/model_test.go", "internal/decision/path_features_test.go", "internal/decision/path_model_test.go",
		"internal/pathplan/diagnosis.go", "internal/pathplan/diagnosis_test.go",
		"internal/pathplan/feedback.go", "internal/pathplan/feedback_batches.go", "internal/pathplan/feedback_batches_test.go", "internal/pathplan/feedback_test.go",
		"internal/pathplan/joint.go", "internal/pathplan/joint_batches.go", "internal/pathplan/joint_feedback.go", "internal/pathplan/joint_test.go",
		"internal/pathplan/pathplan.go", "internal/pathplan/pathplan_test.go", "internal/pathplan/prepared.go", "internal/pathplan/prepared_test.go",
		"internal/pathplan/search.go", "internal/pathplan/search_test.go", "internal/pathplan/session.go", "internal/pathplan/session_test.go",
		"internal/pathplan/source_features.go", "internal/pathplan/source_features_test.go",
		"internal/decision/semantic_features.go", "internal/decision/semantic_features_test.go",
		"internal/decision/split_features.go", "internal/decision/split_features_test.go", "studies/split-context-features-v2/parity.json",
	}
}
