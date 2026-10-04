package main

import "encoding/json"

type resources struct {
	WallMS  float64 `json:"wall_ms"`
	UserS   float64 `json:"user_seconds"`
	SystemS float64 `json:"system_seconds"`
	MaxRSS  int64   `json:"maximum_resident_bytes"`
}

type summary struct {
	Mode             string    `json:"mode"`
	Stage            int       `json:"stage"`
	Replayed         bool      `json:"replayed"`
	Resources        resources `json:"resources"`
	GenerationMS     float64   `json:"generation_ms"`
	RuntimeMS        float64   `json:"runtime_ms"`
	ModelCalls       int       `json:"actual_model_calls"`
	StoredModelCalls int       `json:"stored_generation_model_calls"`
	PredictNS        int64     `json:"stored_prediction_ns"`
	Evaluated        int       `json:"evaluated_candidates"`
	Unattempted      int       `json:"unattempted_candidates"`
	SelectionPassed  int       `json:"selection_passed"`
	SelectionTotal   int       `json:"selection_total"`
	RuntimePassed    int       `json:"runtime_passed"`
	RuntimeTotal     int       `json:"runtime_total"`
	InputSlots       int       `json:"observed_input_slots"`
	Deliveries       int       `json:"observed_edge_deliveries"`
	Fields           int       `json:"observed_record_fields"`
	NativeRuns       int       `json:"native_runs"`
	GoooSHA          string    `json:"gooo_sha256"`
	GoSHA            string    `json:"generated_go_sha256"`
	DriverSHA        string    `json:"driver_sha256"`
}

type fieldType struct {
	Name   string `json:"name"`
	ID     string `json:"id"`
	GoName string `json:"go_name"`
}
type recordType struct {
	Name   string      `json:"name"`
	ID     string      `json:"id"`
	GoName string      `json:"go_name"`
	Fields []fieldType `json:"fields"`
}
type inputPlan struct {
	Port   string `json:"port"`
	Type   string `json:"type"`
	Entity string `json:"entity_id"`
	From   int    `json:"from"`
}
type activity struct {
	Name       string      `json:"name"`
	ID         string      `json:"id"`
	InputType  string      `json:"input_type"`
	OutputType string      `json:"output_type"`
	From       int         `json:"input_from"`
	Inputs     []inputPlan `json:"inputs"`
}
type fieldValue struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}
type inputValue struct {
	Port     string          `json:"port"`
	Entity   string          `json:"entity_id"`
	Producer string          `json:"producer_id"`
	Value    json.RawMessage `json:"value"`
	Fields   []fieldValue    `json:"fields"`
}
type delivery struct {
	Activity     string          `json:"activity_id"`
	Producer     string          `json:"producer_id"`
	Input        json.RawMessage `json:"input"`
	InputFields  []fieldValue    `json:"input_fields"`
	Actual       json.RawMessage `json:"actual"`
	ActualFields []fieldValue    `json:"actual_fields"`
	Expected     json.RawMessage `json:"expected"`
	Inputs       []inputValue    `json:"inputs"`
	Passed       *bool           `json:"passed"`
}
type envelope struct {
	GeneratedNow bool `json:"generated_now"`
	Composition  struct {
		Plan struct {
			Activities []activity   `json:"activities"`
			Records    []recordType `json:"record_types"`
		} `json:"plan"`
		Stage     string `json:"stage"`
		Failure   string `json:"failure"`
		Gooo      string `json:"gooo_source"`
		GoSHA     string `json:"generated_sha256"`
		DriverSHA string `json:"driver_sha256"`
		Elapsed   int64  `json:"elapsed_ns"`
		Steps     []struct {
			Generation struct {
				Report struct {
					Compiler string `json:"compiler_source_sha"`
					Paths    *struct {
						Search struct {
							Evaluated   int `json:"evaluated_candidates"`
							Unattempted int `json:"unattempted_combinations"`
							Selection   struct {
								Calls      int    `json:"local_model_predictions"`
								External   int    `json:"external_provider_calls"`
								Weights    string `json:"model_weights_sha256"`
								Metadata   string `json:"model_metadata_sha256"`
								Prediction *struct {
									NS    int64 `json:"predict_ns"`
									Calls int   `json:"actual_predictions"`
									Mask  int   `json:"proposed_mask"`
								} `json:"three_choice_prediction"`
							} `json:"selection"`
						} `json:"search"`
						Cases []struct {
							Passed bool `json:"passed"`
						} `json:"native_case_results"`
					} `json:"body_paths"`
				} `json:"report"`
			} `json:"generation"`
		} `json:"steps"`
	} `json:"composition"`
	Runtime struct {
		Source   string `json:"producer_source_sha"`
		Stage    string `json:"stage"`
		Failure  string `json:"failure"`
		Replayed bool   `json:"runtime_replayed"`
		Calls    int    `json:"model_calls"`
		Elapsed  int64  `json:"elapsed_ns"`
		Passed   int    `json:"finite_passed"`
		Total    int    `json:"finite_total"`
		Runs     []struct {
			Completed bool `json:"completed"`
			Exit      *int `json:"exit_code"`
		} `json:"runs"`
		Traces []struct {
			Deliveries []delivery `json:"deliveries"`
		} `json:"traces"`
	} `json:"runtime"`
}
