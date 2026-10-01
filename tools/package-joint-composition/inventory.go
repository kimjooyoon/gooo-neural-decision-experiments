package main

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
