package main

import (
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threestudent"
)

const nativeRevision = "774eabb226f88a317c523ce4efa24a08032066a9"
const nativeSDK = "v0.2.13-experimental"

type nativePolicy struct {
	Name      string `json:"name"`
	Candidate string `json:"candidate"`
}

var nativePolicies = [5]nativePolicy{{"selected", "set-feedback/fp32"}, {"uniform-initial", "uniform-initial/fp32"}, {"set-initial", "set-initial/fp32"}, {"set-feedback", "set-feedback/fp32"}, {"offline", "offline"}}

type nativeDocument struct {
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
			Original       string `json:"original_source_sha256"`
			SelectedSource string `json:"selected_source_sha256"`
			Document       string `json:"document_sha256"`
			Tests          string `json:"test_suite_sha256"`
			Bound          bool   `json:"source_base_matched"`
			Binding        struct {
				Equivalent bool   `json:"equivalent"`
				Semantic   string `json:"source_semantic_digest"`
			} `json:"source_binding"`
			Search       pathplan.SearchResult      `json:"search"`
			Progress     []pathplan.SessionProgress `json:"session_progress"`
			Feedback     []pathplan.FeedbackReceipt `json:"feedback_judgments"`
			Unfixed      bool                       `json:"feedback_unfixed"`
			Completeness float64                    `json:"finite_functional_completeness_percent"`
			Cases        []pathplan.TestResult      `json:"native_case_results"`
			Context      *struct {
				Schema       string `json:"schema"`
				Status       string `json:"status"`
				Feature      string `json:"feature_version"`
				Metadata     string `json:"model_metadata_sha256"`
				Semantic     string `json:"source_semantic_sha256"`
				OriginalPlan string `json:"original_plan_sha256"`
				RankedPlan   string `json:"ranked_plan_sha256"`
				Inputs       []struct {
					ID       string `json:"decision_id"`
					SHA      string `json:"input_sha256"`
					Bytes    int    `json:"bytes"`
					Replaced bool   `json:"caller_prefix_replaced"`
				} `json:"inputs"`
				Declared *struct {
					Text      string `json:"text"`
					SHA       string `json:"sha256"`
					Bytes     int    `json:"bytes"`
					Decisions int    `json:"declared_decisions"`
				} `json:"complete_declared_inputs"`
			} `json:"model_context"`
			Timing struct {
				Plan      float64 `json:"plan_prepare_ms"`
				Binding   float64 `json:"source_binding_ms"`
				Load      float64 `json:"model_load_ms"`
				Context   float64 `json:"context_prepare_ms"`
				Search    float64 `json:"bounded_search_ms"`
				Emission  float64 `json:"final_emission_ms"`
				Total     float64 `json:"total_ms"`
				Execution string  `json:"execution_model"`
				Stage     string  `json:"decision_stage"`
			} `json:"timing"`
		} `json:"body_paths"`
	} `json:"report"`
}
type childMetrics struct {
	Started bool    `json:"actual_process_started"`
	Wall    int64   `json:"wall_ns"`
	User    int64   `json:"user_cpu_ns"`
	System  int64   `json:"system_cpu_ns"`
	RSS     int64   `json:"child_max_rss_bytes"`
	CPU     float64 `json:"cpu_percent_of_one_core"`
}
type nativeExecution struct {
	Schema        string       `json:"schema"`
	View          string       `json:"view_id"`
	Policy        nativePolicy `json:"policy"`
	Capture       string       `json:"raw_native_capture_sha256"`
	Source        string       `json:"emitted_go_sha256"`
	NativeCalled  bool         `json:"actual_native_process_called"`
	GoCalled      bool         `json:"actual_compile_and_run_called"`
	Executed      bool         `json:"actual_compiled_go_execution"`
	Values        []int64      `json:"ordered_actual_values"`
	GoStdout      string       `json:"actual_go_stdout"`
	Cases         int          `json:"case_denominator"`
	Codegen       childMetrics `json:"codegen_process_metrics"`
	Execution     childMetrics `json:"compile_and_run_process_metrics"`
	CodegenStderr string       `json:"actual_codegen_stderr"`
	GoStderr      string       `json:"actual_go_stderr"`
	Error         string       `json:"error"`
}
type nativePre struct {
	Schema        string                      `json:"schema"`
	Source        string                      `json:"source_revision"`
	Native        string                      `json:"native_revision"`
	SDK           string                      `json:"sdk"`
	Go            string                      `json:"go"`
	Protocol      string                      `json:"protocol_sha256"`
	Amendment     string                      `json:"storage_continuation_sha256"`
	Dataset       string                      `json:"dataset_sha256"`
	CombinedAudit threestudent.Pin            `json:"combined_sdk_audit_pin"`
	Selected      string                      `json:"calibration_selected_candidate"`
	Models        map[string]pin              `json:"model_pins"`
	Policies      [5]nativePolicy             `json:"policies"`
	IDs           []string                    `json:"ordered_native_view_ids"`
	Binary        threestudent.Pin            `json:"native_binary"`
	Worker        threestudent.Pin            `json:"native_body_worker_binary"`
	GoBinary      threestudent.Pin            `json:"go_binary"`
	Calls         int                         `json:"planned_native_generations"`
	Executions    int                         `json:"planned_compiled_go_executions"`
	Invocations   int                         `json:"planned_ordered_invocations"`
	Prior         map[string]threestudent.Pin `json:"prior_raw_files"`
	PriorBytes    int64                       `json:"prior_raw_bytes"`
	Cap           int64                       `json:"amended_whole_study_raw_cap_bytes"`
	Available     uint64                      `json:"available_disk_bytes_before_phase"`
	CI            bool                        `json:"ci_hint_supplied"`
}
