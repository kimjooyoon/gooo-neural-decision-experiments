package main

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/decision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/jointdecision"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/pathplan"
	"github.com/kimjooyoon/gooo-neural-decision-experiments/internal/threecompositionstudy"
)

type document struct {
	Schema string              `json:"schema"`
	Plan   pathplan.Plan       `json:"path_plan"`
	Cases  []pathplan.TestCase `json:"test_cases"`
	Max    int                 `json:"max_attempts"`
}

type input struct {
	ID             string `json:"decision_id"`
	Text           string `json:"text"`
	SHA            string `json:"input_sha256"`
	Natural        string `json:"natural_intent_sha256"`
	Original       string `json:"original_intent_sha256"`
	SourceFeatures string `json:"source_features_sha256"`
	Bytes          int    `json:"bytes"`
	Replaced       bool   `json:"caller_prefix_replaced"`
}

type export struct {
	Schema   string `json:"schema"`
	Source   string `json:"original_source_sha256"`
	Document string `json:"document_sha256"`
	TestsSHA string `json:"test_suite_sha256"`
	Binding  struct {
		Equivalent bool   `json:"equivalent"`
		Semantic   string `json:"source_semantic_digest"`
	} `json:"source_binding"`
	Context struct {
		Schema   string  `json:"schema"`
		Status   string  `json:"status"`
		Feature  string  `json:"feature_version"`
		Original string  `json:"original_plan_sha256"`
		Semantic string  `json:"source_semantic_sha256"`
		Metadata string  `json:"model_metadata_sha256"`
		Inputs   []input `json:"inputs"`
	} `json:"context"`
	Inputs      []input `json:"inputs"`
	Predictions *int    `json:"model_predictions"`
	Tests       *int    `json:"candidate_tests"`
	Emission    *bool   `json:"selected_emission"`
	Writes      *int    `json:"repository_writes"`
}

func fixture(family string, config, goal int, language string) (document, []byte, threecompositionstudy.FiniteTarget, error) {
	plan, err := threecompositionstudy.Fixture(family, config, goal, language)
	if err != nil {
		return document{}, nil, threecompositionstudy.FiniteTarget{}, err
	}
	target, err := threecompositionstudy.Target(plan, family, config, goal)
	if err != nil {
		return document{}, nil, target, err
	}
	cases, err := threecompositionstudy.Cases(family, config, goal)
	if err != nil {
		return document{}, nil, target, err
	}
	prepared, err := pathplan.Prepare(plan)
	if err != nil {
		return document{}, nil, target, err
	}
	body, err := json.Marshal(prepared.Fallback().GoooBody())
	if err != nil {
		return document{}, nil, target, err
	}
	source := []byte("package threecomposition\nnamespace threecomposition\nentity Integer id \"threecomposition://entity/integer\"\nactivity ChoosePath(Integer) -> Integer computes " + string(body) + "\n")
	return document{"gooo/body-codegen-typed-path-plan/v1", plan, cases, 8}, source, target, nil
}

func inspect(raw []byte, doc document, source []byte) (export, string, error) {
	var value export
	if len(raw) > 1<<20 || privateText.Match(raw) || decision.RejectDuplicateJSONKeys(raw) != nil ||
		json.Unmarshal(raw, &value) != nil {
		return value, "", errors.New("bounded public nonduplicate native JSON required")
	}
	prepared, err := pathplan.Prepare(doc.Plan)
	if err != nil {
		return value, "", err
	}
	docRaw, err := json.Marshal(doc)
	if err != nil {
		return value, "", err
	}
	casesRaw, err := json.Marshal(doc.Cases)
	if err != nil {
		return value, "", err
	}
	if value.Schema != "gooo/compiler-path-input-export/v2" || value.Source != "sha256:"+hash(source) ||
		value.Document != "sha256:"+hash(docRaw) || value.TestsSHA != "sha256:"+hash(casesRaw) ||
		!value.Binding.Equivalent || value.Binding.Semantic == "" || value.Context.Semantic != value.Binding.Semantic ||
		value.Context.Schema != "gooo/compiler-typed-path-context/v3" || value.Context.Status != "ENCODED" ||
		value.Context.Feature != decision.SemanticContextIntentFeatureVersion || value.Context.Metadata != "" ||
		value.Context.Original != prepared.PlanSHA256() || len(value.Inputs) != 3 ||
		!reflect.DeepEqual(value.Inputs, value.Context.Inputs) || value.Predictions == nil || *value.Predictions != 0 ||
		value.Tests == nil || *value.Tests != 0 || value.Writes == nil || *value.Writes != 0 ||
		value.Emission == nil || *value.Emission {
		return value, "", errors.New("complete source-bound zero-prediction native export differs")
	}
	var parts [3]string
	for coordinate, in := range value.Inputs {
		choice := doc.Plan.Decisions[coordinate]
		_, natural, found := strings.Cut(choice.Intent, "intent: ")
		fields, err := prepared.SourceFeatures(choice.ID)
		if err != nil || !found {
			return value, "", errors.New("source facts or natural instruction missing")
		}
		text, err := decision.EncodeSemanticContextInput(fields, natural)
		if err != nil || in.Text != text || in.ID != choice.ID || in.SHA != "sha256:"+hash([]byte(text)) ||
			in.Bytes != len(text) || in.Natural != "sha256:"+hash([]byte(natural)) ||
			in.Original != "sha256:"+hash([]byte(choice.Intent)) || in.SourceFeatures != "sha256:"+hash(fields[:]) || !in.Replaced {
			return value, "", errors.New("complete ordered native input or hash differs")
		}
		parts[coordinate] = text
	}
	text, err := jointdecision.EncodeThree(parts)
	return value, text, err
}
