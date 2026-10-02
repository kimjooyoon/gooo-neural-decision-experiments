package main

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
)

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}
type nativeResult struct {
	Source string `json:"source"`
	Report struct {
		Decision string `json:"decision"`
		Compiler string `json:"compiler_source_sha"`
		Types    bool   `json:"typecheck_passed"`
		Replay   bool   `json:"deterministic_replay"`
		Writes   int    `json:"repository_writes"`
		Paths    struct {
			Original string `json:"original_source_sha256"`
			Document string `json:"document_sha256"`
			Bound    bool   `json:"source_base_matched"`
			Binding  struct {
				Equivalent bool `json:"equivalent"`
			} `json:"source_binding"`
			Search       pathplan.SearchResult      `json:"search"`
			Progress     []pathplan.SessionProgress `json:"session_progress"`
			Feedback     []pathplan.FeedbackReceipt `json:"feedback_judgments"`
			Completeness float64                    `json:"finite_functional_completeness_percent"`
			Cases        []pathplan.TestResult      `json:"native_case_results"`
			Context      *struct {
				Schema   string `json:"schema"`
				Status   string `json:"status"`
				Feature  string `json:"feature_version"`
				Metadata string `json:"model_metadata_sha256"`
				Inputs   []struct {
					ID  string `json:"decision_id"`
					SHA string `json:"input_sha256"`
				} `json:"inputs"`
			} `json:"model_context"`
		} `json:"body_paths"`
	} `json:"report"`
}
